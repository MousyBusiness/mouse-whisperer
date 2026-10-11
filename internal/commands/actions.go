package commands

import (
	"time"

	"github.com/mousybusiness/mouse_whisperer/types"
	log "github.com/sirupsen/logrus"
)

// Callback result types
const (
	ResultSuccess = "success"
	ResultSkip    = "skip"
	ResultFail    = "fail"
)

// Special callback methods
const (
	MethodContinue = "__continue__"
	MethodDone     = "__done__"
	MethodFail     = "__fail__"
)

// Action defines how to execute a command
type Action struct {
	For             string         `toml:"for"`              // ID of the action this callback is for
	Result          string         `toml:"result"`           // Callback result type: "success", "skip", "fail"
	ID              string         `toml:"id"`               // Optional ID for callback matching
	RequiresTrailer bool           `toml:"requires_trailer"` // Action needs trailer input
	Args            map[string]any `toml:"args"`
	Method          string         `toml:"method"`   // "robotgo", "keybd_event", "osascript", "type_str", "active_name"
	Sleep           int            `toml:"sleep"`    // Milliseconds to sleep after execution
	States          []string       `toml:"states"`   // States where this action is active (empty = always active)
	Excludes        []string       `toml:"excludes"` // States where this action is excluded
}

// IsActiveInState checks if this action should be active given the active states
func (a *Action) IsActiveInState(activeStates map[types.ExecutionState]bool) bool {
	// If excludes contains any active state, action is inactive
	for _, s := range a.Excludes {
		if activeStates[types.ExecutionState(s)] {
			return false
		}
	}
	// If states is empty, action is always active
	if len(a.States) == 0 {
		return true
	}
	// Otherwise, action must have at least one of its required states active
	for _, s := range a.States {
		if activeStates[types.ExecutionState(s)] {
			return true
		}
	}
	return false
}

func (a *Action) ToKeyAction() KeyAction {
	if a == nil {
		log.Warn("nil action")
		return KeyAction{}
	}

	ka := KeyAction{}

	if v, ok := a.Args["key"].(string); ok {
		ka.Key = v
	}

	// Handle both int and int64 (TOML parses integers as int64)
	switch v := a.Args["key_code"].(type) {
	case int:
		ka.KeyCode = v
	case int64:
		ka.KeyCode = int(v)
	}

	// Handle both []string and []interface{} (TOML parses arrays as []interface{})
	switch v := a.Args["modifiers"].(type) {
	case []string:
		ka.Modifiers = v
	case []interface{}:
		for _, m := range v {
			if s, ok := m.(string); ok {
				ka.Modifiers = append(ka.Modifiers, s)
			}
		}
	}

	return ka
}

func (a *Action) ToInputAction() InputAction {
	if a == nil {
		return InputAction{}
	}

	ia := InputAction{}

	if v, ok := a.Args["prompt"].(string); ok {
		ia.Prompt = v
	}

	if v, ok := a.Args["break"].(string); ok {
		dur, err := time.ParseDuration(v)
		if err != nil {
			log.Errorf("Invalid break duration: %s, using default 2s", v)
			log.Debugf("Example: 2s, 1.5s, 1000ms, 500ms")
			ia.Break = time.Second
		} else {
			ia.Break = dur
		}
	} else {
		ia.Break = time.Second * 2
	}

	if v, ok := a.Args["timeout"].(string); ok {
		dur, err := time.ParseDuration(v)
		if err != nil {
			log.Errorf("Invalid timeout duration: %s, using default 3s", v)
			log.Debugf("Example: 3s, 2.5s, 1000ms, 500ms")
			ia.Timeout = time.Second * 3
		} else {
			ia.Timeout = dur
		}
	} else {
		ia.Timeout = time.Second * 3
	}

	if v, ok := a.Args["camel"].(bool); ok {
		ia.Camel = v
	}

	if v, ok := a.Args["pascal"].(bool); ok {
		ia.Pascal = v
	}

	if v, ok := a.Args["snake"].(bool); ok {
		ia.Snake = v
	}

	if v, ok := a.Args["kebab"].(bool); ok {
		ia.Kebab = v
	}

	if v, ok := a.Args["lower"].(bool); ok {
		ia.Lower = v
	}

	if v, ok := a.Args["upper"].(bool); ok {
		ia.Upper = v
	}

	if v, ok := a.Args["title"].(bool); ok {
		ia.Title = v
	}

	if v, ok := a.Args["preserve_punctuation"].(bool); ok {
		ia.PreservePunctuation = v
	}

	if v, ok := a.Args["allow_symbols"].(bool); ok {
		ia.AllowSymbols = v
	}

	if v, ok := a.Args["only_symbols"].(bool); ok {
		ia.OnlySymbols = v
	}

	if v, ok := a.Args["only_numbers"].(bool); ok {
		ia.OnlyNumbers = v
	}

	if v, ok := a.Args["only_alphanumeric"].(bool); ok {
		ia.OnlyAlphanumeric = v
	}

	if v, ok := a.Args["no_spaces"].(bool); ok {
		ia.NoSpaces = v
	}

	if v, ok := a.Args["capture_key"].(string); ok {
		ia.CaptureKey = v
	}

	if v, ok := a.Args["voice_model"].(string); ok {
		ia.VoiceModel = v
	}

	return ia
}

type KeyAction struct {
	Key       string   `toml:"key"`       // Key name for robotgo or VK_ constant name for keybd_event
	KeyCode   int      `toml:"key_code"`  // Key code for osascript
	Modifiers []string `toml:"modifiers"` // Modifier keys: "cmd", "shift", "ctrl", "alt"
}

type InputAction struct {
	Prompt     string        `toml:"prompt"`
	Break      time.Duration `toml:"break"`       // Duration string like "2s"
	Timeout    time.Duration `toml:"timeout"`     // Duration string like "5s"
	VoiceModel string        `toml:"voice_model"` // Whisper model: "fast" or "slow"
	Pascal     bool          `toml:"pascal"`
	Camel      bool          `toml:"camel"`
	Snake      bool          `toml:"snake"`
	Kebab      bool          `toml:"kebab"`
	Lower      bool          `toml:"lower"`
	Upper      bool          `toml:"upper"`
	Title      bool          `toml:"title"`

	PreservePunctuation bool `toml:"preserve_punctuation"`
	AllowSymbols        bool `toml:"preserve_symbols"`
	OnlySymbols         bool `toml:"only_symbols"`
	OnlyNumbers         bool `toml:"only_numbers"`
	OnlyAlphanumeric    bool `toml:"only_alphanumeric"`
	NoSpaces            bool `toml:"no_spaces"`

	CaptureKey string `toml:"capture_key"`
}
