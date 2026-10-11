// Package server exposes screen-perception providers over a small local
// HTTP+JSON API: prewarm at utterance start, query targets at command time.
package server

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/api"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

const (
	defaultProvider = "ocr"
	// snapshotTimeout bounds one capture+OCR round-trip; refreshes run on
	// background contexts so one client's disconnect can't kill a capture
	// other waiters have joined.
	snapshotTimeout = 10 * time.Second
)

type Server struct {
	port          int
	defaultTTL    time.Duration
	ttls          map[string]time.Duration
	helperRunning func() bool
	providers     map[string]scan.Provider
	order         []string
	store         *store
	// AXTrusted, when set, surfaces the helper's accessibility-permission
	// state on /healthz — an untrusted helper serves empty ax trees, and
	// this is the diagnostic that says why.
	AXTrusted func() *bool
}

func NewServer(port int, defaultTTL time.Duration, helperRunning func() bool, providers ...scan.Provider) *Server {
	s := &Server{
		port:          port,
		defaultTTL:    defaultTTL,
		ttls:          map[string]time.Duration{},
		helperRunning: helperRunning,
		providers:     map[string]scan.Provider{},
		store:         newStore(),
	}
	for _, p := range providers {
		s.providers[p.Name()] = p
		s.order = append(s.order, p.Name())
	}
	return s
}

// SetTTL overrides the default snapshot max age for one provider — ax trees
// go stale faster than screen captures are worth re-taking.
func (s *Server) SetTTL(provider string, ttl time.Duration) {
	s.ttls[provider] = ttl
}

func (s *Server) ttlFor(provider string) time.Duration {
	if ttl, ok := s.ttls[provider]; ok {
		return ttl
	}
	return s.defaultTTL
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealthz)
	mux.HandleFunc("POST /v1/prewarm", s.handlePrewarm)
	mux.HandleFunc("POST /v1/targets", s.handleTargets)
	return mux
}

// Run serves until ctx is cancelled, binding loopback only.
func (s *Server) Run(ctx context.Context) error {
	srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", s.port), Handler: s.Handler()}
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	resp := api.HealthResponse{
		Status:        "ok",
		Providers:     s.order,
		HelperRunning: s.helperRunning(),
	}
	if s.AXTrusted != nil {
		resp.AXTrusted = s.AXTrusted()
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handlePrewarm(w http.ResponseWriter, r *http.Request) {
	var req api.PrewarmRequest
	if !decodeBody(w, r, &req) {
		return
	}
	p, ok := s.provider(w, req.Provider)
	if !ok {
		return
	}
	status := "warming"
	if !s.store.prewarm(p.Name(), snapshotFn(p, req.Display)) {
		status = "already_warming"
	}
	writeJSON(w, http.StatusAccepted, api.PrewarmResponse{Status: status})
}

func (s *Server) handleTargets(w http.ResponseWriter, r *http.Request) {
	var req api.TargetsRequest
	if !decodeBody(w, r, &req) {
		return
	}
	p, ok := s.provider(w, req.Provider)
	if !ok {
		return
	}

	maxAge := s.ttlFor(p.Name())
	if req.MaxAgeMs != nil {
		maxAge = time.Duration(*req.MaxAgeMs) * time.Millisecond
	}

	snap, fromCache, err := s.store.get(p.Name(), maxAge, snapshotFn(p, req.Display))
	if err != nil {
		writeJSON(w, http.StatusBadGateway, api.ErrorResponse{Error: err.Error()})
		return
	}

	var region *image.Rectangle
	if req.Region != nil {
		r := req.Region.ToImageRect()
		region = &r
	}
	targets, err := p.Query(r.Context(), snap, req.Query, scan.QueryOptions{Region: region, MaxResults: req.MaxResults})
	if err != nil {
		writeJSON(w, http.StatusBadGateway, api.ErrorResponse{Error: err.Error()})
		return
	}

	resp := api.TargetsResponse{
		Provider:   p.Name(),
		SnapshotID: snap.ID,
		CapturedAt: snap.CapturedAt,
		AgeMs:      s.store.now().Sub(snap.CapturedAt).Milliseconds(),
		FromCache:  fromCache,
		Screen: api.ScreenInfo{
			Width:   snap.Screen.Width,
			Height:  snap.Screen.Height,
			Scale:   snap.Screen.Scale,
			Display: snap.Screen.Display,
		},
		Targets: make([]api.Target, len(targets)),
	}
	for i, t := range targets {
		resp.Targets[i] = api.Target{
			ID:         i,
			Label:      t.Label,
			Rect:       api.FromImageRect(t.Rect),
			Confidence: t.Confidence,
			Source:     t.Source,
			Match:      t.Match,
			LineText:   t.LineText,
		}
	}
	if req.IncludeScreenshot && len(snap.ScreenshotPNG) > 0 {
		resp.ScreenshotPNGBase64 = base64.StdEncoding.EncodeToString(snap.ScreenshotPNG)
	}
	log.Infof("targets %q: %d hits (cache=%v age=%dms)", req.Query, len(targets), fromCache, resp.AgeMs)
	writeJSON(w, http.StatusOK, resp)
}

// snapshotFn binds a provider capture to a background context so it outlives
// the request that happened to trigger it.
func snapshotFn(p scan.Provider, display int) func() (*scan.Snapshot, error) {
	return func() (*scan.Snapshot, error) {
		ctx, cancel := context.WithTimeout(context.Background(), snapshotTimeout)
		defer cancel()
		return p.Snapshot(ctx, display)
	}
}

func (s *Server) provider(w http.ResponseWriter, name string) (scan.Provider, bool) {
	if name == "" {
		name = defaultProvider
	}
	p, ok := s.providers[name]
	if !ok {
		writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: fmt.Sprintf("unknown provider %q", name)})
		return nil, false
	}
	return p, true
}

// decodeBody parses a JSON body, tolerating an empty body as the zero value.
func decodeBody(w http.ResponseWriter, r *http.Request, v any) bool {
	err := json.NewDecoder(r.Body).Decode(v)
	if err == nil || errors.Is(err, io.EOF) {
		return true
	}
	writeJSON(w, http.StatusBadRequest, api.ErrorResponse{Error: fmt.Sprintf("bad request: %v", err)})
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
