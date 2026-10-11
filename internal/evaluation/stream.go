package evaluation

import (
	"context"
	"strings"
	"sync"

	"github.com/mousybusiness/mouse_whisperer/internal/asr"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/debuglog"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/symbols"
	"github.com/mousybusiness/mouse_whisperer/pkg/transcribe"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	"github.com/mousybusiness/mouse_whisperer/types"
	log "github.com/sirupsen/logrus"
)

// StreamConfig tunes streaming evaluation.
type StreamConfig struct {
	// StabilityTicks is how many consecutive ticks a token must be
	// unchanged before commands act on it
	StabilityTicks int
	// ProseSalvageMaxTokens caps how short a prose-flipped utterance must
	// be to still get the batch fuzzy/smart-correct/OCR treatment
	ProseSalvageMaxTokens int
	// DictationEnabled turns unmatched speech into the dictation buffer
	// instead of an error
	DictationEnabled bool
	// AccurateDictationFinals re-transcribes dictation finals with the
	// accurate model before they enter the buffer or match control words.
	// Ticks and command chains stay on the fast model.
	AccurateDictationFinals bool
	// AccurateCommandFinals re-transcribes a Final with the accurate model
	// when it fires a trailer command (click/move/select/scan/type): the
	// trailer feeds screen matching, where a fast-model mishearing acts on
	// the wrong thing. Plain command chains stay on the fast model.
	AccurateCommandFinals bool
}

// batchEvaluator is the slice of *Evaluator the streaming path reuses
type batchEvaluator interface {
	EvaluateResultHandled(transcription transcribe.Result, allowSmartCorrection bool) (bool, error)
	EvaluateTranscription(transcription transcribe.Result, allowSmartCorrection bool)
}

type sessionMode int

const (
	modeCommand sessionMode = iota // firing command segments as they stabilize
	modeProse                      // utterance stopped parsing: buffer it
	modeSticky                     // dictation buffer non-empty: append everything
	modeMuted                      // not_listening: defer to the batch gate
)

type streamSession struct {
	id          uint64
	mode        sessionMode
	tokens      []token
	stable      []int // consecutive ticks token i has been unchanged
	firedTokens int   // tokens consumed by executed segments (locked)
	firedSteps  int
	// firedClean is what the locked tokens said when they fired, so a
	// whisper rewrite INSIDE the locked prefix ("up up" → "up-up") is
	// detectable: positions past it can't be trusted (prefixIntact)
	firedClean []string
}

// StreamEvaluator consumes ASR partials and executes command chains on the
// fly ("up down down left right" fires each command as its successor
// confirms the chain), flipping whole utterances to dictation the moment
// they stop parsing as commands.
type StreamEvaluator struct {
	lookup   *commands.CommandLookup
	executor commandExecutor
	state    *state.State
	batch    batchEvaluator
	cfg      StreamConfig

	sess *streamSession

	// Execution is serialized on a worker so a blocking handler (voice
	// prompt, smart correction) never stalls partial consumption; drain
	// is the session-start barrier that keeps buffer state race-free.
	execCh  chan func()
	pending sync.WaitGroup
	queue   func(func())
	drain   func()

	// Overlay hooks, injectable for tests
	sendDictation    func(text, pending string)
	dismissDictation func()
	sendUI           func(string)
	reTranscribe     func(session mic.RecordingSession, prompt string) (string, error)

	// OnUtteranceStart fires once per streaming session, on its first
	// partial — used to prewarm external scanners (screend) while the user
	// is still speaking. Nil-safe.
	OnUtteranceStart func()
}

// NewStreamEvaluator wires the streaming evaluator on top of the batch
// evaluator's lookup, executor and state.
func NewStreamEvaluator(eval *Evaluator, cfg StreamConfig) *StreamEvaluator {
	if cfg.StabilityTicks <= 0 {
		cfg.StabilityTicks = 2
	}
	se := &StreamEvaluator{
		lookup:           eval.lookup,
		executor:         eval.executor,
		state:            eval.state,
		batch:            eval,
		cfg:              cfg,
		execCh:           make(chan func(), 64),
		dismissDictation: func() { overlay.DismissDictation() },
		sendUI:           func(m string) { overlay.SendUIMessage(m) },
	}
	hints := controlHints(eval.lookup)
	se.sendDictation = func(t, p string) { overlay.SendDictation(t, p, hints) }
	se.reTranscribe = func(session mic.RecordingSession, prompt string) (string, error) {
		return transcribe.File(session.FileName, prompt, mic.ModelAccurate)
	}
	se.queue = se.enqueue
	se.drain = se.pending.Wait
	return se
}

