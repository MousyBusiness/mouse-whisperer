package vision

import (
	"image"
	"testing"

	"github.com/mousybusiness/mouse_whisperer/internal/screend/helper"
)

func TestToLogical(t *testing.T) {
	cases := []struct {
		name string
		box  helper.RawBox
		meta helper.CaptureMeta
		want image.Rectangle
	}{
		{
			name: "flips vision bottom-left origin and halves retina pixels",
			box:  helper.RawBox{X: 0.2, Y: 0.6, W: 0.3, H: 0.1},
			meta: helper.CaptureMeta{ImgW: 3000, ImgH: 2000, WidthPts: 1500, HeightPts: 1000},
			// without the Y flip this box would land at y=600..700
			want: image.Rect(300, 300, 750, 400),
		},
		{
			name: "identity at scale 1.0",
			box:  helper.RawBox{X: 0.1, Y: 0.8, W: 0.2, H: 0.1},
			meta: helper.CaptureMeta{ImgW: 1000, ImgH: 500, WidthPts: 1000, HeightPts: 500},
			want: image.Rect(100, 50, 300, 100),
		},
		{
			name: "offsets by the display origin in global points",
			box:  helper.RawBox{X: 0.5, Y: 0, W: 0.5, H: 1},
			meta: helper.CaptureMeta{ImgW: 2000, ImgH: 1000, OriginXPts: 1728, OriginYPts: 100, WidthPts: 1000, HeightPts: 500},
			want: image.Rect(2228, 100, 2728, 600),
		},
		{
			name: "rounds to nearest point instead of truncating",
			box:  helper.RawBox{X: 0.3333, Y: 0, W: 0.3334, H: 1},
			meta: helper.CaptureMeta{ImgW: 1000, ImgH: 100, WidthPts: 1000, HeightPts: 100},
			want: image.Rect(333, 0, 667, 100),
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ToLogical(c.box, c.meta); got != c.want {
				t.Fatalf("ToLogical() = %v, want %v", got, c.want)
			}
		})
	}
}
