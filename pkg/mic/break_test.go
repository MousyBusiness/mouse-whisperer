package mic

import (
	"testing"
	"time"
)

// TestDecideBreak pins every break-path exit (HANDOVER R2): a mistake here
// either double-delivers a session (both evaluators act — duplicate
// keystrokes) or leaks one (a consumer left waiting for a Final).
func TestDecideBreak(t *testing.T) {
	cases := []struct {
		name                      string
		streamed, prompted, noise bool
		want                      breakDecision
	}{
		{
			name: "legacy triggered",
			want: breakDecision{deliverLegacy: true, clearRequest: true},
		},
		{
			name:     "legacy prompted",
			prompted: true,
			want:     breakDecision{deliverLegacy: true, clearRequest: true},
		},
		{
			// the stream Final REPLACES legacy delivery — deliverLegacy here
			// would make both evaluators act (duplicate keystrokes)
			name:     "streamed triggered",
			streamed: true,
			want:     breakDecision{streamFinal: true},
		},
		{
			// prompted takeover: abort the streamed session, then the
			// untouched legacy prompted path takes the recording
			name:     "prompted takeover mid-streamed-utterance",
			streamed: true,
			prompted: true,
			want:     breakDecision{streamFinal: true, streamAborted: true, deliverLegacy: true, clearRequest: true},
		},
		{
			name:  "noise",
			noise: true,
			want:  breakDecision{discardNoise: true},
		},
		{
			// ir must survive a noise false-start: the next real start
			// re-enters the start block with the same request
			name:     "noise while prompted keeps the request",
			prompted: true,
			noise:    true,
			want:     breakDecision{discardNoise: true},
		},
		{
			// a streamed session must never end silently — the consumer
			// would dangle until the next session's ID resets it
			name:     "noise aborts the streamed session",
			streamed: true,
			noise:    true,
			want:     breakDecision{discardNoise: true, streamFinal: true, streamAborted: true},
		},
		{
			name:     "noise with streamed session and pending prompt",
			streamed: true,
			prompted: true,
			noise:    true,
			want:     breakDecision{discardNoise: true, streamFinal: true, streamAborted: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideBreak(tc.streamed, tc.prompted, tc.noise); got != tc.want {
				t.Fatalf("decideBreak(streamed=%v, prompted=%v, noise=%v) = %+v, want %+v",
					tc.streamed, tc.prompted, tc.noise, got, tc.want)
			}
		})
	}
}

// TestDecideBreakInvariants sweeps the whole input space for the properties
// R2 states, independent of the exact table above.
func TestDecideBreakInvariants(t *testing.T) {
	bools := []bool{false, true}
	for _, streamed := range bools {
		for _, prompted := range bools {
			for _, noise := range bools {
				d := decideBreak(streamed, prompted, noise)

				// exactly one owner per session: discarded as noise,
				// delivered legacy, or consumed by a clean stream Final
				owners := 0
				if d.discardNoise {
					owners++
				}
				if d.deliverLegacy {
					owners++
				}
				if d.streamFinal && !d.streamAborted {
					owners++
				}
				if owners != 1 {
					t.Errorf("(%v,%v,%v): %d owners, want exactly 1: %+v", streamed, prompted, noise, owners, d)
				}

				// a streamed session always sees exactly one Final —
				// ending one silently leaves the consumer dangling
				if streamed != d.streamFinal {
					t.Errorf("(%v,%v,%v): streamed=%v but streamFinal=%v", streamed, prompted, noise, streamed, d.streamFinal)
				}

				// an abort flag without a Final to carry it is meaningless
				if d.streamAborted && !d.streamFinal {
					t.Errorf("(%v,%v,%v): streamAborted without streamFinal: %+v", streamed, prompted, noise, d)
				}

				// noise never finalizes or delivers, and keeps the request
				if noise && (d.deliverLegacy || d.clearRequest || !d.discardNoise) {
					t.Errorf("(%v,%v,%v): noise must discard only: %+v", streamed, prompted, noise, d)
				}

				// a request is cleared exactly when its session is delivered
				if d.clearRequest != d.deliverLegacy {
					t.Errorf("(%v,%v,%v): clearRequest=%v deliverLegacy=%v must match", streamed, prompted, noise, d.clearRequest, d.deliverLegacy)
				}
			}
		}
	}
}

// TestHoldForDictation: only a streamed utterance during dictation waits
// past the command break, and only up to the dictation break
func TestHoldForDictation(t *testing.T) {
	dictating := true
	opts := StreamOptions{DictationBreak: 1200 * time.Millisecond, Dictating: func() bool { return dictating }}

	if !opts.holdForDictation(true, 600*time.Millisecond) {
		t.Fatal("dictating stream should hold within the dictation break")
	}
	if opts.holdForDictation(true, 1300*time.Millisecond) {
		t.Fatal("must break once the dictation break passes")
	}
	if opts.holdForDictation(false, 600*time.Millisecond) {
		t.Fatal("prompted/unstreamed sessions keep their own break")
	}
	dictating = false
	if opts.holdForDictation(true, 600*time.Millisecond) {
		t.Fatal("commands outside dictation keep the command break")
	}
	if (StreamOptions{}).holdForDictation(true, 600*time.Millisecond) {
		t.Fatal("unset options must never hold")
	}
}
