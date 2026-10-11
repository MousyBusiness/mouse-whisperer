package evaluation

import (
	"errors"
	"strings"
	"testing"

	"github.com/mousybusiness/mouse_whisperer/internal/asr"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/transcribe"
)

// streamLookup extends the base test commands with dictation controls
func streamLookup() *commands.CommandLookup {
	raw := map[string]map[string]commands.Command{
		"text": {
			"up":        {Triggers: []string{"up"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"down":      {Triggers: []string{"down"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"left":      {Triggers: []string{"left"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"right":     {Triggers: []string{"right"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"left_word": {Triggers: []string{"left word"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"type":      {Triggers: []string{"type"}, Actions: []*commands.Action{{Method: "type_str", RequiresTrailer: true}}},
		},
		"ai": {
			"accept":   {Triggers: []string{"insert"}, DisableFuzzy: true, Actions: []*commands.Action{{Method: "ai_accept"}}},
			"reject":   {Triggers: []string{"reject"}, DisableFuzzy: true, Actions: []*commands.Action{{Method: "ai_reject"}}},
			"clip":     {Triggers: []string{"take"}, Actions: []*commands.Action{{Method: "ai_clip"}}},
			"scratch":  {Triggers: []string{"scratch that"}, States: []string{"dictating"}, Actions: []*commands.Action{{Method: "dictation_scratch"}}},
			"tidy":     {Triggers: []string{"tidy that", "tidy"}, States: []string{"dictating"}, Actions: []*commands.Action{{Method: "dictation_tidy"}}},
			"question": {Triggers: []string{"question that"}, DisableFuzzy: true, States: []string{"dictating"}, Actions: []*commands.Action{{Method: "dictation_punct"}}},
			"replay":   {Triggers: []string{"replay that", "replay"}, DisableFuzzy: true, States: []string{"dictating"}, Actions: []*commands.Action{{Method: "dictation_replay"}}},
			"send_it":  {Triggers: []string{"send it", "send that"}, DisableFuzzy: true, States: []string{"dictating"}, Actions: []*commands.Action{{Method: "dictation_yolo"}}},
			"revise":   {Triggers: []string{"fix that", "adjust that"}, DisableFuzzy: true, States: []string{"dictating"}, Actions: []*commands.Action{{Method: "dictation_revise"}}},
			"undo": {Triggers: []string{"undo"}, Actions: []*commands.Action{
				{ID: "buffer", Method: "dictation_undo", States: []string{"dictating"}},
				{ID: "keys", Method: "keybd_event", Excludes: []string{"dictating"}},
			}},
			"redo": {Triggers: []string{"redo"}, Actions: []*commands.Action{
				{ID: "buffer", Method: "dictation_redo", States: []string{"dictating"}},
				{ID: "keys", Method: "keybd_event", Excludes: []string{"dictating"}},
			}},
			// Mirrors config/text.toml cap_that: forks on the dictating state
			"cap_that": {Triggers: []string{"title that"}, Actions: []*commands.Action{
				{ID: "buffer", Method: "dictation_case", States: []string{"dictating"}},
				{ID: "selection", Method: "text_title_that", Excludes: []string{"dictating"}},
			}},
			// Mirrors config/text.toml delete_word: same trigger forks on
			// the dictating state via action-level states/excludes
			"delete_word": {Triggers: []string{"delete word"}, DisableFuzzy: true, Multiplier: true, Actions: []*commands.Action{
				{ID: "buffer", Method: "dictation_delete_word", States: []string{"dictating"}},
				{ID: "keys", Method: "keybd_event", Excludes: []string{"dictating"}},
			}},
			"delete_that": {Triggers: []string{"delete that"}, DisableFuzzy: true, Multiplier: true, Actions: []*commands.Action{
				{ID: "buffer", Method: "dictation_delete_char", States: []string{"dictating"}},
				{ID: "keys", Method: "keybd_event", Excludes: []string{"dictating"}},
			}},
		},
	}
	return commands.NewCommandLookup(raw, []string{"text", "ai"})
}

type fakeBatch struct {
	handled bool
	calls   []string
}

func (f *fakeBatch) EvaluateResultHandled(t transcribe.Result, allow bool) (bool, error) {
	f.calls = append(f.calls, "fallback:"+t.Text())
	return f.handled, nil
}

func (f *fakeBatch) EvaluateTranscription(t transcribe.Result, allow bool) {
	f.calls = append(f.calls, "muted:"+t.Text())
}

type streamHarness struct {
	se      *StreamEvaluator
	exec    *fakeExecutor
	batch   *fakeBatch
	state   *state.State
	panels  []string // "text|pending" per SendDictation
	dismiss int
	ui      []string
	nextID  uint64
}

func newStreamHarness(t *testing.T, cfg StreamConfig) *streamHarness {
	t.Helper()
	h := &streamHarness{
		exec:  &fakeExecutor{},
		batch: &fakeBatch{},
		state: state.NewBareState(),
	}
	h.se = &StreamEvaluator{
		lookup:   streamLookup(),
		executor: h.exec,
		state:    h.state,
		batch:    h.batch,
		cfg:      cfg,
	}
	// Synchronous execution: ordering is what the tests assert
	h.se.queue = func(fn func()) { fn() }
	h.se.drain = func() {}
	h.se.sendDictation = func(text, pending string) { h.panels = append(h.panels, text+"|"+pending) }
	h.se.dismissDictation = func() { h.dismiss++ }
	h.se.sendUI = func(m string) { h.ui = append(h.ui, m) }
	return h
}

func defaultCfg() StreamConfig {
	return StreamConfig{StabilityTicks: 2, ProseSalvageMaxTokens: 2, DictationEnabled: true}
}

// feed sends one utterance: every entry a tick, the last one the Final
func (h *streamHarness) feed(texts ...string) {
	h.nextID++
	for i, text := range texts {
		h.se.handlePartial(asr.Partial{
			Session: mic.RecordingSession{ID: h.nextID, FileName: "session.wav"},
			Seq:     i + 1,
			Text:    text,
			Final:   i == len(texts)-1,
		})
	}
}

func (h *streamHarness) firedAfter(n int) []executedCall {
	if n > len(h.exec.calls) {
		return nil
	}
	return h.exec.calls[n:]
}

func callNames(calls []executedCall) string {
	var out []string
	for _, c := range calls {
		out = append(out, string(c.name))
	}
	return strings.Join(out, ",")
}

func TestStreamIncrementalFiring(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())

	// Progressive ticks: each command fires once its successor stabilizes
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 1, Text: "up"})
	if len(h.exec.calls) != 0 {
		t.Fatalf("nothing should fire on first tick, got %v", h.exec.calls)
	}
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 2, Text: "up down"})
	if len(h.exec.calls) != 0 {
		t.Fatalf("successor not yet stable, got %v", h.exec.calls)
	}
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 3, Text: "up down down"})
	if callNames(h.exec.calls) != "up" {
		t.Fatalf("expected up fired mid-utterance, got %v", callNames(h.exec.calls))
	}
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 4, Text: "up down down left"})
	if callNames(h.exec.calls) != "up,down" {
		t.Fatalf("got %v", callNames(h.exec.calls))
	}
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 5, Text: "up down down left right", Final: true})
	if callNames(h.exec.calls) != "up,down,down,left,right" {
		t.Fatalf("got %v", callNames(h.exec.calls))
	}
	if len(h.batch.calls) != 0 {
		t.Fatalf("batch fallback should not run: %v", h.batch.calls)
	}
}

