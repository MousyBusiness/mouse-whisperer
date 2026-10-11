//go:build !noocr

package cv

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"slices"

	"gocv.io/x/gocv"
)

// mergeRects merges two rectangles into one that encompasses both.
func mergeRects(a, b image.Rectangle) image.Rectangle {
	return image.Rect(
		min(a.Min.X, b.Min.X), // Leftmost side
		min(a.Min.Y, b.Min.Y), // Topmost side
		max(a.Max.X, b.Max.X), // Rightmost side
		max(a.Max.Y, b.Max.Y), // Bottommost side
	)
}

// expandRect grows a rect by x, y values
func expandRect(a image.Rectangle, x, y int) image.Rectangle {
	return image.Rect(a.Min.X-x, a.Min.Y-y, a.Max.X+x, a.Max.Y+y)
}

// mergeOverlappingRects merges all rectangles that overlap horizontally.
func mergeOverlappingRects(rects []image.Rectangle, x, y int) []image.Rectangle {
	var merged []image.Rectangle
	rerun := false
	for i, a := range rects {
		mergeOccurred := false
		for j, b := range rects {
			if i == j {
				continue
			}

			if a.In(b) {
				continue
			}
			if b.In(a) {
				continue
			}
			expanded := expandRect(a, x, y)
			if expanded.Overlaps(b) {
				merged = append(merged, mergeRects(a, b))
				mergeOccurred = true
				rerun = true
				break
			}
		}
		if !mergeOccurred {
			merged = append(merged, a)
		}
	}

	if rerun {
		return mergeOverlappingRects(merged, x, y)
	}

	return merged
}

// mergeContainedRects merges rects that contain other rects
func mergeContainedRects(rects []image.Rectangle, x, y int) []image.Rectangle {
	var merged []image.Rectangle
	rerun := false
	for i, a := range rects {
		mergeOccurred := false
		for j, b := range rects {
			if i == j {
				continue
			}

			if a.In(b) || b.In(a) || a.Overlaps(b) {
				merged = append(merged, mergeRects(a, b))
				mergeOccurred = true
				rerun = true
				break
			}

		}
		if !mergeOccurred {
			merged = append(merged, a)
		}
	}

	if rerun {
		return mergeOverlappingRects(merged, x, y)
	}

	return merged
}

// ExtractUIElementsRects uses the captured screenshot and generates a new image containing clickable points.
// If debugPOIDir is non-empty, writes intermediate pipeline stages as PNGs into that directory.
func ExtractUIElementsRects(raw image.Image, debugPOIDir string) ([]image.Rectangle, image.Image, error) {
	// Normalize image to ensure correct stride for gocv conversion
	bounds := raw.Bounds()
	normalized := image.NewRGBA(bounds)
	draw.Draw(normalized, bounds, raw, bounds.Min, draw.Src)

	img, _ := gocv.ImageToMatRGB(normalized)
	defer img.Close()

	inverted := img.Clone()
	defer inverted.Close()
	// Invert the image colors
	gocv.BitwiseNot(inverted, &inverted)

	// Preprocess the image
	gray := gocv.NewMat()
	defer gray.Close()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	grayInverted := gocv.NewMat()
	defer grayInverted.Close()
	gocv.CvtColor(inverted, &grayInverted, gocv.ColorBGRToGray)

	blurredInverted := gocv.NewMat()
	defer blurredInverted.Close()
	gocv.GaussianBlur(grayInverted, &blurredInverted, image.Pt(13, 13), 2, 2, gocv.BorderDefault)

	blurredLight := gocv.NewMat()
	defer blurredLight.Close()
	gocv.GaussianBlur(gray, &blurredLight, image.Pt(13, 13), 2, 2, gocv.BorderDefault)

	threshLight := gocv.NewMat()
	defer threshLight.Close()
	gocv.Threshold(blurredLight, &threshLight, 220, 255, gocv.ThresholdBinary)

	threshDark := gocv.NewMat()
	defer threshDark.Close()
	gocv.Threshold(blurredInverted, &threshDark, 190, 255, gocv.ThresholdBinary)

	if debugPOIDir != "" {
		_ = os.MkdirAll(debugPOIDir, 0o755)
		writeStage := func(name string, m gocv.Mat) {
			_ = gocv.IMWrite(filepath.Join(debugPOIDir, name), m)
		}
		writeStage("01_bgr.png", img)
		writeStage("02_gray.png", gray)
		writeStage("03_gray_inverted.png", grayInverted)
		writeStage("04_blur_light.png", blurredLight)
		writeStage("05_blur_inverted.png", blurredInverted)
		writeStage("06_thresh_light.png", threshLight)
		writeStage("07_thresh_dark.png", threshDark)
	}

	// Detect contours - handle both light and dark background scenarios
	contoursLight := gocv.FindContours(threshLight, gocv.RetrievalTree, gocv.ChainApproxSimple)
	defer contoursLight.Close()
	contoursDark := gocv.FindContours(threshDark, gocv.RetrievalTree, gocv.ChainApproxSimple)
	defer contoursDark.Close()

	// Approximate contours to rectangles and filter as needed
	var lightRects []image.Rectangle
	for i := 0; i < contoursLight.Size(); i++ {
		c := contoursLight.At(i)
		rect := gocv.BoundingRect(c)
		if rect.Size().X*rect.Size().Y > 10000 {
			continue
		}
		if rect.Size().X*rect.Size().Y < 100 {
			continue
		}
		lightRects = append(lightRects, rect)
	}

	var darkRects []image.Rectangle
	// Approximate contours to rectangles and filter as needed
	for i := 0; i < contoursDark.Size(); i++ {
		c := contoursDark.At(i)
		rect := gocv.BoundingRect(c)
		if rect.Size().X*rect.Size().Y > 10000 {
			continue
		}
		if rect.Size().X*rect.Size().Y < 100 {
			continue
		}
		darkRects = append(darkRects, rect)
	}

	var allRect []image.Rectangle
	allRect = append(allRect, mergeOverlappingRects(lightRects, 8, 0)...)
	allRect = append(allRect, mergeOverlappingRects(darkRects, 8, 0)...)

	slices.SortFunc(allRect, func(i, j image.Rectangle) int {
		v := 1
		if i.Min.X/5 < j.Min.X/5 {
			v = v - 1
		}
		if i.Min.Y/5 < j.Min.Y/5 {
			v = v - 1
		}
		return v
	})

	allRect = mergeContainedRects(allRect, 0, 0)

	mp := make(map[image.Rectangle]bool)
	for _, v := range allRect {
		mp[v] = true
	}

	var f []image.Rectangle
	for k, _ := range mp {
		f = append(f, k)
	}

	allRect = f
	slices.SortFunc(allRect, func(i, j image.Rectangle) int {
		s1 := i.Size()
		s2 := j.Size()

		a := s1.X * s1.Y
		b := s2.X * s2.Y

		if a < b {
			return -1
		}
		return 1
	})

	alpha := gocv.NewMatWithSize(img.Rows(), img.Cols(), gocv.MatTypeCV8UC4)
	defer alpha.Close()

	paintImage(&alpha, allRect)

	if debugPOIDir != "" {
		_ = gocv.IMWrite(filepath.Join(debugPOIDir, "08_overlay.png"), alpha)
	}

	overlay, err := alpha.ToImage()
	if err != nil {
		return nil, nil, err
	}

	return allRect, overlay, nil
}

