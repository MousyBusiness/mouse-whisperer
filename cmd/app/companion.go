//go:build !nocompanion

package main

import (
	"context"
	"slices"
	"time"

	"github.com/go-vgo/robotgo"
	companionv1 "github.com/mousybusiness/mouse_whisperer/gen/mouse_whisperer/companion/v1"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/companion"
	"github.com/mousybusiness/mouse_whisperer/internal/config"
	"github.com/mousybusiness/mouse_whisperer/internal/evaluation"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/applescript"
	"github.com/mousybusiness/mouse_whisperer/pkg/eye_tracking"

	log "github.com/sirupsen/logrus"
)

// startCompanion serves the companion phone app: commands, typed text and
// touchpad gestures arrive as UDP payloads
func startCompanion(ctx context.Context, cfg *config.Config, eval *evaluation.Evaluator, lstate *state.State) {
	var lastHandledThreeFingerFlingID string
	companionHandler := companion.HandlerFunc(func(payload *companionv1.Payload) {
		if command := payload.GetCommand(); command != "" {
			found, err := eval.EvaluteBackwardsCommand(command, "companion")
			if err != nil {
				log.Errorf("Failed to execute command: %v", err)
			}
			if !found {
				log.Errorf("Command not found: %s", command)
			}
		}
		if keyboard := payload.GetKeyboard(); keyboard != nil {
			if keyboard.GetType() == companionv1.KeyboardEvent_EVENT_TYPE_TYPE_STRING {
				robotgo.Type(keyboard.GetTypeString())
			}
		}
		if mouse := payload.GetMouse(); mouse != nil {
			shift := slices.Contains(mouse.Modifiers, companionv1.Modifier_MODIFIER_SHIFT)
			// ctrl := slices.Contains(mouse.Modifiers, companionv1.Modifier_MODIFIER_CONTROL)
			// alt := slices.Contains(mou se.Modifiers, companionv1.Modifier_MODIFIER_OPTION)
			// cmd := slices.Contains(mouse.Modifiers, companionv1.Modifier_MODIFIER_COMMAND)
			// super := slices.Contains(mouse.Modifiers, companionv1.Modifier_MODIFIER_SUPER)
			pointers := mouse.GetPointerCount()
			pointerSequence := mouse.GetPointerSequence()
			eventType := mouse.GetType()
			dx := int(mouse.GetDeltaX())
			dy := int(mouse.GetDeltaY())

			switch pointers {
			case 1: // Single finger - mouse control
				switch eventType {
				case companionv1.MouseEvent_EVENT_TYPE_CLICK:
					if pointerSequence == 2 {
						log.Debugf("Double click")
						robotgo.Click("left", true)
					} else {
						log.Debugf("Single click")
						robotgo.Click("left")
					}
				case companionv1.MouseEvent_EVENT_TYPE_LONG_PRESS:
					robotgo.Toggle("left", "down")
					lstate.SetMouseDown(true)
					lstate.SetState(state.StateMouseDown, true)
					log.Info("Long press")
				case companionv1.MouseEvent_EVENT_TYPE_DRAG:
					if shift && !lstate.GetMouseDown() {
						robotgo.Toggle("left", "down")
						lstate.SetMouseDown(true)
						lstate.SetState(state.StateMouseDown, true)
					}
					robotgo.MoveRelative(
						int(float64(dx)*lstate.GetCompanionMouseSensitivity()),
						int(float64(dy)*lstate.GetCompanionMouseSensitivity()),
					)
				case companionv1.MouseEvent_EVENT_TYPE_FLING:
					if lstate.GetMouseDown() {
						lstate.SetMouseDown(false)
						lstate.SetState(state.StateMouseDown, false)
						robotgo.Toggle("left", "up")
					}
					robotgo.Toggle("left", "up")
				}

			case 2: // Two finger - scroll and right-click
				switch eventType {
				case companionv1.MouseEvent_EVENT_TYPE_CLICK:
					log.Debugf("Two finger click")
					// Double two-finger tap → right click
					robotgo.Click("right")
				case companionv1.MouseEvent_EVENT_TYPE_DRAG, companionv1.MouseEvent_EVENT_TYPE_FLING:
					robotgo.Scroll(
						int(float64(dx)*cfg.Settings.CompanionScrollSensitivity),
						int(float64(dy)*cfg.Settings.CompanionScrollSensitivity),
					)
				}
			case 3: // Three finger - space switching & Mission Control
				switch eventType {
				case companionv1.MouseEvent_EVENT_TYPE_CLICK:
					log.Debugf("Three finger click")
					// Reserved for future three-finger tap actions
					eyePos, ok := eye_tracking.GetPositionAtTime(time.Now())
					if !ok {
						return
					}
					// scaledV := nearest.Position.Mul(robotgo.ScaleF(robotgo.DisplayID))
					// result := m.screen.FindNearest(scaledV)
					robotgo.Move(int(eyePos.X), int(eyePos.Y))

				case companionv1.MouseEvent_EVENT_TYPE_FLING:
					// Deduplicate by event group ID
					if payload.EventGroupId == lastHandledThreeFingerFlingID {
						return
					}

					flingDist := cfg.Settings.CompanionFlingDistance
					// Horizontal: switch spaces
					if dx > flingDist {
						applescript.ExecuteOsascript(commands.KeyAction{
							KeyCode:   123,
							Modifiers: []string{"control"},
						})
						lastHandledThreeFingerFlingID = payload.EventGroupId

					} else if dx < -flingDist {
						applescript.ExecuteOsascript(commands.KeyAction{
							KeyCode:   124,
							Modifiers: []string{"control"},
						})
						lastHandledThreeFingerFlingID = payload.EventGroupId

					}
					// Vertical: Mission Control / App Windows
					if dy < -flingDist {
						// Three finger up → Mission Control (Ctrl+Up)
						applescript.ExecuteOsascript(commands.KeyAction{
							KeyCode:   126, // Up arrow
							Modifiers: []string{"control"},
						})
						lastHandledThreeFingerFlingID = payload.EventGroupId

					} else if dy > flingDist {
						// Three finger down → Application Windows (Ctrl+Down)
						applescript.ExecuteOsascript(commands.KeyAction{
							KeyCode:   125, // Down arrow
							Modifiers: []string{"control"},
						})
						lastHandledThreeFingerFlingID = payload.EventGroupId

					}
				}
			case 4:
				switch eventType {
				case companionv1.MouseEvent_EVENT_TYPE_CLICK:
					log.Debugf("Four finger click")
					eyePos, ok := eye_tracking.GetPositionAtTime(time.Now())
					if !ok {
						return
					}
					// scaledV := nearest.Position.Mul(robotgo.ScaleF(robotgo.DisplayID))
					// result := m.screen.FindNearest(scaledV)
					robotgo.Move(int(eyePos.X), int(eyePos.Y))
					robotgo.Click("left")
				}

			}

			// Handle finish event regardless of pointer count
			if eventType == companionv1.MouseEvent_EVENT_TYPE_FINISH {
				if lstate.GetMouseDown() {
					lstate.SetMouseDown(false)
					lstate.SetState(state.StateMouseDown, false)
					robotgo.Toggle("left", "up")
				}
			}
		}
	})
	companionServer := companion.NewServer(companion.DefaultPort, companionHandler)
	go func() {
		if err := companionServer.Run(ctx); err != nil {
			log.Errorf("Companion server error: %v", err)
		}
	}()
}
