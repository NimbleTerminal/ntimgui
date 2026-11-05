# `ntimgui` Design Prompt & Technical Specification

> **Origin**: Synthesized from *Terminal IMGUI in Go* (`nimbleterminal.dev/ntimgui`) for the NimbleTerminal ecosystem (`ntcanvas`, `ntpicture`, `ntgpuimage`, `nt3d`, and Bubble Tea v2).

---

## Part 1: Design Prompt

### Background & Context
We are designing **`ntimgui`** (`nimbleterminal.dev/ntimgui`), a lightweight, pure-Go immediate-mode GUI (IMGUI) library inspired by Dear ImGui and designed to layer seamlessly over the NimbleTerminal ecosystem:
- **`charm.land/bubbletea/v2`**: Event-driven terminal application lifecycle (`Update` / `View`).
- **`nimbleterminal.dev/ntcanvas`**: Terminal cell grids (`RuneGrid`, `StyledRuneGrid`, `GraphemeGrid`) and viewport models.
- **`nimbleterminal.dev/ntpicture` & `nimbleterminal.dev/ntgpuimage`**: Pixel/raster terminal graphics (Kitty graphics protocol, shared memory transport, glyph fallback, and offscreen WebGPU readback).
- **`nimbleterminal.dev/nt3d`**: 3D scene engine and Bubble Tea widget where `ntimgui` provides HUD overlays, control panels, and tool palettes without the overhead of managing a retained widget object tree.

### Hard Constraints
1. **Pure Go (Zero CGo)**: Must compile cleanly for native targets and browser WebAssembly (`GOOS=js GOARCH=wasm`).
2. **Backend-Agnostic Core**: The IMGUI core performs only input state resolution, layout math, and abstract Draw List generation. It must not depend directly on terminal escape codes or pixel buffers.
3. **Dual Render Backends**:
   - **Glyph-Oriented Backend**: Renders the Draw List into `ntcanvas` grids or ANSI/Unicode strings (box-drawing characters, fractional block elements, cell styles).
   - **Raster-Oriented Backend**: Renders the Draw List into `*image.RGBA` / `draw.Image` pixel buffers (for compositing over `nt3d` frames or presenting via `ntpicture`).

### Core Architectural Requirements
1. **Immediate-Mode Philosophy**: UI is a direct side-effect of execution flow. Widget calls (`if ui.Button("Rotate") { ... }`) resolve interaction and emit draw commands in a single pass, returning interaction results immediately.
2. **Bridging Continuous IMGUI with Event-Driven Bubble Tea (`Update` vs. `View`)**:
   - During `Update(msg tea.Msg)`: Feed terminal events via `ui.ProcessInput(msg)`, then run `ui.BeginFrame()`, widget declarations, and `ui.EndFrame()` to update `Hot`/`Active`/`Focus` state and build the frame's **Draw List**.
   - During `View()`: Flush the cached **Draw List** through either the Glyph Backend (`ntcanvas`) or the Raster Backend (`*image.RGBA` / `ntpicture`).
   - Expose `WantCaptureMouse()` and `WantCaptureKeyboard()` so host models (like `nt3dwidget`) know when input was consumed by a UI overlay versus when it should pass through to the underlying 3D scene.
3. **Core State & Identity**:
   - Deterministic 64-bit FNV-1a ID hashing of widget labels with Dear ImGui-style `Display Label##unique_id` suffix stripping and an ID stack (`PushID` / `PopID`).
   - Track global `Hot` (hovered), `Active` (clicked/dragged), and `Focus` (keyboard target) widget IDs across frames, plus a lightweight internal `Storage` map keyed by `ID` for transient widget state (e.g., panel collapse, scroll offsets, drag offsets).
4. **Sequential Layout Engine**:
   - Virtual cursor with a stack of layout scopes (`LayoutScope`).
   - Default vertical flow (`FlowVertical`) with `BeginHorizontal()` / `EndHorizontal()` and `SameLine()` grouping, `BeginPanel()` / `EndPanel()` floating containers, padding, spacing, indentation, and clip rectangles.
5. **Dual Coordinate & Rendering Architecture**:
   - Logical coordinates are expressed in **terminal cell units** `(Col, Row)`, matching Bubble Tea mouse event coordinates directly.
   - The Glyph Backend maps `1 unit = 1 terminal cell`.
   - The Raster Backend scales logical cell units by `(CellPixelWidth, CellPixelHeight)` (e.g. `9x18` pixels from `ntpicture`) while rendering sub-cell pixel-accurate shapes, alpha-blended translucent panels, continuous gauge/slider bars, and `golang.org/x/image/font` typography.

