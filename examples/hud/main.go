// Copyright (c) 2026 Neomantra Corp.

package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"nimbleterminal.dev/ntcanvas"
	"nimbleterminal.dev/ntimgui"
	"nimbleterminal.dev/ntimgui/examples/internal/demohelp"
	picture "nimbleterminal.dev/ntpicture"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return demohelp.Tick(time.Second/30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type vec3 struct{ x, y, z float64 }
type edge3 struct {
	a, b vec3
	col  ntimgui.RGBA
}

type demoModel struct {
	session *picture.TerminalSession
	ui      *ntimgui.Context
	glyph   *ntimgui.GlyphBackend
	raster  *ntimgui.RasterBackend
	pic     *picture.Picture

	width, height int

	// 3D scene & camera state
	yaw, pitch  float64
	fov         float64
	zoom        float64
	speed       float64
	density     int
	shape       int // 0=Torus, 1=Sphere, 2=Cube
	backendMode int // 0=Glyph (ntcanvas), 1=Raster (ntpicture)
	autoRotate  bool
	showAxes    bool
	reverseSpin bool
	objectTag   string
	pulses      int

	// Kitty / ntpicture frame pacing state (at most one frame in flight at a time)
	dirty     bool
	awaiting  bool
	submitted uint64
	geometry  picture.RasterGeometry

	// Mouse orbit state (only active when !ui.WantCaptureMouse())
	orbiting       bool
	lastMX, lastMY int
}

func newDemoModel() *demoModel {
	session := picture.NewTerminalSession(picture.TerminalOptions{
		CellPixelWidth:  9,
		CellPixelHeight: 18,
	})
	pic := picture.New(picture.Options{
		Session:  session,
		AutoMode: true,
		StableID: true,
		Render: picture.Config{
			Fit:         picture.FitFill,
			KittyMedium: picture.KittyMediumSharedMemory,
			KittyFormat: picture.KittyFormatZlib,
		},
	})
	_ = pic.SetVisible(false)
	_ = pic.SetSize(80, 24)

	m := &demoModel{
		session:    session,
		ui:         ntimgui.New(ntimgui.WithScreenSize(80, 24), ntimgui.WithTheme(ntimgui.HUDTheme())),
		glyph:      ntimgui.NewGlyphBackend(80, 24),
		raster:     ntimgui.NewRasterBackend(9, 18),
		pic:        pic,
		width:      80,
		height:     24,
		yaw:        0.6,
		pitch:      0.45,
		fov:        65,
		zoom:       3.2,
		speed:      1.0,
		density:    18,
		shape:      0,
		autoRotate: true,
		showAxes:   true,
		objectTag:  "ntimgui-torus",
		dirty:      true,
		geometry:   pic.RasterGeometry(),
	}
	m.buildFrame()
	return m
}

func (m *demoModel) Init() tea.Cmd {
	return tea.Batch(m.pic.Init(), tickCmd())
}

func (m *demoModel) buildFrame() {
	m.ui.BeginFrame()

	// Floating Control Panel on the left
	if m.ui.BeginPanel("nt3d + ntimgui Controls", ntimgui.NewRect(2, 1, 34, 21)) {
		m.ui.Text("Object: %s (Pulses: %d)", m.objectTag, m.pulses)
		m.ui.TextInput("Tag", &m.objectTag, 16)
		m.ui.Separator()

		m.ui.Toggle("Auto-Rotate", &m.autoRotate)
		m.ui.Checkbox("Show 3D Axes", &m.showAxes)
		m.ui.Checkbox("Reverse Spin", &m.reverseSpin)

		if m.ui.CollapsingHeader("3D Geometry", true) {
			m.ui.Indent(2)
			m.ui.RadioButtonInt("Torus Knot", &m.shape, 0)
			m.ui.RadioButtonInt("Geodesic Sphere", &m.shape, 1)
			m.ui.RadioButtonInt("Wire Cube", &m.shape, 2)
			m.ui.Unindent(2)
		}

		if m.ui.CollapsingHeader("Render Backend (g)", true) {
			m.ui.Indent(2)
			m.ui.RadioButtonInt("Glyph (ntcanvas)", &m.backendMode, 0)
			m.ui.RadioButtonInt("Raster (ntpicture)", &m.backendMode, 1)
			m.ui.Unindent(2)
		}

		m.ui.Separator()
		m.ui.SliderFloat("FOV", &m.fov, 35, 110, "%.0f°")
		m.ui.SliderFloat("Zoom", &m.zoom, 1.8, 5.5, "%.1f")
		m.ui.SliderInt("Detail", &m.density, 8, 28)
		m.ui.ProgressBar(float64(m.density)/28.0, 28, fmt.Sprintf("Mesh %d%%", m.density*100/28))

		m.ui.BeginHorizontal()
		if m.ui.Button("Reset") {
			m.yaw, m.pitch = 0.6, 0.45
			m.fov, m.zoom = 65, 3.2
			m.density = 18
		}
		m.ui.Tooltip("Reset 3D camera orbit and FOV to defaults")
		if m.ui.Button("Pulse") {
			m.pulses++
			m.yaw += 0.35
		}
		m.ui.Tooltip("Step camera yaw and increment pulse counter")
		m.ui.EndHorizontal()
	}
	m.ui.EndPanel()

	// Floating Telemetry Panel on the top-right
	teleX := max(38, m.width-34)
	if m.ui.BeginPanel("IMGUI Telemetry", ntimgui.NewRect(teleX, 1, 32, 10)) {
		m.ui.Text("Mouse Cell: (%d, %d)", m.ui.Input.MouseX, m.ui.Input.MouseY)
		m.ui.Text("Capture Mouse: %v", m.ui.WantCaptureMouse())
		m.ui.Text("Capture Keys:  %v", m.ui.WantCaptureKeyboard())
		m.ui.Text("Hot: %v  Active: %v", m.ui.Hot != 0, m.ui.Active != 0)
		m.ui.Text("Camera: yaw=%.2f pitch=%.2f", m.yaw, m.pitch)
		m.ui.Text("Zoom: %.1f (wheel / +/-)", m.zoom)
		m.ui.LabelMuted("Drag background to orbit 3D")
	}
	m.ui.EndPanel()

	m.ui.EndFrame()
}

// submitRasterIfReady submits a new raster frame to ntpicture only when no
// prior frame is still preparing/presenting, preventing job cancellation loops.
func (m *demoModel) submitRasterIfReady() tea.Cmd {
	if m.backendMode != 1 || !m.dirty || m.awaiting || m.pic.Status().Busy {
		return nil
	}
	g := m.pic.RasterGeometry()
	if g.Pixels.X <= 0 || g.Pixels.Y <= 0 {
		return nil
	}
	m.dirty = false
	img := m.renderRasterFrameFor(g)
	before := m.pic.Status().Generation
	cmd := m.pic.SetImageImmutableFor(img, g)
	if m.pic.Status().Generation != before {
		m.awaiting = true
		m.submitted = m.pic.Status().Generation
	}
	return cmd
}

func (m *demoModel) syncBackendVisibility(prevMode int) tea.Cmd {
	if prevMode == m.backendMode {
		return nil
	}
	if m.backendMode == 1 {
		m.dirty = false
		g := m.pic.RasterGeometry()
		img := m.renderRasterFrameFor(g)
		// Update source while hidden so SetVisible(true) schedules a single job.
		_ = m.pic.SetImageImmutableFor(img, g)
		cmd := m.pic.SetVisible(true)
		m.awaiting = m.pic.Status().Busy
		m.submitted = m.pic.Status().Generation
		return cmd
	}
	m.awaiting = false
	return m.pic.SetVisible(false)
}

func (m *demoModel) Help() []key.Binding {
	return []key.Binding{
		demohelp.Key("Pause / Orbit", "space"),
		demohelp.Key("Cycle mesh", "m"),
		demohelp.Key("Zoom in", "+", "="),
		demohelp.Key("Zoom out", "-"),
		demohelp.Key("Reset camera", "r"),
		demohelp.Key("Glyph / Raster", "g"),
		demohelp.Key("Next widget", "tab"),
		demohelp.Key("Quit", "q", "ctrl+c"),
	}
}

func (m *demoModel) Summary() string {
	return fmt.Sprintf("HUD: shape=%d rotate=%v zoom=%.1f backend=%d", m.shape, m.autoRotate, m.zoom, m.backendMode)
}

func (m *demoModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	prevMode := m.backendMode
	prevHot := m.ui.Hot

	switch ev := msg.(type) {
	case picture.PresentedMsg:
		if ev.Owner == m.pic && m.awaiting && ev.Generation >= m.submitted {
			m.awaiting = false
		}
	case picture.StaleFrameMsg:
		if ev.Owner == m.pic {
			m.awaiting = false
			m.dirty = true
		}
	case picture.ErrorMsg:
		if ev.Owner == m.pic {
			m.awaiting = false
		}

	case tickMsg:
		if m.autoRotate {
			dir := 1.0
			if m.reverseSpin {
				dir = -1.0
			}
			m.yaw += 0.035 * m.speed * dir
			m.dirty = true
		}
		m.buildFrame()
		if c := m.pic.Update(msg); c != nil {
			cmds = append(cmds, c)
		}
		if c := m.submitRasterIfReady(); c != nil {
			cmds = append(cmds, c)
		}
		cmds = append(cmds, tickCmd())
		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		if ev.String() == "ctrl+c" || (ev.String() == "q" && !m.ui.WantCaptureKeyboard()) {
			m.session.Close()
			return m, tea.Sequence(m.pic.Close(), tea.Quit)
		}
		if !m.ui.WantCaptureKeyboard() {
			switch ev.String() {
			case "space", " ":
				m.autoRotate = !m.autoRotate
				m.dirty = true
			case "m":
				m.shape = (m.shape + 1) % 3
				m.dirty = true
			case "+", "=":
				m.zoom = max(1.8, m.zoom-0.2)
				m.dirty = true
			case "-", "_":
				m.zoom = min(5.5, m.zoom+0.2)
				m.dirty = true
			case "r":
				m.yaw, m.pitch = 0.4, 0.25
				m.fov, m.zoom = 65, 3.2
				m.density = 18
				m.dirty = true
			case "g":
				m.backendMode = 1 - m.backendMode
			}
		}

	case tea.MouseWheelMsg:
		// Neither HUD panel has scrollable lists, so wheel events always zoom the 3D viewport.
		switch ev.Mouse().Button {
		case tea.MouseWheelUp:
			m.zoom = max(1.8, m.zoom-0.2)
			m.dirty = true
		case tea.MouseWheelDown:
			m.zoom = min(5.5, m.zoom+0.2)
			m.dirty = true
		}

	case tea.WindowSizeMsg:
		if ev.Width > 0 && ev.Height > 0 {
			m.width, m.height = ev.Width, ev.Height
			m.glyph.Resize(ev.Width, ev.Height)
			if c := m.pic.SetSize(ev.Width, ev.Height); c != nil {
				cmds = append(cmds, c)
			}
			m.dirty = true
			m.awaiting = false
		}
	}

	processed := m.ui.ProcessInput(msg)
	m.buildFrame()
	if processed {
		if _, isMotion := msg.(tea.MouseMotionMsg); !isMotion || m.ui.Hot != prevHot || m.ui.Active != 0 {
			m.dirty = true
		}
	}

	// If ntimgui did NOT capture the mouse, allow dragging the background to orbit the 3D camera!
	switch ev := msg.(type) {
	case tea.MouseClickMsg:
		mouse := ev.Mouse()
		if !m.ui.WantCaptureMouse() && mouse.Button == tea.MouseLeft {
			m.orbiting = true
			m.lastMX, m.lastMY = mouse.X, mouse.Y
		} else {
			m.orbiting = false
		}
	case tea.MouseMotionMsg:
		mouse := ev.Mouse()
		if m.orbiting && !m.ui.WantCaptureMouse() {
			dx := mouse.X - m.lastMX
			dy := mouse.Y - m.lastMY
			if dx != 0 || dy != 0 {
				m.lastMX, m.lastMY = mouse.X, mouse.Y
				m.yaw += float64(dx) * 0.05
				m.pitch = max(-1.4, min(1.4, m.pitch+float64(dy)*0.05))
				m.dirty = true
				m.buildFrame()
			}
		}
	case tea.MouseReleaseMsg:
		m.orbiting = false
	}

	if c := m.syncBackendVisibility(prevMode); c != nil {
		cmds = append(cmds, c)
	}

	if c := m.pic.Update(msg); c != nil {
		cmds = append(cmds, c)
	}
	if g := m.pic.RasterGeometry(); g != m.geometry {
		m.geometry = g
		m.awaiting = false
		m.dirty = true
	}
	if c := m.submitRasterIfReady(); c != nil {
		cmds = append(cmds, c)
	}

	return m, tea.Batch(cmds...)
}

func (m *demoModel) sceneEdges() []edge3 {
	var edges []edge3
	n := max(6, m.density)

	switch m.shape {
	case 0: // Torus
		const R, r = 1.05, 0.42
		rings, sides := n, max(8, n*2/3)
		pt := func(i, j int) vec3 {
			u := float64(i) * 2 * math.Pi / float64(rings)
			v := float64(j) * 2 * math.Pi / float64(sides)
			c := R + r*math.Cos(v)
			return vec3{
				x: c * math.Cos(u),
				y: r * math.Sin(v),
				z: c * math.Sin(u),
			}
		}
		for i := 0; i < rings; i++ {
			col := ntimgui.RGB(uint8(90+(i*140)/rings), 185, 255)
			for j := 0; j < sides; j++ {
				p0 := pt(i, j)
				p1 := pt(i+1, j)
				p2 := pt(i, j+1)
				edges = append(edges, edge3{a: p0, b: p1, col: col})
				edges = append(edges, edge3{a: p0, b: p2, col: ntimgui.RGB(70, 210, 190)})
			}
		}
	case 1: // Geodesic Sphere
		latSteps := max(6, n/2)
		lonSteps := n
		pt := func(i, j int) vec3 {
			lat := (float64(i)/float64(latSteps) - 0.5) * math.Pi
			lon := float64(j) * 2 * math.Pi / float64(lonSteps)
			return vec3{
				x: 1.25 * math.Cos(lat) * math.Cos(lon),
				y: 1.25 * math.Sin(lat),
				z: 1.25 * math.Cos(lat) * math.Sin(lon),
			}
		}
		for i := 0; i <= latSteps; i++ {
			for j := 0; j < lonSteps; j++ {
				p0 := pt(i, j)
				if i < latSteps {
					edges = append(edges, edge3{a: p0, b: pt(i+1, j), col: ntimgui.RGB(110, 190, 255)})
				}
				edges = append(edges, edge3{a: p0, b: pt(i, j+1), col: ntimgui.RGB(160, 130, 255)})
			}
		}
	default: // Cube
		s := 1.0
		verts := [8]vec3{
			{-s, -s, -s}, {s, -s, -s}, {s, s, -s}, {-s, s, -s},
			{-s, -s, s}, {s, -s, s}, {s, s, s}, {-s, s, s},
		}
		pairs := [][2]int{
			{0, 1}, {1, 2}, {2, 3}, {3, 0},
			{4, 5}, {5, 6}, {6, 7}, {7, 4},
			{0, 4}, {1, 5}, {2, 6}, {3, 7},
		}
		for _, p := range pairs {
			edges = append(edges, edge3{a: verts[p[0]], b: verts[p[1]], col: ntimgui.RGB(100, 220, 255)})
		}
	}

	if m.showAxes {
		edges = append(edges,
			edge3{a: vec3{0, 0, 0}, b: vec3{1.7, 0, 0}, col: ntimgui.RGB(255, 95, 95)},
			edge3{a: vec3{0, 0, 0}, b: vec3{0, 1.7, 0}, col: ntimgui.RGB(95, 240, 130)},
			edge3{a: vec3{0, 0, 0}, b: vec3{0, 0, 1.7}, col: ntimgui.RGB(95, 160, 255)},
		)
	}
	return edges
}

func (m *demoModel) project(v vec3, w, h, aspect float64) (float64, float64, bool) {
	cy, sy := math.Cos(m.yaw), math.Sin(m.yaw)
	cp, sp := math.Cos(m.pitch), math.Sin(m.pitch)

	// Rotate Y then X
	x1 := v.x*cy - v.z*sy
	z1 := v.x*sy + v.z*cy
	y2 := v.y*cp - z1*sp
	z2 := v.y*sp + z1*cp + m.zoom
	if z2 <= 0.2 {
		return 0, 0, false
	}

	f := 1.0 / math.Tan(m.fov*math.Pi/360.0)
	cx := w * 0.62
	cyScreen := h * 0.52
	scale := math.Min(w, h*aspect) * 0.42 * f

	sx := cx + (x1/z2)*scale*aspect
	syScreen := cyScreen - (y2/z2)*scale
	return sx, syScreen, true
}

func (m *demoModel) renderGlyphScene(grid *ntcanvas.GraphemeGrid) {
	grid.Clear()
	w, h := float64(m.width), float64(m.height)
	for _, e := range m.sceneEdges() {
		x0, y0, ok0 := m.project(e.a, w, h, 2.0)
		x1, y1, ok1 := m.project(e.b, w, h, 2.0)
		if !ok0 || !ok1 {
			continue
		}
		st := ntcanvas.CellStyle{Foreground: e.col.CanvasColor()}
		steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0)))
		if steps == 0 {
			steps = 1
		}
		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			px := int(math.Round(x0 + (x1-x0)*t))
			py := int(math.Round(y0 + (y1-y0)*t))
			if px >= 0 && px < m.width && py >= 0 && py < m.height {
				grid.SetGrapheme(image.Pt(px, py), "·", st)
			}
		}
	}

	if m.objectTag != "" {
		tx, ty, ok := m.project(vec3{0, 1.55, 0}, w, h, 2.0)
		if ok {
			ix := int(math.Round(tx)) - len(m.objectTag)/2
			iy := int(math.Round(ty))
			grid.SetTextWithStyle(image.Pt(ix, iy), m.objectTag, ntcanvas.CellStyle{
				Foreground: ntcanvas.RGBColor(255, 220, 110),
				Attributes: ntcanvas.Bold,
			})
		}
	}
}

