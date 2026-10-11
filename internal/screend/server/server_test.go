package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/api"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

// fakeProvider serves a fixed set of items through the real Rank, mirroring
// what the OCR provider does without any helper process.
type fakeProvider struct {
	name      string
	items     []scan.Item
	png       []byte
	err       error
	snapCalls atomic.Int32
	display   atomic.Int32
}

func (p *fakeProvider) Name() string { return p.name }

func (p *fakeProvider) Snapshot(_ context.Context, display int) (*scan.Snapshot, error) {
	p.snapCalls.Add(1)
	p.display.Store(int32(display))
	if p.err != nil {
		return nil, p.err
	}
	return &scan.Snapshot{
		ID:            fmt.Sprintf("snap-%d", p.snapCalls.Load()),
		CapturedAt:    time.Now(),
		Display:       display,
		Screen:        scan.ScreenInfo{Width: 1728, Height: 1117, Scale: 2.0, Display: display},
		Items:         p.items,
		ScreenshotPNG: p.png,
	}, nil
}

func (p *fakeProvider) Query(_ context.Context, snap *scan.Snapshot, query string, opts scan.QueryOptions) ([]scan.Target, error) {
	targets := scan.Rank(snap.Items, query, opts)
	for i := range targets {
		targets[i].Source = p.name
	}
	return targets, nil
}

func testItems() []scan.Item {
	return []scan.Item{
		{Text: "handler", Line: 0, LineText: "func handler()", Confidence: 0.97, Rect: image.Rect(10, 10, 70, 24)},
		{Text: "handler", Line: 5, LineText: "handler again", Confidence: 0.95, Rect: image.Rect(500, 500, 560, 514)},
		{Text: "unrelated", Line: 9, LineText: "unrelated", Confidence: 0.9, Rect: image.Rect(10, 900, 90, 914)},
	}
}

func newTestServer(t *testing.T, p scan.Provider) (*httptest.Server, *Server) {
	t.Helper()
	srv := NewServer(0, 5*time.Second, func() bool { return true }, p)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts, srv
}

func postJSON[T any](t *testing.T, url string, body any) (int, T) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	var out T
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, out
}

func TestHealthz(t *testing.T) {
	ts, _ := newTestServer(t, &fakeProvider{name: "ocr"})
	resp, err := http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var health api.HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if health.Status != "ok" || len(health.Providers) != 1 || health.Providers[0] != "ocr" || !health.HelperRunning {
		t.Fatalf("unexpected health: %+v", health)
	}
}

func TestTargetsColdThenWarm(t *testing.T) {
	p := &fakeProvider{name: "ocr", items: testItems()}
	ts, _ := newTestServer(t, p)

	status, cold := postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "handler"})
	if status != http.StatusOK {
		t.Fatalf("status = %d", status)
	}
	if cold.FromCache {
		t.Fatal("first query cannot be from cache")
	}
	if len(cold.Targets) != 2 {
		t.Fatalf("targets = %d, want 2: %+v", len(cold.Targets), cold.Targets)
	}
	for i, target := range cold.Targets {
		if target.ID != i {
			t.Fatalf("target %d has id %d — ids must be indexes for hex labels", i, target.ID)
		}
	}
	if cold.Targets[0].Source != "ocr" || cold.Targets[0].Match != "exact" {
		t.Fatalf("unexpected target: %+v", cold.Targets[0])
	}
	if cold.Targets[0].Rect != (api.Rect{X: 10, Y: 10, W: 60, H: 14}) {
		t.Fatalf("rect = %+v", cold.Targets[0].Rect)
	}
	if cold.Screen.Width != 1728 || cold.Screen.Scale != 2.0 {
		t.Fatalf("screen = %+v", cold.Screen)
	}

	_, warm := postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "handler"})
	if !warm.FromCache || warm.SnapshotID != cold.SnapshotID {
		t.Fatalf("second query should hit cache: fromCache=%v", warm.FromCache)
	}
	if p.snapCalls.Load() != 1 {
		t.Fatalf("snapshot calls = %d, want 1", p.snapCalls.Load())
	}
}

