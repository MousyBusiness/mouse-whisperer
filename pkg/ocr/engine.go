//go:build !noocr

package ocr

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"strings"

	"github.com/mousybusiness/mouse_whisperer/pkg/cv"
	"github.com/otiai10/gosseract/v2"
	"gocv.io/x/gocv"
)

// ExtractText performs OCR on the given image and returns all detected text items with their bounding boxes
func ExtractText(img image.Image) ([]TextItem, error) {
	// Normalize image to ensure correct stride for gosseract
	bounds := img.Bounds()
	normalized := image.NewRGBA(bounds)
	draw.Draw(normalized, bounds, img, bounds.Min, draw.Src)

	imgBytes, err := imageToBytes(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to convert image to bytes: %w", err)
	}

	client := gosseract.NewClient()
	defer client.Close()

	// Set image from bytes
	client.SetImageFromBytes(imgBytes)

	// Get bounding boxes with text
	boxes, err := client.GetBoundingBoxes(gosseract.RIL_WORD)
	if err != nil {
		return nil, fmt.Errorf("failed to get bounding boxes: %w", err)
	}

	var items []TextItem
	for _, box := range boxes {
		// Filter out empty or very small text
		text := strings.TrimSpace(box.Word)
		if text == "" || len(text) < 1 {
			continue
		}

		// Convert box coordinates to image.Rectangle
		rect := image.Rect(box.Box.Min.X, box.Box.Min.Y, box.Box.Max.X, box.Box.Max.Y)

		// Filter out very small rectangles (likely noise)
		if rect.Dx() < 1 || rect.Dy() < 1 {
			continue
		}

		items = append(items, TextItem{
			Text: text,
			Rect: rect,
		})
	}

	return items, nil
}

// DrawOverlayWithRegion creates an overlay image with rectangles, hex IDs, and optionally a scan region indicator
func DrawOverlayWithRegion(img image.Image, matches []TextItem, scanRegion *image.Rectangle) ([]image.Rectangle, image.Image, error) {
	bounds := img.Bounds()

	// Convert image to gocv Mat for drawing
	normalized := image.NewRGBA(bounds)
	draw.Draw(normalized, bounds, img, bounds.Min, draw.Src)

	mat, err := gocv.ImageToMatRGB(normalized)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert to mat: %w", err)
	}
	defer mat.Close()

	// Create alpha channel image for overlay
	alpha := gocv.NewMatWithSize(mat.Rows(), mat.Cols(), gocv.MatTypeCV8UC4)
	defer alpha.Close()

	// Draw scan region if provided (in a distinct color)
	if scanRegion != nil {
		cv.PaintScanRegion(*scanRegion, &alpha)
	}

	// Extract rectangles
	var rects []image.Rectangle
	for _, match := range matches {
		rects = append(rects, match.Rect)
	}

	// Paint rectangles using exact bounds (no expansion for OCR)
	for _, rect := range rects {
		cv.PaintRectExact(rect, &alpha)
	}

	// Paint text labels with hex IDs
	for i, rect := range rects {
		cv.PaintText(rect, &alpha, i)
	}

	overlay, err := alpha.ToImage()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert overlay to image: %w", err)
	}

	return rects, overlay, nil
}

// ScaleImageToLogical scales an image from physical to logical coordinates
func ScaleImageToLogical(img image.Image, scaleFactor float64) (image.Image, error) {
	logicalWidth := int(float64(img.Bounds().Dx()) / scaleFactor)
	logicalHeight := int(float64(img.Bounds().Dy()) / scaleFactor)

	// Convert to gocv Mat and resize
	imgBounds := img.Bounds()
	normalized := image.NewRGBA(imgBounds)
	draw.Draw(normalized, imgBounds, img, imgBounds.Min, draw.Src)
	mat, err := gocv.ImageToMatRGB(normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to mat: %w", err)
	}
	defer mat.Close()

	resizedMat := gocv.NewMat()
	defer resizedMat.Close()
	gocv.Resize(mat, &resizedMat, image.Pt(logicalWidth, logicalHeight), 0, 0, gocv.InterpolationLinear)

	logicalImg, err := resizedMat.ToImage()
	if err != nil {
		return nil, fmt.Errorf("failed to convert resized mat to image: %w", err)
	}

	return logicalImg, nil
}

// imageToBytes converts an image to PNG bytes for gosseract
func imageToBytes(img image.Image) ([]byte, error) {
	bounds := img.Bounds()
	if bounds.Dx() == 0 || bounds.Dy() == 0 {
		return nil, fmt.Errorf("image has zero dimensions: %dx%d", bounds.Dx(), bounds.Dy())
	}

	// Convert to grayscale for better OCR performance
	gray := image.NewGray(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			originalColor := img.At(x, y)
			grayColor := color.GrayModel.Convert(originalColor)
			gray.Set(x, y, grayColor)
		}
	}

	// Use standard Go PNG encoder - more reliable than gocv for this
	var buf bytes.Buffer
	if err := png.Encode(&buf, gray); err != nil {
		return nil, fmt.Errorf("failed to encode image as PNG: %w", err)
	}

	if buf.Len() == 0 {
		return nil, fmt.Errorf("encoded PNG is empty")
	}

	return buf.Bytes(), nil
}
