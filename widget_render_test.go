// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	"image"
	"image/color"
	"testing"
)

// These rows contain only single-cell glyphs. Read the grid directly so ANSI
// color settings on the host terminal cannot affect the comparison.
func checkGlyphRow(t *testing.T, glyph *GlyphBackend, y int, want string) {
	t.Helper()
	for x, r := range []rune(want) {
		got := glyph.Grid().Cell(image.Pt(x, y)).Text
		if got == "" {
			got = " "
		}
		if got != string(r) {
			t.Errorf("cell (%d,%d) = %q, want %q in %q", x, y, got, r, want)
		}
	}
}

func TestSliderFloatRendering(t *testing.T) {
	theme := DefaultTheme()
	theme.Normal.BG = RGB(20, 30, 40)
	fill, track := theme.Normal.Accent, theme.Normal.BG

	// A 21-cell row leaves 16 cells for the track, a gap, and the label.
	// At 8 pixels per cell the track occupies the first 128 pixels.
	for _, tc := range []struct {
		name        string
		value       float64
		row         string
		thumbPixelX int
		left, right RGBA
	}{
		{"minimum", 20, "●──────20─────── Gain", 1, track, track},
		{"quarter", 40, "████●──40─────── Gain", 32, fill, track},
		{"maximum", 100, "██████100██████● Gain", 126, fill, fill},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := New(WithScreenSize(21, 2), WithTheme(theme))
			value := tc.value
			ctx.BeginFrame()
			ctx.SetCursorPos(Point{Y: 1})
			ctx.SliderFloat("Gain", &value, 20, 100, "%.0f")
			ctx.EndFrame()

			glyph := NewGlyphBackend(21, 2)
			glyph.Render(ctx.DrawList)
			checkGlyphRow(t, glyph, 1, tc.row)

			img := NewRasterBackend(8, 16).RenderToImage(21, 2, ctx.DrawList)
			// Sample inside the track, above the centered value text.
			for _, sample := range []struct {
				x    int
				want RGBA
			}{{8, tc.left}, {tc.thumbPixelX, theme.Normal.FG}, {120, tc.right}} {
				if got := img.RGBAAt(sample.x, 18); got != color.RGBA(sample.want) {
					t.Errorf("track pixel (%d,18) = %v, want %v", sample.x, got, sample.want)
				}
			}
		})
	}
}

func TestColorEditSwatchRendering(t *testing.T) {
	ctx := New(WithScreenSize(24, 5))
	glyph := NewGlyphBackend(24, 5)
	raster := NewRasterBackend(8, 16)
	for _, frame := range []struct {
		tint RGBA
		row  string
	}{
		{RGB(24, 96, 200), "████ Tint #1860C8"},
		{RGB(210, 128, 12), "████ Tint #D2800C"},
	} {
		tint := frame.tint
		ctx.BeginFrame()
		ctx.ColorEdit("Tint", &tint)
		ctx.EndFrame()

		glyph.Render(ctx.DrawList)
		checkGlyphRow(t, glyph, 0, frame.row)
		img := raster.RenderToImage(24, 5, ctx.DrawList)
		for x := 0; x < 4; x++ {
			cell := glyph.Grid().Cell(image.Pt(x, 0))
			want := frame.tint.CanvasColor()
			if cell.Style.Foreground != want || cell.Style.Background != want {
				t.Errorf("swatch cell %d has style %+v, want tint %v", x, cell.Style, frame.tint)
			}
			if got := img.RGBAAt(x*8+4, 8); got != color.RGBA(frame.tint) {
				t.Errorf("swatch cell %d has pixel %v, want %v", x, got, frame.tint)
			}
		}
		if got := img.RGBAAt(32, 8); got.A != 0 {
			t.Errorf("gap between swatch and label has pixel %v, want transparent", got)
		}
	}
}
