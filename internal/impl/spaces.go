package impl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/applescript"
	"github.com/mousybusiness/mouse_whisperer/pkg/spaces"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	"github.com/lithammer/fuzzysearch/fuzzy"
	log "github.com/sirupsen/logrus"
)

// spaceHelperPath is relative to the repo root, same convention as screend's
// -helper flag default (cmd/screend/main.go).
const spaceHelperPath = "build/space-helper"

// osascript key codes for Ctrl+Left/Right space switching, matching the
// screen_left/screen_right commands in config/mac.toml.
const (
	spaceKeyCodeLeft  = 123
	spaceKeyCodeRight = 124
)

func RegisterWorkspaceHandlers(registry Registry, lstate *state.State) {
	registry.Register("workspace_alias", func(ctx context.Context, args map[string]any) (context.Context, error) {
		ctx, raw, err := captureInput(ctx, lstate, &commands.InputAction{
			Prompt:  "A workspace name e.g. whatsapp, work, private. output: ",
			Break:   750 * time.Millisecond,
			Timeout: 4 * time.Second,
		})
		if err != nil {
			return ctx, fmt.Errorf("failed to capture input: %v", err)
		}

		name := normalizeWorkspaceName(raw)
		if name == "" {
			overlay.SendUIMessage("No workspace name heard")
			return ctx, fmt.Errorf("no workspace name provided")
		}

		snap, err := spaces.Query(spaceHelperPath)
		if err != nil {
			overlay.SendUIMessage("Couldn't read spaces")
			return ctx, err
		}

		lstate.Desktop.SetWorkspaceAlias(name, snap.ActiveID)
		overlay.SendUIMessage(fmt.Sprintf("Screen %d is now '%s'", snap.ActiveIndex(), name))
		log.Infof("Aliased space %d (screen %d) as %q", snap.ActiveID, snap.ActiveIndex(), name)
		return ctx, nil
	})

	registry.Register("workspace_goto", func(ctx context.Context, args map[string]any) (context.Context, error) {
		ctx, raw, err := captureInput(ctx, lstate, &commands.InputAction{
			Prompt:  "An aliased workspace name. output: ",
			Break:   750 * time.Millisecond,
			Timeout: 4 * time.Second,
		})
		if err != nil {
			return ctx, fmt.Errorf("failed to capture input: %v", err)
		}

		name := normalizeWorkspaceName(raw)
		if name == "" {
			overlay.SendUIMessage("No workspace name heard")
			return ctx, fmt.Errorf("no workspace name provided")
		}

		id, ok := lstate.Desktop.WorkspaceAlias(name)
		if !ok {
			// Absorb transcription slop ("what app" -> "whatsapp")
			if matches := fuzzy.RankFindFold(name, lstate.Desktop.WorkspaceAliasNames()); len(matches) > 0 {
				log.Infof("Fuzzy matched workspace %q -> %q", name, matches[0].Target)
				name = matches[0].Target
				id, ok = lstate.Desktop.WorkspaceAlias(name)
			}
		}
		if !ok {
			known := strings.Join(lstate.Desktop.WorkspaceAliasNames(), ", ")
			if known == "" {
				known = "none yet - say 'alias workspace <name>'"
			}
			overlay.SendUIMessage(fmt.Sprintf("No workspace '%s' (%s)", name, known))
			return ctx, fmt.Errorf("unknown workspace alias: %s", name)
		}

		return ctx, gotoSpaceID(id, name)
	})

	registry.Register("workspace_goto_index", func(ctx context.Context, args map[string]any) (context.Context, error) {
		target := intArg(args, "index")
		if target <= 0 {
			return ctx, fmt.Errorf("workspace_goto_index: index argument is required")
		}

		snap, err := spaces.Query(spaceHelperPath)
		if err != nil {
			overlay.SendUIMessage("Couldn't read spaces")
			return ctx, err
		}
		if target > len(snap.Spaces) {
			overlay.SendUIMessage(fmt.Sprintf("No screen %d (only %d)", target, len(snap.Spaces)))
			return ctx, fmt.Errorf("no screen %d, have %d", target, len(snap.Spaces))
		}

		return ctx, walkToSpace(snap.ActiveIndex(), target)
	})
}

// gotoSpaceID re-resolves the space's current position (spaces get reordered)
// and walks there.
func gotoSpaceID(id uint64, name string) error {
	snap, err := spaces.Query(spaceHelperPath)
	if err != nil {
		overlay.SendUIMessage("Couldn't read spaces")
		return err
	}

	target := snap.IndexOf(id)
	if target == 0 {
		overlay.SendUIMessage(fmt.Sprintf("Workspace '%s' no longer exists", name))
		return fmt.Errorf("space %d for alias %q no longer exists", id, name)
	}

	if err := walkToSpace(snap.ActiveIndex(), target); err != nil {
		return err
	}
	overlay.SendUIMessage(fmt.Sprintf("Screen %d: %s", target, name))
	return nil
}

// walkToSpace moves between spaces with Ctrl+Left/Right, one hop at a time.
// Mission Control's direct Ctrl+N shortcuts are off by default, so walking is
// the only path that needs no System Settings changes.
func walkToSpace(current, target int) error {
	if current == 0 {
		return fmt.Errorf("could not determine current space")
	}

	delta := target - current
	keyCode := spaceKeyCodeRight
	if delta < 0 {
		keyCode = spaceKeyCodeLeft
		delta = -delta
	}
	for i := 0; i < delta; i++ {
		if err := applescript.ExecuteOsascript(commands.KeyAction{
			KeyCode:   keyCode,
			Modifiers: []string{"control"},
		}); err != nil {
			return err
		}
		// Let the switch animation start so the next hop isn't swallowed
		robotgo.MilliSleep(250)
	}
	return nil
}

// normalizeWorkspaceName collapses a transcription to a comparable key:
// "What's App" and "whats app" both become "whatsapp".
func normalizeWorkspaceName(s string) string {
	s = util.ToAlphanumericWithSpace(strings.ToLower(strings.TrimSpace(s)))
	return strings.ReplaceAll(s, " ", "")
}

func intArg(args map[string]any, key string) int {
	switch v := args[key].(type) {
	case int64:
		return int(v)
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}