---

## Part 2: Technical Design Specification

### 1. Module & Package Architecture

- **Module Path**: `nimbleterminal.dev/ntimgui`
- **Package Name**: `ntimgui`
- **Go Version**: Go 1.26+ (`CGO_ENABLED=0` compatible, native + `js/wasm`)

| File | Responsibility |
| --- | --- |
| `doc.go` | Package documentation and usage overview |
| `id.go` | `ID` type (`uint64`), FNV-1a hashing, `ParseLabel` (`label##id`), ID stack, and transient per-ID `Storage` |
| `geom.go` | `Point`, `Rect`, alignment, clamping, intersection, and centering helpers |
| `input.go` | `InputState`, `MouseButton`, `KeyCode`, and Bubble Tea v2 `ProcessInput(tea.Msg)` adapter |
| `theme.go` | `RGBA` color, `BorderStyle`, `WidgetStyle`, `Theme` presets (`DefaultTheme`, `HUDTheme`, `HighContrastTheme`) |
| `drawlist.go` | `DrawList`, `Layer` (`LayerBackground`, `LayerWindow`, `LayerOverlay`), `DrawCmd`, `DrawCmdKind`, clip stack |
| `layout.go` | `FlowDir`, `LayoutScope`, cursor positioning, `AllocateRect`, `BeginHorizontal`/`EndHorizontal`, `SameLine` |
| `context.go` | `Context`, `Options`, `BeginFrame`, `EndFrame`, `WantCaptureMouse`, `WantCaptureKeyboard`, focus cycling |
| `widgets.go` | `Label`, `Text`, `Button`, `ButtonSized`, `Checkbox`, `Toggle`, `RadioButton`, `SliderFloat`, `SliderInt`, `ProgressBar`, `CollapsingHeader`, `TextInput`, `Separator`, `Tooltip` |
| `panel.go` | `BeginPanel`, `EndPanel`, draggable/collapsible/scrollable HUD windows, hit-region registration |
| `backend_glyph.go` | `GlyphBackend`: renders `DrawList` to `ntcanvas.GraphemeGrid` / `RuneGrid` / `StyledRuneGrid` or overlays onto existing terminal strings |
| `backend_raster.go` | `RasterBackend`: renders `DrawList` to `*image.RGBA` or alpha-blends over existing `draw.Image` frames using `golang.org/x/image/font` |

---

### 2. Core Types & State Model

#### 2.1 Widget Identity (`ID`)
Widgets are stateless in caller code and identified by a 64-bit hash:
```go
type ID uint64

// ParseLabel splits "Visible Text##HiddenID" into display text ("Visible Text")
// and the hash key ("Visible Text##HiddenID"). A "###ExplicitID" suffix hashes
// only the suffix while displaying the prefix.
func ParseLabel(label string) (display, hashKey string)
```
The `Context` maintains an `idStack []ID` seeded with the FNV-1a offset basis. `PushID(str)` / `PushIDInt(n)` pushes a child seed; `PopID()` restores the parent seed.

#### 2.2 Interaction State (`Hot`, `Active`, `Focus`)
- **`Hot` (`ID`)**: Recomputed every frame. Set when the mouse cursor is inside a widget's clipped hit rectangle (and no higher-Z panel occludes it) and either no widget is `Active` or this widget is `Active`.
- **`Active` (`ID`)**: Persists across frames while the mouse button is held down (e.g. holding a button or dragging a slider or panel title bar). Cleared when the mouse button is released.
- **`Focus` (`ID`)**: Persists across frames for keyboard accessibility. `Tab` / `Shift+Tab` cycles focus across focusable widgets registered during the frame. Focused widgets respond to `Space`/`Enter` (buttons, checkboxes, headers), `Left`/`Right` (sliders, text inputs), and character entry (`TextInput`).

#### 2.3 Bubble Tea v2 Lifecycle & Input Capture
```go
func (m *appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    m.ui.ProcessInput(msg)

    m.ui.BeginFrame()
    if m.ui.BeginPanel("3D Controls", ntimgui.Rect{X: 2, Y: 1, W: 28, H: 12}) {
        if m.ui.Button("Toggle Wireframe") {
            m.engine.ToggleWireframe()
        }
        m.ui.SliderFloat("FOV", &m.fov, 30, 120, "%.0f°")
        m.ui.Checkbox("Shadows", &m.shadows)
    }
    m.ui.EndPanel()
    m.ui.EndFrame()

    // Only forward mouse events to the 3D viewport if ntimgui didn't capture them
    if _, isMouse := msg.(tea.MouseMsg); !isMouse || !m.ui.WantCaptureMouse() {
        m.viewport, cmd = m.viewport.Update(msg)
    }
    return m, cmd
}
```