func TestStreamMultiplierAttach(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.feed("up", "up 3", "up 3 down", "up 3 down", "up 3 down")
	if callNames(h.exec.calls) != "up,down" {
		t.Fatalf("got %v", callNames(h.exec.calls))
	}
	if h.exec.calls[0].multiplier != 3 {
		t.Fatalf("expected up x3, got %d", h.exec.calls[0].multiplier)
	}
	if h.exec.calls[1].multiplier != 1 {
		t.Fatalf("expected down x1, got %d", h.exec.calls[1].multiplier)
	}
}

func TestStreamMultiWordHoldback(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	// "left" must not fire while "left word" is still possible
	h.feed("left", "left word", "left word up", "left word up", "left word up")
	if callNames(h.exec.calls) != "left_word,up" {
		t.Fatalf("got %v", callNames(h.exec.calls))
	}
}

func TestStreamMultiWordResolvesToShort(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.feed("left", "left up", "left up down", "left up down", "left up down")
	if callNames(h.exec.calls) != "left,up,down" {
		t.Fatalf("got %v", callNames(h.exec.calls))
	}
}

func TestStreamRewriteBeforeStabilityNeverFires(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	// "town" gets rewritten to "down" before reaching stability: no prose
	// flip, no misfire
	h.feed("up town", "up down", "up down down", "up down down")
	if callNames(h.exec.calls) != "up,down,down" {
		t.Fatalf("got %v", callNames(h.exec.calls))
	}
	if len(h.panels) != 0 {
		t.Fatalf("no prose panel expected: %v", h.panels)
	}
}