// commandFinalPrompt biases whisper toward command-plus-target utterances:
// trailers are usually code identifiers, not prose words
const commandFinalPrompt = "Click submitHandler. Move batch_size. Select main.go. Scan README."

// accurateFinal re-transcribes the utterance with the accurate model,
// falling back to the fast-model text on any failure. Runs on the exec
// worker: blocking here is the accepted cost of better dictation.
func (se *StreamEvaluator) accurateFinal(session mic.RecordingSession, fastText string) string {
	if !se.cfg.AccurateDictationFinals {
		return fastText
	}
	better, err := se.reTranscribe(session, transcribe.ProsePrompt)
	if err != nil || strings.TrimSpace(better) == "" {
		if err != nil && err.Error() != "noise" {
			log.Warnf("stream: accurate final failed, keeping fast text: %v", err)
		}
		return fastText
	}
	return strings.TrimSpace(better)
}

// controlHints derives the spoken control commands available while
// dictating from config, so the panel's help stays truthful as commands
// are added: any command whose dictating-state-active actions are all
// control methods qualifies.
func controlHints(lookup *commands.CommandLookup) []string {
	dictating := map[types.ExecutionState]bool{state.StateDictating: true}
	var hints []string
	for _, category := range lookup.GetCategoryOrder() {
		for _, entry := range lookup.GetOrderedCommandsForCategory(category) {
			if len(entry.Command.Triggers) == 0 || !entry.Command.IsActiveInState(dictating) {
				continue
			}
			control := 0
			ok := true
			for _, a := range entry.Command.Actions {
				if !a.IsActiveInState(dictating) {
					continue
				}
				if _, isControl := controlMethods[a.Method]; !isControl {
					ok = false
					break
				}
				control++
			}
			if ok && control > 0 {
				hints = append(hints, entry.Command.Triggers[0])
			}
		}
	}
	return hints
}

// Run consumes partials until the context ends or the channel closes.
func (se *StreamEvaluator) Run(ctx context.Context, partials <-chan asr.Partial) {
	go func() {
		for fn := range se.execCh {
			fn()
			se.pending.Done()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			close(se.execCh)
			return
		case p, ok := <-partials:
			if !ok {
				close(se.execCh)
				return
			}
			se.handlePartial(p)
		}
	}
}

func (se *StreamEvaluator) enqueue(fn func()) {
	se.pending.Add(1)
	se.execCh <- fn
}

func (se *StreamEvaluator) handlePartial(p asr.Partial) {
	if se.sess == nil || se.sess.id != p.Session.ID {
		// Barrier: buffer mutations and executions from the previous
		// session must land before this session's mode is decided
		se.drain()
		se.sess = se.newSession(p)
	}
	s := se.sess

	if p.Final {
		defer func() { se.sess = nil }()
	}

	// Empty-text Final is the abort convention: drop the session
	if p.Final && strings.TrimSpace(p.Text) == "" {
		if s.mode == modeProse || s.mode == modeSticky {
			se.queue(se.refreshPanel)
		}
		return
	}

	s.update(normalizeStream(p.Text))

	switch s.mode {
	case modeMuted:
		if p.Final {
			se.queue(func() { se.batch.EvaluateTranscription(result(p), true) })
		}
	case modeSticky:
		if p.Final {
			se.finalSticky(p)
		} else {
			se.sendDictation(se.state.Dictation.Text(), strings.TrimSpace(p.Text))
		}
	case modeProse:
		if p.Final {
			se.finalProse(p)
		} else {
			se.sendDictation(se.state.Dictation.Text(), s.remainderText(p))
		}
	case modeCommand:
		se.evalCommand(p)
	}
}

