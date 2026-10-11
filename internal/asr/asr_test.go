package asr

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/mousybusiness/mouse_whisperer/pkg/mic"
	"github.com/mousybusiness/mouse_whisperer/pkg/transcribe"
)

// makeSessionWav writes a minimal WAV-shaped file: 44 header bytes plus
// nSamples int16 samples. SnapshotWAV copies bytes, so content is arbitrary.
func makeSessionWav(t *testing.T, nSamples int) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.wav")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := f.Write(make([]byte, 44)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < nSamples; i++ {
		if err := binary.Write(f, binary.LittleEndian, int16(1)); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

// stubWhisper serves scripted transcriptions in request order.
// A response of "\x00fail" returns HTTP 500 instead.
func stubWhisper(t *testing.T, responses []string) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	i := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if i >= len(responses) {
			t.Errorf("unexpected whisper request #%d", i+1)
			http.Error(w, "no more responses", http.StatusInternalServerError)
			return
		}
		resp := responses[i]
		i++
		if resp == "\x00fail" {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"text": resp})
	}))
}

type harness struct {
	events chan mic.StreamEvent
	out    <-chan Partial
	cancel context.CancelFunc
}

func newHarness(t *testing.T, responses []string) *harness {
	t.Helper()
	server := stubWhisper(t, responses)
	prevFast := transcribe.FastURL
	transcribe.FastURL = server.URL
	t.Cleanup(func() {
		transcribe.FastURL = prevFast
		server.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	events := make(chan mic.StreamEvent, 8)
	return &harness{events: events, out: Run(ctx, events), cancel: cancel}
}

func (h *harness) tick(session mic.RecordingSession, nSamples int) chan struct{} {
	done := make(chan struct{})
	h.events <- mic.StreamEvent{Session: session, NSamples: nSamples, Done: done}
	return done
}

func (h *harness) final(session mic.RecordingSession, aborted bool) {
	h.events <- mic.StreamEvent{Session: session, Final: true, Aborted: aborted}
}

func (h *harness) collect(t *testing.T, n int) []Partial {
	t.Helper()
	var got []Partial
	for len(got) < n {
		select {
		case p := <-h.out:
			got = append(got, p)
		case <-time.After(3 * time.Second):
			t.Fatalf("timed out after %d partials, want %d", len(got), n)
		}
	}
	return got
}

func (h *harness) expectNone(t *testing.T) {
	t.Helper()
	select {
	case p := <-h.out:
		t.Fatalf("unexpected partial: %+v", p)
	case <-time.After(200 * time.Millisecond):
	}
}

func assertDone(t *testing.T, done chan struct{}) {
	t.Helper()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("tick Done never closed")
	}
}

func TestTicksThenFinal(t *testing.T) {
	wav := makeSessionWav(t, 4000)
	session := mic.RecordingSession{ID: 1, FileName: wav}
	h := newHarness(t, []string{"up", "up down", "up down down", "up down down left"})

	d1 := h.tick(session, 1000)
	d2 := h.tick(session, 2000)
	d3 := h.tick(session, 4000)
	h.final(session, false)

	got := h.collect(t, 4)
	assertDone(t, d1)
	assertDone(t, d2)
	assertDone(t, d3)

	wantTexts := []string{"up", "up down", "up down down", "up down down left"}
	for i, p := range got {
		if p.Seq != i+1 {
			t.Errorf("partial %d: seq %d, want %d", i, p.Seq, i+1)
		}
		if p.Text != wantTexts[i] {
			t.Errorf("partial %d: text %q, want %q", i, p.Text, wantTexts[i])
		}
		if p.Final != (i == 3) {
			t.Errorf("partial %d: final=%v", i, p.Final)
		}
	}
}

func TestNoiseTickSkipped(t *testing.T) {
	wav := makeSessionWav(t, 4000)
	session := mic.RecordingSession{ID: 2, FileName: wav}
	// Second tick transcribes to empty -> "noise" -> skipped, seq stays consecutive
	h := newHarness(t, []string{"up", "", "up down", "up down"})

	d1 := h.tick(session, 1000)
	d2 := h.tick(session, 2000)
	d3 := h.tick(session, 4000)
	h.final(session, false)

	got := h.collect(t, 3)
	assertDone(t, d1)
	assertDone(t, d2)
	assertDone(t, d3)

	if got[0].Seq != 1 || got[1].Seq != 2 || got[2].Seq != 3 {
		t.Fatalf("seqs not consecutive: %d %d %d", got[0].Seq, got[1].Seq, got[2].Seq)
	}
	if !got[2].Final || got[2].Text != "up down" {
		t.Fatalf("bad final: %+v", got[2])
	}
}

func TestFinalOnlySession(t *testing.T) {
	wav := makeSessionWav(t, 500)
	session := mic.RecordingSession{ID: 3, FileName: wav}
	h := newHarness(t, []string{"up"})

	h.final(session, false)
	got := h.collect(t, 1)
	if got[0].Seq != 1 || !got[0].Final || got[0].Text != "up" {
		t.Fatalf("bad final-only partial: %+v", got[0])
	}
}

func TestAbortedWithPriorPartials(t *testing.T) {
	wav := makeSessionWav(t, 4000)
	session := mic.RecordingSession{ID: 4, FileName: wav}
	h := newHarness(t, []string{"up"})

	d1 := h.tick(session, 1000)
	h.final(session, true)

	got := h.collect(t, 2)
	assertDone(t, d1)
	if !got[1].Final || got[1].Text != "" {
		t.Fatalf("expected empty-text cancel final, got %+v", got[1])
	}
}

func TestAbortedWithoutPartialsEmitsNothing(t *testing.T) {
	wav := makeSessionWav(t, 4000)
	session := mic.RecordingSession{ID: 5, FileName: wav}
	h := newHarness(t, nil)

	h.final(session, true)
	h.expectNone(t)
}

func TestFinalTranscriptionErrorFallsBackToLastText(t *testing.T) {
	wav := makeSessionWav(t, 4000)
	session := mic.RecordingSession{ID: 6, FileName: wav}
	h := newHarness(t, []string{"up down", "\x00fail"})

	d1 := h.tick(session, 1000)
	h.final(session, false)

	got := h.collect(t, 2)
	assertDone(t, d1)
	if !got[1].Final || got[1].Text != "up down" {
		t.Fatalf("expected final to fall back to last tick text, got %+v", got[1])
	}
}
