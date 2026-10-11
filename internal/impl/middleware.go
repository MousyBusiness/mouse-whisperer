package impl

import (
	"context"
	"strings"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	log "github.com/sirupsen/logrus"
)

func captureInput(ctx context.Context, state *state.State, override *commands.InputAction) (context.Context, string, error) {
	ex := xctx.Get(ctx)
	a := xctx.GetCurrentAction(ctx).ToInputAction()

	if override != nil {
		a = *override
	}

	text := string(ex.Trailer)
	if text == "" {
		log.Debugf("No trailer text, prompting for input: %s", a.Prompt)
		log.Debugf("Timeout: %dms, Break: %dms", a.Timeout.Milliseconds(), a.Break.Milliseconds())
		p := promptOr(a.Prompt, "A word or sentence. output: ")
		// Parse voice_model string from config to ModelType enum
		model := mic.ParseModel(a.VoiceModel)
		if v, ok := state.Inputter.AwaitInputWithTimeout(p, a.Break, a.Timeout, model); ok {
			text = v
		} else {
			log.Debugf("Input timeout or cancelled, no text captured")
		}
	} else {
		log.Debugf("Using trailer text: %s", text)
	}

	text = applyInputConditions(text, a)
	text = strings.TrimSpace(text)

	ctx = xctx.WithNamedString(ctx, a.CaptureKey, text)
	return ctx, text, nil
}