func (se *StreamEvaluator) newSession(p asr.Partial) *streamSession {
	se.state.SetRecordingStart(p.Session)
	if se.OnUtteranceStart != nil {
		se.OnUtteranceStart()
	}
	mode := modeCommand
	if se.state.HasState(state.StateNotListening) {
		mode = modeMuted
	} else if se.cfg.DictationEnabled && se.state.Dictation.HasText() {
		mode = modeSticky
	}
	return &streamSession{id: p.Session.ID, mode: mode}
}

// normalizeStream applies the batch path's text cleanup (minus the "put"
// symbol special case, which the batch fallback still covers at Final).
// util.Commas is deliberately not applied: tokenize already strips outer
// punctuation for matching, and raw tokens must keep their commas so
// trailer text reaches commands like "type" intact.
func normalizeStream(text string) []token {
	return tokenize(util.CommonMistakes(symbols.NormalizeSymbolInput(text)))
}

// update tracks per-position stability: a change at position i restarts
// the count for every position from i on, since whisper rewrites can
// shift everything downstream
func (s *streamSession) update(tokens []token) {
	m := 0
	for m < len(tokens) && m < len(s.tokens) && tokens[m].clean == s.tokens[m].clean {
		m++
	}
	stable := make([]int, len(tokens))
	for i := range tokens {
		if i < m {
			stable[i] = s.stable[i] + 1
		} else {
			stable[i] = 1
		}
	}
	s.tokens = tokens
	s.stable = stable
}

func (s *streamSession) isStable(i int, need int, final bool) bool {
	return final || s.stable[i] >= need
}

// prefixIntact reports whether the locked prefix still tokenizes to the
// words that fired. Whisper can merge or split words inside the fired
// region ("up up" → "up-up"), which shifts every downstream position —
// scanning the remainder positionally would then drop or misfire commands.
func (s *streamSession) prefixIntact() bool {
	if s.firedTokens > len(s.tokens) {
		return false
	}
	for i, clean := range s.firedClean {
		if s.tokens[i].clean != clean {
			return false
		}
	}
	return true
}

// remainderText is the not-yet-executed part of the utterance. When
// nothing fired this is the raw whisper text (prose must not inherit
// CommonMistakes rewrites); after fires it approximates from tokens.
func (s *streamSession) remainderText(p asr.Partial) string {
	if s.firedTokens == 0 {
		return strings.TrimSpace(p.Text)
	}
	if s.firedTokens >= len(s.tokens) {
		return ""
	}
	return strings.Join(rawWords(s.tokens[s.firedTokens:]), " ")
}

// scanStable greedily parses tokens[firedTokens:] over the stable region.
// A segment fires only once a stable successor confirms the chain keeps
// parsing (waived on Final). broke reports a stable token no trigger
// consumes — the prose signal.
func (se *StreamEvaluator) scanStable(s *streamSession, final bool) (fire []sequenceStep, fireEnd int, broke bool) {
	activeStates := se.state.GetActiveStates()

	stableEnd := s.firedTokens
	for stableEnd < len(s.tokens) && s.isStable(stableEnd, se.cfg.StabilityTicks, final) {
		stableEnd++
	}

	type builtSeg struct {
		step sequenceStep
		end  int
	}
	var segs []builtSeg
	holdWithMatch := false

	i := s.firedTokens
	for i < stableEnd {
		var entry *commands.CommandEntry
		end := i
		for j := stableEnd; j > i; j-- {
			phrase := commands.TriggerPhrase(strings.Join(cleanWords(s.tokens[i:j]), " "))
			if found := se.lookup.FindWithState(phrase, activeStates); found != nil {
				entry = found
				end = j
				break
			}
		}

		if entry == nil {
			// A multi-word trigger may still be forming across the
			// frontier ("left | word"): hold rather than flip to prose
			if !final {
				phrase := commands.TriggerPhrase(strings.Join(cleanWords(s.tokens[i:stableEnd]), " "))
				if se.lookup.HasLongerTrigger(phrase, activeStates) {
					break
				}
			}
			return nil, 0, true
		}

		// Matched right up to the frontier while a longer trigger could
		// still complete: the next word decides, hold
		if end == stableEnd && !final {
			phrase := commands.TriggerPhrase(strings.Join(cleanWords(s.tokens[i:end]), " "))
			if se.lookup.HasLongerTrigger(phrase, activeStates) {
				holdWithMatch = true
				break
			}
		}

		step := sequenceStep{entry: entry, triggerWords: cleanWords(s.tokens[i:end]), multiplier: 1}
		i = end

		// Trailer/prompt commands are greedy: everything after them is
		// their input, and prompting mid-utterance would steal the mic,
		// so they only fire at Final
		if entry.Command.MayPrompt() {
			if !final {
				holdWithMatch = true
				break
			}
			step.trailerWords = rawWords(s.tokens[i:])
			step.trailer = commands.TrailerText(strings.Join(step.trailerWords, " "))
			segs = append(segs, builtSeg{step, len(s.tokens)})
			i = len(s.tokens)
			continue
		}

		if i < stableEnd && entry.Command.Multiplier {
			if m, ok := parseMultiplier(s.tokens[i].clean); ok {
				step.multiplier = m
				i++
			}
		}
		segs = append(segs, builtSeg{step, i})
	}

	fireCount := len(segs)
	if !final && !holdWithMatch {
		// The last segment has no confirming successor yet
		fireCount--
	}
	if fireCount <= 0 {
		return nil, s.firedTokens, false
	}
	steps := make([]sequenceStep, 0, fireCount)
	for _, sg := range segs[:fireCount] {
		steps = append(steps, sg.step)
	}
	return steps, segs[fireCount-1].end, false
}

