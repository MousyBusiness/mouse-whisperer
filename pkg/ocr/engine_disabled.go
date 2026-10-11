//go:build noocr

package ocr

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// ErrDisabled is returned by local OCR calls in a noocr build.
var ErrDisabled = errors.New("local OCR is not built into this binary (noocr)")

func ExtractText(img image.Image) ([]TextItem, error) {
	return nil, ErrDisabled
}

func ScaleImageToLogical(img image.Image, scaleFactor float64) (image.Image, error) {
	return nil, ErrDisabled
}

// DrawOverlayWithRegion paints the numbered target overlay without OpenCV so
// the screend scan path keeps working in a noocr build. It mirrors the gocv
// drawing in engine.go (grey boxes, a translucent label plate at each box's
// top-right corner, decimal labels, optional cyan scan region) using the
// built-in 7x13 bitmap font instead of Hershey.
func DrawOverlayWithRegion(img image.Image, matches []TextItem, scanRegion *image.Rectangle) ([]image.Rectangle, image.Image, error) {
	out := image.NewRGBA(img.Bounds())

	if scanRegion != nil {
		strokeRect(out, *scanRegion, color.RGBA{0, 255, 255, 200}, 2)
	}

	var rects []image.Rectangle
	for _, match := range matches {
		rects = append(rects, match.Rect)
	}
	for _, rect := range rects {
		strokeRect(out, rect, color.RGBA{200, 200, 200, 255}, 1)
	}
	for i, rect := range rects {
		paintLabel(out, rect, i)
	}

	return rects, out, nil
}

func strokeRect(dst *image.RGBA, r image.Rectangle, c color.RGBA, thickness int) {
	src := image.NewUniform(c)
	for k := 0; k < thickness; k++ {
		in := r.Inset(k)
		if in.Empty() {
			return
		}
		draw.Draw(dst, image.Rect(in.Min.X, in.Min.Y, in.Max.X, in.Min.Y+1), src, image.Point{}, draw.Src)
		draw.Draw(dst, image.Rect(in.Min.X, in.Max.Y-1, in.Max.X, in.Max.Y), src, image.Point{}, draw.Src)
		draw.Draw(dst, image.Rect(in.Min.X, in.Min.Y, in.Min.X+1, in.Max.Y), src, image.Point{}, draw.Src)
		draw.Draw(dst, image.Rect(in.Max.X-1, in.Min.Y, in.Max.X, in.Max.Y), src, image.Point{}, draw.Src)
	}
}

// paintLabel uses the same plate geometry as cv.PaintText: decimal, not hex,
// because the labels are spoken back and digits transcribe reliably.
func paintLabel(dst *image.RGBA, rect image.Rectangle, i int) {
	s := fmt.Sprintf("%d", i)
	w := len(s) * 16
	h := 20
	p0 := rect.Min.Add(image.Point{X: rect.Size().X - 1 - w/2, Y: 1})
	if p0.Y-h < 0 {
		p0 = p0.Add(image.Point{X: 0, Y: h - p0.Y})
	}
	plate := image.Rect(p0.X, p0.Y-h, p0.X+w, p0.Y)
	draw.Draw(dst, plate, image.NewUniform(color.RGBA{0, 0, 0, 100}), image.Point{}, draw.Over)

	face := basicfont.Face7x13
	textW := font.MeasureString(face, s).Ceil()
	x := plate.Min.X + (w-textW)/2
	y := plate.Max.Y - (h-face.Ascent)/2
	// drawn twice one pixel apart: the bitmap font is thin next to Hershey
	for _, dx := range []int{0, 1} {
		d := font.Drawer{
			Dst:  dst,
			Src:  image.NewUniform(color.RGBA{255, 255, 255, 255}),
			Face: face,
			Dot:  fixed.P(x+dx, y),
		}
		d.DrawString(s)
	}
}
