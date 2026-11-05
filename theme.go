// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	"image/color"

	"nimbleterminal.dev/ntcanvas"
)

// RGBA represents a non-premultiplied 8-bit sRGB color with alpha.
// An alpha of 0 indicates a transparent color (or default terminal color in glyph mode).
type RGBA struct {
	R, G, B, A uint8
}

// RGB constructs an opaque RGBA color.
func RGB(r, g, b uint8) RGBA {
	return RGBA{R: r, G: g, B: b, A: 255}
}

// ColorRGBA constructs an RGBA color with explicit alpha.
func ColorRGBA(r, g, b, a uint8) RGBA {
	return RGBA{R: r, G: g, B: b, A: a}
}

// IsZero reports whether the color is completely transparent / unset.
func (c RGBA) IsZero() bool {
	return c.A == 0
}

// StdRGBA converts c to a standard library color.NRGBA value.
func (c RGBA) StdNRGBA() color.NRGBA {
	return color.NRGBA{R: c.R, G: c.G, B: c.B, A: c.A}
}

// CanvasColor converts c to an ntcanvas.Color (or zero for default when transparent).
func (c RGBA) CanvasColor() ntcanvas.Color {
	if c.A == 0 {
		return 0
	}
	return ntcanvas.RGBColor(c.R, c.G, c.B)
}

// BorderStyle selects how box outlines are drawn in glyph and raster backends.
type BorderStyle uint8

const (
	BorderNone BorderStyle = iota
	BorderSingle
	BorderRounded
	BorderDouble
	BorderThick
)

// WidgetStyle defines the visual colors and text attributes of a draw primitive.
type WidgetStyle struct {
	FG        RGBA
	BG        RGBA
	Border    RGBA
	Accent    RGBA
	Bold      bool
	Dim       bool
	Underline bool
	Reverse   bool
}

// CellStyle converts WidgetStyle into an ntcanvas.CellStyle.
func (s WidgetStyle) CellStyle() ntcanvas.CellStyle {
	var attr ntcanvas.Attributes
	if s.Bold {
		attr |= ntcanvas.Bold
	}
	if s.Dim {
		attr |= ntcanvas.Dim
	}
	if s.Underline {
		attr |= ntcanvas.Underline
	}
	if s.Reverse {
		attr |= ntcanvas.Reverse
	}
	return ntcanvas.CellStyle{
		Foreground: s.FG.CanvasColor(),
		Background: s.BG.CanvasColor(),
		Attributes: attr,
	}
}

// BorderCellStyle returns the ntcanvas.CellStyle for border runes on this widget.
func (s WidgetStyle) BorderCellStyle() ntcanvas.CellStyle {
	fg := s.Border
	if fg.IsZero() {
		fg = s.FG
	}
	return ntcanvas.CellStyle{
		Foreground: fg.CanvasColor(),
		Background: s.BG.CanvasColor(),
	}
}

// Theme groups semantic widget styles for each interaction state.
type Theme struct {
	Text       WidgetStyle
	Muted      WidgetStyle
	Normal     WidgetStyle
	Hover      WidgetStyle
	Active     WidgetStyle
	Focus      WidgetStyle
	Panel      WidgetStyle
	PanelTitle WidgetStyle
	Input      WidgetStyle
	InputFocus WidgetStyle
	Tooltip    WidgetStyle

	Border      BorderStyle
	PanelBorder BorderStyle

	// CompactButtons controls default Button height: 1 row when true, 3 rows when false.
	CompactButtons bool
}

