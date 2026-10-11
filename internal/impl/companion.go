package impl

import (
	"context"
	"fmt"

	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
)

func RegisterCompanionHandlers(registry Registry, state *state.State) {
	registry.Register("companion_mouse_sensitivity_up", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		change := 0.2 * float64(multiplier)
		newVal := state.GetCompanionMouseSensitivity() + change
		state.SetCompanionMouseSensitivity(newVal)
		overlay.SendUIMessage(fmt.Sprintf("Companion Sensitivity: %.1f", newVal))
		return ctx, nil
	})

	registry.Register("companion_mouse_sensitivity_down", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		change := 0.2 * float64(multiplier)
		newVal := state.GetCompanionMouseSensitivity() - change
		if newVal < 0.2 {
			newVal = 0.2
		}
		state.SetCompanionMouseSensitivity(newVal)
		overlay.SendUIMessage(fmt.Sprintf("Companion Sensitivity: %.1f", newVal))
		return ctx, nil
	})
}
