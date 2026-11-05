// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	"image"
	"math"
	"strings"

	"github.com/mattn/go-runewidth"
	"nimbleterminal.dev/ntcanvas"
)

type borderChars struct {
	tl, tr, bl, br, h, v rune
}

func borderRunes(style BorderStyle) borderChars {
	switch style {
	case BorderRounded:
		return borderChars{tl: '╭', tr: '╮', bl: '╰', br: '╯', h: '─', v: '│'}
	case BorderDouble:
		return borderChars{tl: '╔', tr: '╗', bl: '╚', br: '╝', h: '═', v: '║'}
	case BorderThick:
		return borderChars{tl: '┏', tr: '┓', bl: '┗', br: '┛', h: '━', v: '┃'}
	default:
		return borderChars{tl: '┌', tr: '┐', bl: '└', br: '┘', h: '─', v: '│'}
	}
}

var eighthBlocks = [9]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉', '█'}

// GlyphBackend translates a backend-agnostic DrawList into terminal cells on an
// ntcanvas.GraphemeGrid.
type GlyphBackend struct {
	grid *ntcanvas.GraphemeGrid
}

// NewGlyphBackend creates a GlyphBackend backed by an ntcanvas.GraphemeGrid of
// (width, height) terminal cells.
func NewGlyphBackend(width, height int) *GlyphBackend {
	return &GlyphBackend{
		grid: ntcanvas.NewGraphemeGrid(max(1, width), max(1, height)),
	}
}

// Grid returns the underlying ntcanvas.GraphemeGrid.
func (b *GlyphBackend) Grid() *ntcanvas.GraphemeGrid {
	return b.grid
}

// Resize updates the dimensions of the underlying ntcanvas.GraphemeGrid.
func (b *GlyphBackend) Resize(width, height int) {
	b.grid.Resize(max(1, width), max(1, height))
}

// Render clears the backend's grid and draws all commands in dl.
func (b *GlyphBackend) Render(dl *DrawList) {
	b.grid.Clear()
	RenderToGraphemeGrid(b.grid, dl)
}

// RenderToString clears the grid, renders dl, and returns the formatted ANSI/Unicode string.
func (b *GlyphBackend) RenderToString(dl *DrawList) string {
	b.Render(dl)
	return b.grid.Render(b.grid.Bounds())
}

// OverlayString writes base (a multi-line plain or styled string) into the grid,
// renders dl on top of it, and returns the composited string.
func (b *GlyphBackend) OverlayString(base string, dl *DrawList) string {
	b.grid.Clear()
	if base != "" {
		lines := strings.Split(base, "\n")
		b.grid.SetLines(lines)
	}
	RenderToGraphemeGrid(b.grid, dl)
	return b.grid.Render(b.grid.Bounds())
}

// RenderToGraphemeGrid draws dl directly onto an existing ntcanvas.GraphemeGrid
// without clearing preexisting cells outside the drawn primitives.
func RenderToGraphemeGrid(grid *ntcanvas.GraphemeGrid, dl *DrawList) {
	if grid == nil || dl == nil {
		return
	}
	gb := grid.Bounds()
	gridRect := Rect{X: gb.Min.X, Y: gb.Min.Y, W: gb.Dx(), H: gb.Dy()}

	for _, cmd := range dl.Commands() {
		clip := cmd.Clip.Intersect(gridRect)
		if clip.Empty() {
			continue
		}
		switch cmd.Kind {
		case CmdRect:
			drawGlyphRect(grid, cmd.Bounds, clip, cmd.Style.CellStyle())
		case CmdBox:
			drawGlyphBox(grid, cmd.Bounds, clip, cmd.Border, cmd.Text, cmd.Style)
		case CmdText:
			drawGlyphText(grid, cmd.Bounds, clip, cmd.Text, cmd.Align, cmd.Style.CellStyle())
		case CmdLine:
			drawGlyphLine(grid, cmd.Bounds, clip, cmd.Style.BorderCellStyle())
		case CmdGauge:
			drawGlyphGauge(grid, cmd.Bounds, clip, cmd.Fraction, cmd.ShowThumb, cmd.Text, cmd.Style)
		case CmdSparkline:
			drawGlyphSparkline(grid, cmd.Bounds, clip, cmd.Values, cmd.Histogram, cmd.Text, cmd.Style)
		}
	}
}

func putClipped(grid *ntcanvas.GraphemeGrid, clip Rect, x, y int, cluster string, style ntcanvas.CellStyle) {
	if !clip.Contains(x, y) {
		return
	}
	w := runewidth.StringWidth(cluster)
	if w == 2 && !clip.Contains(x+1, y) {
		return
	}
	grid.SetGrapheme(image.Pt(x, y), cluster, style)
}