func (se *StreamEvaluator) evalCommand(p asr.Partial) {
	s := se.sess

	// A rewrite inside the fired prefix (R5) means positions can't be
	// trusted. Hold mid-utterance — a later tick may restore the prefix —
	// and realign at Final: re-parse the whole utterance and execute only
	// the steps beyond the ones that already fired. The positional parse
	// is kept as the floor so realignment can never do worse than before.
	if s.firedTokens > 0 && !s.prefixIntact() {
		if !p.Final {
			return
		}
		var fallback []sequenceStep
		if fire, _, broke := se.scanStable(s, true); !broke {
			fallback = fire
		}
		firedSteps := s.firedSteps
		se.queue(func() { se.execRealigned(p, firedSteps, fallback) })
		return
	}

	fire, fireEnd, broke := se.scanStable(s, p.Final)

	if broke {
		if !p.Final {
			if se.cfg.DictationEnabled {
				kind := "stream_prose_flip"
				if s.firedSteps > 0 {
					kind = "stream_late_prose"
				}
				se.logStream(kind, s.remainderText(p))
				s.mode = modeProse
				se.sendDictation(se.state.Dictation.Text(), s.remainderText(p))
			}
			return
		}
		if s.firedSteps > 0 {
			tail := s.remainderText(p)
			se.queue(func() {
				se.sendUI("? " + tail)
				se.logStream("stream_tail_discard", tail)
			})
			return
		}
		se.queueFallback(p)
		return
	}

	// A trailer feeds screen matching, where a fast-model mishearing acts on
	// the wrong thing — worth the accurate model's extra beat at Final.
	// Whole-utterance re-parse only: fired history must not be rewritten.
	if p.Final && se.cfg.AccurateCommandFinals && s.firedTokens == 0 && hasTrailerStep(fire) {
		steps := fire
		se.queue(func() { se.execAccurateTrailer(p, steps) })
		s.lockFired(fireEnd, len(fire))
		return
	}

	for _, st := range fire {
		se.queueStep(st)
	}
	if len(fire) > 0 {
		s.lockFired(fireEnd, len(fire))
	}
}

// lockFired advances the positional lock and records what the locked
// tokens said, so later rewrites inside the prefix are detectable.
func (s *streamSession) lockFired(fireEnd, steps int) {
	s.firedTokens = fireEnd
	s.firedSteps += steps
	s.firedClean = cleanWords(s.tokens[:fireEnd])
}

func hasTrailerStep(steps []sequenceStep) bool {
	for _, st := range steps {
		if st.entry.Command.MayPrompt() {
			return true
		}
	}
	return false
}

