// Copyright (c) 2026 Neomantra Corp.

package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"math"
	"os"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"nimbleterminal.dev/ntimgui"
	"nimbleterminal.dev/ntimgui/examples/internal/demohelp"
	picture "nimbleterminal.dev/ntpicture"
)

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return demohelp.Tick(time.Second/20, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

var passNames = []string{
	"01 ShadowCascade",
	"02 GBufferOpaque",
	"03 FurShellPass",
	"04 SkyboxEnv",
	"05 BloomDownUp",
	"06 ntimguiOverlay",
}

type dashboardModel struct {
	session *picture.TerminalSession
	ui      *ntimgui.Context
	glyph   *ntimgui.GlyphBackend
	raster  *ntimgui.RasterBackend
	pic     *picture.Picture

	width, height int
	backendMode   int // 0=Glyph, 1=Raster

	paused       bool
	vsync        bool
	gpuLimit     float64
	instances    int
	selectedPass int
	themeMode    int

	phase     float64
	frameHist []float64
	passTimes []float64

	dirty     bool
	awaiting  bool
	submitted uint64
	geometry  picture.RasterGeometry
}

func newDashboardModel() *dashboardModel {
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
	_ = pic.SetSize(92, 28)

	m := &dashboardModel{
		session:   session,
		ui:        ntimgui.New(ntimgui.WithScreenSize(92, 28), ntimgui.WithTheme(ntimgui.CyberpunkTheme())),
		glyph:     ntimgui.NewGlyphBackend(92, 28),
		raster:    ntimgui.NewRasterBackend(9, 18),
		pic:       pic,
		width:     92,
		height:    28,
		vsync:     true,
		gpuLimit:  16.6,
		instances: 1200,
		frameHist: make([]float64, 36),
		passTimes: make([]float64, len(passNames)),
		dirty:     true,
		geometry:  pic.RasterGeometry(),
	}
	m.stepMetrics()
	m.buildFrame()
	return m
}

func (m *dashboardModel) stepMetrics() {
	scale := float64(m.instances) / 1200.0
	for i := range m.frameHist {
		t := m.phase + float64(i)*0.22
		ms := (7.5 + 3.2*math.Sin(t) + 1.8*math.Cos(t*2.7)) * scale
		m.frameHist[i] = max(1.0, min(24.0, ms))
	}
	for i := range m.passTimes {
		t := m.phase*0.7 + float64(i)*0.9
		m.passTimes[i] = (0.8 + 2.6*math.Abs(math.Sin(t))) * scale
	}
}

func (m *dashboardModel) Init() tea.Cmd {
	return tea.Batch(m.pic.Init(), tickCmd())
}

func (m *dashboardModel) buildFrame() {
	m.ui.BeginFrame()

	// Panel 1: Real-time Frame Pacing & GPU Load
	if m.ui.BeginPanel("GPU Frame Pacing & Telemetry", ntimgui.NewRect(1, 1, 44, 15)) {
		latestMS := m.frameHist[len(m.frameHist)-1]
		fps := 1000.0 / max(1.0, latestMS)
		m.ui.KeyValue("Frame Time / FPS", fmt.Sprintf("%.2f ms (%.0f FPS)", latestMS, fps))
		m.ui.KeyValue("Draw Calls / Tris", fmt.Sprintf("%d / %dk", m.instances/4, m.instances*3))
		m.ui.Separator()

		m.ui.PlotLines("Frame ms", m.frameHist, 40, 3, fmt.Sprintf("Now: %.1f ms", latestMS))
		m.ui.ProgressBar(latestMS/max(1.0, m.gpuLimit), 40, fmt.Sprintf("Budget %.1f / %.1f ms", latestMS, m.gpuLimit))
		m.ui.Separator()

		m.ui.BeginHorizontal()
		m.ui.Toggle("Pause", &m.paused)
		m.ui.Checkbox("VSync", &m.vsync)
		if m.ui.Button("Spike Load") {
			m.instances = min(3000, m.instances+400)
			m.stepMetrics()
		}
		m.ui.EndHorizontal()
	}
	m.ui.EndPanel()

	// Panel 2: Render Pass Breakdown
	if m.ui.BeginPanel("Render Pass Breakdown", ntimgui.NewRect(47, 1, 43, 15)) {
		m.ui.PlotHistogram("Pass Duration", m.passTimes, 39, 3, "6 Active Passes")
		m.ui.Separator()
		m.ui.ListBox("Select Pass", &m.selectedPass, passNames, 6)
		if m.selectedPass >= 0 && m.selectedPass < len(m.passTimes) {
			m.ui.KeyValue("Selected Pass Cost", fmt.Sprintf("%.2f ms", m.passTimes[m.selectedPass]))
		}
	}
	m.ui.EndPanel()

	// Panel 3: Simulation & Theme Controls
	if m.ui.BeginPanel("Simulation & Display Controls", ntimgui.NewRect(1, 17, 89, 10)) {
		m.ui.SliderInt("Scene Instances", &m.instances, 200, 3000)
		m.ui.SliderFloat("Frame Budget", &m.gpuLimit, 8.3, 33.3, "%.1f ms")
		m.ui.Separator()

		m.ui.BeginHorizontal()
		m.ui.Label("Theme:")
		if m.ui.RadioButtonInt("Cyberpunk", &m.themeMode, 0) {
			m.ui.Theme = ntimgui.CyberpunkTheme()
		}
		if m.ui.RadioButtonInt("CRT Amber", &m.themeMode, 1) {
			m.ui.Theme = ntimgui.AmberTheme()
		}
		if m.ui.RadioButtonInt("HUD Dark", &m.themeMode, 2) {
			m.ui.Theme = ntimgui.HUDTheme()
		}
		m.ui.EndHorizontal()

		m.ui.BeginHorizontal()
		m.ui.Label("Backend (g):")
		m.ui.RadioButtonInt("Glyph (ntcanvas)", &m.backendMode, 0)
		m.ui.RadioButtonInt("Raster (ntpicture)", &m.backendMode, 1)
		m.ui.EndHorizontal()
	}
	m.ui.EndPanel()

	m.ui.EndFrame()
}

func (m *dashboardModel) submitRasterIfReady() tea.Cmd {
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

func (m *dashboardModel) syncBackendVisibility(prevMode int) tea.Cmd {
	if prevMode == m.backendMode {
		return nil
	}
	if m.backendMode == 1 {
		m.dirty = false
		g := m.pic.RasterGeometry()
		img := m.renderRasterFrameFor(g)
		_ = m.pic.SetImageImmutableFor(img, g)
		cmd := m.pic.SetVisible(true)
		m.awaiting = m.pic.Status().Busy
		m.submitted = m.pic.Status().Generation
		return cmd
	}
	m.awaiting = false
	return m.pic.SetVisible(false)
}

func (m *dashboardModel) Help() []key.Binding {
	return []key.Binding{
		demohelp.Key("Pause / Resume", "space"),
		demohelp.Key("Cycle theme", "t"),
		demohelp.Key("Glyph / Raster", "g"),
		demohelp.Key("Next widget", "tab"),
		demohelp.Key("Quit", "q", "ctrl+c"),
	}
}

func (m *dashboardModel) Summary() string {
	return fmt.Sprintf("Dashboard: paused=%v theme=%d backend=%d", m.paused, m.themeMode, m.backendMode)
}

func (m *dashboardModel) applyTheme() {
	switch m.themeMode {
	case 1:
		m.ui.Theme = ntimgui.AmberTheme()
	case 2:
		m.ui.Theme = ntimgui.HUDTheme()
	default:
		m.ui.Theme = ntimgui.CyberpunkTheme()
	}
}

func (m *dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if !m.paused {
			m.phase += 0.22
			m.stepMetrics()
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
				m.paused = !m.paused
				m.dirty = true
			case "t":
				m.themeMode = (m.themeMode + 1) % 3
				m.applyTheme()
				m.dirty = true
			case "g":
				m.backendMode = 1 - m.backendMode
			}
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

func (m *dashboardModel) renderRasterFrameFor(g picture.RasterGeometry) *image.RGBA {
	pw, ph := g.Pixels.X, g.Pixels.Y
	cw := pw / max(1, m.width)
	ch := ph / max(1, m.height)
	if cw < 7 || ch < 13 {
		cw, ch = 9, 18
		pw, ph = max(1, m.width)*cw, max(1, m.height)*ch
	}
	m.raster.SetCellPixelSize(cw, ch)

	img := image.NewRGBA(image.Rect(0, 0, pw, ph))
	pix := img.Pix
	for i := 0; i < len(pix); i += 4 {
		pix[i+0] = 10
		pix[i+1] = 8
		pix[i+2] = 20
		pix[i+3] = 255
	}
	m.raster.RenderOver(img, m.ui.DrawList)
	return img
}

func (m *dashboardModel) View() tea.View {
	var content string
	if m.backendMode == 1 {
		content = m.pic.View()
	} else {
		content = m.glyph.RenderToString(m.ui.DrawList)
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeAllMotion
	return v
}

func writeSnapshot(path string, m *dashboardModel) error {
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
	snapshot := flag.String("snapshot", "", "render the dashboard to a PNG file and exit")
	flag.Parse()

	m := newDashboardModel()
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
