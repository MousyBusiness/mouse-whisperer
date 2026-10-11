package asr

import (
	"context"
	"os"

	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/transcribe"
	log "github.com/sirupsen/logrus"
)

// Partial is one incremental transcription of an in-progress utterance.
type Partial struct {
	Session mic.RecordingSession
	Seq     int    // consecutive from 1 within the session
	Text    string // raw whisper text of ALL audio so far this session
	Final   bool   // exactly once per session, highest Seq
}

// Final with empty Text means the session was aborted (noise, prompted
// takeover, device switch): the consumer drops it silently.

// Run consumes mic stream events and emits ordered Partials. A single
// worker goroutine makes Seq order and Final-last structural rather than
// synchronized.
func Run(ctx context.Context, events <-chan mic.StreamEvent) <-chan Partial {
	out := make(chan Partial)
	go func() {
		defer close(out)

		var curID uint64
		seq := 0
		lastText := ""

		for {
			var ev mic.StreamEvent
			var ok bool
			select {
			case <-ctx.Done():
				return
			case ev, ok = <-events:
				if !ok {
					return
				}
			}

			if ev.Session.ID != curID {
				curID = ev.Session.ID
				seq = 0
				lastText = ""
			}

			if ev.Final {
				if ev.Aborted {
					// Only sessions the consumer has already seen need a
					// closing Final; empty text is the cancel convention
					if seq > 0 {
						send(ctx, out, Partial{Session: ev.Session, Seq: seq + 1, Final: true})
					}
					continue
				}
				text, err := transcribe.File(ev.Session.FileName, transcribe.CommandPrompt, mic.ModelFast)
				if err != nil {
					if err.Error() != "noise" {
						log.Warnf("asr: final transcription failed: %v", err)
					}
					text = lastText
				}
				send(ctx, out, Partial{Session: ev.Session, Seq: seq + 1, Text: text, Final: true})
				continue
			}

			if text, ok := transcribeTick(ev); ok {
				seq++
				lastText = text
				send(ctx, out, Partial{Session: ev.Session, Seq: seq, Text: text})
			}
			// Release the mic's in-flight token only after the partial is
			// delivered: a slow consumer back-pressures into tick skipping
			// rather than queueing stale audio.
			close(ev.Done)
		}
	}()
	return out
}

func send(ctx context.Context, out chan<- Partial, p Partial) bool {
	select {
	case out <- p:
		return true
	case <-ctx.Done():
		return false
	}
}

// transcribeTick snapshots the growing session WAV and transcribes it with
// the fast model. Errors and noise results skip the tick entirely so Seq
// stays consecutive over emitted partials.
func transcribeTick(ev mic.StreamEvent) (string, bool) {
	snap, err := mic.SnapshotWAV(ev.Session.FileName, ev.NSamples)
	if err != nil {
		log.Warnf("asr: snapshot failed: %v", err)
		return "", false
	}
	defer os.Remove(snap)

	text, err := transcribe.File(snap, transcribe.CommandPrompt, mic.ModelFast)
	if err != nil {
		if err.Error() != "noise" {
			log.Warnf("asr: tick transcription failed: %v", err)
		}
		return "", false
	}
	return text, true
}
