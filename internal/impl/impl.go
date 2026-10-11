package impl

import (
	"context"
	"strconv"
	"strings"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/pkg/symbols"
	kbd "github.com/micmonay/keybd_event"
	log "github.com/sirupsen/logrus"
)

type Registry interface {
	Register(n commands.CommandName, handler func(ctx context.Context, args map[string]any) (context.Context, error))
	Has(n commands.CommandName) bool
}

// Helper function to simulate keyboard Key function
func key(k int, args ...string) {
	kb, err := kbd.NewKeyBonding()
	if err != nil {
		panic(err)
	}

	preSleep := 150
	kb.SetKeys(k)

	defer func() {
		robotgo.MilliSleep(100)
	}()

	for _, arg := range args {
		if arg == "cmd" {
			robotgo.KeyDown("lcmd")
			robotgo.MilliSleep(preSleep)
			defer func() {
				robotgo.MilliSleep(50)
				robotgo.KeyUp("lcmd")
			}()
		}
		if arg == "shift" {
			robotgo.KeyDown("lshift")
			robotgo.MilliSleep(preSleep)
			defer func() {
				robotgo.MilliSleep(50)
				robotgo.KeyUp("lshift")
			}()
		}
		if arg == "ctrl" {
			robotgo.KeyDown("lctrl")
			robotgo.MilliSleep(preSleep)
			defer func() {
				robotgo.MilliSleep(50)
				robotgo.KeyUp("lctrl")
			}()
		}
		if arg == "alt" {
			robotgo.KeyDown("lalt")
			robotgo.MilliSleep(preSleep)
			defer func() {
				robotgo.MilliSleep(50)
				robotgo.KeyUp("lalt")
			}()
		}
	}

	err = kb.Launching()
	if err != nil {
		panic(err)
	}
}



// Helper for inserting symbols
// Uses wordToSymbol map from symbols.go
func insertSymbol(command string) {
	command = strings.TrimSpace(command)

	// First check if we have a direct mapping
	if symbol, ok := symbols.WordToSymbol[command]; ok {
		robotgo.Type(symbol)
		return
	}

	// Handle special cases that need key events instead of typing
	switch command {
	case "space":
		key(kbd.VK_SPACE)
	case "tab":
		key(kbd.VK_TAB)
	case "enter", "return":
		key(kbd.VK_ENTER)
	default:
		// If no match found, log it for debugging
		log.Debugf("Unknown symbol command: %s", command)
	}
}

// multiplierCommand extracts command and multiplier from a string like "left 3"
func multiplierCommand(s string) (string, int) {
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return "", 1
	}
	if len(parts) == 1 {
		return parts[0], 1
	}
	n, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return parts[0], 1
	}
	return parts[0], n
}
