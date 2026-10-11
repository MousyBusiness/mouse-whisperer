package ax

import (
	"context"
	"image"
	"testing"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

type stubDoer struct {
	lastReq helper.Request
	resp    *helper.Response
}

func (s *stubDoer) Do(_ context.Context, req helper.Request) (*helper.Response, error) {
	s.lastReq = req
	resp := *s.resp
	resp.ID = req.ID
	return &resp, nil
}

func snapshotFor(t *testing.T, elements []helper.RawElement) (*Provider, *scan.Snapshot, *stubDoer) {
	t.Helper()
	stub := &stubDoer{resp: &helper.Response{
		OK:       true,
		Meta:     helper.CaptureMeta{ImgW: 1512, ImgH: 982, WidthPts: 1512, HeightPts: 982},
		Elements: elements,
	}}
	p := newProvider(stub)
	snap, err := p.Snapshot(context.Background(), 0)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	return p, snap, stub
}

func TestSnapshotPassesFramesThroughUntouched(t *testing.T) {
	_, snap, stub := snapshotFor(t, []helper.RawElement{
		{Role: "AXButton", Title: "Save", Enabled: true, X: 100, Y: 200, W: 80, H: 24},
	})
	if stub.lastReq.Op != helper.OpAXQuery {
		t.Fatalf("op = %q, want ax_query", stub.lastReq.Op)
	}
	if len(snap.Items) != 1 {
		t.Fatalf("items = %d, want 1", len(snap.Items))
	}
	// AX frames are already global logical points top-left: any Vision-style
	// Y flip or scale division here is corruption
	if snap.Items[0].Rect != image.Rect(100, 200, 180, 224) {
		t.Fatalf("rect = %v, want (100,200)-(180,224)", snap.Items[0].Rect)
	}
	if snap.Screen.Width != 1512 || snap.Screen.Height != 982 || snap.Screen.Scale != 1 {
		t.Fatalf("screen = %+v", snap.Screen)
	}
}

func TestSnapshotSkipsDisabledAndUntitled(t *testing.T) {
	_, snap, _ := snapshotFor(t, []helper.RawElement{
		{Role: "AXButton", Title: "Save", Enabled: false, X: 0, Y: 0, W: 10, H: 10},
		{Role: "AXButton", Title: "", Enabled: true, X: 0, Y: 20, W: 10, H: 10},
		{Role: "AXButton", Title: "  ", Enabled: true, X: 0, Y: 40, W: 10, H: 10},
		{Role: "AXButton", Title: "Open", Enabled: true, X: 0, Y: 60, W: 10, H: 10},
	})
	if len(snap.Items) != 1 || snap.Items[0].Text != "Open" {
		t.Fatalf("items = %+v, want just Open", snap.Items)
	}
}

func TestMultiWordTitleSplitsIntoWordRun(t *testing.T) {
	_, snap, _ := snapshotFor(t, []helper.RawElement{
		{Role: "AXButton", Title: "Save As", Enabled: true, X: 100, Y: 0, W: 70, H: 20},
	})
	if len(snap.Items) != 2 {
		t.Fatalf("items = %d, want 2 words", len(snap.Items))
	}
	if snap.Items[0].Text != "Save" || snap.Items[1].Text != "As" {
		t.Fatalf("words = %q, %q", snap.Items[0].Text, snap.Items[1].Text)
	}
	if snap.Items[0].Line != snap.Items[1].Line {
		t.Fatal("words of one title must share a line index for phrase matching")
	}
	if snap.Items[0].LineText != "Save As" {
		t.Fatalf("line text = %q", snap.Items[0].LineText)
	}
	// proportional slices: "Save" is runes 0-4 of 7 → x 100..140
	if snap.Items[0].Rect != image.Rect(100, 0, 140, 20) {
		t.Fatalf("first word rect = %v", snap.Items[0].Rect)
	}
}

func TestQueryMatchesPhraseAcrossTitleWords(t *testing.T) {
	p, snap, _ := snapshotFor(t, []helper.RawElement{
		{Role: "AXButton", Title: "Save As", Enabled: true, X: 100, Y: 0, W: 70, H: 20},
		{Role: "AXButton", Title: "Cancel", Enabled: true, X: 0, Y: 40, W: 60, H: 20},
	})
	targets, err := p.Query(context.Background(), snap, "save as", scan.QueryOptions{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(targets) != 1 || targets[0].Match != "phrase" || targets[0].Source != "ax" {
		t.Fatalf("targets = %+v", targets)
	}
	// the phrase union must recover the full element frame
	if targets[0].Rect != image.Rect(100, 0, 170, 20) {
		t.Fatalf("phrase rect = %v, want full frame", targets[0].Rect)
	}

	targets, err = p.Query(context.Background(), snap, "cancel", scan.QueryOptions{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(targets) != 1 || targets[0].Match != "exact" {
		t.Fatalf("targets = %+v", targets)
	}
}
