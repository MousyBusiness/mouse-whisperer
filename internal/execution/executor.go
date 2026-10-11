package execution

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/debuglog"
	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/registry"
	"github.com/mousybusiness/mouse_whisperer/types"
	log "github.com/sirupsen/logrus"
)

type Executor struct {
	lookup   *commands.CommandLookup
	registry *registry.HandlerRegistry
	state
}

type state interface {
	GetActiveStates() map[types.ExecutionState]bool
}

// NewExecutor creates a new command executor
func NewExecutor(lookup *commands.CommandLookup, registry *registry.HandlerRegistry, state state) *Executor {
	return &Executor{
		lookup:   lookup,
		registry: registry,
		state:    state,
	}
}

func (e *Executor) ExecuteCommand(entry *commands.CommandEntry, triggerWords []string, trailerWords []string, trailer commands.TrailerText, multiplier int, invocationSource string) error {
	cmd := entry.Command

	ctx := context.Background()
	ctx = xctx.With(ctx, &xctx.ExecutionContext{
		TriggerWords:     triggerWords,
		TrailerWords:     trailerWords,
		Trailer:          trailer,
		Multiplier:       multiplier,
		Entry:            entry,
		InvocationSource: invocationSource,
	})
	if !entry.Command.Multiplier {
		multiplier = 1
	}

	if len(cmd.Actions) == 0 {
		return fmt.Errorf("command [%s] has no actions", entry.Name)
	}
	log.Infof("!!!!!!!!!! Executing command !!!!!!!!!: %s -> words: %v, trailer: %v,  multiplier: %d", cmd.Triggers[0], triggerWords, trailerWords, multiplier)
	for i, action := range cmd.Actions {
		description := fmt.Sprintf("[%v.%v] action %d: %s", entry.Category, entry.Name, i+1, action.Method)
		log.Infof("********* Executing action *********: %s action: %+v", description, action)

		activeStates := e.GetActiveStates()
		// Check if this action is active in the current state
		if !action.IsActiveInState(activeStates) {
			log.Debugf("%s - skipping action, not active in states %v", description, activeStates)
			// Look for a skip callback
			callback := cmd.FindCallback(action.ID, commands.ResultSkip)
			if callback != nil {
				log.Infof("%s - action not in active state, using skip callback: %s", description, callback.Method)
				ctx = xctx.WithCurrentAction(ctx, callback)
				var done bool
				var err error
				ctx, done, err = e.executeCallback(ctx, callback)
				if err != nil {
					log.Warnf("%s - skip callback failed: %v", description, err)
					return fmt.Errorf("%s - skip callback failed: %w", description, err)
				}
				if done {
					e.logCommandSuccess(entry, invocationSource)
					return nil
				}
				continue
			}
			return fmt.Errorf("%s - no skip callback for action when state condition failed, active states: %v", description, activeStates)
		}

		// Check if this action requires trailer input
		if action.RequiresTrailer && strings.TrimSpace(string(trailer)) == "" {
			// Look for a skip callback
			callback := cmd.FindCallback(action.ID, commands.ResultSkip)
			if callback != nil {
				log.Infof("%s - action requires trailer but none provided, using skip callback: %s", description, callback.Method)
				ctx = xctx.WithCurrentAction(ctx, callback)
				var done bool
				var err error
				ctx, done, err = e.executeCallback(ctx, callback)
				if err != nil {
					log.Warnf("%s - skip callback failed: %v", description, err)
					return fmt.Errorf("%s - skip callback failed: %w", description, err)
				}
				if done {
					e.logCommandSuccess(entry, invocationSource)
					return nil
				}
				continue
			}
			return fmt.Errorf("%s - no skip callback for action when 'required_trailer' condition failed", description)
		}

		cn := commands.CommandName(action.Method)
		h, ok := e.registry.Get(cn)
		if !ok {
			log.Warnf("Handler not found for method: %s", cn)
			continue
		}

		// Set the current action index
		ctx = xctx.WithCurrentAction(ctx, action)

		var err error
		ctx, err = h(ctx, action.Args)
		if err != nil {
			log.Warnf("%s - handler failed: %v", description, err)
			// Look for a fail callback
			callback := cmd.FindCallback(action.ID, commands.ResultFail)
			if callback != nil {
				log.Infof("%s - action failed, using fail callback: %s", description, callback.Method)
				ctx = xctx.WithCurrentAction(ctx, callback)
				var done bool
				ctx, done, err = e.executeCallback(ctx, callback)
				if err != nil {
					return fmt.Errorf("%s - fail callback failed: %w", description, err)
				}
				if done {
					e.logCommandSuccess(entry, invocationSource)
					return nil
				}
				continue
			}
			return fmt.Errorf("%s - handler failed: %w", description, err)
		}

		time.Sleep(time.Duration(action.Sleep) * time.Millisecond)

		// Look for a success callback
		callback := cmd.FindCallback(action.ID, commands.ResultSuccess)
		if callback != nil {
			log.Debugf("%s - action succeeded, using success callback: %s", description, callback.Method)
			ctx = xctx.WithCurrentAction(ctx, callback)
			var done bool
			ctx, done, err = e.executeCallback(ctx, callback)
			if err != nil {
				log.Warnf("%s - success callback failed: %v", description, err)
				return fmt.Errorf("%s - success callback failed: %w", description, err)
			}
			if done {
				e.logCommandSuccess(entry, invocationSource)
				return nil
			}
		}
	}

	e.logCommandSuccess(entry, invocationSource)
	return nil
}

