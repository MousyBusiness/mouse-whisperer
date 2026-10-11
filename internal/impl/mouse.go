package impl

import (
	"context"
	"fmt"
	"image"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	localstate "github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/eye_tracking"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	log "github.com/sirupsen/logrus"
)

func RegisterMouseHandlers(registry Registry, state *localstate.State) {
	registry.Register("screenshot", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Determine dimensions based on variant argument
		variant, _ := args["variant"].(string)
		var width, height int

		switch variant {
		case "tall":
			width = state.CV.ScreenshotTallWidth
			height = state.CV.ScreenshotTallHeight
		case "wide":
			width = state.CV.ScreenshotWideWidth
			height = state.CV.ScreenshotWideHeight
		default:
			width = state.CV.ScreenshotWidth
			height = state.CV.ScreenshotHeight
		}

		// Apply defaults if not configured
		if width <= 0 {
			width = 600
		}
		if height <= 0 {
			height = 600
		}

		// Get position - try eye position first, fall back to mouse
		var x, y int
		if state.GetGhosting() {
			// Use eye position if ghosting is active
			eyePos, ok := eye_tracking.GetPositionAtTime(state.GetRecordingStarted())
			if ok {
				x, y = int(eyePos.X), int(eyePos.Y)
			} else {
				x, y = robotgo.Location()
			}
		} else {
			// Use mouse position
			x, y = robotgo.Location()
		}

		// Capture the region
		img, _, err := state.Screen.CaptureRegionForOCR(x, y, width, height)
		if err != nil {
			return ctx, fmt.Errorf("failed to capture screenshot: %w", err)
		}

		// Save to temp file and copy to clipboard
		if err := copyImageToClipboard(img); err != nil {
			return ctx, fmt.Errorf("failed to copy screenshot to clipboard: %w", err)
		}

		overlay.SendUIMessage("Screenshot copied!")
		log.Infof("Screenshot (%s): %dx%d at (%d, %d) - copied to clipboard", variant, width, height, x, y)
		return ctx, nil
	})

	registry.Register("mouse_click", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.Click("left")
		return ctx, nil
	})

	registry.Register("mouse_double_click", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.Click("left", true)
		return ctx, nil
	})

	registry.Register("mouse_right_click", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.Click("right")
		return ctx, nil
	})

	registry.Register("mouse_select_click", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.KeyDown("shift")
		robotgo.MilliSleep(100)
		robotgo.Click("left")
		robotgo.MilliSleep(100)
		robotgo.KeyUp("shift")
		return ctx, nil
	})

	registry.Register("mouse_command_click", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.KeyDown("lcmd")
		robotgo.MilliSleep(100)
		robotgo.Click("left")
		robotgo.MilliSleep(100)
		robotgo.KeyUp("lcmd")
		return ctx, nil
	})

	// mouse_tap is an alias for mouse_click (simple left click)
	registry.Register("mouse_tap", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.Click("left")
		return ctx, nil
	})

	// mouse_down holds the left mouse button down
	registry.Register("mouse_down", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.Toggle("left", "down")
		state.SetMouseDown(true)
		state.SetState(localstate.StateMouseDown, true)
		overlay.SendMouseDownIndicator(true)
		overlay.SendUIMessage("Mouse down")
		return ctx, nil
	})

	// mouse_up releases the left mouse button
	registry.Register("mouse_up", func(ctx context.Context, args map[string]any) (context.Context, error) {
		robotgo.Toggle("left", "up")
		state.SetMouseDown(false)
		state.SetState(localstate.StateMouseDown, false)
		overlay.SendMouseDownIndicator(false)
		overlay.SendUIMessage("Mouse up")
		return ctx, nil
	})

	// Chained commands - these use the trailer (a spoken target number) to
	// look up rects from the last scan overlay
	registerChainedAction(registry, state, "mouse_move_chained", "move")
	registerChainedAction(registry, state, "mouse_click_chained", "click")
	registerChainedAction(registry, state, "mouse_select_chained", "select")

	// Grid position handler (1-9 numpad layout)
	registry.Register("mouse_position", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		_, _, w, h := robotgo.GetDisplayBounds(robotgo.DisplayID)

		switch multiplier {
		case 1:
			robotgo.Move(w/6, h/6)
		case 2:
			robotgo.Move(w/2, h/6)
		case 3:
			robotgo.Move(w-w/6, h/6)
		case 4:
			robotgo.Move(w/6, h/2)
		case 5:
			robotgo.Move(w/2, h/2)
		case 6:
			robotgo.Move(w-w/6, h/2)
		case 7:
			robotgo.Move(w/6, h-h/6)
		case 8:
			robotgo.Move(w/2, h-h/6)
		case 9:
			robotgo.Move(w-w/6, h-h/6)
		default:
			robotgo.Move(w/2, h/2) // Default to center
		}
		return ctx, nil
	})

	// Center mouse handler
	registry.Register("mouse_center", func(ctx context.Context, args map[string]any) (context.Context, error) {
		_, _, w, h := robotgo.GetDisplayBounds(robotgo.DisplayID)
		robotgo.Move(w/2, h/2)
		return ctx, nil
	})

	/**************************************************************************************************************
	* SCROLLING
	**************************************************************************************************************/

	// Scroll handlers - all scroll handlers jump to ghost position first if ghosting is active
	registry.Register("mouse_scroll_left", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.JumpToGhostIfActive()
		for i := 0; i < multiplier; i++ {
			scrollX(-500, state.NaturalScrolling)
		}
		return ctx, nil
	})

	registry.Register("mouse_scroll_right", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.JumpToGhostIfActive()
		for i := 0; i < multiplier; i++ {
			scrollX(500, state.NaturalScrolling)
		}
		return ctx, nil
	})

	registry.Register("mouse_scroll_up_bit", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.JumpToGhostIfActive()
		for i := 0; i < multiplier; i++ {
			scrollY(-100, 3, state.NaturalScrolling)
		}
		return ctx, nil
	})

	registry.Register("mouse_scroll_up", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.JumpToGhostIfActive()
		for i := 0; i < multiplier; i++ {
			scrollY(-500, 3, state.NaturalScrolling)
		}
		return ctx, nil
	})

	registry.Register("mouse_scroll_down_bit", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.JumpToGhostIfActive()
		for i := 0; i < multiplier; i++ {
			scrollY(100, 3, state.NaturalScrolling)
		}
		return ctx, nil
	})

	registry.Register("mouse_scroll_down", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.JumpToGhostIfActive()
		for i := 0; i < multiplier; i++ {
			scrollY(500, 3, state.NaturalScrolling)
		}
		return ctx, nil
	})

	registry.Register("mouse_scroll_to_bottom", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.JumpToGhostIfActive()
		dir := int(1)
		if state.NaturalScrolling {
			dir = -1
		}

		robotgo.ScrollRelative(0, dir*9990000)
		return ctx, nil
	})

	registry.Register("mouse_scroll_to_top", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.JumpToGhostIfActive()
		dir := int(1)
		if state.NaturalScrolling {
			dir = -1
		}
		robotgo.ScrollRelative(0, dir*-9990000)
		return ctx, nil
	})

	// Jump handlers - instant scroll with no stepped animation (say "jump up 3" for more)
	registry.Register("mouse_jump_up", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.JumpToGhostIfActive()
		dir := 1
		if state.NaturalScrolling {
			dir = -1
		}
		robotgo.ScrollRelative(0, dir*-500*multiplier)
		return ctx, nil
	})

	registry.Register("mouse_jump_down", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.JumpToGhostIfActive()
		dir := 1
		if state.NaturalScrolling {
			dir = -1
		}
		robotgo.ScrollRelative(0, dir*500*multiplier)
		return ctx, nil
	})

	registry.Register("mouse_scroll_continuous", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.JumpToGhostIfActive()
		state.Scrolling = true
		go func() {
			for state.GetScrolling() {
				scrollY(500, 10, state.NaturalScrolling)
				robotgo.MilliSleep(2000)
			}
		}()
		return ctx, nil
	})

	registry.Register("mouse_reverse_scroll", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.JumpToGhostIfActive()
		state.SetScrolling(true)
		go func() {
			for state.GetScrolling() {
				scrollY(-500, 10, state.NaturalScrolling)
				robotgo.MilliSleep(2000)
			}
		}()
		return ctx, nil
	})

	registry.Register("mouse_scroll_fast", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.JumpToGhostIfActive()
		state.SetScrolling(true)
		go func() {
			for state.GetScrolling() {
				scrollY(100, 10, state.NaturalScrolling)
			}
		}()
		return ctx, nil
	})

	registry.Register("mouse_reverse_scroll_fast", func(ctx context.Context, args map[string]any) (context.Context, error) {
		state.JumpToGhostIfActive()
		state.SetScrolling(true)
		go func() {
			for state.GetScrolling() {
				scrollY(-100, 10, state.NaturalScrolling)
			}
		}()
		return ctx, nil
	})

	/**************************************************************************************************************
	* NUDGING
	**************************************************************************************************************/

	registry.Register("mouse_nudge_left", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		distance := state.GetNudge() * multiplier
		robotgo.MoveRelative(-distance, 0)
		return ctx, nil
	})

	registry.Register("mouse_nudge_right", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		distance := state.GetNudge() * multiplier
		robotgo.MoveRelative(distance, 0)
		return ctx, nil
	})

	registry.Register("mouse_nudge_up", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		distance := state.GetNudge() * multiplier
		robotgo.MoveRelative(0, -distance)
		return ctx, nil
	})

	registry.Register("mouse_nudge_down", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		distance := state.GetNudge() * multiplier
		robotgo.MoveRelative(0, distance)
		return ctx, nil
	})

	registry.Register("mouse_increase_nudge", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.SetNudge(state.GetNudge() + 5*multiplier)
		overlay.SendUIMessage(fmt.Sprintf("Nudge: %v [+%v]", state.GetNudge(), 5*multiplier))
		return ctx, nil
	})

	registry.Register("mouse_decrease_nudge", func(ctx context.Context, args map[string]any) (context.Context, error) {
		multiplier := xctx.Get(ctx).Multiplier
		state.SetNudge(state.GetNudge() - 5*multiplier)
		overlay.SendUIMessage(fmt.Sprintf("Nudge: %v [-%v]", state.GetNudge(), 5*multiplier))
		return ctx, nil
	})
}

