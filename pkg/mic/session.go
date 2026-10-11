package mic

import "time"

// DeviceInfo represents information about an audio input device
type DeviceInfo struct {
	Index int
	Name  string
}

type RecordingSession struct {
	ID       uint64 // monotonically increasing per utterance; 0 = unset
	FileName string
	RecordingRequest
	RecordingStarted time.Time
}
