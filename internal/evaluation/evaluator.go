package evaluation

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/config"
	"github.com/mousybusiness/mouse_whisperer/internal/debuglog"
	"github.com/mousybusiness/mouse_whisperer/internal/execution"
	"github.com/mousybusiness/mouse_whisperer/internal/impl"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/symbols"
	"github.com/mousybusiness/mouse_whisperer/pkg/transcribe"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	"github.com/mousybusiness/mouse_whisperer/types"
	log "github.com/sirupsen/logrus"
)

type Inputter interface {
	Enabled() bool
}

// commandExecutor abstracts execution so evaluation can be tested with a fake
type commandExecutor interface {
	ExecuteCommand(entry *commands.CommandEntry, triggerWords []string, trailerWords []string, trailer commands.TrailerText, multiplier int, invocationSource string) error
}

// Evaluator processes voice transcriptions and attemts to locate the appropriate command
type Evaluator struct {
	// Config-based execution
	cfg      *config.Config
	lookup   *commands.CommandLookup
	executor commandExecutor
	state    *state.State
	input    Inputter
}

// NewEvaluator creates a new config-based evaluator
func NewEvaluator(cfg *config.Config, lookup *commands.CommandLookup, executor *execution.Executor, input Inputter, state *state.State) (*Evaluator, error) {
	e := &Evaluator{
		input:    input,
		cfg:      cfg,
		lookup:   lookup,
		executor: executor,
		state:    state,
	}
	return e, nil
}

func (e *Evaluator) Enabled() bool {
	return e.input.Enabled()
}

// EvaluateTranscription processes a voice transcription
func (e *Evaluator) EvaluateTranscription(transcription transcribe.Result, allowSmartCorrection bool) {
	log.Debugf("Evaluating transcription: %s", transcription.Text())

	// Set the recording start time on the executor so handlers can access it

	//if e.awaitingInput {
	//	log.Debugf("Awaiting input")
	//	// Check if user wants to abort - strip punctuation for more robust matching
	//	text := strings.TrimSpace(strings.ToLower(util.TrimTrailingPunctuation(transcription.Text())))
	//	if text == "abort" || text == "bye" || text == "cancel" || text == "nevermind" || text == "never mind" {
	//		log.Warning("Input aborted by user")
	//		overlay.SendUIMessage("Aborted")
	//		transcription.Session.RecordingRequest.Abort()
	//		e.resetInput()
	//		return
	//	}
	//
	//	transcription.Session.RecordingRequest.SetTranscription(transcription.Text())
	//	e.resetInput()
	//	return
	//}

	if e.state.HasState(state.StateNotListening) {
		if !e.allowWhileNotListening(transcription.Text()) {
			fmt.Printf("Not listening\n")
			return
		}
	}

	e.state.SetRecordingStart(transcription.Session)
	if _, err := e.EvaluateResultHandled(transcription, allowSmartCorrection); err != nil {
		log.Error(err)
	}
}

func (e *Evaluator) allowWhileNotListening(trans string) bool {
	t := util.ToWords(util.ToAlphanumericWithSpace(trans))
	log.Debugf("checkStartStop: %s", t)
	// Check against config
	entry := e.lookup.Find(commands.TriggerPhrase(t))
	log.Debugf("checkStartStop: entry: %v", entry)
	if entry != nil {
		log.Debugf("checkStartStop: command: %v", entry.Command)
		if slices.Contains(entry.Command.States, "not_listening") {
			return true
		}
	}

	return false
}