func (e *Executor) logCommandSuccess(entry *commands.CommandEntry, source string) {
	if !debuglog.Enabled() {
		return
	}
	primary := ""
	if len(entry.Command.Triggers) > 0 {
		primary = entry.Command.Triggers[0]
	}
	debuglog.AppendCommand([]string{
		"command",
		source,
		string(entry.Category),
		string(entry.Name),
		primary,
	})
}

// executeCallback executes a callback action and returns whether to stop execution
func (e *Executor) executeCallback(ctx context.Context, callback *commands.Action) (context.Context, bool, error) {
	switch callback.Method {
	case commands.MethodContinue:
		log.Debugf("Callback: %s - continuing", callback.Method)
		return ctx, false, nil
	case commands.MethodDone:
		log.Debugf("Callback: %s - done", callback.Method)
		return ctx, true, nil
	case commands.MethodFail:
		log.Debugf("Callback: %s - fail", callback.Method)
		return ctx, false, errors.New("callback wants fail")
	}

	log.Warnf("Fallback callback: %s - executing handler", callback.Method)
	cn := commands.CommandName(callback.Method)
	h, ok := e.registry.Get(cn)
	if !ok {
		return ctx, false, fmt.Errorf("callback handler not found: %s", callback.Method)
	}

	var err error
	ctx, err = h(ctx, callback.Args)
	if err != nil {
		return ctx, false, fmt.Errorf("callback action [%s] failed: %w", callback.Method, err)
	}

	if callback.Sleep > 0 {
		time.Sleep(time.Duration(callback.Sleep) * time.Millisecond)
	}

	return ctx, false, nil
}

// ExecuteCommand runs a specific command
//func (e *Executor) SExecuteCommand(cmd *commands.Command, parts []string, multiplier int) bool {
//	log.Debugf("Executing command: %s with parts: %v and multiplier: %d", cmd.Triggers[0], parts, multiplier)
//
//	//// Parse and set input config for handlers to use
//	//if cmd.Input != nil {
//	//	parsedInput, err := cmd.Input.Parse()
//	//	if err != nil {
//	//		log.Warnf("Failed to parse input config: %v", err)
//	//	} else {
//	//		e.SetCurrentInputConfig(parsedInput)
//	//	}
//	//} else {
//	//	e.SetCurrentInputConfig(nil)
//	//}
//	//defer e.SetCurrentInputConfig(nil) // Clear after execution
//
//	// Handle complex/chained commands via handlers
//	if cmd.SystemHandler != "" {
//		// Check for fallback: if only trigger word(s) and fallback handler exists
//		if cmd.FallbackHandler != "" && (parts == nil || len(parts) <= 1) {
//			if fallbackHandler, ok := e.handlers[cmd.FallbackHandler]; ok {
//				log.Warnf("=== FALLBACK === No args provided for chained command, using fallback handler: %s -> %s", cmd.Handler, cmd.FallbackHandler)
//				return fallbackHandler(parts, multiplier)
//			}
//			log.Warnf("Fallback handler not found: %s", cmd.FallbackHandler)
//		}
//		// Normal handler execution
//		handler, ok := e.handlers[cmd.Handler]
//		if !ok {
//			log.Warnf("Handler not found: %s", cmd.Handler)
//			return false
//		}
//		return handler(parts, multiplier)
//	}
//
//	// Handle multi-step commands
//	if len(cmd.Actions) > 0 {
//		for _, action := range cmd.Actions {
//			if err := e.executeSpec(&action, multiplier); err != nil {
//				log.Warnf("Action failed: %v", err)
//				// Continue with other actions
//			}
//		}
//		return true
//	}
//
//	// Handle simple commands with primary/fallback
//	if cmd.Primary != nil {
//		repeatCount := 1
//		if cmd.Multiplier && multiplier > 1 {
//			repeatCount = min(multiplier, 20) // Cap at 20 to prevent accidents
//		}
//
//		for i := 0; i < repeatCount; i++ {
//			if err := e.executeSpec(cmd.Primary, 1); err != nil {
//				log.Debugf("Primary execution failed: %v, trying fallback", err)
//				if cmd.Fallback != nil {
//					if err := e.executeSpec(cmd.Fallback, 1); err != nil {
//						log.Warnf("Fallback also failed: %v", err)
//						return false
//					}
//				} else {
//					return false
//				}
//			}
//		}
//		return true
//	}
//
//	return false
//}
