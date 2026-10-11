package mic

import "time"

type RequestType int

const (
	REQUEST_TYPE_TRIGGERED = iota
	REQUEST_TYPE_PROPTED
)

// ModelType represents which Whisper model to use for transcription
type ModelType int

const (
	ModelFast     ModelType = iota // Fast model (tiny/small) - default
	ModelAccurate                  // Slow/accurate model (medium/large)
)

// String returns the string representation of the ModelType
func (m ModelType) String() string {
	switch m {
	case ModelAccurate:
		return "slow"
	case ModelFast:
		return "fast"
	default:
		return "fast"
	}
}

// ParseModel converts a string to ModelType
func ParseModel(s string) ModelType {
	switch s {
	case "slow", "accurate", "medium", "large", "small":
		return ModelAccurate
	case "fast", "quick", "tiny":
		return ModelFast
	default:
		return ModelFast // Default to fast
	}
}

type RecordingRequest struct {
	requetsType       RequestType
	prompt            string
	allowedBreak      time.Duration
	timeout           time.Duration
	model             ModelType // Determines which whisper model to use
	transcripionCh    chan string
	recordedSig       chan struct{}
	recordingStartSig chan struct{}
	timeoutSig        chan struct{}
}

// NewPromptRequest is when a command needs a follow-up input
func NewPromptRequest(prompt string, allowedBreak time.Duration, timeout time.Duration, model ModelType) *RecordingRequest {
	return &RecordingRequest{
		requetsType:       REQUEST_TYPE_PROPTED,
		prompt:            prompt,
		allowedBreak:      allowedBreak,
		timeout:           timeout,
		model:             model,
		timeoutSig:        make(chan struct{}, 1),
		recordedSig:       make(chan struct{}, 1),
		recordingStartSig: make(chan struct{}, 1),
		transcripionCh:    make(chan string, 1),
	}
}

// NewTriggeredRequest is when a recording is started without prompting e.g. a command
func NewTriggeredRequest() *RecordingRequest {
	return &RecordingRequest{
		requetsType:       REQUEST_TYPE_TRIGGERED,
		timeoutSig:        make(chan struct{}, 1),
		recordedSig:       make(chan struct{}, 1),
		recordingStartSig: make(chan struct{}, 1),
		transcripionCh:    make(chan string, 1),
	}
}

func (ir *RecordingRequest) Prompt() string {
	return ir.prompt
}

func (ir *RecordingRequest) Model() ModelType {
	return ir.model
}

func (ir *RecordingRequest) Recorded() <-chan struct{} {
	return ir.recordedSig
}

func (ir *RecordingRequest) RecordingStarted() <-chan struct{} {
	return ir.recordingStartSig
}

func (ir *RecordingRequest) Transcribed() <-chan string {
	return ir.transcripionCh
}

func (ir *RecordingRequest) Timeout() <-chan struct{} {
	return ir.timeoutSig
}

func (ir *RecordingRequest) SetTranscription(s string) {
	ir.transcripionCh <- s
}

func (ir *RecordingRequest) Abort() {
	close(ir.transcripionCh)
}

// IsPrompted returns true if this request is awaiting prompted input (not a triggered command)
func (ir *RecordingRequest) IsPrompted() bool {
	return ir.requetsType == REQUEST_TYPE_PROPTED
}
