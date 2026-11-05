// Copyright (c) 2026 Neomantra Corp.

// Package ntimguiebiten renders ntimgui command snapshots into caller-owned Ebitengine
// images. It is an optional subpackage: importing ntimgui alone does not initialize
// Ebitengine or require a graphics environment.
//
// Construct New and call Backend.Render in an Ebitengine callback. Render accepts
// DrawList.Snapshot output in layer order; it never sorts, mutates, or retains it.
// Publish snapshots through synchronization, such as atomic.Value, together with
// the scene state they describe. Context, input, and widget-bound values stay on
// their owner event loop. All backend methods and font/GPU resources belong to
// the Ebitengine callback goroutine, including construction and Dispose.
//
// RenderOptions defaults to 9x18 pixels per cell. Fractional scales and offsets
// are supported: each absolute edge is rounded to the nearest pixel, so adjacent
// cells share boundaries without accumulated rounding. Offsets are relative to
// dst.Bounds().Min, including subimages. Output is clipped to each command's Clip
// and destination bounds; text is additionally clipped to its original Bounds.
// Clipping never recenters text, changes gauge fractions, or resamples plots.
//
// Colors are straight-alpha ntimgui.RGBA, converted through color.NRGBA for vector
// primitives and text ColorScale. The default font is bundled Go Mono regular
// and bold, automatically sized for the cell scale. WithFontFaces accepts custom
// horizontal faces at a fixed pixel size. Alignment uses measured text advances,
// unlike RasterBackend's fixed seven-pixel advances. Widget arrows, circles,
// blocks, and checkmarks use vector fallbacks with one measured M advance. Other
// Unicode coverage depends on the font. Text is single-line; content after the
// first newline is ignored. Very low raster resolutions (e.g. glyph presentation
// at 1x2 pixels per cell) cannot produce readable typography: use GlyphBackend
// in a separate terminal pane, or a larger raster resolution.
//
// Fills, titles, borders, gauges, and sampled plots follow RasterBackend. Double
// and Thick borders are both two pixels; Rounded cuts the outer corner pixel.
// Degenerate plots are bounded safely. Unlike RasterBackend, Bold selects a bold
// face, Dim reduces foreground alpha to two thirds, Underline draws a one-pixel
// underline, and Reverse swaps foreground/background colors before rendering.
// CmdText still has no implicit background fill. Vector symbols and font edges
// are antialiased; rectangles and sampled plots are pixel-snapped.
//
// Backend owns no GPU images. text/v2 and vector manage their reusable engine
// caches. Dispose drops font references and disables rendering; it never closes
// caller-provided faces or disposes the destination. The package does not run or
// stop an Ebitengine loop, process input, or perform GPU readback. In particular,
// never pass an ebiten.Image to RasterBackend.RenderOver: its per-pixel generic
// image path would read GPU pixels. ntebitengine users render the scene and this
// overlay into the same DrawFunc destination, leaving the single final readback
// and presentation pacing to that adapter.
package ntimguiebiten