// Helper functions for scrolling
func scrollX(x int, naturalScrolling bool) {
	step := 10
	if x < 0 {
		step = -step
	}
	steps := x / step
	if steps < 0 {
		steps = -steps
	}
	// Invert direction for natural scrolling
	if naturalScrolling {
		step = -step
	}
	for i := 0; i < steps; i++ {
		robotgo.Scroll(step, 0)
		robotgo.MilliSleep(3)
	}
}

func scrollY(y int, sleep int, naturalScrolling bool) {
	step := 10
	if y < 0 {
		step = -step
	}
	steps := y / step
	if steps < 0 {
		steps = -steps
	}
	// Invert direction for natural scrolling
	if naturalScrolling {
		step = -step
	}
	for i := 0; i < steps; i++ {
		robotgo.Scroll(0, step)
		robotgo.MilliSleep(sleep)
	}
}

// copyImageToClipboard saves the image to a temp file and copies it to the macOS clipboard
func copyImageToClipboard(img image.Image) error {
	// Create temp file
	tmpFile, err := os.CreateTemp("", "screenshot_*.png")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath) // Clean up temp file after copying

	// Encode image as PNG
	if err := png.Encode(tmpFile, img); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to encode PNG: %w", err)
	}
	tmpFile.Close()

	// Use osascript to copy image to clipboard on macOS
	script := fmt.Sprintf(`set the clipboard to (read (POSIX file "%s") as TIFF picture)`, tmpPath)
	cmd := exec.Command("osascript", "-e", script)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w, output: %s", err, string(output))
	}

	return nil
}