func TestStreamLockedPrefixIgnoresRewrites(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 1, Text: "up down"})
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 2, Text: "up down"})
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 3, Text: "up down down"})
	if callNames(h.exec.calls) != "up" {
		t.Fatalf("setup: got %v", callNames(h.exec.calls))
	}
	// Rewrite inside the fired prefix: must not re-fire or flip
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 4, Text: "cup down down"})
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 1}, Seq: 5, Text: "cup down down left", Final: true})
	if callNames(h.exec.calls) != "up,down,down,left" {
		t.Fatalf("got %v", callNames(h.exec.calls))
	}
}

func TestStreamProseFlip(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.feed("Up", "Up until", "Up until now", "Up until now we waited.", "Up until now we waited.")
	if len(h.exec.calls) != 0 {
		t.Fatalf("nothing must fire from prose, got %v", callNames(h.exec.calls))
	}
	if !h.state.Dictation.HasText() {
		t.Fatal("expected prose in buffer")
	}
	if got := h.state.Dictation.Text(); got != "Up until now we waited." {
		t.Fatalf("buffer got %q", got)
	}
	if !h.state.HasState(state.StateDictating) {
		t.Fatal("expected dictating state")
	}
	// Live panel updated during the flip
	if len(h.panels) == 0 {
		t.Fatal("expected live panel updates")
	}
	last := h.panels[len(h.panels)-1]
	if last != "Up until now we waited.|" {
		t.Fatalf("final panel got %q", last)
	}
}

// TestStreamProseClips pins what "replay that" may re-read: a whole
// prose utterance keeps its recording, a remainder after fired commands
// does not (replaying it would turn the commands into prose)
func TestStreamProseClips(t *testing.T) {
	t.Run("prose utterance records its clip", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.feed("Up until now", "Up until now we waited.", "Up until now we waited.")
		runs := h.state.Dictation.ReplayRuns()
		if len(runs) != 1 || len(runs[0].Clips) != 1 || runs[0].Clips[0] != "session.wav" {
			t.Fatalf("runs got %+v", runs)
		}
	})

	t.Run("sticky append records its clip", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("typed")
		h.state.SetState(state.StateDictating, true)
		h.feed("more prose")
		runs := h.state.Dictation.ReplayRuns()
		if len(runs) != 2 || len(runs[0].Clips) != 0 || len(runs[1].Clips) != 1 {
			t.Fatalf("runs got %+v", runs)
		}
	})

	t.Run("remainder after fired commands has no clip", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.feed("up down", "up down", "up down banana", "up down banana split", "up down banana split")
		if len(h.exec.calls) == 0 {
			t.Fatal("expected commands to fire")
		}
		runs := h.state.Dictation.ReplayRuns()
		if len(runs) != 1 || len(runs[0].Clips) != 0 {
			t.Fatalf("runs got %+v", runs)
		}
	})
}

func TestStreamStickyAppends(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.state.Dictation.Append("The quick brown fox")
	// Even a pure command chain appends while dictating
	h.feed("left right", "left right")
	if len(h.exec.calls) != 0 {
		t.Fatalf("sticky mode must not execute, got %v", callNames(h.exec.calls))
	}
	if got := h.state.Dictation.Text(); got != "The quick brown fox left right" {
		t.Fatalf("buffer got %q", got)
	}
}

