// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// RasterOption configures a RasterBackend.
type RasterOption func(*RasterBackend)

// WithFontFace sets the font.Face used to rasterize widget text.
func WithFontFace(face font.Face) RasterOption {
	return func(b *RasterBackend) {
		if face != nil {
			b.face = face
		}
	}
}

// RasterBackend translates a backend-agnostic DrawList into pixel-accurate
// graphics on an *image.RGBA or draw.Image buffer.
type RasterBackend struct {
	cellW, cellH int
	face         font.Face
}

// NewRasterBackend creates a RasterBackend that maps each logical terminal cell
// to (cellPixelWidth, cellPixelHeight) pixels. If either dimension is <= 0,
// defaults of 9x18 pixels are used.
func NewRasterBackend(cellPixelWidth, cellPixelHeight int, opts ...RasterOption) *RasterBackend {
	if cellPixelWidth <= 0 {
		cellPixelWidth = 9
	}
	if cellPixelHeight <= 0 {
		cellPixelHeight = 18
	}
	b := &RasterBackend{
		cellW: cellPixelWidth,
		cellH: cellPixelHeight,
		face:  basicfont.Face7x13,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	return b
}

// SetCellPixelSize updates the cell-to-pixel scale factors (e.g. from ntpicture.Picture.CellPixelSize).
func (b *RasterBackend) SetCellPixelSize(cellPixelWidth, cellPixelHeight int) {
	if cellPixelWidth > 0 {
		b.cellW = cellPixelWidth
	}
	if cellPixelHeight > 0 {
		b.cellH = cellPixelHeight
	}
}

// CellPixelSize returns the current cell-to-pixel scale factors.
func (b *RasterBackend) CellPixelSize() (cellPixelWidth, cellPixelHeight int) {
	return b.cellW, b.cellH
}

// RenderToImage allocates a new *image.RGBA for a screen of (screenCellsW, screenCellsH)
// cells and renders dl onto it.
func (b *RasterBackend) RenderToImage(screenCellsW, screenCellsH int, dl *DrawList) *image.RGBA {
	pw := max(1, screenCellsW*b.cellW)
	ph := max(1, screenCellsH*b.cellH)
	img := image.NewRGBA(image.Rect(0, 0, pw, ph))
	b.RenderOver(img, dl)
	return img
}

// RenderOver alpha-blends all commands in dl onto dst (such as an existing 3D
// frame from nt3d or ntgpuimage).
func (b *RasterBackend) RenderOver(dst draw.Image, dl *DrawList) {
	if dst == nil || dl == nil {
		return
	}
	dstBounds := dst.Bounds()
	for _, cmd := range dl.Commands() {
		pxBounds := cmd.Bounds.Scaled(b.cellW, b.cellH)
		pxClip := cmd.Clip.Scaled(b.cellW, b.cellH).Intersect(dstBounds)
		if pxClip.Empty() || pxBounds.Intersect(pxClip).Empty() {
			continue
		}
		switch cmd.Kind {
		case CmdRect:
			fillRectAlpha(dst, pxBounds.Intersect(pxClip), cmd.Style.BG)
		case CmdBox:
			b.drawRasterBox(dst, pxBounds, pxClip, cmd.Border, cmd.Text, cmd.Style)
		case CmdText:
			b.drawRasterText(dst, pxBounds, pxClip, cmd.Text, cmd.Align, cmd.Style.FG)
		case CmdLine:
			b.drawRasterLine(dst, pxBounds, pxClip, cmd.Style)
		case CmdGauge:
			b.drawRasterGauge(dst, pxBounds, pxClip, cmd.Fraction, cmd.ShowThumb, cmd.Text, cmd.Style)
		case CmdSparkline:
			b.drawRasterSparkline(dst, pxBounds, pxClip, cmd.Values, cmd.Histogram, cmd.Text, cmd.Style)
		}
	}
}

func blendPixel(dst draw.Image, x, y int, src RGBA) {
	if src.A == 0 {
		return
	}
	if rgba, ok := dst.(*image.RGBA); ok {
		if !image.Pt(x, y).In(rgba.Bounds()) {
			return
		}
		i := rgba.PixOffset(x, y)
		if src.A == 255 {
			rgba.Pix[i+0] = src.R
			rgba.Pix[i+1] = src.G
			rgba.Pix[i+2] = src.B
			rgba.Pix[i+3] = 255
			return
		}
		sa := uint32(src.A)
		invA := 255 - sa
		dr := uint32(rgba.Pix[i+0])
		dg := uint32(rgba.Pix[i+1])
		db := uint32(rgba.Pix[i+2])
		da := uint32(rgba.Pix[i+3])

		rgba.Pix[i+0] = uint8((uint32(src.R)*sa + dr*invA) / 255)
		rgba.Pix[i+1] = uint8((uint32(src.G)*sa + dg*invA) / 255)
		rgba.Pix[i+2] = uint8((uint32(src.B)*sa + db*invA) / 255)
		rgba.Pix[i+3] = uint8(sa + (da*invA)/255)
		return
	}
	draw.Draw(dst, image.Rect(x, y, x+1, y+1), &image.Uniform{C: src.StdNRGBA()}, image.Point{}, draw.Over)
}

func fillRectAlpha(dst draw.Image, r image.Rectangle, c RGBA) {
	if r.Empty() || c.A == 0 {
		return
	}
	if rgba, ok := dst.(*image.RGBA); ok {
		r = r.Intersect(rgba.Bounds())
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				blendPixel(rgba, x, y, c)
			}
		}
		return
	}
	draw.Draw(dst, r, &image.Uniform{C: c.StdNRGBA()}, image.Point{}, draw.Over)
}

