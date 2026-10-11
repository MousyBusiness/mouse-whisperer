package vision

import (
	"context"
	"encoding/base64"
	"image"
	"testing"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

// stubDoer answers every request with a canned response, recording the last
// request for assertions.
type stubDoer struct {
	lastReq helper.Request
	resp    *helper.Response
	err     error
}

func (s *stubDoer) Do(_ context.Context, req helper.Request) (*helper.Response, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	resp := *s.resp
	resp.ID = req.ID
	return &resp, nil
}

func TestOCRProviderSnapshotAndQuery(t *testing.T) {
	png := []byte("not-a-real-png") // provider must pass bytes through untouched
	stub := &stubDoer{resp: &helper.Response{
		OK:   true,
		Meta: helper.CaptureMeta{ImgW: 3000, ImgH: 2000, WidthPts: 1500, HeightPts: 1000},
		Observations: []helper.RawBox{
			{Text: "handler", Line: 0, LineText: "func handler()", Conf: 0.97, X: 0.2, Y: 0.6, W: 0.3, H: 0.1},
			{Text: "other", Line: 1, LineText: "other", Conf: 0.9, X: 0.1, Y: 0.1, W: 0.1, H: 0.05},
		},
		PNGBase64: base64.StdEncoding.EncodeToString(png),
	}}
	p := newOCRProvider(stub, "", false)

	snap, err := p.Snapshot(context.Background(), 1)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	req := stub.lastReq
	if req.Op != helper.OpCaptureOCR || req.IncludeImage || req.Level != "accurate" || req.Display != 1 {
		t.Errorf("unexpected capture request: %+v", req)
	}
	if snap.Screen.Width != 1500 || snap.Screen.Height != 1000 || snap.Screen.Scale != 2.0 || snap.Display != 1 {
		t.Fatalf("screen = %+v display = %d", snap.Screen, snap.Display)
	}
	if len(snap.Items) != 2 {
		t.Fatalf("items = %d, want 2", len(snap.Items))
	}
	if snap.Items[0].Rect != image.Rect(300, 300, 750, 400) {
		t.Fatalf("item rect = %v (coordinate conversion broken)", snap.Items[0].Rect)
	}
	if string(snap.ScreenshotPNG) != string(png) {
		t.Fatalf("screenshot bytes mangled: %q", snap.ScreenshotPNG)
	}
	if snap.ID == "" || snap.CapturedAt.IsZero() {
		t.Fatalf("snapshot missing identity: %+v", snap)
	}

	targets, err := p.Query(context.Background(), snap, "handler", scan.QueryOptions{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(targets) != 1 || targets[0].Match != "exact" || targets[0].Source != "ocr" {
		t.Fatalf("targets = %+v", targets)
	}
}
