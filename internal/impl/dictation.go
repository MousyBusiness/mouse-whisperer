package impl

import (
	"context"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	localstate "github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	log "github.com/sirupsen/logrus"
)

// RegisterDictationHandlers registers the dictation buffer and suggestion
// commit handlers (accept/take/reject, scratch, undo, case, punctuation);
// none of them need an LLM
func RegisterDictationHandlers(registry Registry, state *localstate.State) {
	registry.Register("ai_clip", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Dictation buffer takes priority over an AI suggestion; "take" is
		// a commit: copy and clear
		if state.Dictation.HasText() {
			text := state.Dictation.TakeAll()
			state.SetState(localstate.StateDictating, false)
			overlay.DismissDictation()
			if err := util.WriteTextToClipboard(text); err != nil {
				log.Error(err)
				return ctx, err
			}
			overlay.ShowToast("Success", "Copied to clipboard!", "info")
			return ctx, nil
		}

		defer overlay.DismissAIResponse()
		if state.Coding.PendingSuggestion != nil {
			err := util.WriteTextToClipboard(*state.Coding.PendingSuggestion)
			if err != nil {
				log.Error(err)
				return ctx, err
			}
			overlay.ShowToast("Success", "Copied to clipboard!", "info")
		}
		return ctx, nil
	})

	registry.Register("ai_accept", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Dictation buffer takes priority over an AI suggestion
		if state.Dictation.HasText() {
			text := state.Dictation.TakeAll()
			state.SetState(localstate.StateDictating, false)
			overlay.DismissDictation()
			robotgo.Type(text)
			return ctx, nil
		}

		defer overlay.DismissAIResponse()
		if state.Coding.PendingSuggestion != nil {
			robotgo.Type(*state.Coding.PendingSuggestion)
			return ctx, nil
		}
		// Accept inline IDE suggestion
		robotgo.KeyTap("enter", "lcmd")
		return ctx, nil
	})

	registry.Register("ai_reject", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Dictation buffer takes priority over an AI suggestion
		if state.Dictation.HasText() {
			state.Dictation.Clear()
			state.SetState(localstate.StateDictating, false)
			overlay.DismissDictation()
			return ctx, nil
		}

		defer overlay.DismissAIResponse()
		if state.Coding.PendingSuggestion != nil {
			state.Coding.PendingSuggestion = nil
			return ctx, nil
		}
		// Reject inline IDE suggestion
		robotgo.KeyTap("escape")
		return ctx, nil
	})

	registry.Register("dictation_scratch", func(ctx context.Context, args map[string]any) (context.Context, error) {
		if _, ok := state.Dictation.PopLast(); !ok {
			return ctx, nil
		}
		if state.Dictation.HasText() {
			overlay.SendDictation(state.Dictation.Text(), "", nil)
			return ctx, nil
		}
		state.SetState(localstate.StateDictating, false)
		overlay.DismissDictation()
		return ctx, nil
	})

	registry.Register("dictation_delete_char", func(ctx context.Context, args map[string]any) (context.Context, error) {
		n := 1
		if ex := xctx.Get(ctx); ex != nil && ex.Multiplier > 1 {
			n = ex.Multiplier
		}
		if !state.Dictation.PopChars(n) {
			return ctx, nil
		}
		if state.Dictation.HasText() {
			overlay.SendDictation(state.Dictation.Text(), "", nil)
			return ctx, nil
		}
		state.SetState(localstate.StateDictating, false)
		overlay.DismissDictation()
		return ctx, nil
	})

	registry.Register("dictation_delete_word", func(ctx context.Context, args map[string]any) (context.Context, error) {
		n := 1
		if ex := xctx.Get(ctx); ex != nil && ex.Multiplier > 1 {
			n = ex.Multiplier
		}
		if !state.Dictation.PopWords(n) {
			return ctx, nil
		}
		if state.Dictation.HasText() {
			overlay.SendDictation(state.Dictation.Text(), "", nil)
			return ctx, nil
		}
		state.SetState(localstate.StateDictating, false)
		overlay.DismissDictation()
		return ctx, nil
	})

	registry.Register("dictation_yolo", func(ctx context.Context, args map[string]any) (context.Context, error) {
		if !state.Dictation.HasText() {
			return ctx, nil
		}
		text := state.Dictation.TakeAll()
		state.SetState(localstate.StateDictating, false)
		overlay.DismissDictation()
		robotgo.Type(text)
		// Give the target app a beat to register the text before submitting
		time.Sleep(100 * time.Millisecond)
		robotgo.KeyTap("enter")
		return ctx, nil
	})

	// refreshDictationPanel applies the StateDictating <=> buffer-non-empty
	// invariant after history navigation
	refreshDictationPanel := func() {
		if state.Dictation.HasText() {
			state.SetState(localstate.StateDictating, true)
			overlay.SendDictation(state.Dictation.Text(), "", nil)
			return
		}
		state.SetState(localstate.StateDictating, false)
		overlay.DismissDictation()
	}

	registry.Register("dictation_undo", func(ctx context.Context, args map[string]any) (context.Context, error) {
		n := 1
		if ex := xctx.Get(ctx); ex != nil && ex.Multiplier > 1 {
			n = ex.Multiplier
		}
		changed := false
		for i := 0; i < n && state.Dictation.Undo(); i++ {
			changed = true
		}
		if changed {
			refreshDictationPanel()
		}
		return ctx, nil
	})

	registry.Register("dictation_redo", func(ctx context.Context, args map[string]any) (context.Context, error) {
		n := 1
		if ex := xctx.Get(ctx); ex != nil && ex.Multiplier > 1 {
			n = ex.Multiplier
		}
		changed := false
		for i := 0; i < n && state.Dictation.Redo(); i++ {
			changed = true
		}
		if changed {
			refreshDictationPanel()
		}
		return ctx, nil
	})

	registry.Register("dictation_case", func(ctx context.Context, args map[string]any) (context.Context, error) {
		upper := true
		if v, ok := args["case"].(string); ok && v == "lower" {
			upper = false
		}
		if !state.Dictation.SetLeadingCase(upper) {
			return ctx, nil
		}
		overlay.SendDictation(state.Dictation.Text(), "", nil)
		return ctx, nil
	})

	registry.Register("dictation_punct", func(ctx context.Context, args map[string]any) (context.Context, error) {
		mark := "?"
		if v, ok := args["mark"].(string); ok && v != "" {
			mark = v
		}
		if !state.Dictation.SetTrailingPunct(mark) {
			return ctx, nil
		}
		overlay.SendDictation(state.Dictation.Text(), "", nil)
		return ctx, nil
	})
}