// execAccurateTrailer re-reads the utterance with the accurate model before
// executing trailer command(s). The accurate parse wins only when the whole
// utterance still resolves to commands; any downgrade keeps the fast parse.
// Runs on the exec worker, like the dictation finals it mirrors.
func (se *StreamEvaluator) execAccurateTrailer(p asr.Partial, fast []sequenceStep) {
	steps := fast
	text, err := se.reTranscribe(p.Session, commandFinalPrompt)
	switch {
	case err != nil:
		if err.Error() != "noise" {
			log.Warnf("stream: accurate command final failed, keeping fast text: %v", err)
		}
	case strings.TrimSpace(text) != "":
		tmp := &streamSession{}
		tmp.update(normalizeStream(text))
		if better, _, broke := se.scanStable(tmp, true); !broke && len(better) > 0 {
			se.logStream("stream_accurate_trailer", text)
			steps = better
		}
	}
	se.execSteps(steps)
}

// execRealigned recovers a Final whose fired prefix was re-tokenized:
// parse the whole utterance fresh (accurate re-read first when enabled,
// then the fast text) and execute only the steps beyond the firedSteps
// already executed — fired history is never rewritten. When no candidate
// fully parses past the fired steps, the positional fallback runs, so
// this path is never worse than the pre-realign behavior. Runs on the
// exec worker. Skipping by step count is best-effort: a re-parse that
// restructures the fired region (e.g. absorbing a word as a multiplier)
// may still shift the remainder.
func (se *StreamEvaluator) execRealigned(p asr.Partial, firedSteps int, fallback []sequenceStep) {
	candidates := []string{}
	if se.cfg.AccurateCommandFinals {
		better, err := se.reTranscribe(p.Session, commandFinalPrompt)
		if err != nil {
			if err.Error() != "noise" {
				log.Warnf("stream: realign re-read failed, using fast text: %v", err)
			}
		} else if strings.TrimSpace(better) != "" {
			candidates = append(candidates, strings.TrimSpace(better))
		}
	}
	candidates = append(candidates, strings.TrimSpace(p.Text))

	for _, text := range candidates {
		tmp := &streamSession{}
		tmp.update(normalizeStream(text))
		if steps, _, broke := se.scanStable(tmp, true); !broke && len(steps) > firedSteps {
			se.logStream("stream_realign", text)
			se.execSteps(steps[firedSteps:])
			return
		}
	}
	se.logStream("stream_realign_miss", strings.TrimSpace(p.Text))
	se.execSteps(fallback)
}

// execSteps executes parsed steps in order on the current goroutine (the
// exec worker).
func (se *StreamEvaluator) execSteps(steps []sequenceStep) {
	for _, st := range steps {
		label := strings.Join(st.triggerWords, " ")
		se.sendUI(label)
		if err := se.executor.ExecuteCommand(st.entry, st.triggerWords, st.trailerWords, st.trailer, st.multiplier, "voice_stream"); err != nil {
			log.Errorf("stream: command [%s] failed: %v", label, err)
		}
		se.logStream("stream_fire", label)
	}
}

func (se *StreamEvaluator) queueStep(st sequenceStep) {
	se.queue(func() { se.execSteps([]sequenceStep{st}) })
}

// queueFallback routes an unparsed utterance through the full batch
// pipeline (sequence, backwards+fuzzy, smart correction, OCR); when even
// that fails and dictation is on, the utterance becomes prose.
//
// With accurate command finals on, one accurate-model re-read runs first:
// fast text like "move,cloudcode" (comma, no space) breaks the stream parse
// — normalizeStream deliberately keeps commas — and the batch pipeline
// would then execute its trailer from the fast transcription, sidestepping
// the accurate pass command_accurate_finals promises trailer commands. The
// accurate parse wins only when the whole utterance resolves to commands;
// anything else falls through with the original fast result, so prose
// handling and its fidelity are untouched. Short prose utterances that
// never ticked pay the extra beat here, same accepted cost as accurate
// dictation finals.
func (se *StreamEvaluator) queueFallback(p asr.Partial) {
	text := strings.TrimSpace(p.Text)
	se.queue(func() {
		if se.cfg.AccurateCommandFinals && se.accurateRescue(p) {
			return
		}
		handled, err := se.batch.EvaluateResultHandled(result(p), true)
		if handled {
			return
		}
		if se.cfg.DictationEnabled {
			se.appendProse(text, p.Session.FileName)
			return
		}
		if err != nil {
			log.Error(err)
		}
	})
}