// DefaultTheme returns a dark Dear ImGui-inspired theme suitable for both
// terminal cell grids and translucent 3D raster HUD overlays.
func DefaultTheme() Theme {
	return Theme{
		Text: WidgetStyle{
			FG: RGB(230, 235, 245),
		},
		Muted: WidgetStyle{
			FG:  RGB(140, 150, 170),
			Dim: true,
		},
		Normal: WidgetStyle{
			FG:     RGB(225, 232, 245),
			BG:     ColorRGBA(42, 54, 76, 230),
			Border: RGB(78, 98, 134),
			Accent: RGB(76, 154, 255),
		},
		Hover: WidgetStyle{
			FG:     RGB(255, 255, 255),
			BG:     ColorRGBA(62, 86, 128, 240),
			Border: RGB(115, 162, 235),
			Accent: RGB(105, 178, 255),
			Bold:   true,
		},
		Active: WidgetStyle{
			FG:     RGB(255, 255, 255),
			BG:     ColorRGBA(38, 108, 196, 250),
			Border: RGB(150, 200, 255),
			Accent: RGB(140, 205, 255),
			Bold:   true,
		},
		Focus: WidgetStyle{
			FG:     RGB(245, 250, 255),
			BG:     ColorRGBA(50, 68, 98, 240),
			Border: RGB(240, 198, 90),
			Accent: RGB(96, 170, 255),
		},
		Panel: WidgetStyle{
			FG:     RGB(220, 226, 238),
			BG:     ColorRGBA(22, 27, 36, 215),
			Border: RGB(64, 78, 104),
			Accent: RGB(76, 154, 255),
		},
		PanelTitle: WidgetStyle{
			FG:     RGB(240, 246, 255),
			BG:     ColorRGBA(36, 50, 76, 235),
			Border: RGB(86, 112, 156),
			Accent: RGB(105, 178, 255),
			Bold:   true,
		},
		Input: WidgetStyle{
			FG:     RGB(230, 236, 248),
			BG:     ColorRGBA(16, 20, 28, 235),
			Border: RGB(70, 86, 116),
			Accent: RGB(76, 154, 255),
		},
		InputFocus: WidgetStyle{
			FG:     RGB(255, 255, 255),
			BG:     ColorRGBA(24, 32, 46, 245),
			Border: RGB(110, 175, 255),
			Accent: RGB(110, 175, 255),
		},
		Tooltip: WidgetStyle{
			FG:     RGB(245, 248, 255),
			BG:     ColorRGBA(14, 18, 24, 245),
			Border: RGB(120, 145, 185),
		},
		Border:         BorderRounded,
		PanelBorder:    BorderRounded,
		CompactButtons: false,
	}
}

// HUDTheme returns a compact translucent theme tailored for floating 3D control panels.
func HUDTheme() Theme {
	t := DefaultTheme()
	t.CompactButtons = true
	t.Panel.BG = ColorRGBA(18, 22, 30, 195)
	return t
}

// AmberTheme returns a retro phosphor-amber terminal theme.
func AmberTheme() Theme {
	return Theme{
		Text:           WidgetStyle{FG: RGB(255, 196, 64)},
		Muted:          WidgetStyle{FG: RGB(168, 118, 28), Dim: true},
		Normal:         WidgetStyle{FG: RGB(255, 200, 72), BG: ColorRGBA(48, 32, 8, 235), Border: RGB(180, 120, 24), Accent: RGB(255, 176, 0)},
		Hover:          WidgetStyle{FG: RGB(255, 235, 150), BG: ColorRGBA(78, 52, 12, 245), Border: RGB(255, 196, 64), Accent: RGB(255, 210, 80), Bold: true},
		Active:         WidgetStyle{FG: RGB(28, 18, 4), BG: ColorRGBA(255, 188, 42, 250), Border: RGB(255, 230, 128), Accent: RGB(255, 215, 90), Bold: true},
		Focus:          WidgetStyle{FG: RGB(255, 220, 110), BG: ColorRGBA(64, 42, 10, 240), Border: RGB(255, 230, 120), Accent: RGB(255, 196, 64)},
		Panel:          WidgetStyle{FG: RGB(245, 186, 54), BG: ColorRGBA(24, 16, 4, 220), Border: RGB(145, 96, 20), Accent: RGB(255, 176, 0)},
		PanelTitle:     WidgetStyle{FG: RGB(255, 220, 110), BG: ColorRGBA(62, 40, 8, 240), Border: RGB(195, 134, 28), Accent: RGB(255, 196, 64), Bold: true},
		Input:          WidgetStyle{FG: RGB(255, 206, 84), BG: ColorRGBA(16, 10, 2, 240), Border: RGB(140, 92, 18), Accent: RGB(255, 176, 0)},
		InputFocus:     WidgetStyle{FG: RGB(255, 235, 140), BG: ColorRGBA(32, 20, 4, 245), Border: RGB(255, 200, 64), Accent: RGB(255, 200, 64)},
		Tooltip:        WidgetStyle{FG: RGB(255, 225, 120), BG: ColorRGBA(18, 12, 2, 250), Border: RGB(220, 155, 35)},
		Border:         BorderSingle,
		PanelBorder:    BorderDouble,
		CompactButtons: true,
	}
}