func TestTargetsForceFresh(t *testing.T) {
	p := &fakeProvider{name: "ocr", items: testItems()}
	ts, _ := newTestServer(t, p)

	postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "handler"})
	zero := 0
	_, resp := postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "handler", MaxAgeMs: &zero})
	if resp.FromCache {
		t.Fatal("max_age_ms=0 must force a fresh capture")
	}
	if p.snapCalls.Load() != 2 {
		t.Fatalf("snapshot calls = %d, want 2", p.snapCalls.Load())
	}
}

func TestTargetsRegionAndEmptyQuery(t *testing.T) {
	p := &fakeProvider{name: "ocr", items: testItems()}
	ts, _ := newTestServer(t, p)

	_, resp := postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{
		Query:  "handler",
		Region: &api.Rect{X: 0, Y: 0, W: 100, H: 100},
	})
	if len(resp.Targets) != 1 || resp.Targets[0].Rect.X != 10 {
		t.Fatalf("region filter failed: %+v", resp.Targets)
	}

	_, all := postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: ""})
	if len(all.Targets) != 3 {
		t.Fatalf("empty query should list everything, got %d", len(all.Targets))
	}
}

func TestTargetsScreenshot(t *testing.T) {
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 4))); err != nil {
		t.Fatalf("encode fixture png: %v", err)
	}
	p := &fakeProvider{name: "ocr", items: testItems(), png: buf.Bytes()}
	ts, _ := newTestServer(t, p)

	_, with := postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "handler", IncludeScreenshot: true})
	img, err := with.DecodeScreenshot()
	if err != nil {
		t.Fatalf("DecodeScreenshot: %v", err)
	}
	if img.Bounds().Dx() != 4 {
		t.Fatalf("screenshot bounds = %v", img.Bounds())
	}

	_, without := postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "handler"})
	if without.ScreenshotPNGBase64 != "" {
		t.Fatal("screenshot must be omitted unless requested")
	}
}

func TestTargetsErrors(t *testing.T) {
	ts, _ := newTestServer(t, &fakeProvider{name: "ocr", err: errors.New("no permission")})

	status, errResp := postJSON[api.ErrorResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "x", Provider: "nope"})
	if status != http.StatusBadRequest {
		t.Fatalf("unknown provider status = %d (%+v)", status, errResp)
	}

	status, errResp = postJSON[api.ErrorResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "x"})
	if status != http.StatusBadGateway || errResp.Error == "" {
		t.Fatalf("provider failure: status = %d, err = %q", status, errResp.Error)
	}

	resp, err := http.Post(ts.URL+"/v1/targets", "application/json", bytes.NewReader([]byte("{bad json")))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad json status = %d", resp.StatusCode)
	}
}

func TestPrewarm(t *testing.T) {
	p := &fakeProvider{name: "ocr", items: testItems()}
	ts, _ := newTestServer(t, p)

	status, resp := postJSON[api.PrewarmResponse](t, ts.URL+"/v1/prewarm", api.PrewarmRequest{Provider: "ocr", Display: 0})
	if status != http.StatusAccepted || resp.Status != "warming" {
		t.Fatalf("prewarm: status=%d resp=%+v", status, resp)
	}

	// wait for the background capture, then confirm the query is a cache hit
	deadline := time.After(2 * time.Second)
	for p.snapCalls.Load() == 0 {
		select {
		case <-deadline:
			t.Fatal("prewarm never captured")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	_, targets := postJSON[api.TargetsResponse](t, ts.URL+"/v1/targets", api.TargetsRequest{Query: "handler"})
	if !targets.FromCache {
		t.Fatal("query after prewarm should be a cache hit")
	}
	if p.snapCalls.Load() != 1 {
		t.Fatalf("snapshot calls = %d, want 1", p.snapCalls.Load())
	}

	status, errResp := postJSON[api.ErrorResponse](t, ts.URL+"/v1/prewarm", api.PrewarmRequest{Provider: "nope"})
	if status != http.StatusBadRequest {
		t.Fatalf("unknown provider prewarm status = %d (%+v)", status, errResp)
	}
}