func paintImage(mat *gocv.Mat, allRect []image.Rectangle) {
	// Paint rectangles
	for _, rect := range allRect {
		PaintRect(rect, mat)
	}

	// Paint text last
	for i, rect := range allRect {
		PaintText(rect, mat, i)
	}

}

func PaintRect(rect image.Rectangle, mat *gocv.Mat) {
	r := rect
	x := rect.Size().X
	minX := 50
	if x < minX {
		diffX := minX - x
		halfDiff := diffX / 2
		p1 := r.Min.Sub(image.Point{
			X: halfDiff,
			Y: 0,
		})
		p2 := r.Max.Add(image.Point{
			X: halfDiff,
			Y: 0,
		})
		r = image.Rect(p1.X, p1.Y, p2.X, p2.Y)
	}
	minY := 30
	y := rect.Size().Y
	if y < minY {
		diffY := minY - y
		halfDiff := diffY / 2
		p1 := r.Min.Sub(image.Point{
			X: 0,
			Y: halfDiff,
		})
		p2 := r.Max.Add(image.Point{
			X: 0,
			Y: halfDiff,
		})
		r = image.Rect(p1.X, p1.Y, p2.X, p2.Y)
	}
	gocv.Rectangle(mat, r, color.RGBA{200, 200, 200, 255}, 1)
}

// PaintRectExact draws a rectangle without any size expansion (for OCR)
func PaintRectExact(rect image.Rectangle, mat *gocv.Mat) {
	gocv.Rectangle(mat, rect, color.RGBA{200, 200, 200, 255}, 1)
}

// PaintScanRegion draws a scan region boundary in a distinct color (cyan/blue)
func PaintScanRegion(rect image.Rectangle, mat *gocv.Mat) {
	// Draw with thicker line and cyan color to distinguish from match rectangles
	gocv.Rectangle(mat, rect, color.RGBA{0, 255, 255, 200}, 2)
}

func PaintText(rect image.Rectangle, mat *gocv.Mat, i int) {
	// decimal, not hex: these labels are spoken back by the user and digits
	// are the only thing the transcriber reproduces reliably
	s := fmt.Sprintf("%d", i)
	w := len(s) * 16
	h := 20
	p0 := rect.Min.Add(image.Point{X: rect.Size().X - 1 - w/2, Y: 1})
	if p0.Y-h < 0 {
		p0 = p0.Add(image.Point{X: 0, Y: h - p0.Y})
	}
	p1 := p0.Add(image.Point{X: w, Y: -h})
	gocv.Rectangle(mat, image.Rect(p0.X, p0.Y, p1.X, p1.Y), color.RGBA{0, 0, 0, 100}, -1)
	gocv.PutText(mat, s, p0.Add(image.Point{X: 1, Y: -2}), gocv.FontHersheyPlain, 1.5, color.RGBA{255, 255, 255, 255}, 2)
}

func PaintDot(mat *gocv.Mat, center image.Point, col color.RGBA) {
	gocv.Circle(mat, center, 10, col, -1)
}