// EvaluateResultHandled runs the full single-utterance pipeline (sequence,
// backwards+fuzzy, smart correction, OCR fallback) and reports whether
// anything consumed the utterance. A found-but-failed command counts as
// handled. Smart correct will use a larger model to improve accuracy.
func (e *Evaluator) EvaluateResultHandled(transcription transcribe.Result, allowSmartCorrection bool) (bool, error) {
	tt := transcription.Text()
	// ASR may emit the literal symbol instead of its spoken name (e.g. "/" instead of "slash");
	// SplitTriggerAndSymbol below drops pure-symbol input, so normalize first.
	tt = symbols.NormalizeSymbolInput(tt)
	word, symbol := symbols.SplitTriggerAndSymbol(tt)
	if word == "put" {
		tt = word + " " + symbol
	} else {
		tt = word
	}

	tt = util.CommonMistakes(strings.TrimSpace(tt))

	input := tt
	chainableInput := util.Commas(tt)

	log.Debugf("Input: %s", input)
	log.Debugf("Chainable input: %s", chainableInput)

	// Try a full command sequence first, e.g. "up up up down left right",
	// so multi-command utterances aren't swallowed as one command + trailer
	handled, err := e.EvaluateSequence(chainableInput, "voice")
	if err != nil {
		return true, err
	}
	if handled {
		overlay.SendUIMessage(chainableInput)
		return true, nil
	}

	// Then try a single command with trailer
	handled, err = e.EvaluteBackwardsCommand(chainableInput, "voice")
	if err != nil {
		return true, err
	}
	if handled {
		overlay.SendUIMessage(chainableInput)
		return true, nil
	}

	// A bare number while the scan overlay is up picks that target — the
	// digit triggers only cover 0-9, so "fifty" lands here. Must run before
	// the overlay_hex reset below, which would dismiss the overlay.
	if impl.TryPickTarget(e.state, tt) {
		if debuglog.Enabled() {
			debuglog.AppendCommand([]string{"pick_target", "voice", "", "", tt})
		}
		return true, nil
	}

	// ------------------- TODO REFACTOR THIS -------------------
	//
	//// Process comma-separated commands
	//parts := strings.Split(input, ",")
	//for _, part := range parts {
	//	cleaned := util.Commas(part)
	//	m := getMultiplier(util.TrimTrailingPunctuation(cleaned))
	//	multiplier := 1
	//	if m != nil {
	//		multiplier = *m
	//	}
	//
	//	command := util.ToWords(cleaned)
	//
	//	// Try config-based execution first
	//	if e.executeFromConfig(transcription.Session, command, multiplier) {
	//		log.Debugf("[Consumed] config")
	//		overlay.SendUIMessage(command)
	//		return nil
	//	}
	//
	//	// Fallback to mouse subsystem for complex mouse commands
	//	if e.mouse.PerformCommand(transcription.Session, command, multiplier) {
	//		log.Debugf("[Consumed] mouse (fallback)")
	//		overlay.SendUIMessage(command)
	//		return nil
	//	}
	//
	//	log.Debugf("fallthrough, allow correction: %v", allowSmartCorrection)
	//
	// If we're in overlay_hex state and command not found, reset to default
	// This prevents being stuck in overlay mode when trying to use non-overlay commands
	if e.state.HasState(state.StateOverlayHex) {
		log.Info("Command not found in overlay_hex state - resetting to default")
		e.state.ClearOverlay()
	}

	if allowSmartCorrection {
		tran, err := transcribe.Do(transcription.Session, mic.ModelAccurate)
		if err != nil {
			return false, errors.New("Smart correction failed")
		}

		log.Debugf("Smart reevaluation\noriginal: %v\nimproved: %v", transcription.Text(), tran.Text())
		return e.EvaluateResultHandled(transcription.Copy(strings.ToLower(tran.Text())), false)
	}

	// Try OCR move fallback with configurable region size
	// Only for short phrases (max 1 word for now)
	fallbackRegionSize := e.state.GetOCRRegionSizeFallback()
	if impl.TryOCRMoveFallback(e.state, tt, fallbackRegionSize, 1) {
		if debuglog.Enabled() {
			debuglog.AppendCommand([]string{"ocr_fallback", "voice", "", "", tt})
		}
		overlay.SendUIMessage(tt)
		return true, nil
	}

	return false, errors.New("no command found")
}

type sequenceStep struct {
	entry        *commands.CommandEntry
	triggerWords []string
	trailerWords []string
	trailer      commands.TrailerText
	multiplier   int
}

// token is a single input word: clean is used for trigger matching, raw is
// preserved so trailer text keeps its original case and punctuation
type token struct {
	raw   string
	clean string
}

// outerPunctuation strips leading/trailing non-alphanumerics from a lowered
// word; interior punctuation stays so words like "readme.md" survive
var outerPunctuation = regexp.MustCompile(`^[^a-z0-9]+|[^a-z0-9]+$`)

// tokenize splits input into tokens for trigger matching. Whisper adds
// punctuation and capitalization ("Up, up. Down!") which must never reach
// trigger lookup; pure-punctuation words are noise and dropped entirely
func tokenize(input string) []token {
	var tokens []token
	for _, w := range strings.Fields(input) {
		clean := outerPunctuation.ReplaceAllString(strings.ToLower(w), "")
		if clean == "" {
			continue
		}
		tokens = append(tokens, token{raw: w, clean: clean})
	}
	return tokens
}

// EvaluateSequence interprets the input as consecutive commands, e.g.
// "up up up down left right" or "up 3 down". It only succeeds when every
// word is consumed by known triggers (plus optional multipliers and one
// trailing free-text trailer), so any other input falls through to the
// single-command path.
func (e *Evaluator) EvaluateSequence(t string, invocationSource string) (bool, error) {
	steps := splitSequence(e.lookup, tokenize(t), e.state.GetActiveStates())
	if len(steps) < 2 {
		return false, nil
	}

	log.Infof("++++++++ EvaluateSequence +++++++++: %d commands from: %s", len(steps), t)
	for _, s := range steps {
		if err := e.executor.ExecuteCommand(s.entry, s.triggerWords, s.trailerWords, s.trailer, s.multiplier, invocationSource); err != nil {
			return true, fmt.Errorf("sequence command [%s] failed: %w", strings.Join(s.triggerWords, " "), err)
		}
	}
	return true, nil
}

