package impl

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/go-vgo/robotgo"
	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/config"
	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/applescript"
	"github.com/mousybusiness/mouse_whisperer/pkg/symbols"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	"github.com/iancoleman/strcase"
	kbd "github.com/micmonay/keybd_event"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func RegisterSystemHandlers(registry Registry, lstate *state.State) {
	registry.Register("stop", func(ctx context.Context, _ map[string]any) (context.Context, error) {
		lstate.Stop()
		return ctx, nil
	})

	registry.Register("reset_keys", func(ctx context.Context, _ map[string]any) (context.Context, error) {
		// Release all modifier keys - both left and right variants
		// Use both naming conventions for compatibility
		keysToRelease := []string{
			"lcmd", "rcmd", "cmd",
			"lshift", "rshift", "shift", "lshft", "rshft",
			"lctrl", "rctrl", "ctrl",
			"lalt", "ralt", "alt",
		}
		for _, key := range keysToRelease {
			robotgo.KeyUp(key)
		}
		log.Warnf("cleaning keys")
		return ctx, nil
	})

	registry.Register("robotgo", func(ctx context.Context, args map[string]any) (context.Context, error) {
		a := xctx.GetCurrentAction(ctx).ToKeyAction()
		multiplier := xctx.Get(ctx).Multiplier
		for i := 0; i < multiplier; i++ {
			if err := executeRobotgo(a); err != nil {
				return ctx, err
			}
		}
		return ctx, nil
	})

	registry.Register("keybd_event", func(ctx context.Context, args map[string]any) (context.Context, error) {
		a := xctx.GetCurrentAction(ctx).ToKeyAction()
		multiplier := xctx.Get(ctx).Multiplier
		for i := 0; i < multiplier; i++ {
			if err := executeKeybdEvent(a); err != nil {
				return ctx, err
			}
		}
		return ctx, nil
	})

	registry.Register("osascript", func(ctx context.Context, args map[string]any) (context.Context, error) {
		a := xctx.GetCurrentAction(ctx).ToKeyAction()
		multiplier := xctx.Get(ctx).Multiplier
		for i := 0; i < multiplier; i++ {
			if err := applescript.ExecuteOsascript(a); err != nil {
				return ctx, err
			}
		}
		return ctx, nil
	})

	registry.Register("applescript", func(ctx context.Context, args map[string]any) (context.Context, error) {
		script, ok := args["script"].(string)
		if !ok || script == "" {
			return ctx, fmt.Errorf("applescript: script argument is required")
		}
		cmd := exec.Command("osascript", "-e", script)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return ctx, fmt.Errorf("applescript: failed to execute: %w, output: %s", err, string(output))
		}
		return ctx, nil
	})

	registry.Register("type_str", func(ctx context.Context, args map[string]any) (context.Context, error) {
		if t, ok := args["text"].(string); ok {
			if t != "" {
				log.Warnf("========== Executing text from arg: %s =========", t)
				return ctx, executeTypeStr(t)
			}
		}

		var err error
		var text string
		ctx, text, err = captureInput(ctx, lstate, nil)
		if err != nil {
			return ctx, err
		}

		text = strings.TrimSpace(text)
		log.Warnf("========== Executing text: %s =========", text)
		return ctx, executeTypeStr(text)
	})

	registry.Register("type_num", func(ctx context.Context, args map[string]any) (context.Context, error) {
		if t, ok := args["number"]; ok {
			switch v := t.(type) {
			case string:
				if t != "" {
					log.Warnf("========== Executing number from arg: %s =========", t)
					return ctx, executeTypeStr(v)
				}
			case int32:
				log.Warnf("========== Executing number from arg: %d =========", t)
				return ctx, executeTypeStr(strconv.Itoa(int(v)))
			case int64:
				log.Warnf("========== Executing number from arg: %d =========", t)
				return ctx, executeTypeStr(strconv.Itoa(int(v)))
			case int:
				log.Warnf("========== Executing number from arg: %d =========", t)
				return ctx, executeTypeStr(strconv.Itoa(v))
			case float32:
				log.Warnf("========== Executing number from arg: %f =========", t)
				return ctx, executeTypeStr(strconv.FormatFloat(float64(v), 'f', -1, 32))
			case float64:
				log.Warnf("========== Executing number from arg: %f =========", t)
				return ctx, executeTypeStr(strconv.FormatFloat(float64(v), 'f', -1, 64))
			default:
				return ctx, fmt.Errorf("invalid number, type: %T", t)
			}
		}

		a := xctx.GetCurrentAction(ctx).ToInputAction()

		var err error
		var text string
		ctx, text, err = captureInput(ctx, lstate, &commands.InputAction{
			Prompt:  "A code file line number e.g. 123. output: ",
			Break:   a.Break,
			Timeout: a.Timeout,
		})
		if err != nil {
			return ctx, err
		}
		if text == "" {
			return ctx, fmt.Errorf("no number entered")
		}

		_, err = strconv.ParseInt(text, 10, 64)
		if err != nil {
			return ctx, fmt.Errorf("invalid number, text: %s", text)
		}

		return ctx, executeTypeStr(text)
	})

	registry.Register("capture_input", func(ctx context.Context, args map[string]any) (context.Context, error) {
		var err error
		ctx, _, err = captureInput(ctx, lstate, nil)
		return ctx, err
	})

	registry.Register("reload_config", func(ctx context.Context, args map[string]any) (context.Context, error) {
		configPath := "config"
		cfg, err := config.Load(configPath)
		if err != nil {
			overlay.SendUIMessage("Config reload failed!")
			return ctx, fmt.Errorf("failed to reload config: %w", err)
		}

		cfg.PruneUnhandled(registry.Has)
		lookup := commands.NewCommandLookup(cfg.RawCommands, cfg.Settings.CategoryOrder)
		lstate.SetLookup(lookup)

		log.Info("Config reloaded successfully")
		overlay.SendUIMessage("Config reloaded!")
		return ctx, nil
	})

}