func strokeRectAlpha(dst draw.Image, r image.Rectangle, clip image.Rectangle, thickness int, rounded bool, c RGBA) {
	if r.Empty() || c.A == 0 || thickness <= 0 {
		return
	}
	for t := 0; t < thickness; t++ {
		x0, y0 := r.Min.X+t, r.Min.Y+t
		x1, y1 := r.Max.X-1-t, r.Max.Y-1-t
		if x1 <= x0 || y1 <= y0 {
			break
		}
		for x := x0; x <= x1; x++ {
			if rounded && t == 0 && (x == x0 || x == x1) {
				continue
			}
			if image.Pt(x, y0).In(clip) {
				blendPixel(dst, x, y0, c)
			}
			if image.Pt(x, y1).In(clip) {
				blendPixel(dst, x, y1, c)
			}
		}
		for y := y0 + 1; y < y1; y++ {
			if image.Pt(x0, y).In(clip) {
				blendPixel(dst, x0, y, c)
			}
			if image.Pt(x1, y).In(clip) {
				blendPixel(dst, x1, y, c)
			}
		}
	}
}

func (b *RasterBackend) drawRasterBox(dst draw.Image, pxBounds, pxClip image.Rectangle, border BorderStyle, title string, style WidgetStyle) {
	fillRectAlpha(dst, pxBounds.Intersect(pxClip), style.BG)

	if title != "" {
		headerH := min(b.cellH, pxBounds.Dy())
		headerRect := image.Rect(pxBounds.Min.X, pxBounds.Min.Y, pxBounds.Max.X, pxBounds.Min.Y+headerH)
		headerBG := style.Border
		headerBG.A = max(180, style.BG.A)
		fillRectAlpha(dst, headerRect.Intersect(pxClip), headerBG)
		textRect := image.Rect(headerRect.Min.X+b.cellW, headerRect.Min.Y, headerRect.Max.X-b.cellW, headerRect.Max.Y)
		b.drawRasterText(dst, textRect, pxClip, title, AlignLeft, style.FG)
	}

	if border != BorderNone {
		borderCol := style.Border
		if borderCol.IsZero() {
			borderCol = style.FG
		}
		thickness := 1
		if border == BorderThick || border == BorderDouble {
			thickness = 2
		}
		strokeRectAlpha(dst, pxBounds, pxClip, thickness, border == BorderRounded, borderCol)
	}
}

