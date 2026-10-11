//go:build novoice

package mic

import (
	"context"
	"errors"
)

// ErrDisabled is returned by audio-device calls in a build without voice.
var ErrDisabled = errors.New("voice is not built into this binary (novoice)")

// Run never records in a novoice build; it only waits for shutdown.
func Run(ctx context.Context, ch chan RecordingSession, inputRequestCh <-chan *RecordingRequest, streamCh chan<- StreamEvent, opts StreamOptions) {
	<-ctx.Done()
}

func EnumerateInputDevices() ([]DeviceInfo, error) {
	return nil, ErrDisabled
}

func RequestDeviceChange(deviceIndex int) error {
	return ErrDisabled
}
