//go:build noocr

package cv

import (
	"errors"
	"image"
)

// ErrDisabled is returned by the OpenCV pipeline in a noocr build.
var ErrDisabled = errors.New("local OpenCV pipeline is not built into this binary (noocr)")

func ExtractUIElementsRects(raw image.Image, debugPOIDir string) ([]image.Rectangle, image.Image, error) {
	return nil, nil, ErrDisabled
}