func (b *RasterBackend) drawRasterText(dst draw.Image, pxBounds, pxClip image.Rectangle, text string, align Align, fg RGBA) {
	if text == "" || fg.A == 0 {
		return
	}
	vis := pxBounds.Intersect(pxClip)
	if vis.Empty() {
		return
	}

	const glyphW = 7
	runes := []rune(text)
	adv := len(runes) * glyphW
	metrics := b.face.Metrics()
	ascent := metrics.Ascent.Ceil()
	descent := metrics.Descent.Ceil()
	fontH := ascent + descent

	x := pxBounds.Min.X
	switch align {
	case AlignCenter:
		x = pxBounds.Min.X + (pxBounds.Dx()-adv)/2
	case AlignRight:
		x = pxBounds.Max.X - adv
	}
	topY := pxBounds.Min.Y + (pxBounds.Dy()-fontH)/2
	baseY := topY + ascent
	midY := pxBounds.Min.Y + pxBounds.Dy()/2

	clippedDst := &clippedImage{
		dst:  dst,
		clip: vis,
	}
	d := &font.Drawer{
		Dst:  clippedDst,
		Src:  image.NewUniform(color.NRGBA(fg)),
		Face: b.face,
	}

	for _, r := range runes {
		cx := x + glyphW/2
		switch r {
		case '▾': // Downward triangle
			for row := 0; row < 4; row++ {
				for dx := -(3 - row); dx <= (3 - row); dx++ {
					if image.Pt(cx+dx, midY-1+row).In(vis) {
						blendPixel(dst, cx+dx, midY-1+row, fg)
					}
				}
			}
		case '▸': // Rightward triangle
			for col := 0; col < 4; col++ {
				for dy := -(3 - col); dy <= (3 - col); dy++ {
					if image.Pt(cx-1+col, midY+dy).In(vis) {
						blendPixel(dst, cx-1+col, midY+dy, fg)
					}
				}
			}
		case '◂': // Leftward triangle
			for col := 0; col < 4; col++ {
				for dy := -(3 - col); dy <= (3 - col); dy++ {
					if image.Pt(cx+1-col, midY+dy).In(vis) {
						blendPixel(dst, cx+1-col, midY+dy, fg)
					}
				}
			}
		case '●', '◉': // Filled circle
			for dy := -3; dy <= 3; dy++ {
				for dx := -3; dx <= 3; dx++ {
					if dx*dx+dy*dy <= 10 && image.Pt(cx+dx, midY+dy).In(vis) {
						blendPixel(dst, cx+dx, midY+dy, fg)
					}
				}
			}
		case '○': // Hollow circle
			for dy := -3; dy <= 3; dy++ {
				for dx := -3; dx <= 3; dx++ {
					d2 := dx*dx + dy*dy
					if d2 >= 5 && d2 <= 11 && image.Pt(cx+dx, midY+dy).In(vis) {
						blendPixel(dst, cx+dx, midY+dy, fg)
					}
				}
			}
		case '─': // Horizontal bar
			for dx := 0; dx < glyphW; dx++ {
				if image.Pt(x+dx, midY).In(vis) {
					blendPixel(dst, x+dx, midY, fg)
				}
			}
		case '▏': // Vertical cursor bar
			for y := topY + 1; y < topY+fontH-1; y++ {
				if image.Pt(x+1, y).In(vis) {
					blendPixel(dst, x+1, y, fg)
				}
			}
		case '°': // Degree ring
			degY := topY + 3
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					d2 := dx*dx + dy*dy
					if d2 >= 2 && d2 <= 5 && image.Pt(cx+dx, degY+dy).In(vis) {
						blendPixel(dst, cx+dx, degY+dy, fg)
					}
				}
			}
		case '█': // Full block swatch
			blockRect := image.Rect(x, topY+1, x+glyphW, topY+fontH-1)
			fillRectAlpha(dst, blockRect.Intersect(vis), fg)
		default:
			d.Dot = fixed.P(x, baseY)
			d.DrawString(string(r))
		}
		x += glyphW
	}
}

func (b *RasterBackend) drawRasterLine(dst draw.Image, pxBounds, pxClip image.Rectangle, style WidgetStyle) {
	col := style.Border
	if col.IsZero() {
		col = style.FG
	}
	if pxBounds.Dx() >= pxBounds.Dy() {
		midY := pxBounds.Min.Y + pxBounds.Dy()/2
		lineRect := image.Rect(pxBounds.Min.X, midY, pxBounds.Max.X, midY+1)
		fillRectAlpha(dst, lineRect.Intersect(pxClip), col)
	} else {
		midX := pxBounds.Min.X + pxBounds.Dx()/2
		lineRect := image.Rect(midX, pxBounds.Min.Y, midX+1, pxBounds.Max.Y)
		fillRectAlpha(dst, lineRect.Intersect(pxClip), col)
	}
}