// accurateRescue re-reads the utterance with the accurate model and
// executes it when it fully resolves to commands, reporting whether it did.
// Runs on the exec worker.
func (se *StreamEvaluator) accurateRescue(p asr.Partial) bool {
	text, err := se.reTranscribe(p.Session, commandFinalPrompt)
	if err != nil {
		if err.Error() != "noise" {
			log.Warnf("stream: accurate fallback failed, keeping fast text: %v", err)
		}
		return false
	}
	if strings.TrimSpace(text) == "" {
		return false
	}
	tmp := &streamSession{}
	tmp.update(normalizeStream(text))
	steps, _, broke := se.scanStable(tmp, true)
	if broke || len(steps) == 0 {
		return false
	}
	se.logStream("stream_accurate_fallback", text)
	se.execSteps(steps)
	return true
}

func (se *StreamEvaluator) finalProse(p asr.Partial) {
	s := se.sess
	text := s.remainderText(p)
	salvageable := s.firedSteps == 0 && len(s.tokens) <= se.cfg.ProseSalvageMaxTokens
	// Accurate re-transcription covers the whole utterance, so it can only
	// replace the fast text when nothing was executed from it
	wholeUtterance := s.firedSteps == 0
	se.queue(func() {
		if salvageable && !se.state.Dictation.HasText() {
			if handled, _ := se.batch.EvaluateResultHandled(result(p), true); handled {
				se.logStream("stream_salvage", text)
				se.refreshPanel()
				return
			}
		}
		if wholeUtterance {
			text = se.accurateFinal(p.Session, text)
			// The accurate model may reveal a command chain the fast ticks
			// garbled into prose. Full-parse only: prose can never fully
			// consume into >=2 segments, so this cannot steal dictation.
			if se.rescueChain(text) {
				se.refreshPanel()
				return
			}
			se.appendProse(text, p.Session.FileName)
			return
		}
		// commands fired from this recording: replaying it would turn
		// them into prose, so the remainder carries no clip
		se.appendProse(text, "")
	})
}

// rescueChain executes text when it fully parses into a command chain
// (every token consumed, at least two segments). Runs on the exec worker,
// so commands execute directly rather than re-queueing.
func (se *StreamEvaluator) rescueChain(text string) bool {
	steps := splitSequence(se.lookup, normalizeStream(text), se.state.GetActiveStates())
	if len(steps) < 2 {
		return false
	}
	se.logStream("stream_rescue", text)
	for _, st := range steps {
		label := strings.Join(st.triggerWords, " ")
		se.sendUI(label)
		if err := se.executor.ExecuteCommand(st.entry, st.triggerWords, st.trailerWords, st.trailer, st.multiplier, "voice_stream"); err != nil {
			log.Errorf("stream: rescued command [%s] failed: %v", label, err)
			return true
		}
	}
	return true
}

func (se *StreamEvaluator) finalSticky(p asr.Partial) {
	se.queue(func() {
		// Accurate text improves both control-word recognition ("send it")
		// and what lands in the buffer
		text := se.accurateFinal(p.Session, strings.TrimSpace(p.Text))
		if m := se.controlWord(text); m != nil {
			label := strings.Join(m.triggerWords, " ")
			if err := se.executor.ExecuteCommand(m.entry, m.triggerWords, m.trailerWords, m.trailer, m.multiplier, "voice_stream"); err != nil {
				log.Errorf("stream: control command [%s] failed: %v", label, err)
			}
			se.logStream(m.kind, label)
			return
		}
		se.appendProse(text, p.Session.FileName)
	})
}

// controlMethods are the only handlers a standalone utterance may invoke
// while dictating; everything else is prose
var controlMethods = map[string]string{
	"ai_accept":             "dictation_commit",
	"ai_clip":               "dictation_commit",
	"ai_reject":             "dictation_discard",
	"dictation_scratch":     "dictation_scratch",
	"dictation_delete_char": "dictation_delete_char",
	"dictation_delete_word": "dictation_delete_word",
	"dictation_tidy":        "dictation_tidy",
	"dictation_case":        "dictation_case",
	"dictation_punct":       "dictation_punct",
	"dictation_yolo":        "dictation_commit",
	"dictation_revise":      "dictation_revise",
	"dictation_undo":        "dictation_undo",
	"dictation_redo":        "dictation_redo",
	"dictation_replay":      "dictation_replay",
}

