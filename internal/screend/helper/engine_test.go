package helper

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeSpawner stands in for launching the native helper. Each spawn wires a
// pipe pair; the test reads requests from reqs and answers via respond.
type fakeSpawner struct {
	mu       sync.Mutex
	count    int
	spawnErr error
	reqs     chan Request
	stdoutW  *io.PipeWriter
	killed   bool
}

func newFakeSpawner() *fakeSpawner {
	return &fakeSpawner{reqs: make(chan Request, 4)}
}

func (s *fakeSpawner) spawn() (*helperProc, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count++
	if s.spawnErr != nil {
		return nil, s.spawnErr
	}
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	s.stdoutW = stdoutW
	s.killed = false
	go func() {
		rd := bufio.NewReader(stdinR)
		for {
			line, err := rd.ReadBytes('\n')
			if err != nil {
				return
			}
			var req Request
			if json.Unmarshal(line, &req) == nil {
				s.reqs <- req
			}
		}
	}()
	return &helperProc{
		stdin:  stdinW,
		stdout: stdoutR,
		kill: func() {
			s.mu.Lock()
			s.killed = true
			s.mu.Unlock()
			stdinR.CloseWithError(errors.New("killed"))
			stdoutW.CloseWithError(errors.New("killed"))
		},
		wait: func() error { return nil },
	}, nil
}

func (s *fakeSpawner) respond(resp Response) {
	b, _ := json.Marshal(resp)
	s.mu.Lock()
	w := s.stdoutW
	s.mu.Unlock()
	w.Write(append(b, '\n'))
}

func (s *fakeSpawner) spawnCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count
}

func (s *fakeSpawner) wasKilled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.killed
}

func (s *fakeSpawner) crash() {
	s.mu.Lock()
	w := s.stdoutW
	s.mu.Unlock()
	w.CloseWithError(errors.New("helper crashed"))
}

func answerNext(s *fakeSpawner, t *testing.T) {
	t.Helper()
	select {
	case req := <-s.reqs:
		s.respond(Response{ID: req.ID, OK: true})
	case <-time.After(2 * time.Second):
		t.Error("helper never received a request")
	}
}

func TestEngineHappyPath(t *testing.T) {
	s := newFakeSpawner()
	eng := newEngineWithSpawn(s.spawn)
	go answerNext(s, t)
	resp, err := eng.Do(context.Background(), Request{Op: OpCaptureOCR})
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if !resp.OK {
		t.Fatalf("response not OK: %+v", resp)
	}
	if s.spawnCount() != 1 {
		t.Fatalf("spawn count = %d, want 1", s.spawnCount())
	}
}

func TestEngineIDMismatch(t *testing.T) {
	s := newFakeSpawner()
	eng := newEngineWithSpawn(s.spawn)
	go func() {
		req := <-s.reqs
		s.respond(Response{ID: req.ID + 99, OK: true})
	}()
	if _, err := eng.Do(context.Background(), Request{Op: OpPing}); err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("want id mismatch error, got %v", err)
	}
	if !s.wasKilled() {
		t.Fatal("engine should kill helper after protocol violation")
	}
}

func TestEngineHelperError(t *testing.T) {
	s := newFakeSpawner()
	eng := newEngineWithSpawn(s.spawn)
	go func() {
		req := <-s.reqs
		s.respond(Response{ID: req.ID, OK: false, Error: "no permission"})
	}()
	_, err := eng.Do(context.Background(), Request{Op: OpCaptureOCR})
	if err == nil || !strings.Contains(err.Error(), "no permission") {
		t.Fatalf("want helper error, got %v", err)
	}
	// a clean protocol-level error is not a crash: same process keeps serving
	if s.wasKilled() {
		t.Fatal("helper should not be killed for an ok:false response")
	}
	go answerNext(s, t)
	if _, err := eng.Do(context.Background(), Request{Op: OpCaptureOCR}); err != nil {
		t.Fatalf("second request after helper error: %v", err)
	}
	if s.spawnCount() != 1 {
		t.Fatalf("spawn count = %d, want 1", s.spawnCount())
	}
}

