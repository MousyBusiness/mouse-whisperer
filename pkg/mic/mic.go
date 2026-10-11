//go:build !novoice

package mic

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/gordonklaus/portaudio"
	log "github.com/sirupsen/logrus"
)

var (
	// deviceChangeCh allows runtime switching of audio input device
	deviceChangeCh = make(chan int, 1)
	currentDevice  *portaudio.DeviceInfo
)

// sendStreamFinal notifies the streaming worker that the current session
// ended. Non-blocking: the mic loop must never stall on a 4ms read cadence.
func sendStreamFinal(streamCh chan<- StreamEvent, snip *snippet, aborted bool) {
	ev := StreamEvent{Session: snip.session(), Final: true, Aborted: aborted}
	select {
	case streamCh <- ev:
	default:
		log.Errorf("Stream channel full, dropping final event for session %d", snip.id)
	}
}

// Function to check if audio data contains a break (silence)
func isSilent(audio []int32) bool {
	for _, sample := range audio {
		amp := math.Abs(float64(sample))
		//fmt.Println(amp)
		if amp >= silenceThreshold {
			return false // Audio data contains non-silent part
		}
	}
	return true // Audio data is silent
}

// Run captures audio continuously. streamCh, when non-nil, receives
// mid-utterance snapshot ticks and session-final events for triggered
// sessions (streaming transcription); nil disables streaming entirely.
func Run(ctx context.Context, ch chan RecordingSession, inputRequestCh <-chan *RecordingRequest, streamCh chan<- StreamEvent, opts StreamOptions) {
	if opts.TickInterval <= 0 {
		opts.TickInterval = 350 * time.Millisecond
	}
	if opts.MaxDuration <= 0 {
		opts.MaxDuration = 12 * time.Second
	}
	portaudio.Initialize()
	defer portaudio.Terminate()
	queue := make([]int32, 0)
	in := make([]int32, 64)

	device, err := portaudio.DefaultInputDevice()
	chk(err)
	currentDevice = device

	devices, _ := portaudio.Devices()
	deviceIndex := -1
	for i, d := range devices {
		if d == device {
			deviceIndex = i
			break
		}
	}

	log.Infof("Selected audio input device: %s (index: %d, channels: %d, sample rate: %.0f Hz)",
		device.Name, deviceIndex, device.MaxInputChannels, device.DefaultSampleRate)

	stream, err := portaudio.OpenDefaultStream(channelCount, 0, sampleRate, len(in), in)
	chk(err)
	defer func() {
		log.Warnf("Audio stream closed")
		stream.Close()
	}()

	chk(stream.Start())

	// Assume silent
	var snip *snippet
	started := false
	silent := true
	silentAt := time.Now()
	var ir *RecordingRequest
	breakThreshold := commandBreakThreshold

	// startSignaledFor is the request whose recordingStartSig has already been
	// closed. A NOISE false-start (below) resets started/snip but keeps ir set,
	// so the next real start re-enters the start block with the same request;
	// without this guard we'd close an already-closed channel and panic.
	var startSignaledFor *RecordingRequest

	// Streaming state. tickInFlight tracks worker business, not session
	// state, so it deliberately survives session resets.
	streamSession := false
	lastTick := time.Time{}
	var tickInFlight chan struct{}
	go func(breakThreshold *int) {
		for {
			select {
			case <-ctx.Done():
				log.Warnf("Command looper exit")
				return
			case i := <-inputRequestCh:
				ir = i
				silentAt = time.Now()
				if i == nil {
					*breakThreshold = commandBreakThreshold
				} else {
					*breakThreshold = inputBreakThreshold
					if i.allowedBreak != 0 {
						*breakThreshold = int(i.allowedBreak.Milliseconds())
					}
				}
			}

		}
	}(&breakThreshold)

	for {
		select {
		case <-ctx.Done():
			log.Warnf("Mic closing")
			chk(stream.Stop())
			return
		case newDeviceIndex := <-deviceChangeCh:
			log.Infof("Device change requested to index %d", newDeviceIndex)

			// Get all devices
			devices, err := portaudio.Devices()
			if err != nil {
				log.Errorf("Failed to enumerate devices: %v", err)
				continue
			}

			// Validate device index
			if newDeviceIndex < 0 || newDeviceIndex >= len(devices) {
				log.Errorf("Invalid device index %d (valid range: 0-%d)", newDeviceIndex, len(devices)-1)
				continue
			}

			newDevice := devices[newDeviceIndex]
			if newDevice.MaxInputChannels == 0 {
				log.Errorf("Device %d (%s) has no input channels", newDeviceIndex, newDevice.Name)
				continue
			}

			// Stop and close current stream
			log.Infof("Stopping current stream...")
			if err := stream.Stop(); err != nil {
				log.Errorf("Failed to stop stream: %v", err)
				// Try to continue anyway
			}
			if err := stream.Close(); err != nil {
				log.Errorf("Failed to close stream: %v", err)
			}

			// Open new stream with selected device
			log.Infof("Opening stream with device: %s (index: %d)", newDevice.Name, newDeviceIndex)
			streamParams := portaudio.StreamParameters{
				Input: portaudio.StreamDeviceParameters{
					Device:   newDevice,
					Channels: channelCount,
					Latency:  newDevice.DefaultLowInputLatency,
				},
				SampleRate:      sampleRate,
				FramesPerBuffer: len(in),
			}

			stream, err = portaudio.OpenStream(streamParams, in)
			if err != nil {
				log.Errorf("Failed to open stream with device %d: %v", newDeviceIndex, err)
				// Try to revert to default device
				log.Warnf("Reverting to default device")
				defaultDevice, defErr := portaudio.DefaultInputDevice()
				if defErr != nil {
					log.Errorf("Failed to get default device: %v - audio will be unavailable", defErr)
					continue
				}
				stream, err = portaudio.OpenDefaultStream(channelCount, 0, sampleRate, len(in), in)
				if err != nil {
					log.Errorf("Failed to open default stream: %v - audio will be unavailable", err)
					continue
				}
				currentDevice = defaultDevice
			} else {
				currentDevice = newDevice
			}

			// Start the new stream
			if err := stream.Start(); err != nil {
				log.Errorf("Failed to start stream: %v", err)
				continue
			}

			log.Infof("Successfully switched to device: %s (index: %d)", currentDevice.Name, newDeviceIndex)

			// Close out a streamed session before dropping it so the
			// consumer isn't left waiting for a Final
			if streamSession && snip != nil {
				sendStreamFinal(streamCh, snip, true)
			}
			streamSession = false

			// Clear buffer state
			queue = queue[:0]
			started = false
			snip = nil
			silent = true
			silentAt = time.Now()

			continue
		default:
		}

		if err := stream.Read(); err != nil {
			// Input overflow means we didn't read fast enough - samples were lost but we can continue
			if err.Error() == "Input overflowed" {
				log.Warnf("Audio input overflow - some samples lost")
				continue
			}
			chk(err)
		}

		// Check if the audio data contains a break (silence)
		if isSilent(in) {
			if !silent {
				silentAt = time.Now()
				silent = true
			}

			if !started {
				queue = append(queue, in...)
				if len(queue) > 64*64 {
					queue = queue[64:]
				}

				if ir != nil {
					if time.Since(silentAt) > ir.timeout {
						log.Warn("Input request timeout")
						close(ir.timeoutSig)
						ir = nil

						log.Debug("Clearing buffer")
						// clear the buffer
						queue = queue[:0]
						started = false
						snip = nil
					}
				}
				continue
			}

		} else {
			if !started {
				started = true
				snip = createSnippetWav()
				// Prompted sessions never stream; their audio belongs to
				// the waiting command, not the evaluator
				streamSession = streamCh != nil && ir == nil
				lastTick = snip.created

				// Signal that recording has started (once per request; see
				// startSignaledFor)
				if ir != nil && startSignaledFor != ir {
					close(ir.recordingStartSig)
					startSignaledFor = ir
				}

				// Prefill with queue

			}
			silent = false
			silentAt = time.Time{}
		}

		if silent && time.Since(silentAt) > time.Millisecond*time.Duration(breakThreshold) &&
			!opts.holdForDictation(streamSession, time.Since(silentAt)) {
			d := decideBreak(streamSession, ir != nil, silentAt.Sub(snip.created) < time.Millisecond*45)
			streamSession = false

			if d.discardNoise {
				fmt.Println("NOISE")
				if d.streamFinal {
					sendStreamFinal(streamCh, snip, d.streamAborted)
				}
				started = false
				snip = nil
				continue
			}

			log.Infof("Sending mic recording: %s, ir: %v", snip.name, ir != nil)
			snip.finalizeWav()
			//snip.filterWav()
			req := NewTriggeredRequest()
			if ir != nil {
				req = ir
			}

			// The WAV must be finalized (headers correct) before the
			// streaming worker hears the session ended
			if d.streamFinal {
				sendStreamFinal(streamCh, snip, d.streamAborted)
			}

			close(req.recordedSig)

			if d.deliverLegacy {
				ch <- RecordingSession{
					ID:               snip.id,
					FileName:         snip.name,
					RecordingRequest: *req,
					RecordingStarted: snip.created,
				}
			}

			if d.clearRequest {
				ir = nil // Clear input request after it's been consumed
			}
			started = false
			snip = nil
			continue
		}

		if len(queue) > 0 {
			for _, sample := range queue {
				// Convert from 32-bit to 16-bit by shifting right (assuming the samples are in the lower bits)
				// You might need to adjust this if your audio samples are scaled differently.
				var sample16 int16 = int16(sample >> 16) // Adjust this line if needed for your audio data
				chk(binary.Write(snip.file, binary.LittleEndian, sample16))
			}

			// Pre-roll samples are part of the wav: without this the header
			// undercounts and readers drop the final 256ms of audio
			snip.nSamples += len(queue)
			queue = queue[:0]
		}

		// These writes must stay unbuffered (no bufio): SnapshotWAV reads
		// the live file concurrently and relies on nSamples-counted bytes
		// being on disk.
		for _, sample := range in {
			// Convert from 32-bit to 16-bit by shifting right (assuming the samples are in the lower bits)
			// You might need to adjust this if your audio samples are scaled differently.
			var sample16 int16 = int16(sample >> 16) // Adjust this line if needed for your audio data
			chk(binary.Write(snip.file, binary.LittleEndian, sample16))
		}

		snip.nSamples += len(in)

		// Mid-utterance snapshot ticks for streaming transcription. Skip
		// (never queue) while the previous tick is still being processed.
		if streamSession && ir == nil && time.Since(snip.created) < opts.MaxDuration {
			if tickInFlight != nil {
				select {
				case <-tickInFlight:
					tickInFlight = nil
				default:
				}
			}
			if tickInFlight == nil && time.Since(lastTick) >= opts.TickInterval {
				done := make(chan struct{})
				select {
				case streamCh <- StreamEvent{Session: snip.session(), NSamples: snip.nSamples, Done: done}:
					tickInFlight = done
					lastTick = time.Now()
				default: // worker saturated: drop this tick
				}
			}
		}
	}
}

// EnumerateInputDevices returns a list of all available audio input devices
func EnumerateInputDevices() ([]DeviceInfo, error) {
	devices, err := portaudio.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to enumerate devices: %w", err)
	}

	var inputDevices []DeviceInfo
	for i, device := range devices {
		if device.MaxInputChannels > 0 {
			inputDevices = append(inputDevices, DeviceInfo{
				Index: i,
				Name:  device.Name,
			})
		}
	}

	return inputDevices, nil
}

// GetCurrentDevice returns the currently selected audio input device
func GetCurrentDevice() *portaudio.DeviceInfo {
	return currentDevice
}

// RequestDeviceChange requests a change to a specific audio input device by index
func RequestDeviceChange(deviceIndex int) error {
	select {
	case deviceChangeCh <- deviceIndex:
		return nil
	default:
		return fmt.Errorf("device change request channel is full")
	}
}