func TestStreamControlWords(t *testing.T) {
	t.Run("standalone insert executes", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.feed("Insert.")
		if callNames(h.exec.calls) != "accept" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("insert inside prose appends", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.feed("insert this word")
		if len(h.exec.calls) != 0 {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if got := h.state.Dictation.Text(); got != "hello world insert this word" {
			t.Fatalf("buffer got %q", got)
		}
	})

	t.Run("scratch that executes", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("scratch that")
		if callNames(h.exec.calls) != "scratch" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("tidy that executes", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("messy. text,with. breaks")
		h.state.SetState(state.StateDictating, true)
		h.feed("Tidy that.")
		if callNames(h.exec.calls) != "tidy" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("replay that executes", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.AppendClip("with the final result", "a.wav")
		h.state.SetState(state.StateDictating, true)
		h.feed("Replay that.")
		if callNames(h.exec.calls) != "replay" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("question that executes", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("is this the right approach.")
		h.state.SetState(state.StateDictating, true)
		h.feed("Question that.")
		if callNames(h.exec.calls) != "question" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("delete word routes to buffer action while dictating", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("delete word")
		if callNames(h.exec.calls) != "delete_word" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if h.exec.calls[0].multiplier != 1 {
			t.Fatalf("multiplier got %d", h.exec.calls[0].multiplier)
		}
	})

	t.Run("delete word with spoken count", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello there big world")
		h.state.SetState(state.StateDictating, true)
		h.feed("Delete word three.")
		if callNames(h.exec.calls) != "delete_word" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if h.exec.calls[0].multiplier != 3 {
			t.Fatalf("multiplier got %d", h.exec.calls[0].multiplier)
		}
	})

	t.Run("send it commits while dictating", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("Send it.")
		if callNames(h.exec.calls) != "send_it" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("send that alias commits while dictating", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("send that")
		if callNames(h.exec.calls) != "send_it" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("send it inside prose appends", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("send it to me tomorrow")
		if len(h.exec.calls) != 0 {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if got := h.state.Dictation.Text(); got != "hello world send it to me tomorrow" {
			t.Fatalf("buffer got %q", got)
		}
	})

	t.Run("delete that routes to char deletion with spoken count", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("Delete that 5.")
		if callNames(h.exec.calls) != "delete_that" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if h.exec.calls[0].multiplier != 5 {
			t.Fatalf("multiplier got %d", h.exec.calls[0].multiplier)
		}
	})

	t.Run("fix that carries the instruction as trailer", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("the piracy of user data matters")
		h.state.SetState(state.StateDictating, true)
		h.feed("Fix that, I was talking about privacy, not piracy.")
		if callNames(h.exec.calls) != "revise" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if got := h.exec.calls[0].trailer; got != "I was talking about privacy, not piracy" {
			t.Fatalf("trailer got %q", got)
		}
	})

	t.Run("adjust that alias carries instruction", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("adjust that make it shorter")
		if callNames(h.exec.calls) != "revise" || h.exec.calls[0].trailer != "make it shorter" {
			t.Fatalf("got %v trailer %q", callNames(h.exec.calls), h.exec.calls[0].trailer)
		}
	})

	t.Run("bare fix that executes with empty trailer", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("fix that")
		if callNames(h.exec.calls) != "revise" || h.exec.calls[0].trailer != "" {
			t.Fatalf("got %v trailer %q", callNames(h.exec.calls), h.exec.calls[0].trailer)
		}
	})

	t.Run("prose not starting with a trigger appends", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("we need to fix that bug soon")
		if len(h.exec.calls) != 0 {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if got := h.state.Dictation.Text(); got != "hello world we need to fix that bug soon" {
			t.Fatalf("buffer got %q", got)
		}
	})

	t.Run("instruction keeps trailing numbers", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("fix that split it into 3")
		if callNames(h.exec.calls) != "revise" || h.exec.calls[0].trailer != "split it into 3" {
			t.Fatalf("got %v trailer %q", callNames(h.exec.calls), h.exec.calls[0].trailer)
		}
		if h.exec.calls[0].multiplier != 1 {
			t.Fatalf("multiplier got %d", h.exec.calls[0].multiplier)
		}
	})

	t.Run("undo routes to buffer history while dictating", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("Undo.")
		if callNames(h.exec.calls) != "undo" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("undo with spoken count", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("undo 2")
		if callNames(h.exec.calls) != "undo" || h.exec.calls[0].multiplier != 2 {
			t.Fatalf("got %v multiplier %d", callNames(h.exec.calls), h.exec.calls[0].multiplier)
		}
	})

	t.Run("redo routes to buffer history while dictating", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("redo")
		if callNames(h.exec.calls) != "redo" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("title that routes to case action while dictating", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("Title that.")
		if callNames(h.exec.calls) != "cap_that" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("delete word without dictating state appends as prose", func(t *testing.T) {
		// Buffer non-empty but state missing: the buffer action is
		// inactive and the keystroke action is not a control method
		h := newStreamHarness(t, defaultCfg())
		h.state.Dictation.Append("hello world")
		h.feed("delete word")
		if len(h.exec.calls) != 0 {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if got := h.state.Dictation.Text(); got != "hello world delete word" {
			t.Fatalf("buffer got %q", got)
		}
	})
}

func TestStreamSalvage(t *testing.T) {
	t.Run("short garbled utterance goes to batch first", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.batch.handled = true
		h.feed("scrol down", "scrol down", "scrol down")
		if len(h.batch.calls) != 1 || h.batch.calls[0] != "fallback:scrol down" {
			t.Fatalf("batch calls: %v", h.batch.calls)
		}
		if h.state.Dictation.HasText() {
			t.Fatal("salvaged utterance must not enter the buffer")
		}
	})

	t.Run("long prose skips batch entirely", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.batch.handled = true
		h.feed("up until now we waited", "up until now we waited", "up until now we waited")
		if len(h.batch.calls) != 0 {
			t.Fatalf("batch must not see long prose: %v", h.batch.calls)
		}
		if !h.state.Dictation.HasText() {
			t.Fatal("expected prose in buffer")
		}
	})

	t.Run("unsalvageable short utterance becomes prose", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.batch.handled = false
		h.feed("blorp", "blorp")
		if len(h.batch.calls) != 1 {
			t.Fatalf("batch calls: %v", h.batch.calls)
		}
		if got := h.state.Dictation.Text(); got != "blorp" {
			t.Fatalf("buffer got %q", got)
		}
	})
}

