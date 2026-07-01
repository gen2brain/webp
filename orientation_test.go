package webp

import (
	"image"
	"testing"
)

// rgbaWrapper hides the concrete *image.RGBA type so applyOrientation takes the generic path.
type rgbaWrapper struct{ *image.RGBA }

func TestApplyOrientationFastPath(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 5, 3))
	for i := range src.Pix {
		src.Pix[i] = byte(i)
	}

	for o := 2; o <= 8; o++ {
		fast := applyOrientation(src, o).(*image.RGBA)
		slow := applyOrientation(rgbaWrapper{src}, o).(*image.RGBA)

		if fast.Bounds() != slow.Bounds() {
			t.Fatalf("orientation %d: bounds %v vs %v", o, fast.Bounds(), slow.Bounds())
		}

		for y := 0; y < fast.Bounds().Dy(); y++ {
			for x := 0; x < fast.Bounds().Dx(); x++ {
				if fast.RGBAAt(x, y) != slow.RGBAAt(x, y) {
					t.Fatalf("orientation %d: pixel (%d,%d) %v vs %v", o, x, y, fast.RGBAAt(x, y), slow.RGBAAt(x, y))
				}
			}
		}
	}
}
