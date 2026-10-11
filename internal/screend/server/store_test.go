package server

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

func snapAt(at time.Time) *scan.Snapshot {
	return &scan.Snapshot{ID: at.String(), CapturedAt: at}
}

func TestStoreCachesWithinMaxAge(t *testing.T) {
	now := time.Unix(1000, 0)
	s := newStore()
	s.now = func() time.Time { return now }

	var calls atomic.Int32
	refresh := func() (*scan.Snapshot, error) {
		calls.Add(1)
		return snapAt(now), nil
	}

	snap1, fromCache, err := s.get("ocr", 5*time.Second, refresh)
	if err != nil || fromCache {
		t.Fatalf("first get: snap=%v fromCache=%v err=%v", snap1, fromCache, err)
	}
	snap2, fromCache, err := s.get("ocr", 5*time.Second, refresh)
	if err != nil || !fromCache || snap2 != snap1 {
		t.Fatalf("second get should serve cache: fromCache=%v err=%v", fromCache, err)
	}
	if calls.Load() != 1 {
		t.Fatalf("refresh calls = %d, want 1", calls.Load())
	}

	now = now.Add(6 * time.Second) // past max age
	_, fromCache, err = s.get("ocr", 5*time.Second, refresh)
	if err != nil || fromCache {
		t.Fatalf("stale get should refresh: fromCache=%v err=%v", fromCache, err)
	}
	if calls.Load() != 2 {
		t.Fatalf("refresh calls = %d, want 2", calls.Load())
	}
}

func TestStoreSingleFlight(t *testing.T) {
	s := newStore()
	var calls atomic.Int32
	release := make(chan struct{})
	refresh := func() (*scan.Snapshot, error) {
		calls.Add(1)
		<-release
		return snapAt(time.Now()), nil
	}

	var wg sync.WaitGroup
	snaps := make([]*scan.Snapshot, 10)
	for i := range 10 {
		wg.Go(func() {
			snap, _, err := s.get("ocr", 0, refresh)
			if err != nil {
				t.Errorf("get: %v", err)
			}
			snaps[i] = snap
		})
	}
	time.Sleep(50 * time.Millisecond) // let the getters pile onto the flight
	close(release)
	wg.Wait()

	if calls.Load() != 1 {
		t.Fatalf("refresh calls = %d, want 1", calls.Load())
	}
	for i, snap := range snaps {
		if snap != snaps[0] {
			t.Fatalf("getter %d received a different snapshot", i)
		}
	}
}

func TestStorePrewarmCoalesces(t *testing.T) {
	s := newStore()
	release := make(chan struct{})
	var calls atomic.Int32
	refresh := func() (*scan.Snapshot, error) {
		calls.Add(1)
		<-release
		return snapAt(time.Now()), nil
	}

	if !s.prewarm("ocr", refresh) {
		t.Fatal("first prewarm should start a flight")
	}
	if s.prewarm("ocr", refresh) {
		t.Fatal("second prewarm should coalesce onto the in-flight capture")
	}
	close(release)

	// the prewarmed snapshot must serve a subsequent get without re-capturing
	var getCalls atomic.Int32
	getRefresh := func() (*scan.Snapshot, error) {
		getCalls.Add(1)
		return snapAt(time.Now()), nil
	}
	deadline := time.After(2 * time.Second)
	for {
		_, fromCache, err := s.get("ocr", time.Minute, getRefresh)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		if fromCache {
			break
		}
		select {
		case <-deadline:
			t.Fatal("prewarmed snapshot never became visible to get")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	if calls.Load() != 1 {
		t.Fatalf("prewarm refresh calls = %d, want 1", calls.Load())
	}
}

func TestStorePrewarmSkipsFreshCache(t *testing.T) {
	now := time.Unix(1000, 0)
	s := newStore()
	s.now = func() time.Time { return now }

	var calls atomic.Int32
	refresh := func() (*scan.Snapshot, error) {
		calls.Add(1)
		return snapAt(now), nil
	}

	if !s.prewarm("ocr", refresh) {
		t.Fatal("first prewarm should capture")
	}
	// wait for the async flight to land in the cache
	deadline := time.After(2 * time.Second)
	for {
		if _, fromCache, _ := s.get("ocr", time.Minute, refresh); fromCache {
			break
		}
		select {
		case <-deadline:
			t.Fatal("prewarm capture never landed")
		default:
			time.Sleep(5 * time.Millisecond)
		}
	}

	if s.prewarm("ocr", refresh) {
		t.Fatal("prewarm must skip while the cache is fresh")
	}
	if calls.Load() != 1 {
		t.Fatalf("refresh calls = %d, want 1", calls.Load())
	}

	now = now.Add(2 * time.Second) // past prewarmMinAge
	if !s.prewarm("ocr", refresh) {
		t.Fatal("prewarm should capture once the cache has aged")
	}
}

func TestStoreErrorDoesNotPoisonCache(t *testing.T) {
	now := time.Unix(1000, 0)
	s := newStore()
	s.now = func() time.Time { return now }

	good := snapAt(now)
	_, _, err := s.get("ocr", time.Minute, func() (*scan.Snapshot, error) { return good, nil })
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	now = now.Add(2 * time.Minute)
	_, _, err = s.get("ocr", time.Minute, func() (*scan.Snapshot, error) { return nil, errors.New("helper down") })
	if err == nil {
		t.Fatal("refresh failure should surface")
	}

	// the failed refresh must not have clobbered the last good snapshot
	snap, fromCache, err := s.get("ocr", 3*time.Minute, func() (*scan.Snapshot, error) { return nil, errors.New("helper down") })
	if err != nil || !fromCache || snap != good {
		t.Fatalf("stale-but-good snapshot lost: snap=%v fromCache=%v err=%v", snap, fromCache, err)
	}
}

func TestStoreKeysByProvider(t *testing.T) {
	s := newStore()
	ocrSnap := snapAt(time.Now())
	llmSnap := snapAt(time.Now())

	got, _, _ := s.get("ocr", time.Minute, func() (*scan.Snapshot, error) { return ocrSnap, nil })
	if got != ocrSnap {
		t.Fatal("ocr snapshot mismatch")
	}
	var llmCalls atomic.Int32
	got, fromCache, _ := s.get("vision_llm", time.Minute, func() (*scan.Snapshot, error) {
		llmCalls.Add(1)
		return llmSnap, nil
	})
	if fromCache || got != llmSnap || llmCalls.Load() != 1 {
		t.Fatal("providers must not share cache entries")
	}
}
