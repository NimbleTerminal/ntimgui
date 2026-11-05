// Copyright (c) 2026 Neomantra Corp.

// Package geometry contains the CPU-only cell-to-pixel transform.
package geometry

import (
	"image"
	"math"

	"nimbleterminal.dev/ntimgui"
)

// Transform maps logical cell edges to destination coordinates. Offset is
// relative to destination Min, including for an Ebitengine subimage.
type Transform struct{ ScaleX, ScaleY, OffsetX, OffsetY float64 }

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }

// Valid rejects invalid scales and offsets before conversion to integers.
func (t Transform) Valid() bool {
	return finite(t.ScaleX) && finite(t.ScaleY) && t.ScaleX > 0 && t.ScaleY > 0 && finite(t.OffsetX) && finite(t.OffsetY)
}

// Rect rounds absolute edges, never separately rounded cell widths. Invalid,
// empty, or unrepresentable rectangles produce an empty rectangle.
func (t Transform) Rect(r ntimgui.Rect, origin image.Point) image.Rectangle {
	if r.Empty() || !t.Valid() {
		return image.Rectangle{}
	}
	x0 := float64(origin.X) + t.OffsetX + float64(r.X)*t.ScaleX
	y0 := float64(origin.Y) + t.OffsetY + float64(r.Y)*t.ScaleY
	x1 := float64(origin.X) + t.OffsetX + (float64(r.X)+float64(r.W))*t.ScaleX
	y1 := float64(origin.Y) + t.OffsetY + (float64(r.Y)+float64(r.H))*t.ScaleY
	// Stay representable on both native and WASM targets (and in float32 GPU coordinates).
	for _, v := range []float64{x0, y0, x1, y1} {
		if !finite(v) || math.Abs(v) > 1<<30 {
			return image.Rectangle{}
		}
	}
	return image.Rect(int(math.Round(x0)), int(math.Round(y0)), int(math.Round(x1)), int(math.Round(y1)))
}