// CyberpunkTheme returns a high-contrast neon cyan and magenta theme.
func CyberpunkTheme() Theme {
	return Theme{
		Text:           WidgetStyle{FG: RGB(220, 250, 255)},
		Muted:          WidgetStyle{FG: RGB(120, 140, 180), Dim: true},
		Normal:         WidgetStyle{FG: RGB(0, 245, 255), BG: ColorRGBA(26, 18, 48, 230), Border: RGB(180, 40, 220), Accent: RGB(255, 42, 165)},
		Hover:          WidgetStyle{FG: RGB(255, 255, 255), BG: ColorRGBA(58, 24, 96, 245), Border: RGB(0, 245, 255), Accent: RGB(0, 245, 255), Bold: true},
		Active:         WidgetStyle{FG: RGB(16, 10, 28), BG: ColorRGBA(0, 245, 255, 250), Border: RGB(255, 255, 255), Accent: RGB(255, 42, 165), Bold: true},
		Focus:          WidgetStyle{FG: RGB(255, 240, 100), BG: ColorRGBA(44, 20, 76, 240), Border: RGB(255, 230, 0), Accent: RGB(0, 245, 255)},
		Panel:          WidgetStyle{FG: RGB(215, 245, 255), BG: ColorRGBA(14, 10, 28, 215), Border: RGB(145, 35, 185), Accent: RGB(255, 42, 165)},
		PanelTitle:     WidgetStyle{FG: RGB(0, 250, 255), BG: ColorRGBA(46, 16, 78, 240), Border: RGB(255, 42, 165), Accent: RGB(0, 245, 255), Bold: true},
		Input:          WidgetStyle{FG: RGB(0, 245, 255), BG: ColorRGBA(10, 6, 20, 240), Border: RGB(120, 30, 160), Accent: RGB(255, 42, 165)},
		InputFocus:     WidgetStyle{FG: RGB(255, 255, 255), BG: ColorRGBA(22, 12, 42, 245), Border: RGB(0, 245, 255), Accent: RGB(0, 245, 255)},
		Tooltip:        WidgetStyle{FG: RGB(255, 255, 255), BG: ColorRGBA(20, 8, 36, 250), Border: RGB(0, 245, 255)},
		Border:         BorderThick,
		PanelBorder:    BorderThick,
		CompactButtons: true,
	}
}

// LightTheme returns a clean daylight theme with dark typography on light panels.
func LightTheme() Theme {
	return Theme{
		Text:           WidgetStyle{FG: RGB(28, 34, 46)},
		Muted:          WidgetStyle{FG: RGB(105, 115, 132)},
		Normal:         WidgetStyle{FG: RGB(24, 32, 48), BG: ColorRGBA(212, 220, 234, 240), Border: RGB(140, 154, 178), Accent: RGB(36, 112, 224)},
		Hover:          WidgetStyle{FG: RGB(10, 18, 32), BG: ColorRGBA(186, 204, 235, 245), Border: RGB(64, 124, 218), Accent: RGB(28, 98, 208), Bold: true},
		Active:         WidgetStyle{FG: RGB(255, 255, 255), BG: ColorRGBA(36, 112, 224, 250), Border: RGB(20, 78, 170), Accent: RGB(24, 92, 200), Bold: true},
		Focus:          WidgetStyle{FG: RGB(20, 28, 42), BG: ColorRGBA(200, 214, 238, 245), Border: RGB(210, 120, 20), Accent: RGB(36, 112, 224)},
		Panel:          WidgetStyle{FG: RGB(30, 36, 48), BG: ColorRGBA(238, 242, 248, 230), Border: RGB(156, 168, 188), Accent: RGB(36, 112, 224)},
		PanelTitle:     WidgetStyle{FG: RGB(18, 26, 42), BG: ColorRGBA(210, 220, 236, 245), Border: RGB(130, 146, 172), Accent: RGB(36, 112, 224), Bold: true},
		Input:          WidgetStyle{FG: RGB(20, 26, 38), BG: ColorRGBA(252, 253, 255, 245), Border: RGB(150, 164, 186), Accent: RGB(36, 112, 224)},
		InputFocus:     WidgetStyle{FG: RGB(10, 14, 24), BG: ColorRGBA(255, 255, 255, 255), Border: RGB(36, 112, 224), Accent: RGB(36, 112, 224)},
		Tooltip:        WidgetStyle{FG: RGB(24, 30, 42), BG: ColorRGBA(250, 252, 255, 250), Border: RGB(110, 128, 156)},
		Border:         BorderRounded,
		PanelBorder:    BorderRounded,
		CompactButtons: true,
	}
}