func (b *RasterBackend) drawRasterGauge(dst draw.Image, pxBounds, pxClip image.Rectangle, fraction float64, showThumb bool, overlay string, style WidgetStyle) {
	fraction = max(0, min(1, fraction))
	trackRect := pxBounds.Inset(1)
	if trackRect.Empty() {
		trackRect = pxBounds
	}
	fillRectAlpha(dst, trackRect.Intersect(pxClip), style.BG)

	fillCol := style.Accent
	if fillCol.IsZero() {
		fillCol = style.FG
	}
	fillW := int(math.Round(fraction * float64(trackRect.Dx())))
	if fillW > 0 {
		fillRect := image.Rect(trackRect.Min.X, trackRect.Min.Y, trackRect.Min.X+fillW, trackRect.Max.Y)
		fillRectAlpha(dst, fillRect.Intersect(pxClip), fillCol)
	}

	borderCol := style.Border
	if borderCol.IsZero() {
		borderCol = style.FG
	}
	strokeRectAlpha(dst, trackRect, pxClip, 1, true, borderCol)

	if showThumb {
		thumbW := max(4, b.cellW/2)
		cx := trackRect.Min.X + int(math.Round(fraction*float64(max(0, trackRect.Dx()-1))))
		thumbRect := image.Rect(cx-thumbW/2, pxBounds.Min.Y+1, cx+thumbW/2+1, pxBounds.Max.Y-1)
		fillRectAlpha(dst, thumbRect.Intersect(pxClip), style.FG)
	}

	if overlay != "" {
		b.drawRasterText(dst, pxBounds, pxClip, overlay, AlignCenter, style.FG)
	}
}

func (b *RasterBackend) drawRasterSparkline(dst draw.Image, pxBounds, pxClip image.Rectangle, values []float64, histogram bool, overlay string, style WidgetStyle) {
	if len(values) == 0 {
		return
	}
	plotRect := pxBounds.Inset(1)
	if plotRect.Empty() {
		plotRect = pxBounds
	}
	fillRectAlpha(dst, plotRect.Intersect(pxClip), style.BG)

	accent := style.Accent
	if accent.IsZero() {
		accent = style.FG
	}
	areaFill := accent
	areaFill.A = 75

	w := plotRect.Dx()
	h := plotRect.Dy()
	if w > 0 && h > 0 {
		if histogram {
			n := len(values)
			for i, v := range values {
				v = max(0, min(1, v))
				x0 := plotRect.Min.X + (i*w)/n
				x1 := plotRect.Min.X + ((i+1)*w)/n
				if x1 > x0+1 {
					x1--
				}
				barH := int(math.Round(v * float64(h-2)))
				if v > 0 && barH == 0 {
					barH = 1
				}
				barRect := image.Rect(x0, plotRect.Max.Y-1-barH, max(x0+1, x1), plotRect.Max.Y-1)
				fillRectAlpha(dst, barRect.Intersect(pxClip), accent)
			}
		} else {
			prevX, prevY := 0, 0
			for px := 0; px < w; px++ {
				idx := px * len(values) / w
				if idx >= len(values) {
					idx = len(values) - 1
				}
				v := max(0, min(1, values[idx]))
				x := plotRect.Min.X + px
				y := plotRect.Max.Y - 2 - int(math.Round(v*float64(max(1, h-4))))
				fillColRect := image.Rect(x, y, x+1, plotRect.Max.Y-1)
				fillRectAlpha(dst, fillColRect.Intersect(pxClip), areaFill)
				if px > 0 {
					yMin, yMax := min(prevY, y), max(prevY, y)
					lineSeg := image.Rect(x, yMin, x+1, yMax+1)
					fillRectAlpha(dst, lineSeg.Intersect(pxClip), accent)
				} else {
					lineSeg := image.Rect(x, y, x+1, y+1)
					fillRectAlpha(dst, lineSeg.Intersect(pxClip), accent)
				}
				prevX, prevY = x, y
				_ = prevX
			}
		}
	}

	borderCol := style.Border
	if borderCol.IsZero() {
		borderCol = style.FG
	}
	strokeRectAlpha(dst, plotRect, pxClip, 1, true, borderCol)

	if overlay != "" {
		headerRect := image.Rect(plotRect.Min.X+4, plotRect.Min.Y+1, plotRect.Max.X-4, min(plotRect.Max.Y, plotRect.Min.Y+b.cellH))
		b.drawRasterText(dst, headerRect, pxClip, overlay, AlignRight, style.FG)
	}
}

type clippedImage struct {
	dst  draw.Image
	clip image.Rectangle
}

func (c *clippedImage) ColorModel() color.Model { return c.dst.ColorModel() }
func (c *clippedImage) Bounds() image.Rectangle { return c.clip }
func (c *clippedImage) At(x, y int) color.Color { return c.dst.At(x, y) }
func (c *clippedImage) Set(x, y int, col color.Color) {
	if image.Pt(x, y).In(c.clip) {
		c.dst.Set(x, y, col)
	}
}