// trailerControlMethods may take the rest of the utterance as free-form
// input ("fix that I meant privacy, not piracy") via trigger-prefix
// matching rather than the exact-match rule
var trailerControlMethods = map[string]bool{
	"dictation_revise": true,
}

type controlMatch struct {
	entry        *commands.CommandEntry
	triggerWords []string
	trailerWords []string
	trailer      commands.TrailerText
	kind         string
	multiplier   int
}

// controlWord matches a standalone utterance against dictation controls.
// Exact match first (with an optional trailing spoken count, "delete word
// three"); then trigger-prefix matching, longest first, for the few
// trailer-consuming controls, with the raw remainder as their instruction.
// No fuzzy matching: prose containing "insert" must never trigger.
func (se *StreamEvaluator) controlWord(text string) *controlMatch {
	tokens := tokenize(util.TrimTrailingPunctuation(text))
	if len(tokens) == 0 {
		return nil
	}
	clean := cleanWords(tokens)

	norm := strings.Join(clean, " ")
	multiplier := 1
	if len(clean) > 1 {
		if m, ok := parseMultiplier(clean[len(clean)-1]); ok {
			multiplier = m
			norm = strings.Join(clean[:len(clean)-1], " ")
		}
	}
	if entry, kind, _ := se.controlEntry(norm); entry != nil {
		return &controlMatch{entry: entry, triggerWords: strings.Fields(norm), kind: kind, multiplier: multiplier}
	}

	for n := len(clean) - 1; n >= 1; n-- {
		entry, kind, trailerCapable := se.controlEntry(strings.Join(clean[:n], " "))
		if entry == nil || !trailerCapable {
			continue
		}
		trailerWords := rawWords(tokens[n:])
		return &controlMatch{
			entry:        entry,
			triggerWords: clean[:n],
			trailerWords: trailerWords,
			trailer:      commands.TrailerText(strings.Join(trailerWords, " ")),
			kind:         kind,
			multiplier:   1,
		}
	}
	return nil
}

// controlEntry returns the command for the phrase when every state-active
// action is a dictation control (commands like "delete word" fork on the
// dictating state, so inactive keystroke actions are ignored)
func (se *StreamEvaluator) controlEntry(phrase string) (*commands.CommandEntry, string, bool) {
	if phrase == "" {
		return nil, "", false
	}
	entry := se.lookup.Find(commands.TriggerPhrase(phrase))
	if entry == nil {
		return nil, "", false
	}
	activeStates := se.state.GetActiveStates()
	if !entry.Command.IsActiveInState(activeStates) {
		return nil, "", false
	}
	kind := ""
	active := 0
	trailerCapable := false
	for _, a := range entry.Command.Actions {
		if !a.IsActiveInState(activeStates) {
			continue
		}
		active++
		k, ok := controlMethods[a.Method]
		if !ok {
			return nil, "", false
		}
		if kind == "" {
			kind = k
		}
		if trailerControlMethods[a.Method] {
			trailerCapable = true
		}
	}
	if active == 0 {
		return nil, "", false
	}
	return entry, kind, trailerCapable
}

// appendProse runs on the exec worker so buffer changes stay ordered with
// command execution. clip is the recording text was read from in full
// ("replay that" re-reads it), empty when it has none.
func (se *StreamEvaluator) appendProse(text, clip string) {
	if strings.TrimSpace(text) == "" {
		se.refreshPanel()
		return
	}
	se.state.Dictation.AppendClip(text, clip)
	se.state.SetState(state.StateDictating, true)
	se.sendDictation(se.state.Dictation.Text(), "")
	se.logStream("dictation_append", text)
}

func (se *StreamEvaluator) refreshPanel() {
	if se.state.Dictation.HasText() {
		se.sendDictation(se.state.Dictation.Text(), "")
	} else {
		se.dismissDictation()
	}
}

func (se *StreamEvaluator) logStream(kind, detail string) {
	if !debuglog.Enabled() {
		return
	}
	debuglog.AppendCommand([]string{kind, "voice_stream", "", "", detail})
}

func result(p asr.Partial) transcribe.Result {
	return transcribe.Result{Session: p.Session}.Copy(strings.TrimSpace(p.Text))
}
