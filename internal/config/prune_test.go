package config

import (
	"slices"
	"testing"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/types"
)

func TestPruneUnhandled(t *testing.T) {
	cfg := &Config{
		Settings: Settings{CategoryOrder: []string{"mouse", "ai"}},
		RawCommands: map[string]map[string]commands.Command{
			"mouse": {
				// optional branch: the missing action has a skip callback
				"click": {
					Actions: []*commands.Action{
						{ID: "ocr", Method: "ocr_click", RequiresTrailer: true, Args: map[string]any{"prompt": "?"}},
					},
					ActionCallbacks: []*commands.Action{
						{For: "ocr", Result: commands.ResultSkip, Method: "mouse_click"},
					},
				},
				// fallback-only: the primary action is available
				"move": {
					Actions: []*commands.Action{
						{ID: "ocr", Method: "mouse_click"},
					},
					ActionCallbacks: []*commands.Action{
						{For: "ocr", Result: commands.ResultSkip, Method: "gaze_move"},
						{For: "ocr", Result: commands.ResultFail, Method: "gaze_move"},
					},
				},
			},
			"ai": {
				"tidy": {Actions: []*commands.Action{{Method: "ai_tidy"}}},
			},
		},
	}
	has := func(n commands.CommandName) bool { return n == "mouse_click" }

	changes := cfg.PruneUnhandled(has)

	want := []string{
		"ai.tidy (dropped: ai_tidy)",
		"mouse.click (action ocr_click -> skip)",
		"mouse.move (callback gaze_move -> __continue__)",
		"mouse.move (callback gaze_move -> __fail__)",
	}
	if !slices.Equal(changes, want) {
		t.Fatalf("changes = %q, want %q", changes, want)
	}
	if _, ok := cfg.RawCommands["ai"]; ok {
		t.Error("empty category ai kept")
	}
	if !slices.Equal(cfg.Settings.CategoryOrder, []string{"mouse"}) {
		t.Errorf("CategoryOrder = %v", cfg.Settings.CategoryOrder)
	}

	click := cfg.RawCommands["mouse"]["click"]
	if click.Actions[0].IsActiveInState(map[types.ExecutionState]bool{}) {
		t.Error("skipped action is still active")
	}
	if click.ConsumesTrailer() || click.MayPrompt() {
		t.Error("skipped action still claims the trailer or a prompt")
	}
	if cb := click.FindCallback("ocr", commands.ResultSkip); cb == nil || cb.Method != "mouse_click" {
		t.Errorf("skip callback = %v, want mouse_click", cb)
	}
}
