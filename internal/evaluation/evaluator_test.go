package evaluation

import (
	"errors"
	"strings"
	"testing"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/types"
)

func testLookup() *commands.CommandLookup {
	raw := map[string]map[string]commands.Command{
		"text": {
			"up":        {Triggers: []string{"up"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"down":      {Triggers: []string{"down"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"left":      {Triggers: []string{"left"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"right":     {Triggers: []string{"right"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"left_word": {Triggers: []string{"left word"}, Multiplier: true, Actions: []*commands.Action{{Method: "keybd_event"}}},
			"type":      {Triggers: []string{"type"}, Actions: []*commands.Action{{Method: "type_str", RequiresTrailer: true}}},
			"pick":      {Triggers: []string{"pick"}, Actions: []*commands.Action{{Method: "hint_pick"}}},
		},
	}
	return commands.NewCommandLookup(raw, []string{"text"})
}

type executedCall struct {
	name       commands.CommandName
	multiplier int
	trailer    string
}

type fakeExecutor struct {
	calls  []executedCall
	failOn int // 1-based index of the call that should fail, 0 = never
}

func (f *fakeExecutor) ExecuteCommand(entry *commands.CommandEntry, triggerWords []string, trailerWords []string, trailer commands.TrailerText, multiplier int, invocationSource string) error {
	f.calls = append(f.calls, executedCall{name: entry.Name, multiplier: multiplier, trailer: string(trailer)})
	if f.failOn == len(f.calls) {
		return errors.New("handler failed")
	}
	return nil
}

func testEvaluator(failOn int) (*Evaluator, *fakeExecutor) {
	fake := &fakeExecutor{failOn: failOn}
	e := &Evaluator{
		lookup:   testLookup(),
		executor: fake,
		state:    &state.State{Global: &state.GlobalState{}},
	}
	return e, fake
}

func steps(t *testing.T, input string) []sequenceStep {
	t.Helper()
	return splitSequence(testLookup(), tokenize(input), map[types.ExecutionState]bool{})
}

func triggers(s []sequenceStep) []string {
	var out []string
	for _, step := range s {
		out = append(out, strings.Join(step.triggerWords, " "))
	}
	return out
}

func names(calls []executedCall) string {
	var out []string
	for _, c := range calls {
		out = append(out, string(c.name))
	}
	return strings.Join(out, ",")
}

func TestTokenize(t *testing.T) {
	t.Run("strips punctuation and case for matching, keeps raw", func(t *testing.T) {
		got := tokenize("Up, up. Down!")
		if len(got) != 3 {
			t.Fatalf("expected 3 tokens, got %d: %+v", len(got), got)
		}
		wantClean := []string{"up", "up", "down"}
		wantRaw := []string{"Up,", "up.", "Down!"}
		for i := range got {
			if got[i].clean != wantClean[i] {
				t.Errorf("token %d clean: got %q, want %q", i, got[i].clean, wantClean[i])
			}
			if got[i].raw != wantRaw[i] {
				t.Errorf("token %d raw: got %q, want %q", i, got[i].raw, wantRaw[i])
			}
		}
	})

	t.Run("drops pure punctuation tokens", func(t *testing.T) {
		if got := tokenize("... - up"); len(got) != 1 || got[0].clean != "up" {
			t.Fatalf("expected single 'up' token, got %+v", got)
		}
	})

	t.Run("keeps interior punctuation", func(t *testing.T) {
		got := tokenize("readme.md")
		if len(got) != 1 || got[0].clean != "readme.md" {
			t.Fatalf("expected 'readme.md', got %+v", got)
		}
	})

	t.Run("cleans numbers", func(t *testing.T) {
		got := tokenize("3.")
		if len(got) != 1 || got[0].clean != "3" {
			t.Fatalf("expected '3', got %+v", got)
		}
	})
}

func TestSplitSequence(t *testing.T) {
	t.Run("repeated directions", func(t *testing.T) {
		got := steps(t, "up up up down left right")
		want := []string{"up", "up", "up", "down", "left", "right"}
		if strings.Join(triggers(got), ",") != strings.Join(want, ",") {
			t.Fatalf("got %v, want %v", triggers(got), want)
		}
		for _, s := range got {
			if s.multiplier != 1 {
				t.Fatalf("expected multiplier 1, got %d", s.multiplier)
			}
		}
	})

	t.Run("whisper punctuation and capitalization", func(t *testing.T) {
		got := steps(t, "Up, up, up. Down! Left, right.")
		want := []string{"up", "up", "up", "down", "left", "right"}
		if strings.Join(triggers(got), ",") != strings.Join(want, ",") {
			t.Fatalf("got %v, want %v", triggers(got), want)
		}
	})

	t.Run("per command multiplier", func(t *testing.T) {
		got := steps(t, "up 3 down 2 left")
		if len(got) != 3 {
			t.Fatalf("expected 3 steps, got %v", triggers(got))
		}
		if got[0].multiplier != 3 || got[1].multiplier != 2 || got[2].multiplier != 1 {
			t.Fatalf("wrong multipliers: %d %d %d", got[0].multiplier, got[1].multiplier, got[2].multiplier)
		}
	})

	t.Run("multiplier with punctuation", func(t *testing.T) {
		got := steps(t, "Up 3, down 2.")
		if len(got) != 2 || got[0].multiplier != 3 || got[1].multiplier != 2 {
			t.Fatalf("got %v with %+v", triggers(got), got)
		}
	})

	t.Run("spoken number multiplier", func(t *testing.T) {
		got := steps(t, "up three down")
		if len(got) != 2 || got[0].multiplier != 3 {
			t.Fatalf("got %v with multipliers %+v", triggers(got), got)
		}
	})

	t.Run("longest trigger wins", func(t *testing.T) {
		got := steps(t, "left word up")
		want := []string{"left word", "up"}
		if strings.Join(triggers(got), ",") != strings.Join(want, ",") {
			t.Fatalf("got %v, want %v", triggers(got), want)
		}
	})

	t.Run("unmatched residue aborts", func(t *testing.T) {
		if got := steps(t, "up banana"); got != nil {
			t.Fatalf("expected nil, got %v", triggers(got))
		}
	})

	t.Run("trailer command consumes remainder", func(t *testing.T) {
		got := steps(t, "up up type hello world")
		if len(got) != 3 {
			t.Fatalf("expected 3 steps, got %v", triggers(got))
		}
		if string(got[2].trailer) != "hello world" {
			t.Fatalf("expected trailer 'hello world', got %q", got[2].trailer)
		}
	})

	t.Run("trailer keeps original case and punctuation", func(t *testing.T) {
		got := steps(t, "up up Type Hello, World!")
		if len(got) != 3 {
			t.Fatalf("expected 3 steps, got %v", triggers(got))
		}
		if string(got[2].trailer) != "Hello, World!" {
			t.Fatalf("expected trailer 'Hello, World!', got %q", got[2].trailer)
		}
	})

	t.Run("leading trailer command keeps current behaviour", func(t *testing.T) {
		// One step only, so EvaluateSequence declines and the backwards
		// matcher handles it as before
		got := steps(t, "type up down")
		if len(got) != 1 {
			t.Fatalf("expected 1 step, got %v", triggers(got))
		}
	})

	t.Run("number not consumed without multiplier support", func(t *testing.T) {
		if got := steps(t, "up pick 3"); got != nil {
			t.Fatalf("expected nil, got %v", triggers(got))
		}
	})
}

func TestEvaluateSequence(t *testing.T) {
	t.Run("executes whisper style utterance in order", func(t *testing.T) {
		e, fake := testEvaluator(0)
		handled, err := e.EvaluateSequence("Up, up, up, down, left, right.", "test")
		if err != nil || !handled {
			t.Fatalf("handled=%v err=%v", handled, err)
		}
		if got, want := names(fake.calls), "up,up,up,down,left,right"; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	})

	t.Run("all caps", func(t *testing.T) {
		e, fake := testEvaluator(0)
		handled, err := e.EvaluateSequence("UP DOWN", "test")
		if err != nil || !handled {
			t.Fatalf("handled=%v err=%v", handled, err)
		}
		if got, want := names(fake.calls), "up,down"; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	})

	t.Run("multipliers with punctuation", func(t *testing.T) {
		e, fake := testEvaluator(0)
		handled, err := e.EvaluateSequence("Up 3, down 2.", "test")
		if err != nil || !handled {
			t.Fatalf("handled=%v err=%v", handled, err)
		}
		if len(fake.calls) != 2 || fake.calls[0].multiplier != 3 || fake.calls[1].multiplier != 2 {
			t.Fatalf("got calls %+v", fake.calls)
		}
	})

	t.Run("single command declines", func(t *testing.T) {
		e, fake := testEvaluator(0)
		handled, err := e.EvaluateSequence("up.", "test")
		if err != nil || handled {
			t.Fatalf("handled=%v err=%v", handled, err)
		}
		if len(fake.calls) != 0 {
			t.Fatalf("expected no calls, got %+v", fake.calls)
		}
	})

	t.Run("unknown word declines without executing", func(t *testing.T) {
		e, fake := testEvaluator(0)
		handled, err := e.EvaluateSequence("up banana", "test")
		if err != nil || handled {
			t.Fatalf("handled=%v err=%v", handled, err)
		}
		if len(fake.calls) != 0 {
			t.Fatalf("expected no calls, got %+v", fake.calls)
		}
	})

	t.Run("failure stops the sequence", func(t *testing.T) {
		e, fake := testEvaluator(2)
		handled, err := e.EvaluateSequence("up down left", "test")
		if !handled || err == nil {
			t.Fatalf("handled=%v err=%v", handled, err)
		}
		if len(fake.calls) != 2 {
			t.Fatalf("expected 2 calls before stopping, got %+v", fake.calls)
		}
	})

	t.Run("trailer command receives trailer", func(t *testing.T) {
		e, fake := testEvaluator(0)
		handled, err := e.EvaluateSequence("up up type hello world", "test")
		if err != nil || !handled {
			t.Fatalf("handled=%v err=%v", handled, err)
		}
		if len(fake.calls) != 3 || fake.calls[2].trailer != "hello world" {
			t.Fatalf("got calls %+v", fake.calls)
		}
	})
}