func drawGlyphRect(grid *ntcanvas.GraphemeGrid, bounds, clip Rect, style ntcanvas.CellStyle) {
	vis := bounds.Intersect(clip)
	if vis.Empty() {
		return
	}
	grid.FillRect(vis.ImageRect(), " ", style)
}

func drawGlyphBox(grid *ntcanvas.GraphemeGrid, bounds, clip Rect, border BorderStyle, title string, style WidgetStyle) {
	cellStyle := style.CellStyle()
	drawGlyphRect(grid, bounds, clip, cellStyle)

	if border == BorderNone || bounds.W < 2 || bounds.H < 2 {
		if title != "" {
			titleRect := Rect{X: bounds.X + 1, Y: bounds.Y, W: max(0, bounds.W-2), H: 1}
			drawGlyphText(grid, titleRect, clip, title, AlignLeft, cellStyle)
		}
		return
	}

	br := borderRunes(border)
	bStyle := style.BorderCellStyle()

	x0, y0 := bounds.X, bounds.Y
	x1, y1 := bounds.Right()-1, bounds.Bottom()-1

	putClipped(grid, clip, x0, y0, string(br.tl), bStyle)
	putClipped(grid, clip, x1, y0, string(br.tr), bStyle)
	putClipped(grid, clip, x0, y1, string(br.bl), bStyle)
	putClipped(grid, clip, x1, y1, string(br.br), bStyle)

	for x := x0 + 1; x < x1; x++ {
		putClipped(grid, clip, x, y0, string(br.h), bStyle)
		putClipped(grid, clip, x, y1, string(br.h), bStyle)
	}
	for y := y0 + 1; y < y1; y++ {
		putClipped(grid, clip, x0, y, string(br.v), bStyle)
		putClipped(grid, clip, x1, y, string(br.v), bStyle)
	}

	if title != "" && bounds.W > 4 {
		titleStyle := cellStyle
		titleStyle.Attributes |= ntcanvas.Bold
		label := " " + title + " "
		titleRect := Rect{X: x0 + 1, Y: y0, W: bounds.W - 2, H: 1}
		drawGlyphText(grid, titleRect, clip, label, AlignLeft, titleStyle)
	}
}

func drawGlyphText(grid *ntcanvas.GraphemeGrid, bounds, clip Rect, text string, align Align, style ntcanvas.CellStyle) {
	vis := bounds.Intersect(clip)
	if vis.Empty() || text == "" {
		return
	}
	tw := displayWidth(text)
	startX := bounds.X
	switch align {
	case AlignCenter:
		startX = bounds.X + (bounds.W-tw)/2
	case AlignRight:
		startX = bounds.Right() - tw
	}

	y := bounds.Y
	if y < clip.Y || y >= clip.Bottom() {
		return
	}

	curX := startX
	for _, r := range text {
		rw := runewidth.RuneWidth(r)
		if rw <= 0 {
			continue
		}
		if curX >= clip.Right() || curX >= bounds.Right() {
			break
		}
		if curX >= clip.X && curX >= bounds.X && curX+rw <= clip.Right() && curX+rw <= bounds.Right() {
			cell := grid.Cell(image.Pt(curX, y))
			merged := style
			if merged.Background == 0 && cell.Style.Background != 0 {
				merged.Background = cell.Style.Background
			}
			grid.SetGrapheme(image.Pt(curX, y), string(r), merged)
		}
		curX += rw
	}
}

func drawGlyphLine(grid *ntcanvas.GraphemeGrid, bounds, clip Rect, style ntcanvas.CellStyle) {
	vis := bounds.Intersect(clip)
	if vis.Empty() {
		return
	}
	if bounds.W >= bounds.H {
		y := bounds.Y
		for x := vis.X; x < vis.Right(); x++ {
			cell := grid.Cell(image.Pt(x, y))
			st := style
			if st.Background == 0 {
				st.Background = cell.Style.Background
			}
			putClipped(grid, clip, x, y, "─", st)
		}
	} else {
		x := bounds.X
		for y := vis.Y; y < vis.Bottom(); y++ {
			cell := grid.Cell(image.Pt(x, y))
			st := style
			if st.Background == 0 {
				st.Background = cell.Style.Background
			}
			putClipped(grid, clip, x, y, "│", st)
		}
	}
}

