package commands

import (
	"github.com/mousybusiness/mouse_whisperer/types"
	log "github.com/sirupsen/logrus"
)

// Command represents a single voice command configuration
type Command struct {
	Triggers        []string  `toml:"triggers"`
	Description     string    `toml:"description"`
	DisableFuzzy    bool      `toml:"disable_fuzzy"`
	Multiplier      bool      `toml:"multiplier"`
	Actions         []*Action `toml:"actions"`          // For multi-step commands
	ActionCallbacks []*Action `toml:"action_callbacks"` // Callbacks for action results (success, skip, fail)
	States          []string  `toml:"states"`           // States where this command is active (empty = always active)
	Excludes        []string  `toml:"excludes"`         // States where this command is excluded
}

// FindCallback returns the callback action for the given action ID and result type.
// Result types: ResultSuccess, ResultSkip, ResultFail
// Defaults: ResultSuccess→MethodContinue, ResultFail→MethodFail, ResultSkip→nil (explicit only)
func (c *Command) FindCallback(actionID, result string) *Action {
	log.Debugf("Finding callback for action ID: %s, result: %s", actionID, result)
	if actionID == "" {
		d := defaultCallback(result)
		if d != nil {
			log.Debugf("No action ID provided, returning default for result: %s -> %s", result, d.Method)
		} else {
			log.Debugf("No action ID provided, returning default for result: %s -> nil", result)
		}
		return d
	}

	// First, look for exact match (for + result)
	for _, cb := range c.ActionCallbacks {
		if cb.For == actionID && cb.Result == result {
			log.Debugf("Found exact callback match: %s", cb.Method)
			return cb
		}
	}

	// Second, look for action-only match (for without result specified)
	for _, cb := range c.ActionCallbacks {
		if cb.For == actionID && cb.Result == "" {
			log.Debugf("Found action-only callback match: %s", cb.Method)
			return cb
		}
	}

	// Apply defaults
	d := defaultCallback(result)
	if d != nil {
		log.Debugf("No callback found, using default for result: %s -> %s", result, d.Method)
	} else {
		log.Debugf("No callback found, using default for result: %s -> nil", result)
	}
	return d
}

// defaultCallback returns the default callback action for a result type
func defaultCallback(result string) *Action {
	switch result {
	case ResultSuccess:
		return &Action{Method: MethodContinue}
	case ResultFail:
		return &Action{Method: MethodFail}
	default:
		// ResultSkip and others have no default - must be explicit
		return nil
	}
}

// ConsumesTrailer reports whether any action reads trailer text, meaning the
// words after the trigger are input for this command rather than further commands
func (c *Command) ConsumesTrailer() bool {
	for _, a := range c.Actions {
		if a.RequiresTrailer {
			return true
		}
	}
	return false
}

// MayPrompt reports whether any action could open a voice prompt
// (captureInput calls AwaitInputWithTimeout when no trailer is present).
// Such commands must not fire mid-utterance: the mic attaches a pending
// prompt request to the recording currently in progress.
func (c *Command) MayPrompt() bool {
	for _, a := range c.Actions {
		if a.RequiresTrailer {
			return true
		}
		if _, ok := a.Args["prompt"]; ok {
			return true
		}
	}
	return false
}

// IsActiveInState checks if this command should be active given the active states
func (c *Command) IsActiveInState(activeStates map[types.ExecutionState]bool) bool {
	// If excludes contains any active state, command is inactive
	for _, s := range c.Excludes {
		if activeStates[types.ExecutionState(s)] {
			return false
		}
	}
	// If states is empty, command is always active
	if len(c.States) == 0 {
		return true
	}
	// Otherwise, command must have at least one of its required states active
	for _, s := range c.States {
		if activeStates[types.ExecutionState(s)] {
			return true
		}
	}
	return false
}
