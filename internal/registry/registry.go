package registry

import (
	"context"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/features"
	"github.com/mousybusiness/mouse_whisperer/internal/impl"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
)

type Handler = func(ctx context.Context, args map[string]any) (context.Context, error)

// HandlerRegistry manages all command handlers
type HandlerRegistry struct {
	handlers map[commands.CommandName]Handler
}

// NewHandlerRegistry creates a new handler registry
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[commands.CommandName]Handler),
	}
}

// Register adds a handler to the registry
func (r *HandlerRegistry) Register(n commands.CommandName, handler Handler) {
	r.handlers[n] = handler
}

// Get retrieves a handler by name
func (r *HandlerRegistry) Get(n commands.CommandName) (Handler, bool) {
	h, ok := r.handlers[n]
	return h, ok
}

// Has reports whether a handler is registered; config pruning uses it to
// drop commands that belong to features compiled out of this binary
func (r *HandlerRegistry) Has(n commands.CommandName) bool {
	_, ok := r.handlers[n]
	return ok
}

// RegisterAll registers the core handlers plus those of every optional
// feature built into this binary (see internal/features)
func (r *HandlerRegistry) RegisterAll(s *state.State) {
	impl.RegisterSystemHandlers(r, s)
	impl.RegisterHintsHandlers(r, s)
	impl.RegisterMacHandlers(r, s)
	impl.RegisterTextHandlers(r, s)
	impl.RegisterMouseHandlers(r, s)
	impl.RegisterDictationHandlers(r, s)

	if features.Voice {
		impl.RegisterVoiceHandlers(r, s)
	}
	if features.Eye {
		impl.RegisterGazeHandlers(r, s)
	}
	// scan/click-text commands need something to resolve targets: local OCR,
	// or a screend client (nil when screend is compiled out or disabled in
	// settings, so the commands are pruned instead of always failing)
	if features.OCR || s.ScanClient != nil {
		impl.RegisterOCRHandlers(r, s)
	}
	if features.OCR {
		impl.RegisterElementHandlers(r, s)
	}
	if features.Companion {
		impl.RegisterCompanionHandlers(r, s)
	}
	if features.AI {
		impl.RegisterAIHandlers(r, s)
	}
	if features.Workspace {
		impl.RegisterWorkspaceHandlers(r, s)
	}
}