func TestStreamTrailerHold(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.feed("up", "up up", "up up type", "up up type Hello,", "up up type Hello, World!", "up up type Hello, World!")
	if callNames(h.exec.calls) != "up,up,type" {
		t.Fatalf("got %v", callNames(h.exec.calls))
	}
	last := h.exec.calls[2]
	if last.trailer != "Hello, World!" {
		t.Fatalf("trailer got %q", last.trailer)
	}
}

func TestStreamFinalOnlySession(t *testing.T) {
	t.Run("chain fires in order", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.feed("up down left")
		if callNames(h.exec.calls) != "up,down,left" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("single command fires directly", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.feed("up 3.")
		if callNames(h.exec.calls) != "up" || h.exec.calls[0].multiplier != 3 {
			t.Fatalf("got %+v", h.exec.calls)
		}
	})
}

func TestControlHints(t *testing.T) {
	got := controlHints(streamLookup())
	want := map[string]bool{
		"insert": true, "reject": true, "take": true, "scratch that": true,
		"tidy that": true, "send it": true, "title that": true, "delete word": true,
		"delete that": true, "fix that": true, "undo": true, "redo": true,
		"question that": true, "replay that": true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %d hints %v, want %d", len(got), got, len(want))
	}
	for _, h := range got {
		if !want[h] {
			t.Fatalf("unexpected hint %q in %v", h, got)
		}
	}
}