// registerChainedAction registers a "<verb> <number>" handler that acts on a
// numbered rect from the last scan overlay (labels are decimal).
func registerChainedAction(registry Registry, state *localstate.State, method commands.CommandName, action string) {
	registry.Register(method, func(ctx context.Context, args map[string]any) (context.Context, error) {
		var trailer string
		var err error
		ctx, trailer, err = captureInput(ctx, state, &commands.InputAction{
			Prompt:  "Target number: ",
			Break:   750 * time.Millisecond,
			Timeout: 3 * time.Second,
		})
		if err != nil {
			return ctx, fmt.Errorf("failed to capture input: %v", err)
		}
		if strings.TrimSpace(trailer) == "" {
			return ctx, fmt.Errorf("no target number provided")
		}

		// digits or words: whisper renders "fifty" either way
		index, ok := util.ParseSpokenNumber(trailer)
		if !ok {
			return ctx, fmt.Errorf("invalid target number: %s", trailer)
		}

		rects := state.GetOCRRects()
		if len(rects) == 0 {
			return ctx, fmt.Errorf("no scan targets available")
		}
		if index < 0 || index >= len(rects) {
			return ctx, fmt.Errorf("target %d out of range (max: %d)", index, len(rects)-1)
		}

		rect := rects[index]
		log.Infof("%s target %d at %v", action, index, rect)
		state.ClearOverlay()
		performTargetAction(rect, action)
		return ctx, nil
	})
}
