package xctx

import (
	"context"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
)

type ExecutionContext struct {
	TriggerWords []string
	TrailerWords []string
	Trailer      commands.TrailerText
	Multiplier   int
	Entry        *commands.CommandEntry
	// InvocationSource is how the command was invoked: "voice", "companion", etc.
	InvocationSource string
}
type commandContextKey struct{}

func With(ctx context.Context, e *ExecutionContext) context.Context {
	return context.WithValue(ctx, commandContextKey{}, e)
}

func Get(ctx context.Context) *ExecutionContext {
	e, ok := ctx.Value(commandContextKey{}).(*ExecutionContext)
	if !ok {
		return nil
	}

	return e
}

type currentActionKey struct{}

func WithCurrentAction(ctx context.Context, a *commands.Action) context.Context {
	return context.WithValue(ctx, currentActionKey{}, a)
}

func GetCurrentAction(ctx context.Context) *commands.Action {
	a, ok := ctx.Value(currentActionKey{}).(*commands.Action)
	if !ok {
		return nil
	}

	return a
}

func WithNamedString(ctx context.Context, name string, value string) context.Context {
	return context.WithValue(ctx, name, value)
}

func GetNamedString(ctx context.Context, name string) string {
	v, ok := ctx.Value(name).(string)
	if !ok {
		return ""
	}

	return v
}
