package vision

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
	"github.com/mousybusiness/mouse_whisperer/internal/screend/scan"
)

// TestVisionIntegration exercises the real Swift helper + Vision framework
// against a committed fixture. It is the only test that touches native code.
func TestVisionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test skipped with -short")
	}
	helperBin, err := filepath.Abs("../../../build/screend-helper")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(helperBin); err != nil {
		t.Skip("build/screend-helper missing — run `make helper` first")
	}
	fixture, err := filepath.Abs("testdata/known_text.png")
	if err != nil {
		t.Fatal(err)
	}

	eng := helper.NewEngine(helperBin)
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := eng.Do(ctx, helper.Request{Op: helper.OpOCRFile, Path: fixture})
	if err != nil {
		t.Fatalf("ocr_file: %v", err)
	}

	items := make(map[string]scan.Item)
	for _, ob := range resp.Observations {
		rect := ToLogical(ob, resp.Meta)
		if rect.Empty() {
			t.Errorf("empty rect for %q", ob.Text)
		}
		if _, seen := items[ob.Text]; !seen {
			items[ob.Text] = scan.Item{Text: ob.Text, Line: ob.Line, Rect: rect}
		}
	}

	for _, want := range []string{"handler", "whisperer", "scan", "RegisterOCRHandlers", "mouse", "app"} {
		if _, ok := items[want]; !ok {
			t.Errorf("word %q not recognized; got %v", want, keys(items))
		}
	}
	if t.Failed() {
		t.FailNow()
	}

	// fixture layout: "handler whisperer scan" is the TOP line, "mouse
	// whisperer app" the BOTTOM — verifies the Y flip end to end
	if items["handler"].Rect.Min.Y >= items["mouse"].Rect.Min.Y {
		t.Errorf("Y flip broken: handler(y=%d) should be above mouse(y=%d)",
			items["handler"].Rect.Min.Y, items["mouse"].Rect.Min.Y)
	}
	if items["handler"].Rect.Min.X >= items["whisperer"].Rect.Min.X {
		t.Errorf("x order broken: handler(x=%d) should be left of whisperer(x=%d)",
			items["handler"].Rect.Min.X, items["whisperer"].Rect.Min.X)
	}
	if items["mouse"].Line != items["app"].Line {
		t.Errorf("line grouping broken: mouse(line=%d) vs app(line=%d)",
			items["mouse"].Line, items["app"].Line)
	}
	if items["handler"].Line == items["mouse"].Line {
		t.Error("distinct rows must land on distinct lines")
	}
}

func keys(m map[string]scan.Item) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
