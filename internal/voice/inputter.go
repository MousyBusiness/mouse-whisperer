package voice

import (
	"sync"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/features"
	"github.com/mousybusiness/mouse_whisperer/internal/overlay"
	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	log "github.com/sirupsen/logrus"
)

type Input struct {
	listening             bool
	awaitingInput         bool
	awaitingInputSignalCh chan *mic.RecordingRequest

	sync.RWMutex
}

func NewInput(signalCh chan *mic.RecordingRequest) *Input {
	return &Input{
		awaitingInputSignalCh: signalCh,
	}
}

// Enabled returns whether the evaluator is listening
func (e *Input) Enabled() bool {
	e.RLock()
	defer e.RUnlock()

	return e.listening
}

// AwaitInputWithTimeout implements the Inputter interface
func (e *Input) AwaitInputWithTimeout(prompt string, allowedBreak, timeout time.Duration, model mic.ModelType) (string, bool) {
	if !features.Voice {
		// nothing would ever consume the request: a command that needs
		// spoken input must say so via its trailer instead
		log.Warnf("Voice prompt %q unavailable in a novoice build", prompt)
		return "", false
	}
	log.Infof("Awaiting input with model: %s", model)
	go func() {
		overlay.SendRecordingIndicator(true, int32(time.Minute.Milliseconds()))
	}()
	defer func() {
		go func() {
			overlay.SendRecordingIndicator(false, 0)
		}()
	}()

	e.Lock()
	e.awaitingInput = true
	e.Unlock()

	defer e.resetInput()

	r := mic.NewPromptRequest(prompt, allowedBreak, timeout, model)

	e.awaitingInputSignalCh <- r

	// First, wait for recording to START or initial timeout
	select {
	case <-r.RecordingStarted():
		log.Info("Recording started, waiting for completion...")
	case <-r.Timeout():
		log.Warn("Initial timeout - no recording started")
		return "", false
	case <-time.After(time.Minute):
		log.Error("Hard timeout waiting for recording to start")
		return "", false
	}

	// Now wait for recording to complete
	select {
	case <-r.Recorded():
		log.Info("Recording completed")
		// continue for transcription
	case <-time.After(time.Minute):
		log.Error("Hard timeout waiting for recording to complete")
		return "", false
	}

	select {
	case s, ok := <-r.Transcribed():
		if !ok {
			log.Warnf("Input cancelled")
			return "", false
		}

		//if e.awaitingInput {
		//	log.Debugf("Awaiting input")
		//	// Check if user wants to abort - strip punctuation for more robust matching
		//	text := strings.TrimSpace(strings.ToLower(util.TrimTrailingPunctuation(transcription.Text())))
		//	if text == "abort" || text == "bye" || text == "cancel" || text == "nevermind" || text == "never mind" {
		//		log.Warning("Input aborted by user")
		//		overlay.SendUIMessage("Aborted")
		//		Session.RecordingRequest.Abort()
		//		return
		//	}
		//
		//	transcription.Session.RecordingRequest.SetTranscription(transcription.Text())
		//	return
		//}

		return util.TrimTrailingPunctuation(s), true
	case <-time.After(time.Minute):
		log.Error("Hard timeout waiting for transcription")
		return "", false
	}
}

func (e *Input) resetInput() {
	log.Warnf("Resetting input")
	e.Lock()
	e.awaitingInput = false
	e.Unlock()
	e.awaitingInputSignalCh <- nil
	go func() {
		overlay.SendRecordingIndicator(false, 0)
	}()
}
