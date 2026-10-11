package impl

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/transcribe"
	log "github.com/sirupsen/logrus"
)

// RegisterVoiceHandlers registers the commands that control the microphone
// itself (pausing listening, choosing the audio input device) and the
// dictation replay, which re-reads recorded audio
func RegisterVoiceHandlers(registry Registry, lstate *state.State) {
	registry.Register("dictation_replay", func(ctx context.Context, args map[string]any) (context.Context, error) {
		return ctx, replayDictation(lstate, args)
	})

	registry.Register("enable_voice", func(ctx context.Context, _ map[string]any) (context.Context, error) {
		lstate.SetState(state.StateNotListening, false)
		overlay.SendUIMessage("Started listening!")
		overlay.SendNotListeningIndicator(false)
		return ctx, nil
	})

	registry.Register("disable_voice", func(ctx context.Context, _ map[string]any) (context.Context, error) {
		lstate.SetState(state.StateNotListening, true)
		overlay.SendUIMessage("Stopped listening!")
		overlay.SendNotListeningIndicator(true)
		return ctx, nil
	})

	registry.Register("show_inputs", func(ctx context.Context, args map[string]any) (context.Context, error) {
		// Import mic package is needed at the top of the file
		// For now, we'll need to add the import
		devices, err := lstate.EnumerateAudioInputs()
		if err != nil {
			overlay.SendUIMessage("Failed to list audio inputs")
			log.Errorf("Failed to enumerate audio inputs: %v", err)
			return ctx, err
		}

		if len(devices) == 0 {
			overlay.SendUIMessage("No audio input devices found")
			return ctx, nil
		}

		// Store devices in state for pick handler
		lstate.Global.AudioInputList = devices
		lstate.Global.ViewingInputs = true

		// Reset other navigation state
		lstate.Global.CurrentCategory = ""
		lstate.Global.CurrentCommands = nil

		// Build list of available devices with indices
		var items []string
		for i, device := range devices {
			items = append(items, fmt.Sprintf("%d. %s", i+1, device.Name))
		}
		items = append(items, "", "Say 'pick N' to select input")

		// Send to overlay
		overlay.SendCommandsList("AUDIO INPUTS", items, 20)
		return ctx, nil
	})
}

// replayDictation re-transcribes the dictation buffer from its recordings
// ("replay that"). Each run of recorded segments is joined into one WAV,
// so whisper hears whole sentences instead of the pause-split fragments
// it transcribed live; text without recordings is kept as-is. The accurate
// model is the default; args.model picks another ("fast").
func replayDictation(lstate *state.State, args map[string]any) error {
	text := lstate.Dictation.Text()
	runs := lstate.Dictation.ReplayRuns()
	recorded := false
	for _, r := range runs {
		recorded = recorded || len(r.Clips) > 0
	}
	if !recorded {
		overlay.ShowToast("Replay", "no recorded audio in the buffer", "info")
		return nil
	}

	model := mic.ModelAccurate
	if v, ok := args["model"].(string); ok && v != "" {
		model = mic.ParseModel(v)
	}

	overlay.SendDictation(text, "· replaying ·", nil)
	for i, r := range runs {
		if len(r.Clips) == 0 {
			continue
		}
		replayed, err := transcribeClips(r.Clips, model)
		if err != nil {
			overlay.SendDictation(text, "", nil)
			overlay.ShowToast("Replay failed", err.Error(), "error")
			return err
		}
		// noise or an empty read keeps what the live pass heard
		if replayed != "" {
			runs[i].Text = replayed
		}
	}

	lstate.Dictation.ReplaceRuns(runs)
	if lstate.Dictation.HasText() {
		overlay.SendDictation(lstate.Dictation.Text(), "", nil)
	} else {
		lstate.SetState(state.StateDictating, false)
		overlay.DismissDictation()
	}
	log.Infof("dictation replay (%s model): %q -> %q", model, text, lstate.Dictation.Text())
	return nil
}

// transcribeClips joins the recordings and transcribes them as one;
// whisper's "noise" (silence-only) result is an empty string, not an error
func transcribeClips(clips []string, model mic.ModelType) (string, error) {
	joined, err := mic.JoinWAVs(clips)
	if err != nil {
		return "", err
	}
	defer os.Remove(joined)
	text, err := transcribe.FileLong(joined, transcribe.ProsePrompt, model)
	if err != nil && err.Error() == "noise" {
		return "", nil
	}
	// whisper breaks long reads into lines; the buffer wants one flow
	return strings.Join(strings.Fields(text), " "), err
}