func (m *demoModel) renderRasterFrameFor(g picture.RasterGeometry) *image.RGBA {
	pw, ph := g.Pixels.X, g.Pixels.Y
	cw := pw / max(1, m.width)
	ch := ph / max(1, m.height)
	if cw < 7 || ch < 13 {
		cw, ch = 9, 18
		pw, ph = max(1, m.width)*cw, max(1, m.height)*ch
	}
	m.raster.SetCellPixelSize(cw, ch)

	img := image.NewRGBA(image.Rect(0, 0, pw, ph))

	// Gradient backdrop
	for y := 0; y < ph; y++ {
		for x := 0; x < pw; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(12 + (x*18)/pw),
				G: uint8(18 + (y*28)/ph),
				B: uint8(34 + (x*24)/pw),
				A: 255,
			})
		}
	}

	// Draw 3D edges in pixel space
	wf, hf := float64(pw), float64(ph)
	aspect := 1.0
	if g.DisplayAspect > 0 && ph > 0 {
		// Compensate when raster pixels are non-square (e.g. half-block glyph fallback)
		pixelAspect := (wf / hf) / g.DisplayAspect
		if pixelAspect > 0 {
			aspect = pixelAspect
		}
	}
	for _, e := range m.sceneEdges() {
		x0, y0, ok0 := m.project(e.a, wf, hf, aspect)
		x1, y1, ok1 := m.project(e.b, wf, hf, aspect)
		if !ok0 || !ok1 {
			continue
		}
		col := color.RGBA{R: e.col.R, G: e.col.G, B: e.col.B, A: 255}
		steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0)))
		if steps == 0 {
			steps = 1
		}
		for i := 0; i <= steps; i++ {
			t := float64(i) / float64(steps)
			px := int(math.Round(x0 + (x1-x0)*t))
			py := int(math.Round(y0 + (y1-y0)*t))
			if px >= 0 && px < pw && py >= 0 && py < ph {
				img.SetRGBA(px, py, col)
			}
		}
	}

	// Composite the ntimgui DrawList directly on top of the 3D pixel buffer!
	m.raster.RenderOver(img, m.ui.DrawList)
	return img
}

func (m *demoModel) View() tea.View {
	var content string
	if m.backendMode == 1 {
		content = m.pic.View()
	} else {
		grid := m.glyph.Grid()
		m.renderGlyphScene(grid)
		ntimgui.RenderToGraphemeGrid(grid, m.ui.DrawList)
		content = grid.Render(grid.Bounds())
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

func writeSnapshot(path string, m *demoModel) error {
	img := m.renderRasterFrameFor(picture.RasterGeometry{
		Pixels:        image.Pt(m.width*9, m.height*18),
		DisplayAspect: float64(m.width*9) / float64(m.height*18),
	})
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	encErr := png.Encode(f, img)
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	return closeErr
}

func main() {
	snapshot := flag.String("snapshot", "", "render the 3D scene + HUD via RasterBackend to a PNG file and exit")
	flag.Parse()

	m := newDemoModel()
	defer m.session.Close()
	defer func() { _ = m.pic.Close() }()

	if *snapshot != "" {
		if err := writeSnapshot(*snapshot, m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	demohelp.Main(m)
}
