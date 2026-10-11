package impl

import (
	"context"
	"strings"

	"github.com/go-vgo/robotgo"
	"github.com/go-vgo/robotgo/clipboard"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/iancoleman/strcase"
	kbd "github.com/micmonay/keybd_event"
)

// RegisterHandlers registers all handlers with the registry
func RegisterTextHandlers(registry Registry, state *state.State) {
	registry.Register("text_title_that", func(ctx context.Context, args map[string]any) (context.Context, error) {
		key(kbd.VK_C, "cmd")
		clip, err := clipboard.ReadAll()
		if err != nil {
			return ctx, err
		}
		robotgo.Type(strings.ToUpper(string(clip[0])) + clip[1:])
		return ctx, nil
	})

	registry.Register("text_lowercase_that", func(ctx context.Context, args map[string]any) (context.Context, error) {
		key(kbd.VK_C, "cmd")
		clip, err := clipboard.ReadAll()
		if err != nil {
			return ctx, err
		}
		robotgo.Type(strings.ToLower(clip))
		return ctx, nil
	})

	registry.Register("text_uppercase_that", func(ctx context.Context, args map[string]any) (context.Context, error) {
		key(kbd.VK_C, "cmd")
		clip, err := clipboard.ReadAll()
		if err != nil {
			return ctx, err
		}
		robotgo.Type(strings.ToUpper(clip))
		return ctx, nil
	})

	registry.Register("text_kebab_that", func(ctx context.Context, args map[string]any) (context.Context, error) {
		key(kbd.VK_C, "cmd")
		clip, err := clipboard.ReadAll()
		if err != nil {
			return ctx, err
		}
		robotgo.Type(strcase.ToKebab(clip))
		return ctx, nil
	})

	registry.Register("text_snake_that", func(ctx context.Context, args map[string]any) (context.Context, error) {
		key(kbd.VK_C, "cmd")
		clip, err := clipboard.ReadAll()
		if err != nil {
			return ctx, err
		}
		robotgo.Type(strcase.ToSnake(clip))
		return ctx, nil
	})

	registry.Register("text_camel_that", func(ctx context.Context, args map[string]any) (context.Context, error) {
		key(kbd.VK_C, "cmd")
		clip, err := clipboard.ReadAll()
		if err != nil {
			return ctx, err
		}
		robotgo.Type(strcase.ToLowerCamel(clip))
		return ctx, nil
	})

	registry.Register("text_pascal_that", func(ctx context.Context, args map[string]any) (context.Context, error) {
		key(kbd.VK_C, "cmd")
		clip, err := clipboard.ReadAll()
		if err != nil {
			return ctx, err
		}
		robotgo.Type(strcase.ToCamel(clip))
		return ctx, nil
	})
}
