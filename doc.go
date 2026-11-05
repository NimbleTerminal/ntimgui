// Copyright (c) 2026 Neomantra Corp.

// Package ntimgui implements a pure-Go immediate-mode GUI (IMGUI) library for the
// NimbleTerminal ecosystem (Bubble Tea v2, ntcanvas, ntpicture, ntgpuimage, and nt3d).
//
// In immediate mode, user interfaces are declared inline during execution flow
// without retaining a widget object tree. During Bubble Tea's Update phase,
// callers feed terminal events via [Context.ProcessInput], open a frame with
// [Context.BeginFrame], invoke widget functions such as [Context.Button] or
// [Context.SliderFloat], and close the frame with [Context.EndFrame].
//
// Widget evaluation resolves interaction state (Hot, Active, and Focus) and
// populates a backend-agnostic [DrawList]. During Bubble Tea's View phase (or
// frame compositing), the cached DrawList is rendered through either:
//   - [GlyphBackend]: renders into [ntcanvas] terminal cell grids or overlays
//     onto existing ANSI/Unicode terminal strings.
//   - [RasterBackend]: renders pixel-accurate boxes, gauges, and typography
//     into [*image.RGBA] buffers or alpha-blends HUD overlays directly onto
//     3D/GPU frames.
package ntimgui