func TestStreamAccurateFinals(t *testing.T) {
	accurateCfg := func() StreamConfig {
		cfg := defaultCfg()
		cfg.AccurateDictationFinals = true
		return cfg
	}

	t.Run("prose append uses the accurate transcription", func(t *testing.T) {
		h := newStreamHarness(t, accurateCfg())
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "Up until now we waited patiently.", nil
		}
		h.feed("Up until now we wait", "Up until now we waited", "Up until now we waited")
		if got := h.state.Dictation.Text(); got != "Up until now we waited patiently." {
			t.Fatalf("buffer got %q", got)
		}
	})

	t.Run("sticky control word matched on accurate text", func(t *testing.T) {
		h := newStreamHarness(t, accurateCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		// Fast model garbled the commit word; accurate model heard it right
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "send it", nil
		}
		h.feed("sent hit")
		if callNames(h.exec.calls) != "send_it" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
	})

	t.Run("accurate final rescues a garbled command chain", func(t *testing.T) {
		h := newStreamHarness(t, accurateCfg())
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "up down down", nil
		}
		// Three garbled tokens: over the salvage cap, flips to prose
		h.feed("cup town brown", "cup town brown", "cup town brown")
		if callNames(h.exec.calls) != "up,down,down" {
			t.Fatalf("got %v", callNames(h.exec.calls))
		}
		if h.state.Dictation.HasText() {
			t.Fatal("rescued chain must not enter the buffer")
		}
		if h.dismiss == 0 {
			t.Fatal("expected the live panel to be cleared after rescue")
		}
	})

	t.Run("accurate prose still appends, no partial rescue", func(t *testing.T) {
		h := newStreamHarness(t, accurateCfg())
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			// starts with a command word but does not fully parse
			return "up until now we waited", nil
		}
		h.feed("cup until now we waited", "cup until now we waited", "cup until now we waited")
		if len(h.exec.calls) != 0 {
			t.Fatalf("nothing may execute from prose, got %v", callNames(h.exec.calls))
		}
		if got := h.state.Dictation.Text(); got != "up until now we waited" {
			t.Fatalf("buffer got %q", got)
		}
	})

	t.Run("falls back to fast text on error", func(t *testing.T) {
		h := newStreamHarness(t, accurateCfg())
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "", errors.New("accurate server down")
		}
		h.feed("more prose here")
		if got := h.state.Dictation.Text(); got != "hello world more prose here" {
			t.Fatalf("buffer got %q", got)
		}
	})

	t.Run("late prose keeps the fast tail", func(t *testing.T) {
		h := newStreamHarness(t, accurateCfg())
		called := false
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			called = true
			return "WHOLE UTTERANCE", nil
		}
		// "up" fires, then the utterance flips to prose: the accurate text
		// covers the fired part too, so it must not replace the tail
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 42}, Seq: 1, Text: "up down"})
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 42}, Seq: 2, Text: "up down"})
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 42}, Seq: 3, Text: "up down banana"})
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 42}, Seq: 4, Text: "up down banana split"})
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 42}, Seq: 5, Text: "up down banana split", Final: true})
		if called {
			t.Fatal("re-transcription must be skipped after commands fired")
		}
	})

	t.Run("disabled flag never re-transcribes", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			t.Fatal("must not be called when disabled")
			return "", nil
		}
		h.state.Dictation.Append("hello world")
		h.state.SetState(state.StateDictating, true)
		h.feed("more prose")
		if got := h.state.Dictation.Text(); got != "hello world more prose" {
			t.Fatalf("buffer got %q", got)
		}
	})
}

func TestStreamMutedDelegates(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.state.SetState(state.StateNotListening, true)
	h.feed("up down", "up down")
	if len(h.exec.calls) != 0 {
		t.Fatalf("muted must not fire: %v", callNames(h.exec.calls))
	}
	if len(h.batch.calls) != 1 || h.batch.calls[0] != "muted:up down" {
		t.Fatalf("batch calls: %v", h.batch.calls)
	}
}

func TestStreamAbortDropsSession(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 1, Text: "Up until"})
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 2, Text: "Up until now"})
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 3, Final: true})
	if h.state.Dictation.HasText() {
		t.Fatal("aborted session must not append")
	}
	if h.dismiss == 0 {
		t.Fatal("expected the live panel to be cleared")
	}
	if len(h.exec.calls) != 0 || len(h.batch.calls) != 0 {
		t.Fatalf("aborted session must be a no-op: %v %v", h.exec.calls, h.batch.calls)
	}
}

func TestStreamTailDiscardAtFinal(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 3}, Seq: 1, Text: "up down"})
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 3}, Seq: 2, Text: "up down"})
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 3}, Seq: 3, Text: "up down down"})
	if callNames(h.exec.calls) != "up" {
		t.Fatalf("setup: %v", callNames(h.exec.calls))
	}
	// Tail token only appears at Final: discard rather than half-executing
	h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 3}, Seq: 4, Text: "up down down blorp", Final: true})
	if callNames(h.exec.calls) != "up" {
		t.Fatalf("tail must not fire: %v", callNames(h.exec.calls))
	}
	if h.state.Dictation.HasText() {
		t.Fatal("discarded tail must not append")
	}
	found := false
	for _, m := range h.ui {
		if strings.HasPrefix(m, "? ") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected discard notice, got %v", h.ui)
	}
}

func TestStreamDictationDisabled(t *testing.T) {
	cfg := defaultCfg()
	cfg.DictationEnabled = false
	h := newStreamHarness(t, cfg)
	h.feed("Up until now", "Up until now", "Up until now")
	if h.state.Dictation.HasText() {
		t.Fatal("dictation disabled: nothing may enter the buffer")
	}
	// Falls back to batch (today's behavior) instead of prose
	if len(h.batch.calls) != 1 {
		t.Fatalf("batch calls: %v", h.batch.calls)
	}
	if len(h.panels) != 0 {
		t.Fatalf("no panel when dictation disabled: %v", h.panels)
	}
}

