package impl

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mousybusiness/mouse_whisperer/internal/screenclient"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/api"
	"github.com/mousybusiness/mouse_whisperer/internal/state"
)

func target(x, y, w, h int) api.Target {
	return api.Target{Rect: api.Rect{X: x, Y: y, W: w, H: h}}
}

func TestPickTarget(t *testing.T) {
	cases := []struct {
		name        string
		targets     []api.Target
		gazeOK      bool
		gx, gy      int
		wantIdx     int
		wantOverlay bool
	}{
		{
			name:    "no targets",
			wantIdx: -1,
		},
		{
			name:    "single target acts without gaze",
			targets: []api.Target{target(500, 500, 60, 14)},
			wantIdx: 0,
		},
		{
			name:    "single target acts even when gaze is far away",
			targets: []api.Target{target(1500, 900, 60, 14)},
			gazeOK:  true, gx: 10, gy: 10,
			wantIdx: 0,
		},
		{
			name: "gaze picks the nearest target within radius",
			targets: []api.Target{
				target(100, 100, 60, 14),
				target(400, 400, 60, 14),
				target(1200, 800, 60, 14),
			},
			gazeOK: true, gx: 420, gy: 390,
			wantIdx: 1,
		},
		{
			name: "gaze with every target beyond radius disambiguates",
			targets: []api.Target{
				target(1200, 100, 60, 14),
				target(1200, 800, 60, 14),
			},
			gazeOK: true, gx: 10, gy: 500,
			wantIdx: -1, wantOverlay: true,
		},
		{
			name: "multiple targets without gaze disambiguates",
			targets: []api.Target{
				target(100, 100, 60, 14),
				target(400, 400, 60, 14),
			},
			wantIdx: -1, wantOverlay: true,
		},
	}

	const radius = 600
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			idx, overlay := pickTarget(c.targets, c.gazeOK, c.gx, c.gy, radius)
			if idx != c.wantIdx || overlay != c.wantOverlay {
				t.Fatalf("pickTarget() = (%d, %v), want (%d, %v)", idx, overlay, c.wantIdx, c.wantOverlay)
			}
		})
	}
}

// scanState builds a State whose ScanClient answers /v1/targets from canned
// per-provider responses; a nil entry answers HTTP 502.
func scanState(t *testing.T, providers []string, responses map[string]*api.TargetsResponse) *state.State {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req api.TargetsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		resp, ok := responses[req.Provider]
		if !ok || resp == nil {
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(api.ErrorResponse{Error: "provider down"})
			return
		}
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return &state.State{ScanClient: screenclient.New(srv.URL), ScanProviders: providers}
}

func match(provider, tier string) *api.TargetsResponse {
	return &api.TargetsResponse{Provider: provider, Targets: []api.Target{{Label: "x", Match: tier}}}
}

func TestFindTargetsProviderFallback(t *testing.T) {
	axOCR := []string{api.ProviderAX, api.ProviderOCR}
	cases := []struct {
		name         string
		providers    []string
		responses    map[string]*api.TargetsResponse
		wantProvider string
		wantTargets  int
		wantErr      bool
	}{
		{
			name:      "ax strong match wins without asking ocr",
			providers: axOCR,
			responses: map[string]*api.TargetsResponse{
				api.ProviderAX:  match(api.ProviderAX, "exact"),
				api.ProviderOCR: match(api.ProviderOCR, "exact"),
			},
			wantProvider: api.ProviderAX, wantTargets: 1,
		},
		{
			name:      "empty ax yields to ocr",
			providers: axOCR,
			responses: map[string]*api.TargetsResponse{
				api.ProviderAX:  {Provider: api.ProviderAX},
				api.ProviderOCR: match(api.ProviderOCR, "substring"),
			},
			wantProvider: api.ProviderOCR, wantTargets: 1,
		},
		{
			name:      "fuzzy-only ax yields to a strong ocr match",
			providers: axOCR,
			responses: map[string]*api.TargetsResponse{
				api.ProviderAX:  match(api.ProviderAX, "fuzzy"),
				api.ProviderOCR: match(api.ProviderOCR, "exact"),
			},
			wantProvider: api.ProviderOCR, wantTargets: 1,
		},
		{
			name:      "fuzzy ax still beats empty ocr",
			providers: axOCR,
			responses: map[string]*api.TargetsResponse{
				api.ProviderAX:  match(api.ProviderAX, "fuzzy"),
				api.ProviderOCR: {Provider: api.ProviderOCR},
			},
			wantProvider: api.ProviderAX, wantTargets: 1,
		},
		{
			name:      "ax provider error falls through to ocr",
			providers: axOCR,
			responses: map[string]*api.TargetsResponse{
				api.ProviderOCR: match(api.ProviderOCR, "exact"),
			},
			wantProvider: api.ProviderOCR, wantTargets: 1,
		},
		{
			name:      "all empty reports a clean no-match, not an error",
			providers: axOCR,
			responses: map[string]*api.TargetsResponse{
				api.ProviderAX:  {Provider: api.ProviderAX},
				api.ProviderOCR: {Provider: api.ProviderOCR},
			},
			wantProvider: api.ProviderOCR, wantTargets: 0,
		},
		{
			name:      "all providers down errors so callers use the legacy path",
			providers: axOCR,
			responses: map[string]*api.TargetsResponse{},
			wantErr:   true,
		},
		{
			name:      "unset provider order defaults to ocr alone",
			providers: nil,
			responses: map[string]*api.TargetsResponse{
				api.ProviderOCR: match(api.ProviderOCR, "fuzzy"),
			},
			wantProvider: api.ProviderOCR, wantTargets: 1,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := scanState(t, c.providers, c.responses)
			resp, err := findTargets(s, "save")
			if c.wantErr {
				if err == nil {
					t.Fatalf("want error, got %+v", resp)
				}
				return
			}
			if err != nil {
				t.Fatalf("findTargets: %v", err)
			}
			if resp.Provider != c.wantProvider || len(resp.Targets) != c.wantTargets {
				t.Fatalf("got provider=%s targets=%d, want %s/%d", resp.Provider, len(resp.Targets), c.wantProvider, c.wantTargets)
			}
		})
	}
}
