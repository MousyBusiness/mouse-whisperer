package registry

import (
	"strings"
	"testing"

	"github.com/mousybusiness/mouse_whisperer/internal/config"
	"github.com/mousybusiness/mouse_whisperer/internal/features"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
)

// Every command in the shipped config must resolve to a registered handler
// when all features are built in; anything pruned here is a config typo or
// a handler that was renamed without updating the TOML.
func TestShippedConfigFullyHandled(t *testing.T) {
	cfg, err := config.Load("../../config")
	if err != nil {
		t.Fatal(err)
	}
	r := NewHandlerRegistry()
	r.RegisterAll(state.NewBareState())

	changes := cfg.PruneUnhandled(r.Has)
	if len(features.Disabled()) > 0 {
		t.Logf("built without %v, adapted: %s", features.Disabled(), strings.Join(changes, ", "))
		return
	}
	if len(changes) > 0 {
		t.Errorf("commands with no handler: %s", strings.Join(changes, ", "))
	}
}