func applyInputConditions(text string, conditions commands.InputAction) string {
	if !conditions.PreservePunctuation {
		text = util.TrimTrailingPunctuation(text)
	}

	if !conditions.AllowSymbols {
		if conditions.PreservePunctuation {
			text = util.NoSymbolsExceptPunctuation(text)
		} else {
			text = util.NoSymbols(text)
		}
	}

	if conditions.OnlyNumbers {
		text = util.ToAlphanumericWithSpace(text)
		text = util.ReplaceNumberWords(text)
		text = util.OnlyNumbers(text)
	}

	if conditions.OnlySymbols {
		text = util.OnlySymbols(text)
		if t, ok := symbols.WordToSymbol[text]; ok {
			text = t
		}
	}

	if conditions.OnlyAlphanumeric {
		text = util.ToAlphanumericWithSpace(text)
	}

	if conditions.NoSpaces {
		text = strings.ReplaceAll(text, " ", "")
	}

	if conditions.Snake {
		return strcase.ToSnake(text)
	}
	if conditions.Kebab {
		return strcase.ToKebab(text)
	}
	if conditions.Camel {
		return strcase.ToLowerCamel(text)
	}
	if conditions.Pascal {
		return strcase.ToCamel(text)
	}
	if conditions.Title {
		return cases.Title(language.English).String(text)
	}
	if conditions.Lower {
		return strings.ToLower(text)
	}
	if conditions.Upper {
		return strings.ToUpper(text)
	}

	return text
}

func promptOr(prompt string, or string) string {
	if prompt == "" {
		return or
	}
	return prompt
}

func executeRobotgo(action commands.KeyAction) error {
	if action.Key == "" {
		return fmt.Errorf("robotgo: key is required")
	}

	key := strings.ToLower(action.Key)

	// Convert modifiers to robotgo format
	var modifiers []interface{}
	for _, mod := range action.Modifiers {
		modifiers = append(modifiers, translateModifierToRobotgo(mod))
	}

	if len(modifiers) > 0 {
		robotgo.KeyTap(key, modifiers...)
	} else {
		robotgo.KeyTap(key)
	}

	return nil
}

