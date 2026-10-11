package mic

import "time"

// StreamEvent is emitted by the mic loop at snapshot points during a
// triggered recording session so audio can be transcribed while speech
// is still in progress.
type StreamEvent struct {
	// Session identifies the utterance. Stream events always carry a zero
	// RecordingRequest: prompted sessions never stream.
	Session RecordingSession
	// NSamples is the number of samples on disk at snapshot time (ticks only)
	NSamples int
	// Final marks the end of the session (silence break)
	Final bool
	// Aborted is set on Final when there is no usable audio: noise-guard
	// discard, prompted takeover mid-utterance, or a device switch
	Aborted bool
	// Done is closed by the consumer once a tick is fully processed; the
	// mic skips further ticks while it is open (skip, never queue)
	Done chan struct{}
}

// StreamOptions controls mid-utterance snapshotting.
type StreamOptions struct {
	TickInterval time.Duration // how often to snapshot while speaking
	MaxDuration  time.Duration // stop ticking beyond this utterance length
	// DictationBreak is the silence that ends a streamed utterance while
	// Dictating reports true: longer than the command break, so natural
	// pauses don't split sentences into fragments whisper reads without
	// context. Zero, or a nil Dictating, keeps the command break.
	DictationBreak time.Duration
	Dictating      func() bool
}

// holdForDictation reports whether a streamed utterance that has been
// silent for silence (already past the command break) should keep
// recording because dictation is in progress. Dictating is only consulted
// once the command break has passed, keeping it off the 4ms read path.
func (o StreamOptions) holdForDictation(streamed bool, silence time.Duration) bool {
	return streamed && o.Dictating != nil && silence <= o.DictationBreak && o.Dictating()
}
