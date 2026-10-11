package voice

import (
	"context"
	"strings"

	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/transcribe"
	"github.com/mousybusiness/mouse_whisperer/pkg/util"
	log "github.com/sirupsen/logrus"
)

// isAbortPhrase reports whether a prompted-input utterance means "cancel"
// rather than being the input itself
func isAbortPhrase(text string) bool {
	switch strings.ToLower(strings.TrimSpace(util.TrimTrailingPunctuation(text))) {
	case "abort", "cancel", "nevermind", "never mind":
		return true
	}
	return false
}

type app struct {
	requestCh       chan *mic.RecordingRequest
	recordingsCh    chan mic.RecordingSession
	evaluator       Evaluator
	transcriptionCh chan *transcribe.Result
}

func (a *app) processVoiceRecordings(ctx context.Context, eval Evaluator) {
	a.evaluator = eval
	for {
		select {
		case session := <-a.recordingsCh:
			go func(session mic.RecordingSession) {

				// Get the model type from the session
				model := session.RecordingRequest.Model()

				log.Infof("Transcribing with model: %s", model)

				t, err := transcribe.Do(session, model)
				if err != nil {
					log.Println(err)
					return
				}

				a.transcriptionCh <- t

			}(session)
		case t := <-a.transcriptionCh:
			if t == nil {
				log.Errorf("Transcription channel closed")
				return
			}

			// Check if this was a prompted input request (e.g., from "fast" command awaiting text)
			// If so, route the transcription back to the request instead of evaluating as a command
			if t.Session.IsPrompted() {
				if isAbortPhrase(t.Text()) {
					log.Infof("Aborting prompted request: %s", t.Text())
					t.Session.RecordingRequest.Abort()
					continue
				}
				log.Infof("Routing transcription to prompted request: %s", t.Text())
				t.Session.SetTranscription(t.Text())
				continue
			}

			// Only allow smart correction if we used the fast model
			// If we already used the slow model, smart correction would just use the same model again
			allowSmartCorrection := t.Session.RecordingRequest.Model() == mic.ModelFast
			if !allowSmartCorrection {
				log.Debugf("Smart correction disabled - already using slow/accurate model")
			}

			go eval.EvaluateTranscription(*t, allowSmartCorrection)
		case <-ctx.Done():
			log.Warnf("Voice looper exit")
			return
		}
	}
}

type Evaluator interface {
	Enabled() bool
	EvaluateTranscription(transcription transcribe.Result, allowSmartCorrection bool)
}

func Run(ctx context.Context, eval Evaluator, requestCh chan *mic.RecordingRequest, sessionsCh chan mic.RecordingSession, streamCh chan mic.StreamEvent, streamOpts mic.StreamOptions) {
	a := &app{
		requestCh:       requestCh,
		recordingsCh:    sessionsCh,
		transcriptionCh: make(chan *transcribe.Result),
		evaluator:       nil,
	}
	go mic.Run(ctx, a.recordingsCh, a.requestCh, streamCh, streamOpts)

	go a.processVoiceRecordings(ctx, eval)
}