func drawGlyphGauge(grid *ntcanvas.GraphemeGrid, bounds, clip Rect, fraction float64, showThumb bool, overlay string, style WidgetStyle) {
	vis := bounds.Intersect(clip)
	if vis.Empty() || bounds.W <= 0 {
		return
	}
	fraction = max(0, min(1, fraction))
	y := bounds.Y
	if y < clip.Y || y >= clip.Bottom() {
		return
	}

	fillFG := style.Accent
	if fillFG.IsZero() {
		fillFG = style.FG
	}
	trackBG := style.BG
	trackStyle := ntcanvas.CellStyle{
		Foreground: fillFG.CanvasColor(),
		Background: trackBG.CanvasColor(),
	}
	emptyStyle := ntcanvas.CellStyle{
		Foreground: style.Border.CanvasColor(),
		Background: trackBG.CanvasColor(),
	}

	totalEighths := int(math.Round(fraction * float64(bounds.W*8)))
	fullCells := totalEighths / 8
	remEighths := totalEighths % 8

	thumbX := bounds.X + int(math.Round(fraction*float64(max(0, bounds.W-1))))

	for i := 0; i < bounds.W; i++ {
		x := bounds.X + i
		if !clip.Contains(x, y) {
			continue
		}
		if showThumb && x == thumbX {
			thumbStyle := ntcanvas.CellStyle{
				Foreground: style.FG.CanvasColor(),
				Background: fillFG.CanvasColor(),
				Attributes: ntcanvas.Bold,
			}
			putClipped(grid, clip, x, y, "●", thumbStyle)
			continue
		}
		switch {
		case i < fullCells:
			putClipped(grid, clip, x, y, "█", trackStyle)
		case i == fullCells && remEighths > 0:
			putClipped(grid, clip, x, y, string(eighthBlocks[remEighths]), trackStyle)
		default:
			putClipped(grid, clip, x, y, "─", emptyStyle)
		}
	}

	if overlay != "" {
		tw := displayWidth(overlay)
		startX := bounds.X + (bounds.W-tw)/2
		curX := startX
		for _, r := range overlay {
			rw := runewidth.RuneWidth(r)
			if rw <= 0 {
				continue
			}
			if curX >= clip.X && curX+rw <= clip.Right() && curX >= bounds.X && curX+rw <= bounds.Right() {
				cellIdx := curX - bounds.X
				var st ntcanvas.CellStyle
				if cellIdx < fullCells {
					st = ntcanvas.CellStyle{
						Foreground: trackBG.CanvasColor(),
						Background: fillFG.CanvasColor(),
						Attributes: ntcanvas.Bold,
					}
				} else {
					st = ntcanvas.CellStyle{
						Foreground: style.FG.CanvasColor(),
						Background: trackBG.CanvasColor(),
						Attributes: ntcanvas.Bold,
					}
				}
				grid.SetGrapheme(image.Pt(curX, y), string(r), st)
			}
			curX += rw
		}
	}
}

var verticalEighths = [9]rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

func drawGlyphSparkline(grid *ntcanvas.GraphemeGrid, bounds, clip Rect, values []float64, histogram bool, overlay string, style WidgetStyle) {
	vis := bounds.Intersect(clip)
	if vis.Empty() || bounds.W <= 0 || bounds.H <= 0 || len(values) == 0 {
		return
	}
	bgStyle := style.CellStyle()
	drawGlyphRect(grid, bounds, clip, bgStyle)

	accent := style.Accent
	if accent.IsZero() {
		accent = style.FG
	}
	barStyle := ntcanvas.CellStyle{
		Foreground: accent.CanvasColor(),
		Background: style.BG.CanvasColor(),
	}

	totalSubRows := bounds.H * 8
	for col := 0; col < bounds.W; col++ {
		idx := col * len(values) / bounds.W
		if idx >= len(values) {
			idx = len(values) - 1
		}
		v := max(0, min(1, values[idx]))
		level := int(math.Round(v * float64(totalSubRows)))
		if v > 0 && level == 0 {
			level = 1
		}
		x := bounds.X + col
		if !histogram && bounds.H > 1 {
			// Line mode on multi-row chart: plot the active row cell
			rowFromBottom := min(bounds.H-1, level/8)
			rem := max(1, level%8)
			y := bounds.Bottom() - 1 - rowFromBottom
			putClipped(grid, clip, x, y, string(verticalEighths[rem]), barStyle)
			continue
		}
		for rowFromBottom := 0; rowFromBottom < bounds.H; rowFromBottom++ {
			y := bounds.Bottom() - 1 - rowFromBottom
			cellUnits := level - rowFromBottom*8
			if cellUnits <= 0 {
				continue
			}
			if cellUnits >= 8 {
				putClipped(grid, clip, x, y, "█", barStyle)
			} else {
				putClipped(grid, clip, x, y, string(verticalEighths[cellUnits]), barStyle)
			}
		}
	}

	if overlay != "" {
		textRect := Rect{X: bounds.X + 1, Y: bounds.Y, W: max(1, bounds.W-2), H: 1}
		ovStyle := ntcanvas.CellStyle{
			Foreground: style.FG.CanvasColor(),
			Background: style.BG.CanvasColor(),
			Attributes: ntcanvas.Bold,
		}
		drawGlyphText(grid, textRect, clip, overlay, AlignRight, ovStyle)
	}
}
