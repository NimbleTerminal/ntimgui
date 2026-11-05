// Copyright (c) 2026 Neomantra Corp.

package ntimguiebiten

import (
	"bytes"
	"image"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"nimbleterminal.dev/ntimgui"
	"nimbleterminal.dev/ntimgui/ebiten/internal/geometry"
)

// Option configures a Backend.
type Option func(*Backend)

// WithFontFaces supplies horizontal text/v2 faces at the caller's desired pixel
// size. A nil regular face keeps the automatic bundled Go Mono font. A nil bold
// face uses regular for bold text. Faces remain caller-owned; do not mutate them
// concurrently with rendering. Custom faces are not resized with the cell scale.
func WithFontFaces(regular, bold text.Face) Option {
	return func(b *Backend) {
		if regular != nil {
			b.regular, b.bold = regular, bold
		}
	}
}

// RenderOptions maps logical cells to destination pixels. Zero scales default to
// 9x18. Other scales must be finite and positive. Offsets are pixels relative to
// dst.Bounds().Min, including when dst is a subimage. Invalid options draw nothing.
// Hit testing remains in logical cells and is the caller's responsibility.
type RenderOptions struct{ ScaleX, ScaleY, OffsetX, OffsetY float64 }

// Backend renders snapshots directly on the GPU. All methods, including New
// and Dispose, belong to the Ebitengine callback goroutine. Do not copy a Backend.
// It never retains a destination or command slice and never reads pixels.
type Backend struct {
	regular, bold         text.Face
	autoRegular, autoBold *text.GoTextFace
	disposed              bool
}

// New constructs a reusable renderer. The default Go Mono regular/bold faces
// are bundled in x/image (no runtime downloads) and sized to fit the cell scale.
// text/v2 owns and reuses its glyph GPU cache; the backend owns no GPU images.
func New(opts ...Option) *Backend {
	b := &Backend{}
	for _, opt := range opts {
		if opt != nil {
			opt(b)
		}
	}
	if b.regular == nil {
		source, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
		if err != nil {
			panic(err)
		}
		bold, err := text.NewGoTextFaceSource(bytes.NewReader(gomonobold.TTF))
		if err != nil {
			panic(err)
		}
		b.autoRegular = &text.GoTextFace{Source: source}
		b.autoBold = &text.GoTextFace{Source: bold}
		b.regular, b.bold = b.autoRegular, b.autoBold
	}
	return b
}

// Dispose releases backend references, including its font sources, and makes
// future Render calls no-ops. It is idempotent. It does not dispose caller fonts,
// destinations, or text/v2's engine-managed glyph cache. Call on the callback side.
func (b *Backend) Dispose() {
	b.regular, b.bold, b.autoRegular, b.autoBold = nil, nil, nil, nil
	b.disposed = true
}

// Render alpha-composites commands in their supplied order without clearing dst.
// Use DrawList.Snapshot or ntimgui.SnapshotCommands before crossing goroutines;
// callers must not mutate commands or referenced Values during rendering.
func (b *Backend) Render(dst *ebiten.Image, commands []ntimgui.DrawCmd, opts RenderOptions) {
	if b.disposed || dst == nil {
		return
	}
	if opts.ScaleX == 0 {
		opts.ScaleX = 9
	}
	if opts.ScaleY == 0 {
		opts.ScaleY = 18
	}
	tr := geometry.Transform{ScaleX: opts.ScaleX, ScaleY: opts.ScaleY, OffsetX: opts.OffsetX, OffsetY: opts.OffsetY}
	if !tr.Valid() {
		return
	}
	if b.autoRegular != nil {
		size := math.Min(opts.ScaleY*.78, opts.ScaleX/.6)
		b.autoRegular.Size, b.autoBold.Size = size, size
	}
	origin := dst.Bounds().Min
	for _, cmd := range commands {
		r := tr.Rect(cmd.Bounds, origin)
		clip := tr.Rect(cmd.Clip, origin).Intersect(dst.Bounds())
		if r.Empty() || clip.Empty() || r.Intersect(clip).Empty() {
			continue
		}
		// SubImage clips output in destination coordinates; layout still uses r.
		target := dst.SubImage(clip).(*ebiten.Image)
		style := cmd.Style
		if style.Reverse {
			style.FG, style.BG = style.BG, style.FG
		}
		if style.Dim {
			style.FG.A = uint8(uint16(style.FG.A) * 2 / 3)
		}
		switch cmd.Kind {
		case ntimgui.CmdRect:
			fill(target, r, style.BG)
		case ntimgui.CmdBox:
			b.box(target, r, cmd, style, opts)
		case ntimgui.CmdText:
			b.drawText(target, r, cmd.Text, cmd.Align, style)
		case ntimgui.CmdLine:
			col := fallback(style.Border, style.FG)
			if r.Dx() >= r.Dy() {
				fill(target, rect(r.Min.X, r.Min.Y+r.Dy()/2, r.Max.X, r.Min.Y+r.Dy()/2+1), col)
			} else {
				fill(target, rect(r.Min.X+r.Dx()/2, r.Min.Y, r.Min.X+r.Dx()/2+1, r.Max.Y), col)
			}
		case ntimgui.CmdGauge:
			b.gauge(target, r, cmd, style, opts)
		case ntimgui.CmdSparkline:
			b.plot(target, r, cmd, style, opts)
		}
	}
}

func fallback(c, other ntimgui.RGBA) ntimgui.RGBA {
	if c.IsZero() {
		return other
	}
	return c
}
func unit(v float64) float64 {
	if math.IsNaN(v) {
		return 0
	}
	return math.Max(0, math.Min(1, v))
}
func fill(dst *ebiten.Image, r image.Rectangle, c ntimgui.RGBA) {
	if r.Empty() || c.A == 0 {
		return
	}
	vector.FillRect(dst, float32(r.Min.X), float32(r.Min.Y), float32(r.Dx()), float32(r.Dy()), c.StdNRGBA(), false)
}