func executeKeybdEvent(action commands.KeyAction) error {
	if action.Key == "" {
		return fmt.Errorf("keybd_event: key is required")
	}

	keyCode, ok := vkCodeMap[action.Key]
	if !ok {
		return fmt.Errorf("keybd_event: unknown key code: %s", action.Key)
	}

	// Use the Key function from keyboard package logic
	kb, err := kbd.NewKeyBonding()
	if err != nil {
		return fmt.Errorf("keybd_event: failed to create key bonding: %w", err)
	}

	preSleep := 150
	kb.SetKeys(keyCode)

	// Handle modifiers with manual key down/up for reliability
	for _, mod := range action.Modifiers {
		switch mod {
		case "cmd":
			robotgo.KeyDown("lcmd")
			robotgo.MilliSleep(preSleep)
			defer func() {
				robotgo.MilliSleep(50)
				robotgo.KeyUp("lcmd")
			}()
		case "shift":
			robotgo.KeyDown("lshift")
			robotgo.MilliSleep(preSleep)
			defer func() {
				robotgo.MilliSleep(50)
				robotgo.KeyUp("lshift")
			}()
		case "ctrl":
			robotgo.KeyDown("lctrl")
			robotgo.MilliSleep(preSleep)
			defer func() {
				robotgo.MilliSleep(50)
				robotgo.KeyUp("lctrl")
			}()
		case "alt":
			robotgo.KeyDown("lalt")
			robotgo.MilliSleep(preSleep)
			defer func() {
				robotgo.MilliSleep(50)
				robotgo.KeyUp("lalt")
			}()
		}
	}

	if err := kb.Launching(); err != nil {
		return fmt.Errorf("keybd_event: failed to launch: %w", err)
	}

	robotgo.MilliSleep(100)
	return nil
}

func executeTypeStr(text string) error {
	if text == "" {
		return fmt.Errorf("type_str: text is required")
	}
	robotgo.Type(text)
	return nil
}

// translateModifierToRobotgo converts modifier names to robotgo format
func translateModifierToRobotgo(mod string) string {
	switch mod {
	case "cmd", "command":
		return "lcmd"
	case "shift":
		return "lshift"
	case "ctrl", "control":
		return "lctrl"
	case "alt", "option":
		return "lalt"
	default:
		return mod
	}
}

// vkCodeMap maps key names to keybd_event virtual key codes
var vkCodeMap = map[string]int{
	// Letters
	"VK_A": kbd.VK_A, "VK_B": kbd.VK_B, "VK_C": kbd.VK_C, "VK_D": kbd.VK_D,
	"VK_E": kbd.VK_E, "VK_F": kbd.VK_F, "VK_G": kbd.VK_G, "VK_H": kbd.VK_H,
	"VK_I": kbd.VK_I, "VK_J": kbd.VK_J, "VK_K": kbd.VK_K, "VK_L": kbd.VK_L,
	"VK_M": kbd.VK_M, "VK_N": kbd.VK_N, "VK_O": kbd.VK_O, "VK_P": kbd.VK_P,
	"VK_Q": kbd.VK_Q, "VK_R": kbd.VK_R, "VK_S": kbd.VK_S, "VK_T": kbd.VK_T,
	"VK_U": kbd.VK_U, "VK_V": kbd.VK_V, "VK_W": kbd.VK_W, "VK_X": kbd.VK_X,
	"VK_Y": kbd.VK_Y, "VK_Z": kbd.VK_Z,

	// Numbers
	"VK_0": kbd.VK_0, "VK_1": kbd.VK_1, "VK_2": kbd.VK_2, "VK_3": kbd.VK_3,
	"VK_4": kbd.VK_4, "VK_5": kbd.VK_5, "VK_6": kbd.VK_6, "VK_7": kbd.VK_7,
	"VK_8": kbd.VK_8, "VK_9": kbd.VK_9,

	// Function keys
	"VK_F1": kbd.VK_F1, "VK_F2": kbd.VK_F2, "VK_F3": kbd.VK_F3, "VK_F4": kbd.VK_F4,
	"VK_F5": kbd.VK_F5, "VK_F6": kbd.VK_F6, "VK_F7": kbd.VK_F7, "VK_F8": kbd.VK_F8,
	"VK_F9": kbd.VK_F9, "VK_F10": kbd.VK_F10, "VK_F11": kbd.VK_F11, "VK_F12": kbd.VK_F12,

	// Navigation
	"VK_UP": kbd.VK_UP, "VK_DOWN": kbd.VK_DOWN, "VK_LEFT": kbd.VK_LEFT, "VK_RIGHT": kbd.VK_RIGHT,
	"VK_HOME": kbd.VK_HOME, "VK_END": kbd.VK_END, "VK_PAGEUP": kbd.VK_PAGEUP, "VK_PAGEDOWN": kbd.VK_PAGEDOWN,

	// Special keys
	"VK_ENTER": kbd.VK_ENTER, "VK_TAB": kbd.VK_TAB, "VK_SPACE": kbd.VK_SPACE,
	"VK_DELETE": kbd.VK_DELETE, "VK_ESC": kbd.VK_ESC,

	// Punctuation
	"VK_SLASH":     kbd.VK_SLASH,
	"VK_BACKSLASH": kbd.VK_BACKSLASH,
	"VK_GRAVE":     kbd.VK_GRAVE,
}