func TestOnUtteranceStartFiresOncePerSession(t *testing.T) {
	h := newStreamHarness(t, defaultCfg())
	var starts int
	h.se.OnUtteranceStart = func() { starts++ }

	h.feed("up", "up", "up")
	if starts != 1 {
		t.Fatalf("hook fired %d times for one session, want 1", starts)
	}

	h.feed("down", "down")
	if starts != 2 {
		t.Fatalf("hook fired %d times after two sessions, want 2", starts)
	}
}

func TestAccurateCommandFinals(t *testing.T) {
	cfg := defaultCfg()
	cfg.AccurateCommandFinals = true

	t.Run("accurate text replaces the trailer", func(t *testing.T) {
		h := newStreamHarness(t, cfg)
		var gotPrompt string
		h.se.reTranscribe = func(_ mic.RecordingSession, prompt string) (string, error) {
			gotPrompt = prompt
			return "type bat", nil
		}
		h.feed("type bad")
		if len(h.exec.calls) != 1 || h.exec.calls[0].trailer != "bat" {
			t.Fatalf("calls: %+v", h.exec.calls)
		}
		if gotPrompt == transcribe.ProsePrompt || gotPrompt == "" {
			t.Fatalf("command finals must use the command prompt, got %q", gotPrompt)
		}
	})

	t.Run("unparseable accurate text keeps the fast parse", func(t *testing.T) {
		h := newStreamHarness(t, cfg)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "tie bad", nil
		}
		h.feed("type bad")
		if len(h.exec.calls) != 1 || h.exec.calls[0].trailer != "bad" {
			t.Fatalf("calls: %+v", h.exec.calls)
		}
	})

	t.Run("plain chains never re-transcribe", func(t *testing.T) {
		h := newStreamHarness(t, cfg)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			t.Error("chains without trailer commands must not pay for the accurate model")
			return "", nil
		}
		h.feed("up down", "up down")
		if s := callNames(h.exec.calls); s != "up,down" {
			t.Fatalf("calls: %s", s)
		}
	})

	t.Run("disabled flag never re-transcribes", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			t.Error("reTranscribe called with accurate command finals disabled")
			return "", nil
		}
		h.feed("type bad")
		if len(h.exec.calls) != 1 || h.exec.calls[0].trailer != "bad" {
			t.Fatalf("calls: %+v", h.exec.calls)
		}
	})

	t.Run("mid-utterance fires keep the fast trailer", func(t *testing.T) {
		h := newStreamHarness(t, cfg)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			t.Error("history must not be rewritten after mid-utterance fires")
			return "", nil
		}
		h.feed("up", "up", "up type", "up type bad", "up type bad")
		if s := callNames(h.exec.calls); s != "up,type" {
			t.Fatalf("calls: %s", s)
		}
		if h.exec.calls[1].trailer != "bad" {
			t.Fatalf("trailer: %+v", h.exec.calls[1])
		}
	})
}

// Whisper can re-tokenize INSIDE the fired prefix — a real tiny-model
// session transcribed "up up right" as ticks "up up" then a Final of
// "up-up right.", merging the fired region so the positional remainder
// silently dropped the second "up" (R5). The realign path recovers it.
func TestStreamRealignAfterPrefixRewrite(t *testing.T) {
	fireFirstUp := func(t *testing.T, h *streamHarness) {
		t.Helper()
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 1, Text: "up up"})
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 2, Text: "up up"})
		if callNames(h.exec.calls) != "up" {
			t.Fatalf("setup: expected the first up to fire, got %v", callNames(h.exec.calls))
		}
	}

	t.Run("accurate re-read recovers the swallowed command", func(t *testing.T) {
		cfg := defaultCfg()
		cfg.AccurateCommandFinals = true
		h := newStreamHarness(t, cfg)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "up up right", nil
		}
		fireFirstUp(t, h)
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 3, Text: "up-up right.", Final: true})
		if got := callNames(h.exec.calls); got != "up,up,right" {
			t.Fatalf("got %v, want up,up,right (second up must not be dropped)", got)
		}
		if len(h.batch.calls) != 0 {
			t.Fatalf("batch must not run: %v", h.batch.calls)
		}
	})

	t.Run("holds mid-utterance while the prefix is rewritten", func(t *testing.T) {
		cfg := defaultCfg()
		cfg.AccurateCommandFinals = true
		h := newStreamHarness(t, cfg)
		fireFirstUp(t, h)
		// stable misaligned ticks must not fire the shifted remainder
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 3, Text: "up-up right left"})
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 4, Text: "up-up right left"})
		if got := callNames(h.exec.calls); got != "up" {
			t.Fatalf("misaligned tick fired anyway: %v", got)
		}
	})

	t.Run("realign miss falls back to the positional parse", func(t *testing.T) {
		cfg := defaultCfg()
		cfg.AccurateCommandFinals = true
		h := newStreamHarness(t, cfg)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "banana banana", nil
		}
		fireFirstUp(t, h)
		h.se.handlePartial(asr.Partial{Session: mic.RecordingSession{ID: 9}, Seq: 3, Text: "up-up right.", Final: true})
		// neither candidate parses past the fired step: the pre-realign
		// positional behavior is the floor
		if got := callNames(h.exec.calls); got != "up,right" {
			t.Fatalf("got %v, want up,right", got)
		}
	})
}