func TestEngineRespawnsAfterCrash(t *testing.T) {
	s := newFakeSpawner()
	eng := newEngineWithSpawn(s.spawn)
	go func() {
		<-s.reqs
		s.crash()
	}()
	if _, err := eng.Do(context.Background(), Request{Op: OpCaptureOCR}); err == nil {
		t.Fatal("want error when helper dies mid-request")
	}
	go answerNext(s, t)
	if _, err := eng.Do(context.Background(), Request{Op: OpCaptureOCR}); err != nil {
		t.Fatalf("do after respawn: %v", err)
	}
	if s.spawnCount() != 2 {
		t.Fatalf("spawn count = %d, want 2", s.spawnCount())
	}
}

func TestEngineContextTimeout(t *testing.T) {
	s := newFakeSpawner()
	eng := newEngineWithSpawn(s.spawn)
	go func() { <-s.reqs }() // swallow the request, never respond
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := eng.Do(ctx, Request{Op: OpCaptureOCR}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("want DeadlineExceeded, got %v", err)
	}
	if !s.wasKilled() {
		t.Fatal("engine should kill a wedged helper")
	}
	go answerNext(s, t)
	if _, err := eng.Do(context.Background(), Request{Op: OpCaptureOCR}); err != nil {
		t.Fatalf("do after timeout respawn: %v", err)
	}
	if s.spawnCount() != 2 {
		t.Fatalf("spawn count = %d, want 2", s.spawnCount())
	}
}

func TestEngineSerializesRequests(t *testing.T) {
	s := newFakeSpawner()
	eng := newEngineWithSpawn(s.spawn)

	first := <-launch(eng, s) // in flight: helper has the request, no response yet

	done2 := make(chan error, 1)
	go func() {
		_, err := eng.Do(context.Background(), Request{Op: OpCaptureOCR})
		done2 <- err
	}()

	select {
	case <-s.reqs:
		t.Fatal("second request reached helper while first still in flight")
	case <-time.After(100 * time.Millisecond):
	}

	s.respond(Response{ID: first.ID, OK: true})
	answerNext(s, t)
	if err := <-done2; err != nil {
		t.Fatalf("second request: %v", err)
	}
}

// launch starts a Do() in the background and yields the request the helper saw.
func launch(eng *Engine, s *fakeSpawner) <-chan Request {
	ch := make(chan Request, 1)
	go eng.Do(context.Background(), Request{Op: OpCaptureOCR})
	go func() { ch <- <-s.reqs }()
	return ch
}

func TestEngineSpawnBackoff(t *testing.T) {
	s := newFakeSpawner()
	s.spawnErr = errors.New("helper binary missing")
	eng := newEngineWithSpawn(s.spawn)
	now := time.Unix(1000, 0)
	eng.now = func() time.Time { return now }

	if _, err := eng.Do(context.Background(), Request{Op: OpPing}); err == nil {
		t.Fatal("want spawn error")
	}
	if _, err := eng.Do(context.Background(), Request{Op: OpPing}); err == nil {
		t.Fatal("want backoff error")
	}
	if s.spawnCount() != 1 {
		t.Fatalf("spawn retried during backoff window: count = %d", s.spawnCount())
	}

	now = now.Add(600 * time.Millisecond)
	s.mu.Lock()
	s.spawnErr = nil
	s.mu.Unlock()
	go answerNext(s, t)
	if _, err := eng.Do(context.Background(), Request{Op: OpPing}); err != nil {
		t.Fatalf("do after backoff elapsed: %v", err)
	}
	if s.spawnCount() != 2 {
		t.Fatalf("spawn count = %d, want 2", s.spawnCount())
	}
}

func TestEngineCachesAXTrusted(t *testing.T) {
	s := newFakeSpawner()
	eng := newEngineWithSpawn(s.spawn)
	if eng.AXTrusted() != nil {
		t.Fatal("ax trust must be unknown before any ping answers")
	}
	trusted := true
	go func() {
		req := <-s.reqs
		s.respond(Response{ID: req.ID, OK: true, AXTrusted: &trusted})
	}()
	if err := eng.Ping(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if got := eng.AXTrusted(); got == nil || !*got {
		t.Fatalf("AXTrusted = %v, want true", got)
	}
}
