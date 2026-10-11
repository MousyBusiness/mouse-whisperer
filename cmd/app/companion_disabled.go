//go:build nocompanion

package main

import (
	"context"

	"github.com/mousybusiness/mouse_whisperer/internal/config"
	"github.com/mousybusiness/mouse_whisperer/internal/evaluation"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
)

func startCompanion(ctx context.Context, cfg *config.Config, eval *evaluation.Evaluator, lstate *state.State) {
}
