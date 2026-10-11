package config

import (
	"fmt"
	"slices"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
)

// PruneUnhandled adapts the commands to the handlers this binary registers
// (a feature compiled out with a build tag, or a scan path with nothing to
// answer it), so disabled features neither match speech nor show up in hints:
//   - an action naming a missing handler that has an explicit skip callback
//     is an optional branch: it is pinned to an unreachable state so the
//     executor always takes the skip path, and stops consuming the trailer
//   - any other missing action drops the command, since running the rest of
//     its sequence would only do part of the job
//   - a callback naming a missing handler is only a fallback, so it becomes
//     the default for its result (__fail__ on fail, otherwise __continue__)
//
// Categories left empty are dropped too. Returns one "category.command
// (detail)" entry per change.
func (c *Config) PruneUnhandled(has func(commands.CommandName) bool) []string {
	var changes []string
	for category, cmds := range c.RawCommands {
		for name, cmd := range cmds {
			if missing := missingHandler(cmd, has); missing != nil {
				changes = append(changes, fmt.Sprintf("%s.%s (dropped: %s)", category, name, missing.Method))
				delete(cmds, name)
				continue
			}
			for i, a := range cmd.Actions {
				if handled(a, has) {
					continue
				}
				skipped := *a
				skipped.States = []string{stateUnavailable}
				skipped.Excludes = nil
				skipped.RequiresTrailer = false
				skipped.Args = nil
				changes = append(changes, fmt.Sprintf("%s.%s (action %s -> skip)", category, name, a.Method))
				cmd.Actions[i] = &skipped
			}
			for i, cb := range cmd.ActionCallbacks {
				if handled(cb, has) {
					continue
				}
				replacement := *cb
				replacement.Method = commands.MethodContinue
				if cb.Result == commands.ResultFail {
					replacement.Method = commands.MethodFail
				}
				changes = append(changes, fmt.Sprintf("%s.%s (callback %s -> %s)", category, name, cb.Method, replacement.Method))
				cmd.ActionCallbacks[i] = &replacement
			}
		}
		if len(cmds) == 0 {
			delete(c.RawCommands, category)
		}
	}
	c.Settings.CategoryOrder = slices.DeleteFunc(slices.Clone(c.Settings.CategoryOrder), func(category string) bool {
		_, ok := c.RawCommands[category]
		return !ok
	})
	slices.Sort(changes)
	return changes
}

// stateUnavailable is never active, so an action pinned to it always skips
const stateUnavailable = "__unavailable__"

// missingHandler returns the first action whose handler isn't registered and
// that has no skip callback to fall back on, or nil when the command can run
func missingHandler(cmd commands.Command, has func(commands.CommandName) bool) *commands.Action {
	for _, a := range cmd.Actions {
		if !handled(a, has) && !hasSkipCallback(cmd, a) {
			return a
		}
	}
	return nil
}

func hasSkipCallback(cmd commands.Command, a *commands.Action) bool {
	if a.ID == "" {
		return false
	}
	for _, cb := range cmd.ActionCallbacks {
		if cb.For == a.ID && cb.Result == commands.ResultSkip {
			return true
		}
	}
	return false
}

func handled(a *commands.Action, has func(commands.CommandName) bool) bool {
	switch a.Method {
	case "", commands.MethodContinue, commands.MethodDone, commands.MethodFail:
		return true
	}
	return has(commands.CommandName(a.Method))
}
