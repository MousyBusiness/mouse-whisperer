//go:build noocr

package ocr

import (
	"image"
	"image/color"
	"testing"
)

func TestDrawOverlayPureGo(t *testing.T) {
	canvas := image.NewRGBA(image.Rect(0, 0, 400, 300))
	items := []TextItem{
		{Text: "a", Rect: image.Rect(100, 100, 160, 120)},
		{Text: "b", Rect: image.Rect(200, 5, 260, 25)},
	}
	rects, img, err := DrawOverlay(canvas, items)
	if err != nil {
		t.Fatal(err)
	}
	if len(rects) != 2 || rects[0] != items[0].Rect || rects[1] != items[1].Rect {
		t.Fatalf("rects = %v", rects)
	}
	if img.Bounds() != canvas.Bounds() {
		t.Fatalf("bounds = %v, want %v", img.Bounds(), canvas.Bounds())
	}
	grey := color.RGBA{200, 200, 200, 255}
	if got := img.At(100, 110); got != grey {
		t.Errorf("box edge = %v, want %v", got, grey)
	}
	if _, _, _, a := img.At(130, 110).RGBA(); a != 0 {
		t.Errorf("box interior should stay transparent, alpha = %d", a)
	}
	// the label plate sits above the box's top-right corner
	if _, _, _, a := img.At(160, 90).RGBA(); a == 0 {
		t.Error("expected a label plate above the first box")
	}
}
