package ax

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
)

// TestAXIntegration exercises the real Swift helper against the live
// frontmost application. Beyond needing the helper binary, it requires
// Accessibility permission for the process tree running the test, so it is
// opt-in rather than -short-gated: run it from a trusted terminal with
// AX_INTEGRATION=1 while any titled window has focus.
func TestAXIntegration(t *testing.T) {
	if os.Getenv("AX_INTEGRATION") == "" {
		t.Skip("set AX_INTEGRATION=1 to run against the real helper (needs Accessibility permission and a frontmost window)")
	}
	helperBin, err := filepath.Abs("../../../build/screend-helper")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(helperBin); err != nil {
		t.Skip("build/screend-helper missing — run `make helper` first")
	}

	eng := helper.NewEngine(helperBin)
	defer eng.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := eng.Ping(ctx); err != nil {
		t.Fatalf("ping: %v", err)
	}
	if trusted := eng.AXTrusted(); trusted == nil || !*trusted {
		t.Fatalf("helper is not accessibility-trusted (ax_trusted=%v); grant Accessibility to this terminal", trusted)
	}

	p := NewProvider(eng)
	snap, err := p.Snapshot(ctx, 0)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if snap.Screen.Width <= 0 || snap.Screen.Height <= 0 || snap.Screen.Scale != 1 {
		t.Fatalf("screen = %+v", snap.Screen)
	}
	t.Logf("frontmost window yielded %d titled element words", len(snap.Items))
	for i, it := range snap.Items {
		if i >= 20 {
			break
		}
		t.Logf("  %-30q line=%d rect=%v", it.Text, it.Line, it.Rect)
	}
	if len(snap.Items) == 0 {
		t.Error("no actionable titled elements in the frontmost window — either the app has no AX tree or the walk is broken")
	}
	for _, it := range snap.Items {
		if it.Rect.Empty() {
			t.Errorf("empty rect for %q", it.Text)
		}
	}
}