// A parse that breaks at Final (nothing fired) gets one accurate re-read
// before conceding to the batch pipeline — the comma-glued "move,cloudcode"
// case, which normalizeStream keeps as a single unmatchable token.
func TestAccurateFallbackAtFinal(t *testing.T) {
	cfg := defaultCfg()
	cfg.AccurateCommandFinals = true

	t.Run("comma-glued trailer is rescued by the accurate re-read", func(t *testing.T) {
		h := newStreamHarness(t, cfg)
		var gotPrompt string
		h.se.reTranscribe = func(_ mic.RecordingSession, prompt string) (string, error) {
			gotPrompt = prompt
			return "type cloud code", nil
		}
		h.feed("type,cloudcode")
		if s := callNames(h.exec.calls); s != "type" {
			t.Fatalf("calls: %s", s)
		}
		if h.exec.calls[0].trailer != "cloud code" {
			t.Fatalf("trailer: %+v", h.exec.calls[0])
		}
		if len(h.batch.calls) != 0 {
			t.Fatalf("batch pipeline must not run when the accurate parse succeeds: %v", h.batch.calls)
		}
		if gotPrompt != commandFinalPrompt {
			t.Fatalf("rescue must use the command prompt, got %q", gotPrompt)
		}
	})

	t.Run("unparseable accurate text falls through with the fast text", func(t *testing.T) {
		h := newStreamHarness(t, cfg)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "still gibberish", nil
		}
		h.feed("type,cloudcode")
		if len(h.exec.calls) != 0 {
			t.Fatalf("calls: %+v", h.exec.calls)
		}
		if len(h.batch.calls) != 1 || h.batch.calls[0] != "fallback:type,cloudcode" {
			t.Fatalf("batch calls: %v", h.batch.calls)
		}
		// prose keeps the original fast text, not the command-biased re-read
		if got := h.state.Dictation.Text(); got != "type,cloudcode" {
			t.Fatalf("dictation buffer: %q", got)
		}
	})

	t.Run("partially-parsing accurate text does not execute", func(t *testing.T) {
		h := newStreamHarness(t, cfg)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "up banana", nil
		}
		h.feed("up,banana")
		if len(h.exec.calls) != 0 {
			t.Fatalf("a partial parse must not execute its prefix: %+v", h.exec.calls)
		}
		if len(h.batch.calls) != 1 {
			t.Fatalf("batch calls: %v", h.batch.calls)
		}
	})

	t.Run("re-read failure still reaches the batch pipeline", func(t *testing.T) {
		h := newStreamHarness(t, cfg)
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			return "", errors.New("server down")
		}
		h.feed("type,cloudcode")
		if len(h.batch.calls) != 1 {
			t.Fatalf("batch calls: %v", h.batch.calls)
		}
	})

	t.Run("disabled flag skips the re-read entirely", func(t *testing.T) {
		h := newStreamHarness(t, defaultCfg())
		h.se.reTranscribe = func(mic.RecordingSession, string) (string, error) {
			t.Error("reTranscribe called with accurate command finals disabled")
			return "", nil
		}
		h.feed("type,cloudcode")
		if len(h.batch.calls) != 1 {
			t.Fatalf("batch calls: %v", h.batch.calls)
		}
	})
}