// splitSequence greedily matches the longest trigger at each position.
// Returns the steps only if the entire input segments into commands;
// returns nil on any unmatched residue so the caller can fall back.
func splitSequence(lookup *commands.CommandLookup, tokens []token, activeStates map[types.ExecutionState]bool) []sequenceStep {
	var steps []sequenceStep
	i := 0
	for i < len(tokens) {
		var entry *commands.CommandEntry
		end := i
		for j := len(tokens); j > i; j-- {
			if found := lookup.FindWithState(commands.TriggerPhrase(strings.Join(cleanWords(tokens[i:j]), " ")), activeStates); found != nil {
				entry = found
				end = j
				break
			}
		}
		if entry == nil {
			return nil
		}

		step := sequenceStep{entry: entry, triggerWords: cleanWords(tokens[i:end]), multiplier: 1}
		i = end

		if entry.Command.ConsumesTrailer() {
			// Trailer commands are greedy: the rest of the utterance is their
			// input, e.g. "up up type hello world"
			step.trailerWords = rawWords(tokens[i:])
			step.trailer = commands.TrailerText(strings.Join(step.trailerWords, " "))
			i = len(tokens)
		} else if i < len(tokens) && entry.Command.Multiplier {
			if m, ok := parseMultiplier(tokens[i].clean); ok {
				step.multiplier = m
				i++
			}
		}
		steps = append(steps, step)
	}
	return steps
}

func cleanWords(tokens []token) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = t.clean
	}
	return out
}

func rawWords(tokens []token) []string {
	out := make([]string, len(tokens))
	for i, t := range tokens {
		out[i] = t.raw
	}
	return out
}

var numberWords = map[string]int{
	"one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
	"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
}

// parseMultiplier accepts digits and spoken numbers; CommonMistakes only
// converts number words at the end of the utterance, not mid-sequence
func parseMultiplier(word string) (int, bool) {
	if m, err := strconv.Atoi(word); err == nil && m > 0 {
		return m, true
	}
	m, ok := numberWords[word]
	return m, ok
}

// EvaluteBackwardsCommand tries to match a command from the end of the input
// so that we can correctly capture the trailer
func (e *Evaluator) EvaluteBackwardsCommand(t string, invocationSource string) (bool, error) {
	parts := strings.Split(t, " ")
	activeStates := e.state.GetActiveStates()

	// Extract multiplier from the last part if it's a number
	multiplier := 1
	if len(parts) > 0 {
		if m, err := strconv.Atoi(parts[len(parts)-1]); err == nil && m > 0 {
			multiplier = m
		}
	}

	log.Infof("++++++++ evaluteBackwardsCommand +++++++++: %s (multiplier: %d)", parts, multiplier)

	// Try to match progressively longer prefixes for multi-word triggers
	// e.g., for "scan for readme.md", try "scan for", then "scan"
	for i := len(parts); i > 0; i-- {
		var args []string
		if i < len(parts) {
			args = parts[i:]
		}
		phrase := commands.TriggerPhrase(strings.Join(parts[:i], " "))
		trailer := commands.TrailerText(strings.Replace(t, string(phrase), "", 1))
		found, err := e.TryFindAndExecute(phrase, args, trailer, multiplier, activeStates, invocationSource)
		if err != nil {
			return true, fmt.Errorf("command found but failed to execute: %w", err)
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

// TryFindAndExecute runs a chained command with state-aware lookup
func (e *Evaluator) TryFindAndExecute(t commands.TriggerPhrase, trailerWords []string, trailer commands.TrailerText, multiplier int, activeStates map[types.ExecutionState]bool, invocationSource string) (bool, error) {
	log.Infof("++++++++ ExecuteChainedWithState +++++++++: %s %v", t, trailerWords)

	triggerWords := strings.Split(string(t), " ")

	// Check for direct match first
	entry := e.lookup.FindWithState(t, activeStates)
	if entry != nil {
		return true, e.executor.ExecuteCommand(entry, triggerWords, trailerWords, trailer, multiplier, invocationSource)
	}

	// Final fallback: ranked fuzzy match against all triggers with high confidence
	// This handles transcription typos like "okay" -> "ok", "coma" -> "comma"
	if rankedEntry, matchedTrigger := e.lookup.FindRankedFuzzy(commands.IncorrectTriggerPhrase(t), activeStates); rankedEntry != nil {
		log.Infof("Ranked fuzzy matched '%s' -> trigger '%s'", t, matchedTrigger)
		return true, e.executor.ExecuteCommand(rankedEntry, triggerWords, trailerWords, trailer, multiplier, invocationSource)
	}

	return false, nil
}

//
//// Helper function - same as in evaluate.go
//func getMultiplier(word string) *int {
//	reg := regexp.MustCompile(`[^\w\s,]`)
//	cleaned := reg.ReplaceAllString(word, "")
//	reg2 := regexp.MustCompile(`^.*?(\d+)$`)
//	match := reg2.FindStringSubmatch(cleaned)
//	if match == nil {
//		return nil
//	}
//	if len(match) > 1 {
//		multiplier, err := strconv.Atoi(match[1])
//		if err != nil {
//			fmt.Printf("Error converting '%s' to int: %s\n", match[1], err)
//			return nil
//		}
//		return &multiplier
//	}
//	return nil
//}
