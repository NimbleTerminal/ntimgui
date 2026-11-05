# Examples

Run these from the repository root. Each browser demo publishes its keys through
[booba](https://github.com/NimbleTerminal/booba) so the same actions appear
as buttons above the terminal.

| Command | What it shows | Controls |
| --- | --- | --- |
| `task run-catalog` | Four-tab UX element catalog (Controls, Plots, Trees & Lists, Themes) + live IMGUI State Inspector | `]`/`[`: switch tab, `t`: cycle theme, `b`: cycle border, `g`: Glyph/Raster, `Tab`: next widget |
| `task run-dashboard` | Live GPU frame-time sparklines, pass breakdown histograms, and simulation controls | `Space`: pause/resume, `t`: cycle theme, `g`: Glyph/Raster, `Tab`: next widget |
| `task run-hud` | Interactive 3D wireframe viewport with draggable floating HUD panels and mouse/keyboard capture | `Space`: pause/orbit, `m`: cycle mesh, `+`/`-` or wheel: zoom, `r`: reset camera, `g`: Glyph/Raster |

Use `q` or `Ctrl+C` to quit the terminal demos. `task serve-wasm-site` builds the WASM demos and
serves all six interactive presets at `http://localhost:8000/web/`.

The `examples` module owns `booba` and `ntpicture`. Browser builds select
`wasm.work` and its Bubble Tea replacement so no demo-only dependencies are
added to `ntimgui`'s root module.

## Standalone Ebitengine window

Run `task run-ebiten` (or `go -C examples run ./ebiten`) from the repository root.
This opens a regular resizable desktop window with animated circles and a
floating ntimgui panel. Drag its title, adjust speed, toggle pause/light theme, or
click outside the panel to pause. Tab/Shift+Tab, arrows, Enter, and Space operate
widgets; Escape closes the window.

The [single source file](ebiten/main.go) maps mouse pixels to a fixed 80×30 cell
grid, declares widgets in Update, and renders a command snapshot in Draw. Window
resizing changes the fractional cell-to-pixel scale. Ebitengine serializes these
callbacks, so no atomic handoff is needed. The backend is created lazily in Draw
and reused; Escape disposes it on the callback side. Closing the native window
ends the engine and releases its process-lifetime caches.

This example uses no terminal adapter, ntebitengine, or GPU readback. It needs a
working desktop graphics environment and is separate from the terminal browser
gallery. Bundled fonts use [the Go font license](../ebiten/FONT-LICENSE.txt).
