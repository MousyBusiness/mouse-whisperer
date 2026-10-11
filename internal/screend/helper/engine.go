package helper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

const (
	initialBackoff = 500 * time.Millisecond
	maxBackoff     = 5 * time.Second
)

type helperProc struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
	kill   func()
	wait   func() error
}

// Engine supervises the persistent native helper: lazy spawn, one request in
// flight at a time, kill + respawn on crash, wedge, or protocol violation,
// with backoff between failed spawns.
type Engine struct {
	mu           sync.Mutex
	spawn        func() (*helperProc, error)
	now          func() time.Time
	proc         *helperProc
	rd           *bufio.Reader
	nextID       int64
	backoff      time.Duration
	nextSpawnAt  time.Time
	lastSpawnErr error
	// running mirrors proc != nil without taking mu, so health checks never
	// queue behind an in-flight capture
	running atomic.Bool
	// axTrusted caches the accessibility-permission fact from the last ping
	// response, lock-free for the same reason as running: an untrusted helper
	// fails ax queries with empty trees, and healthz must be able to say why
	// without queueing behind a capture.
	axTrusted atomic.Pointer[bool]
}

func NewEngine(helperPath string) *Engine {
	return newEngineWithSpawn(func() (*helperProc, error) {
		cmd := exec.Command(helperPath)
		cmd.Stderr = os.Stderr
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return nil, err
		}
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		if err := cmd.Start(); err != nil {
			return nil, err
		}
		return &helperProc{
			stdin:  stdin,
			stdout: stdout,
			kill: func() {
				stdin.Close()
				cmd.Process.Kill()
			},
			wait: cmd.Wait,
		}, nil
	})
}

func newEngineWithSpawn(spawn func() (*helperProc, error)) *Engine {
	return &Engine{spawn: spawn, now: time.Now, backoff: initialBackoff}
}

// Ping round-trips the helper, spawning it if needed. Called at server start
// so helper problems surface before the first real scan.
func (e *Engine) Ping(ctx context.Context) error {
	_, err := e.Do(ctx, Request{Op: OpPing})
	return err
}

// Running reports whether a helper process is currently alive. Lock-free so
// it stays responsive while a capture holds the engine.
func (e *Engine) Running() bool {
	return e.running.Load()
}

// AXTrusted reports the accessibility-permission state from the most recent
// ping, or nil when no ping has answered yet. Lock-free.
func (e *Engine) AXTrusted() *bool {
	return e.axTrusted.Load()
}

func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.teardown()
}

func (e *Engine) Do(ctx context.Context, req Request) (*Response, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.ensureProc(); err != nil {
		return nil, err
	}

	e.nextID++
	req.ID = e.nextID

	if err := writeRequest(e.proc.stdin, req); err != nil {
		e.teardown()
		return nil, fmt.Errorf("write to helper: %w", err)
	}

	type result struct {
		resp *Response
		err  error
	}
	ch := make(chan result, 1)
	go func(rd *bufio.Reader) {
		resp, err := readResponse(rd)
		ch <- result{resp, err}
	}(e.rd)

	select {
	case <-ctx.Done():
		e.teardown() // unblocks the reader goroutine with EOF
		return nil, ctx.Err()
	case r := <-ch:
		switch {
		case r.err != nil:
			e.teardown()
			return nil, r.err
		case r.resp.ID != req.ID:
			e.teardown()
			return nil, fmt.Errorf("helper response id mismatch: got %d, want %d", r.resp.ID, req.ID)
		case !r.resp.OK:
			// clean protocol-level failure; the process is still healthy
			return nil, fmt.Errorf("helper: %s", r.resp.Error)
		}
		if r.resp.AXTrusted != nil {
			e.axTrusted.Store(r.resp.AXTrusted)
		}
		return r.resp, nil
	}
}

func (e *Engine) ensureProc() error {
	if e.proc != nil {
		return nil
	}
	if e.now().Before(e.nextSpawnAt) {
		return fmt.Errorf("helper in restart backoff: %w", e.lastSpawnErr)
	}
	p, err := e.spawn()
	if err != nil {
		e.lastSpawnErr = err
		e.nextSpawnAt = e.now().Add(e.backoff)
		e.backoff = min(e.backoff*2, maxBackoff)
		return fmt.Errorf("spawn helper: %w", err)
	}
	e.backoff = initialBackoff
	e.proc = p
	e.rd = bufio.NewReader(p.stdout)
	e.running.Store(true)
	return nil
}

func (e *Engine) teardown() {
	if e.proc == nil {
		return
	}
	e.proc.kill()
	e.proc.wait()
	e.proc = nil
	e.rd = nil
	e.running.Store(false)
}
