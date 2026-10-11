package server

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

// prewarmMinAge is how young the cache may be for a prewarm to be skipped:
// back-to-back utterances shouldn't burn a capture each when the last one is
// still fresher than any staleness bound callers use.
const prewarmMinAge = 1500 * time.Millisecond

// store caches the latest snapshot per provider. A prewarm at utterance
// start fills it so the query at command time is a cache hit; concurrent
// refreshes for one provider coalesce into a single capture.
type store struct {
	mu      sync.Mutex
	now     func() time.Time
	entries map[string]*entry
}

type entry struct {
	snap   *scan.Snapshot
	flight *flight
}

type flight struct {
	done chan struct{}
	snap *scan.Snapshot
	err  error
}

func newStore() *store {
	return &store{now: time.Now, entries: map[string]*entry{}}
}

// get returns the cached snapshot when it is younger than maxAge, otherwise
// refreshes (joining an in-flight refresh if one is already running). A
// failed refresh is returned as an error but never evicts the last good
// snapshot.
func (s *store) get(provider string, maxAge time.Duration, refresh func() (*scan.Snapshot, error)) (*scan.Snapshot, bool, error) {
	s.mu.Lock()
	e := s.entry(provider)
	if e.snap != nil && s.now().Sub(e.snap.CapturedAt) <= maxAge {
		snap := e.snap
		s.mu.Unlock()
		return snap, true, nil
	}
	f := e.flight
	if f == nil {
		f = s.startFlightLocked(e, refresh)
	}
	s.mu.Unlock()

	<-f.done
	return f.snap, false, f.err
}

// prewarm kicks a background refresh and reports whether it started one;
// false means a capture was already in flight (coalesced) or the cache is
// fresh enough that another capture would buy nothing.
func (s *store) prewarm(provider string, refresh func() (*scan.Snapshot, error)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e := s.entry(provider)
	if e.flight != nil {
		return false
	}
	if e.snap != nil && s.now().Sub(e.snap.CapturedAt) < prewarmMinAge {
		return false
	}
	s.startFlightLocked(e, refresh)
	return true
}

func (s *store) entry(provider string) *entry {
	e, ok := s.entries[provider]
	if !ok {
		e = &entry{}
		s.entries[provider] = e
	}
	return e
}

func (s *store) startFlightLocked(e *entry, refresh func() (*scan.Snapshot, error)) *flight {
	f := &flight{done: make(chan struct{})}
	e.flight = f
	go func() {
		start := time.Now()
		f.snap, f.err = refresh()
		if f.err != nil {
			log.Warnf("capture failed after %s: %v", time.Since(start).Round(time.Millisecond), f.err)
		} else {
			log.Infof("captured %d items in %s", len(f.snap.Items), time.Since(start).Round(time.Millisecond))
		}
		s.mu.Lock()
		if f.err == nil {
			e.snap = f.snap
		}
		e.flight = nil
		s.mu.Unlock()
		close(f.done)
	}()
	return f
}