// Disjoint strips avoid double-blending translucent corners. Rounded follows
// RasterBackend's one-pixel corner cut; Double and Thick both use two pixels.
func border(dst *ebiten.Image, r image.Rectangle, width int, rounded bool, c ntimgui.RGBA) {
	for n := 0; n < width; n++ {
		q := r.Inset(n)
		if q.Dx() < 2 || q.Dy() < 2 {
			break
		}
		cut := 0
		if rounded && n == 0 {
			cut = 1
		}
		fill(dst, rect(q.Min.X+cut, q.Min.Y, q.Max.X-cut, q.Min.Y+1), c)
		fill(dst, rect(q.Min.X+cut, q.Max.Y-1, q.Max.X-cut, q.Max.Y), c)
		fill(dst, rect(q.Min.X, q.Min.Y+1, q.Min.X+1, q.Max.Y-1), c)
		fill(dst, rect(q.Max.X-1, q.Min.Y+1, q.Max.X, q.Max.Y-1), c)
	}
}
func inset(r image.Rectangle) image.Rectangle {
	q := r.Inset(1)
	if q.Empty() {
		return r
	}
	return q
}
func (b *Backend) box(dst *ebiten.Image, r image.Rectangle, cmd ntimgui.DrawCmd, s ntimgui.WidgetStyle, o RenderOptions) {
	fill(dst, r, s.BG)
	if cmd.Text != "" {
		header := r
		header.Max.Y = min(r.Max.Y, r.Min.Y+max(1, int(math.Round(o.ScaleY))))
		col := s.Border
		col.A = max(180, s.BG.A)
		fill(dst, header, col)
		pad := int(math.Round(o.ScaleX))
		header.Min.X += pad
		header.Max.X -= pad
		b.drawText(dst, header, cmd.Text, ntimgui.AlignLeft, s)
	}
	if cmd.Border != ntimgui.BorderNone {
		w := 1
		if cmd.Border == ntimgui.BorderThick || cmd.Border == ntimgui.BorderDouble {
			w = 2
		}
		border(dst, r, w, cmd.Border == ntimgui.BorderRounded, fallback(s.Border, s.FG))
	}
}
func (b *Backend) gauge(dst *ebiten.Image, r image.Rectangle, cmd ntimgui.DrawCmd, s ntimgui.WidgetStyle, o RenderOptions) {
	track := inset(r)
	fraction := unit(cmd.Fraction)
	fill(dst, track, s.BG)
	bar := track
	bar.Max.X = bar.Min.X + int(math.Round(fraction*float64(track.Dx())))
	fill(dst, bar, fallback(s.Accent, s.FG))
	border(dst, track, 1, true, fallback(s.Border, s.FG))
	if cmd.ShowThumb {
		w := max(4, int(math.Round(o.ScaleX/2)))
		x := track.Min.X + int(math.Round(fraction*float64(max(0, track.Dx()-1))))
		fill(dst, rect(x-w/2, r.Min.Y+1, x+w/2+1, r.Max.Y-1), s.FG)
	}
	b.drawText(dst, r, cmd.Text, ntimgui.AlignCenter, s)
}
func (b *Backend) plot(dst *ebiten.Image, r image.Rectangle, cmd ntimgui.DrawCmd, s ntimgui.WidgetStyle, o RenderOptions) {
	if len(cmd.Values) == 0 {
		return
	}
	p := inset(r)
	fill(dst, p, s.BG)
	accent := fallback(s.Accent, s.FG)
	area := accent
	area.A = 75
	w, h := p.Dx(), p.Dy()
	if cmd.Histogram {
		for i, value := range cmd.Values {
			v := unit(value)
			x0 := p.Min.X + i*w/len(cmd.Values)
			x1 := p.Min.X + (i+1)*w/len(cmd.Values)
			if x1 > x0+1 {
				x1--
			}
			height := int(math.Round(v * float64(max(0, h-2))))
			if v > 0 && height == 0 {
				height = 1
			}
			fill(dst, rect(x0, p.Max.Y-1-height, max(x0+1, x1), p.Max.Y-1).Intersect(p), accent)
		}
	} else {
		// Match raster's nearest-sample staircase and translucent area. Restrict the
		// iteration to visible columns without changing sample indices or joins.
		visible := p.Intersect(dst.Bounds())
		pointY := func(x int) int {
			v := unit(cmd.Values[min(len(cmd.Values)-1, (x-p.Min.X)*len(cmd.Values)/w)])
			return max(p.Min.Y, p.Max.Y-2-int(math.Round(v*float64(max(1, h-4)))))
		}
		for x := visible.Min.X; x < visible.Max.X; x++ {
			y := pointY(x)
			prev := y
			if x > p.Min.X {
				prev = pointY(x - 1)
			}
			fill(dst, rect(x, y, x+1, p.Max.Y-1), area)
			fill(dst, rect(x, min(prev, y), x+1, max(prev, y)+1).Intersect(p), accent)
		}
	}
	border(dst, p, 1, true, fallback(s.Border, s.FG))
	if cmd.Text != "" {
		header := rect(p.Min.X+4, p.Min.Y+1, p.Max.X-4, min(p.Max.Y, p.Min.Y+int(math.Round(o.ScaleY))))
		b.drawText(dst, header, cmd.Text, ntimgui.AlignRight, s)
	}
}

func rect(x0, y0, x1, y1 int) image.Rectangle {
	return image.Rectangle{Min: image.Pt(x0, y0), Max: image.Pt(x1, y1)}
}
