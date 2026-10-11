package impl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/commands"
	"github.com/mousybusiness/mouse_whisperer/internal/execution/xctx"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	localstate "github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/ai"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	log "github.com/sirupsen/logrus"
)

// RegisterAIHandlers registers the ollama-backed handlers: free-form AI
// prompts and the LLM dictation edits (tidy, revise)
func RegisterAIHandlers(registry Registry, state *localstate.State) {
	registry.Register("ai_ollama", func(ctx context.Context, args map[string]any) (context.Context, error) {
		_, t, err := captureInput(ctx, state, nil)
		if err != nil {
			return ctx, err
		}

		var aiPrompt string
		if v, ok := args["ai_prompt"].(string); ok {
			aiPrompt = v
			if v == "" {
				return ctx, errors.New("ai_prompt required")
			}
		} else {
			return ctx, errors.New("ai_prompt required")
		}

		var model string
		if v, ok := args["model"].(string); ok && v != "" {
			model = v
		} else {
			model = ai.DefaultModel
		}

		fullPrompt := fmt.Sprintf("%v: %v", aiPrompt, t)
		log.Debugf("Sending to Ollama (model: %v): %v", model, fullPrompt)
		response, err := ai.SendToOllama(model, fullPrompt)
		if err != nil {
			return ctx, err
		}

		log.Debugf("Response from Ollama: %v", response)

		cleaned := util.StripAIFormating(response)
		if cleaned == "" {
			return ctx, errors.New("no response from ollama")
		}
		state.Coding.PendingSuggestion = &cleaned
		overlay.ConfirmAIResponsePosition(t, cleaned)

		return ctx, nil
	})

	// defaultRevisePrompt applies a spoken instruction ("I was talking
	// about privacy, not piracy") to the dictated buffer
	const defaultRevisePrompt = "The text between the <text> tags is dictated prose. Apply the " +
		"instruction between the <instruction> tags to it, everywhere in the text it applies — " +
		"a correction like 'I meant X not Y' must replace every occurrence of Y. Keep everything " +
		"the instruction does not concern exactly as-is. Respond with ONLY the revised text " +
		"wrapped in <output></output> tags."

	registry.Register("dictation_revise", func(ctx context.Context, args map[string]any) (context.Context, error) {
		text := state.Dictation.Text()
		if strings.TrimSpace(text) == "" {
			return ctx, nil
		}
		instruction := ""
		if ex := xctx.Get(ctx); ex != nil {
			instruction = strings.TrimSpace(string(ex.Trailer))
		}
		if instruction == "" {
			// No instruction spoken with the trigger: show the recording
			// indicator and await one. "abort"/"cancel" or silence cancels.
			overlay.SendDictation(text, "· instruction? ·", nil)
			var err error
			ctx, instruction, err = captureInput(ctx, state, &commands.InputAction{
				Prompt:              "An instruction for revising dictated text. output: ",
				Break:               2 * time.Second,
				Timeout:             5 * time.Second,
				VoiceModel:          "accurate",
				AllowSymbols:        true,
				PreservePunctuation: true,
			})
			if err != nil {
				overlay.SendDictation(text, "", nil)
				return ctx, err
			}
			instruction = strings.TrimSpace(instruction)
			if instruction == "" {
				// Aborted or timed out: leave the buffer untouched
				overlay.SendDictation(text, "", nil)
				return ctx, nil
			}
		}

		model := ai.DefaultModel
		if v, ok := args["model"].(string); ok && v != "" {
			model = v
		}
		prompt := defaultRevisePrompt
		if v, ok := args["ai_prompt"].(string); ok && v != "" {
			prompt = v
		}

		overlay.SendDictation(text, "· revising ·", nil)
		full := fmt.Sprintf("%v\n\n<instruction>\n%v\n</instruction>\n\n<text>\n%v\n</text>", prompt, instruction, text)
		response, err := ai.SendToOllama(model, full)
		if err != nil {
			overlay.SendDictation(text, "", nil)
			overlay.ShowToast("Revise failed", err.Error(), "error")
			return ctx, err
		}

		cleaned, tagged := util.ExtractTagged(response, "output")
		if !tagged {
			cleaned = util.StripAIFormating(response)
		}
		if cleaned == "" {
			overlay.SendDictation(text, "", nil)
			overlay.ShowToast("Revise failed", "empty response", "error")
			return ctx, errors.New("revise: empty response from ollama")
		}

		state.Dictation.ReplaceAll(cleaned)
		overlay.SendDictation(cleaned, "", nil)
		return ctx, nil
	})

	// defaultTidyPrompt reflows pause-punctuated dictation ("this text is.
	// an example. of something,I have constructed") into cohesive prose.
	// The tag contract lets us extract exactly the rewrite and discard any
	// preamble/commentary small models like to add.
	const defaultTidyPrompt = "The text between the <input> tags was dictated with speech pauses, " +
		"so it contains spurious periods, commas and ellipses in the middle of sentences, plus " +
		"broken capitalization. Rewrite it as cohesive, correctly punctuated prose. Keep the " +
		"original wording and meaning; do not add new content. Respond with ONLY the rewritten " +
		"text wrapped in <output></output> tags."

	registry.Register("dictation_tidy", func(ctx context.Context, args map[string]any) (context.Context, error) {
		text := state.Dictation.Text()
		if strings.TrimSpace(text) == "" {
			return ctx, nil
		}

		model := ai.DefaultModel
		if v, ok := args["model"].(string); ok && v != "" {
			model = v
		}
		prompt := defaultTidyPrompt
		if v, ok := args["ai_prompt"].(string); ok && v != "" {
			prompt = v
		}

		overlay.SendDictation(text, "· tidying ·", nil)
		response, err := ai.SendToOllama(model, fmt.Sprintf("%v\n\n<input>\n%v\n</input>", prompt, text))
		if err != nil {
			overlay.SendDictation(text, "", nil)
			overlay.ShowToast("Tidy failed", err.Error(), "error")
			return ctx, err
		}

		cleaned, tagged := util.ExtractTagged(response, "output")
		if !tagged {
			// Model ignored the tag contract: fall back to the whole
			// response minus code fences
			cleaned = util.StripAIFormating(response)
		}
		if cleaned == "" {
			overlay.SendDictation(text, "", nil)
			overlay.ShowToast("Tidy failed", "empty response", "error")
			return ctx, errors.New("tidy: empty response from ollama")
		}

		state.Dictation.ReplaceAll(cleaned)
		overlay.SendDictation(cleaned, "", nil)
		return ctx, nil
	})
}
