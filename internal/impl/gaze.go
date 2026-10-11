package impl

import (
	"context"
	"fmt"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	localstate "github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/eye_tracking"
)

// RegisterGazeHandlers registers the mouse commands that only make sense
// with an eye tracker: gaze moves, follow modes, ghost and tunnel vision.
func RegisterGazeHandlers(registry Registry, state *localstate.State) {
	registry.Register("mouse_snapping", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.SetSnapping(!state.GetSnapping())
		return ctx, nil
	})

	registry.Register("mouse_nudging", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.SetNudging(!state.GetNudging())
		return ctx, nil
	})

	registry.Register("mouse_jumping", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.SetJumping(!state.GetJumping())
		return ctx, nil
	})

	registry.Register("mouse_ghost", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.SetGhosting(!state.GetGhosting())
		overlay.SendGhostVisibility(state.GetGhosting())
		if state.GetGhosting() {
			state.SetState(localstate.StateGhost, true)
			overlay.SendUIMessage("Ghost ON")
		} else {
			state.SetState(localstate.StateGhost, false)
			overlay.SendUIMessage("Ghost OFF")
		}
		return ctx, nil
	})

	registry.Register("mouse_tunnel_vision", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.SetTunnelVision(!state.GetTunnelVision())
		// Default values: radius=600, opacity=0.85, blendDistance=0.5
		overlay.SendTunnelVision(state.GetTunnelVision(), 600, 0.85, 0.5)
		if state.GetTunnelVision() {
			overlay.SendUIMessage("Tunnel vision ON")
		} else {
			overlay.SendUIMessage("Tunnel vision OFF")
		}
		return ctx, nil
	})

	registry.Register("mouse_move", func(ctx context.Context, args map[string]any) (context.Context, error) {
		eyePos, ok := eye_tracking.GetPositionAtTime(state.GetRecordingStarted())
		if !ok {
			return ctx, fmt.Errorf("no eye position found at recording start")
		}
		robotgo.Move(int(eyePos.X), int(eyePos.Y))
		return ctx, nil
	})

	registry.Register("mouse_snap", func(ctx context.Context, args map[string]any) (context.Context, error) {
		eyePos, ok := eye_tracking.GetPositionAtTime(state.GetRecordingStarted())
		if !ok {
			return ctx, fmt.Errorf("no eye position found at recording start")
		}
		scaledV := eyePos.Mul(robotgo.ScaleF(robotgo.DisplayID))
		result, ok := state.Screen.FindNearest(scaledV)
		if !ok {
			result = eyePos
		}
		robotgo.Move(int(result.X), int(result.Y))
		return ctx, nil
	})

	registry.Register("mouse_move_click", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Move to ghost position and click (only works in ghost mode)
		eyePos, ok := eye_tracking.GetPositionAtTime(state.GetRecordingStarted())
		if !ok {
			return ctx, fmt.Errorf("no eye position found at recording start")
		}
		robotgo.Move(int(eyePos.X), int(eyePos.Y))
		robotgo.MilliSleep(50)
		robotgo.Click("left")
		return ctx, nil
	})

	// Jump is same as snap for now
	registry.Register("mouse_jump", func(ctx context.Context, args map[string]any) (context.Context, error) {
		eyePos, ok := eye_tracking.GetPositionAtTime(state.GetRecordingStarted())
		if !ok {
			return ctx, fmt.Errorf("no eye position found at recording start")
		}
		robotgo.Move(int(eyePos.X), int(eyePos.Y))
		return ctx, nil
	})

	// Follow mode toggles
	registry.Register("mouse_quick_follow", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.SetJumping(true)
		state.SetSnapping(false)
		state.SetNudging(false)
		overlay.SendUIMessage("Quick follow enabled")
		return ctx, nil
	})

	registry.Register("mouse_smooth_follow", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.SetNudging(true)
		state.SetSnapping(false)
		state.SetJumping(false)
		overlay.SendUIMessage("Smooth follow enabled")
		return ctx, nil
	})

	registry.Register("mouse_follow", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.SetSnapping(true)
		state.SetJumping(false)
		state.SetNudging(false)
		overlay.SendUIMessage("Follow enabled")
		return ctx, nil
	})

	// Nudge handler (snap to eye position with nudge mode active)
	registry.Register("mouse_nudge", func(ctx context.Context, args map[string]any) (context.Context, error) {
		eyePos, ok := eye_tracking.GetPositionAtTime(state.GetRecordingStarted())
		if !ok {
			return ctx, fmt.Errorf("no eye position found at recording start")
		}
		robotgo.Move(int(eyePos.X), int(eyePos.Y))
		state.SetNudging(true)
		return ctx, nil
	})
}
