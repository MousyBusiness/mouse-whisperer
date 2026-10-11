package applescript

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
)

var osascriptKeyCodeMap = map[string]int{
	// Arrow keys
	"left": 123, "right": 124, "down": 125, "up": 126,
	// Enter
	"enter": 36, "return": 36,
	// Letters
	"h": 4, "m": 46, "e": 14, "t": 17, "d": 2, "f": 3, "g": 5,
	"u": 32, "i": 34, "j": 38, "k": 40,
	// Numbers
	"3": 20, "4": 21,
	// Function keys
	"f3": 99, "f11": 103, "f12": 111,
	// Space
	"space": 49,
}

func ExecuteOsascript(spec commands.KeyAction) error {
	if spec.KeyCode == 0 && spec.Key == "" {
		return fmt.Errorf("osascript: key_code or key is required")
	}

	keyCode := spec.KeyCode
	if keyCode == 0 {
		// Try to look up the key code from the key name
		if code, ok := osascriptKeyCodeMap[spec.Key]; ok {
			keyCode = code
		} else {
			return fmt.Errorf("osascript: unknown key: %s", spec.Key)
		}
	}

	script := BuildOsascriptCommand(keyCode, spec.Modifiers)
	cmd := exec.Command("bash", "-c", script)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("osascript: failed to execute: %w, output: %s", err, string(output))
	}

	return nil
}

func BuildOsascriptCommand(keyCode int, modifiers []string) string {
	if len(modifiers) == 0 {
		return fmt.Sprintf("osascript -e 'tell application \"System Events\" to key code %d'", keyCode)
	}

	// Build modifier string
	var modStrs []string
	for _, mod := range modifiers {
		switch mod {
		case "cmd", "command":
			modStrs = append(modStrs, "command down")
		case "shift":
			modStrs = append(modStrs, "shift down")
		case "ctrl", "control":
			modStrs = append(modStrs, "control down")
		case "alt", "option":
			modStrs = append(modStrs, "option down")
		}
	}

	if len(modStrs) == 1 {
		return fmt.Sprintf("osascript -e 'tell application \"System Events\" to key code %d using %s'", keyCode, modStrs[0])
	}

	return fmt.Sprintf("osascript -e 'tell application \"System Events\" to key code %d using {%s}'", keyCode, strings.Join(modStrs, ", "))
}
