package vision

import (
	"image"
	"math"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
)

// ToLogical converts a Vision-normalized bottom-left-origin box into a
// top-left-origin logical (point) rect in global screen coordinates — the
// space robotgo and the overlay consume. Scale is derived per axis from
// captured pixels vs display points (Retina = 2.0).
func ToLogical(b helper.RawBox, m helper.CaptureMeta) image.Rectangle {
	sx := float64(m.ImgW) / m.WidthPts
	sy := float64(m.ImgH) / m.HeightPts

	pxX0 := b.X * float64(m.ImgW)
	pxX1 := (b.X + b.W) * float64(m.ImgW)
	pxY0 := (1 - b.Y - b.H) * float64(m.ImgH) // flip Y
	pxY1 := (1 - b.Y) * float64(m.ImgH)

	return image.Rect(
		round(pxX0/sx+m.OriginXPts), round(pxY0/sy+m.OriginYPts),
		round(pxX1/sx+m.OriginXPts), round(pxY1/sy+m.OriginYPts),
	)
}

func round(f float64) int { return int(math.Round(f)) }
