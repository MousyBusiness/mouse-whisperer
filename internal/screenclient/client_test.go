package screenclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/api"
)

func TestFindTargets(t *testing.T) {
	var gotPath atomic.Value
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath.Store(r.URL.Path)
		var req api.TargetsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Query != "handler" || !req.IncludeScreenshot {
			t.Errorf("unexpected request: %+v", req)
		}
		json.NewEncoder(w).Encode(api.TargetsResponse{
			Provider:  "ocr",
			FromCache: true,
			Targets:   []api.Target{{ID: 0, Label: "handler", Rect: api.Rect{X: 10, Y: 20, W: 60, H: 14}}},
		})
	}))
	defer ts.Close()

	c := New(ts.URL)
	resp, err := c.FindTargets(context.Background(), api.TargetsRequest{Query: "handler", IncludeScreenshot: true})
	if err != nil {
		t.Fatalf("FindTargets: %v", err)
	}
	if gotPath.Load() != "/v1/targets" {
		t.Fatalf("path = %v", gotPath.Load())
	}
	if len(resp.Targets) != 1 || resp.Targets[0].Label != "handler" || !resp.FromCache {
		t.Fatalf("response = %+v", resp)
	}
}

func TestFindTargetsServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(api.ErrorResponse{Error: "helper down"})
	}))
	defer ts.Close()

	if _, err := New(ts.URL).FindTargets(context.Background(), api.TargetsRequest{Query: "x"}); err == nil {
		t.Fatal("non-200 must surface as error")
	}
}

func TestFindTargetsServerDown(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	ts.Close() // connection refused from here on

	start := time.Now()
	_, err := New(ts.URL).FindTargets(context.Background(), api.TargetsRequest{Query: "x"})
	if err == nil {
		t.Fatal("want connection error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("refused connection took %s — must fail fast for the handler fallback", elapsed)
	}
}

func TestPrewarmSwallowsErrors(t *testing.T) {
	var calls atomic.Int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/v1/prewarm" {
			t.Errorf("path = %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(api.PrewarmResponse{Status: "warming"})
	}))
	defer ts.Close()

	c := New(ts.URL)
	c.Prewarm()
	if calls.Load() != 1 {
		t.Fatalf("prewarm calls = %d", calls.Load())
	}

	ts.Close()
	c.Prewarm() // must not panic or block on a dead server
}
