package geometry

import (
	"image"
	"math"
	"nimbleterminal.dev/ntimgui"
	"testing"
)

func TestFractionalEdgesAndOffset(t *testing.T) {
	tr := Transform{ScaleX: 2.5, ScaleY: 3.25, OffsetX: .5, OffsetY: 1.25}
	origin := image.Pt(11, 19)
	whole := tr.Rect(ntimgui.Rect{X: -2, Y: 1, W: 12, H: 2}, origin)
	if whole != image.Rect(7, 24, 37, 30) {
		t.Fatal(whole)
	}
	for x := -2; x < 9; x++ {
		a := tr.Rect(ntimgui.Rect{X: x, Y: 1, W: 1, H: 2}, origin)
		b := tr.Rect(ntimgui.Rect{X: x + 1, Y: 1, W: 1, H: 2}, origin)
		if a.Max.X != b.Min.X {
			t.Fatal("gap", a, b)
		}
	}
}
func TestInvalidGeometry(t *testing.T) {
	for _, tr := range []Transform{{}, {ScaleX: -1, ScaleY: 2}, {ScaleX: math.NaN(), ScaleY: 2}, {ScaleX: 1, ScaleY: 2, OffsetX: math.Inf(1)}} {
		if !tr.Rect(ntimgui.Rect{W: 1, H: 1}, image.Point{}).Empty() {
			t.Fatal(tr)
		}
	}
}