---

### 3. Layout Engine

Each `LayoutScope` on the layout stack tracks:
- `Bounds Rect`: Content region available to this scope.
- `Cursor Point`: Next widget allocation position `(X, Y)`.
- `Flow FlowDir`: `FlowVertical` (default) or `FlowHorizontal`.
- `Spacing Point`: Horizontal and vertical gap between sequential widgets.
- `Indent int`: Left margin offset inside vertical flows.
- `Max Point`: Maximum extent reached by children in this scope.
- `LastRect Rect`: Bounding box of the most recently allocated widget (used by `SameLine()` and `Tooltip()`).

`AllocateRect(w, h int) Rect` places a widget at `Cursor`, updates `LastRect` and `Max`, and advances `Cursor`:
- In `FlowVertical`: `Cursor.Y += h + Spacing.Y`
- In `FlowHorizontal`: `Cursor.X += w + Spacing.X`

---

### 4. Abstract Draw List

To decouple widget logic from rendering, widgets emit commands into layered buckets (`LayerBackground`, `LayerWindow`, `LayerOverlay`) so floating panels and tooltips always draw above background widgets:

```go
type DrawCmdKind uint8

const (
    CmdRect  DrawCmdKind = iota // Filled rectangle
    CmdBox                      // Bordered box with optional fill and title
    CmdText                     // Positioned/aligned text
    CmdLine                     // Horizontal or vertical line
    CmdGauge                    // Fractional [0..1] progress/slider bar with optional thumb
)

type DrawCmd struct {
    Kind      DrawCmdKind
    Layer     Layer
    Bounds    Rect
    Clip      Rect
    Text      string
    Align     Align
    Style     WidgetStyle
    Border    BorderStyle
    Fraction  float64
    ShowThumb bool
}
```

---

### 5. Render Backends

#### 5.1 Glyph-Oriented Backend (`GlyphBackend`)
- Targets `ntcanvas.GraphemeGrid` (or `RuneGrid` / `StyledRuneGrid`) using `nimbleterminal.dev/ntcanvas`.
- **Coordinates**: 1 logical unit = 1 terminal cell `(Col, Row)`.
- **Primitives**:
  - `CmdRect`: Fills clipped cells with spaces and `ntcanvas.CellStyle` background/foreground colors.
  - `CmdBox`: Draws Unicode border runes (`┌─┐│└─┘`, `╭─╮│╰─╯`, `╔═╗║╚═╝`, `┏━┓┃┗━┛`) around `Bounds`, fills interior, and renders optional title in the top border.
  - `CmdText`: Measures display width (grapheme-aware), aligns (`AlignLeft`, `AlignCenter`, `AlignRight`), clips to `Clip`, and writes to the grid.
  - `CmdLine`: Draws horizontal (`─`) or vertical (`│`) rules.
  - `CmdGauge`: Renders smooth fractional progress using Unicode eighth-blocks (`█▉▊▋▌▍▎▏`) or track/thumb glyphs (`━●─`).
- **Compositing**: Provides `RenderToGrid(grid, dl)`, `RenderToString(dl)`, and `OverlayString(baseView, dl)` for zero-friction Bubble Tea `View()` integration.

#### 5.2 Raster-Oriented Backend (`RasterBackend`)
- Targets `*image.RGBA` or any `draw.Image` (compatible with `ntpicture`, `ntgpuimage`, and `nt3d` framebuffers).
- **Coordinates**: Scales logical cell coordinates `(X, Y, W, H)` by `(CellPixelWidth, CellPixelHeight)` (default `9x18`, configurable to match `ntpicture.Picture.CellPixelSize()`).
- **Primitives**:
  - `CmdRect` & `CmdBox`: Alpha-blended Porter-Duff source-over fill (supporting translucent HUD panels over 3D scenes), crisp 1px/2px borders, and title bar highlights.
  - `CmdText`: Renders glyphs using `golang.org/x/image/font` (`basicfont.Face7x13` default, or custom `font.Face`) clipped to the pixel clip rectangle.
  - `CmdLine`: Pixel-accurate horizontal/vertical lines.
  - `CmdGauge`: Pixel-accurate track bar, continuous fractional fill (`int(fraction * pixelWidth)`), and raised thumb handle.
