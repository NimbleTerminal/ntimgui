// Copyright (c) 2026 Neomantra Corp.

package ntimguiebiten

import (
	"image"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"nimbleterminal.dev/ntimgui"
)

// Symbols used by ntimgui have vector fallbacks even with a limited custom font.
// Reserve one measured M advance for each symbol, preserving real measurements
// and shaping within ordinary text runs (including proportional fonts).
const symbols = "▾▸◂▴▼▶◀▲●◉○─▏█✓✔☑☐"

type textRun struct {
	value   string
	symbol  rune
	advance float64
}

func textRuns(value string, face text.Face) ([]textRun, float64) {
	value, _, _ = strings.Cut(value, "\n")
	value = strings.ReplaceAll(value, "\r", "")
	var runs []textRun
	var width float64
	start := 0
	appendText := func(s string) {
		if s != "" {
			a := text.Advance(s, face)
			runs = append(runs, textRun{value: s, advance: a})
			width += a
		}
	}
	for i, r := range value {
		if !strings.ContainsRune(symbols, r) {
			continue
		}
		appendText(value[start:i])
		a := text.Advance("M", face)
		runs = append(runs, textRun{symbol: r, advance: a})
		width += a
		start = i + len(string(r))
	}
	appendText(value[start:])
	return runs, width
}
func (b *Backend) drawText(dst *ebiten.Image, r image.Rectangle, value string, align ntimgui.Align, s ntimgui.WidgetStyle) {
	if value == "" || s.FG.A == 0 || r.Empty() {
		return
	}
	visible := r.Intersect(dst.Bounds())
	if visible.Empty() {
		return
	}
	dst = dst.SubImage(visible).(*ebiten.Image)
	face := b.regular
	if s.Bold && b.bold != nil {
		face = b.bold
	}
	runs, width := textRuns(value, face)
	metrics := face.Metrics()
	height := metrics.HAscent + metrics.HDescent
	x := float64(r.Min.X)
	switch align {
	case ntimgui.AlignCenter:
		x += (float64(r.Dx()) - width) / 2
	case ntimgui.AlignRight:
		x = float64(r.Max.X) - width
	}
	y := float64(r.Min.Y) + (float64(r.Dy())-height)/2
	start := x
	for _, run := range runs {
		if run.symbol != 0 {
			drawSymbol(dst, run.symbol, float32(x), float32(y), float32(run.advance), float32(height), s.FG)
		} else {
			op := &text.DrawOptions{}
			op.GeoM.Translate(x, y)
			op.ColorScale.ScaleWithColor(s.FG.StdNRGBA())
			text.Draw(dst, run.value, face, op)
		}
		x += run.advance
	}
	if s.Underline {
		vector.FillRect(dst, float32(start), float32(y+metrics.HAscent+1), float32(width), 1, s.FG.StdNRGBA(), false)
	}
}
func drawSymbol(dst *ebiten.Image, r rune, x, y, w, h float32, c ntimgui.RGBA) {
	col := c.StdNRGBA()
	cx, cy := x+w/2, y+h/2
	radius := min(w*.38, h*.24)
	stroke := max(float32(1), radius*.3)
	check := func(x0, y0, x1, y1, x2, y2 float32) {
		var p vector.Path
		p.MoveTo(x0, y0)
		p.LineTo(x1, y1)
		p.LineTo(x2, y2)
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(col)
		vector.StrokePath(dst, &p, &vector.StrokeOptions{Width: stroke, LineJoin: vector.LineJoinRound}, op)
	}
	switch r {
	case '●', '◉':
		vector.FillCircle(dst, cx, cy, radius, col, true)
	case '○':
		vector.StrokeCircle(dst, cx, cy, radius, stroke, col, true)
	case '─':
		vector.FillRect(dst, x, cy, w, 1, col, false)
	case '▏':
		vector.FillRect(dst, x+1, y+1, 1, max(0, h-2), col, false)
	case '█':
		vector.FillRect(dst, x, y+1, w, max(0, h-2), col, false)
	case '✓', '✔':
		check(x+w*.1, cy, x+w*.4, y+h*.7, x+w*.9, y+h*.3)
	case '☐', '☑':
		vector.StrokeRect(dst, x+1, cy-radius, max(0, w-2), radius*2, stroke, col, true)
		if r == '☑' {
			check(x+w*.15, cy, x+w*.4, cy+radius*.6, x+w*.85, cy-radius*.6)
		}
	default:
		var p vector.Path
		switch r {
		case '▾', '▼':
			p.MoveTo(cx-radius, cy-radius/2)
			p.LineTo(cx+radius, cy-radius/2)
			p.LineTo(cx, cy+radius)
		case '▴', '▲':
			p.MoveTo(cx-radius, cy+radius/2)
			p.LineTo(cx+radius, cy+radius/2)
			p.LineTo(cx, cy-radius)
		case '▸', '▶':
			p.MoveTo(cx-radius/2, cy-radius)
			p.LineTo(cx-radius/2, cy+radius)
			p.LineTo(cx+radius, cy)
		case '◂', '◀':
			p.MoveTo(cx+radius/2, cy-radius)
			p.LineTo(cx+radius/2, cy+radius)
			p.LineTo(cx-radius, cy)
		}
		p.Close()
		op := &vector.DrawPathOptions{AntiAlias: true}
		op.ColorScale.ScaleWithColor(col)
		vector.FillPath(dst, &p, nil, op)
	}
}
