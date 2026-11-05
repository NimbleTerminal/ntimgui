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

var (
	catalogTabs  = []string{"Controls", "Plots", "Trees & Lists", "Themes"}
	qualityItems = []string{"Low (Fast)", "Medium", "High (MSAA 4x)", "Ultra (Raytraced)"}
	shaderItems  = []string{
		"pbr_metallic.wgsl",
		"fur_shells.wgsl",
		"shadow_depth.wgsl",
		"plasma_proc.wgsl",
		"bloom_downsample.wgsl",
		"tonemap_aces.wgsl",
		"wireframe_aa.wgsl",
		"fog_exponential.wgsl",
	}
	themeNames  = []string{"HUD Compact (Default)", "ImGui Dark (Boxed)", "Retro CRT Amber", "Neon Cyberpunk", "Daylight Light"}
	borderNames = []string{"Rounded (╭─╮)", "Single (┌─┐)", "Double (╔═╗)", "Thick (┏━┓)"}
)

type catalogModel struct {
	session *picture.TerminalSession
	ui      *ntimgui.Context
	glyph   *ntimgui.GlyphBackend
	raster  *ntimgui.RasterBackend
	pic     *picture.Picture

	width, height int
	backendMode   int // 0=Glyph (ntcanvas), 1=Raster (ntpicture)

	// Catalog state
	activeTab int

	// Tab 1: Controls
	counter      int
	enableVSync  bool
	enableHDR    bool
	enableBloom  bool
	aaMode       int
	qualityIdx   int
	profileLabel string

	// Tab 2: Sliders, Plots & Color
	exposure    float64
	roughness   float64
	threadCount int
	animateWave bool
	phase       float64
	waveData    []float64
	histData    []float64
	accentColor ntimgui.RGBA

	// Tab 3: Trees, Lists & Tables
	selectedShader int

	// Tab 4: Themes & Borders
	themeIdx  int
	borderIdx int

	// Event log
	events []string

	// Kitty / ntpicture pacing
	dirty     bool
	awaiting  bool
	submitted uint64
	geometry  picture.RasterGeometry
}

func newCatalogModel() *catalogModel {
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

	m := &catalogModel{
		session:      session,
		ui:           ntimgui.New(ntimgui.WithScreenSize(92, 28), ntimgui.WithTheme(ntimgui.HUDTheme())),
		glyph:        ntimgui.NewGlyphBackend(92, 28),
		raster:       ntimgui.NewRasterBackend(9, 18),
		pic:          pic,
		width:        92,
		height:       28,
		enableVSync:  true,
		enableHDR:    true,
		aaMode:       1,
		qualityIdx:   2,
		profileLabel: "studio-default",
		exposure:     1.25,
		roughness:    0.35,
		threadCount:  8,
		animateWave:  true,
		waveData:     make([]float64, 32),
		histData:     make([]float64, 12),
		accentColor:  ntimgui.RGB(76, 154, 255),
		events: []string{
			"Welcome to the ntimgui UX Catalog!",
			"Click tabs, drag panels, or press Tab.",
			"Press 'g' to toggle Glyph / Raster.",
		},
		dirty:    true,
		geometry: pic.RasterGeometry(),
	}
	m.updatePlots()
	m.buildFrame()
	return m
}

func (m *catalogModel) logEvent(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	m.events = append(m.events, msg)
	if len(m.events) > 8 {
		m.events = m.events[len(m.events)-8:]
	}
}

func (m *catalogModel) updatePlots() {
	for i := range m.waveData {
		t := m.phase + float64(i)*0.28
		m.waveData[i] = 0.5 + 0.35*math.Sin(t) + 0.15*math.Cos(t*2.3)
	}
	for i := range m.histData {
		t := m.phase*0.8 + float64(i)*0.55
		m.histData[i] = 0.15 + 0.85*math.Abs(math.Sin(t))
	}
}

func (m *catalogModel) applySelectedTheme() {
	var th ntimgui.Theme
	switch m.themeIdx {
	case 1:
		th = ntimgui.DefaultTheme()
	case 2:
		th = ntimgui.AmberTheme()
	case 3:
		th = ntimgui.CyberpunkTheme()
	case 4:
		th = ntimgui.LightTheme()
	default:
		th = ntimgui.HUDTheme()
	}
	borders := []ntimgui.BorderStyle{
		ntimgui.BorderRounded,
		ntimgui.BorderSingle,
		ntimgui.BorderDouble,
		ntimgui.BorderThick,
	}
	if m.borderIdx >= 0 && m.borderIdx < len(borders) {
		th.Border = borders[m.borderIdx]
		th.PanelBorder = borders[m.borderIdx]
	}
	m.ui.Theme = th
}

func (m *catalogModel) Init() tea.Cmd {
	return tea.Batch(m.pic.Init(), tickCmd())
}

func (m *catalogModel) buildFrame() {
	m.ui.BeginFrame()

	// Left Window: Main UX Element Catalog
	if m.ui.BeginPanel("ntimgui UX Element Catalog", ntimgui.NewRect(1, 1, 52, 25)) {
		if m.ui.TabBar("catalog_tabs", &m.activeTab, catalogTabs) {
			m.logEvent("Switched to tab: %s", catalogTabs[m.activeTab])
		}
		m.ui.Separator()

		switch m.activeTab {
		case 0: // 1. Controls
			m.ui.Label("Push Buttons (Compact & Boxed):")
			m.ui.BeginHorizontal()
			if m.ui.Button("Increment") {
				m.counter++
				m.logEvent("Button(Increment) -> %d", m.counter)
			}
			m.ui.Tooltip("Click or press Space/Enter to increment counter")
			if m.ui.Button("Reset##cnt") {
				m.counter = 0
				m.logEvent("Counter reset to 0")
			}
			m.ui.Text("Count: %d", m.counter)
			m.ui.EndHorizontal()

			if m.ui.ButtonSized("Boxed 3-Row Action Button", 32, 3) {
				m.counter += 10
				m.logEvent("Boxed Button clicked (+10 -> %d)", m.counter)
			}
			m.ui.Separator()

			m.ui.Label("Checkboxes & Pill Toggles:")
			m.ui.BeginHorizontal()
			if m.ui.Checkbox("VSync", &m.enableVSync) {
				m.logEvent("Checkbox(VSync) = %v", m.enableVSync)
			}
			if m.ui.Toggle("HDR Pipeline", &m.enableHDR) {
				m.logEvent("Toggle(HDR) = %v", m.enableHDR)
			}
			if m.ui.Checkbox("Bloom", &m.enableBloom) {
				m.logEvent("Checkbox(Bloom) = %v", m.enableBloom)
			}
			m.ui.EndHorizontal()
			m.ui.Separator()

			m.ui.Label("Radio Buttons & Combo Selector:")
			m.ui.BeginHorizontal()
			if m.ui.RadioButtonInt("Off", &m.aaMode, 0) {
				m.logEvent("AA Mode -> Off")
			}
			if m.ui.RadioButtonInt("FXAA", &m.aaMode, 1) {
				m.logEvent("AA Mode -> FXAA")
			}
			if m.ui.RadioButtonInt("MSAA 4x", &m.aaMode, 2) {
				m.logEvent("AA Mode -> MSAA 4x")
			}
			m.ui.EndHorizontal()

			if m.ui.Combo("Quality", &m.qualityIdx, qualityItems) {
				m.logEvent("Combo(Quality) -> %s", qualityItems[m.qualityIdx])
			}
			if m.ui.TextInput("Profile", &m.profileLabel, 22) {
				m.logEvent("TextInput(Profile) -> %q", m.profileLabel)
			}

		case 1: // 2. Sliders & Plots
			m.ui.Label("Numeric Sliders & Progress Gauge:")
			if m.ui.SliderFloat("Exposure", &m.exposure, 0.1, 4.0, "%.2f EV") {
				m.logEvent("SliderFloat(Exposure) -> %.2f", m.exposure)
			}
			if m.ui.SliderFloat("Roughness", &m.roughness, 0.0, 1.0, "%.2f") {
				m.logEvent("SliderFloat(Roughness) -> %.2f", m.roughness)
			}
			if m.ui.SliderInt("Threads", &m.threadCount, 1, 32) {
				m.logEvent("SliderInt(Threads) -> %d", m.threadCount)
			}
			m.ui.ProgressBar(float64(m.threadCount)/32.0, 46, fmt.Sprintf("Worker Pool %d/32 (%d%%)", m.threadCount, m.threadCount*100/32))
			m.ui.Separator()

			m.ui.BeginHorizontal()
			m.ui.Label("Inline Sparkline & Histogram:")
			m.ui.Toggle("Animate", &m.animateWave)
			m.ui.EndHorizontal()
			m.ui.PlotLines("Waveform", m.waveData, 46, 2, "sin(t)+cos(2.3t)")
			m.ui.PlotHistogram("Spectrum", m.histData, 46, 2, "FFT Bands")
			m.ui.Separator()

			if m.ui.ColorEdit("Accent Swatch", &m.accentColor) {
				m.logEvent("ColorEdit -> #%02X%02X%02X", m.accentColor.R, m.accentColor.G, m.accentColor.B)
			}

		case 2: // 3. Trees & Lists
			if m.ui.CollapsingHeader("Hierarchical Scene Tree (TreeNode)", true) {
				if m.ui.TreeNode("Root Scene (nt3d.Scene)", true) {
					if m.ui.TreeNode("SpotTheCow (Mesh 5,856 tris)", true) {
						m.ui.KeyValue("Shader", shaderItems[m.selectedShader])
						m.ui.TreePop()
					}
					m.ui.TreePop()
				}
			}
			m.ui.Separator()

			if m.ui.ListBox("Active Shader (Click or Wheel)", &m.selectedShader, shaderItems, 5) {
				m.logEvent("ListBox -> %s", shaderItems[m.selectedShader])
			}
			m.ui.Separator()

			colW := []int{16, 12, 14}
			m.ui.TableRow([]string{"Pass Name", "Time (ms)", "Triangles"}, colW, true)
			m.ui.TableRow([]string{"ShadowMap", "1.12 ms", "14,200"}, colW, false)
			m.ui.TableRow([]string{"Opaque Pbr", "3.48 ms", "28,400"}, colW, false)
			m.ui.TableRow([]string{"ntimgui Overlay", "0.18 ms", "48 cmds"}, colW, false)

		case 3: // 4. Themes & Borders
			m.ui.Label("Select Color Theme Preset:")
			for i, name := range themeNames {
				if m.ui.RadioButtonInt(name, &m.themeIdx, i) {
					m.applySelectedTheme()
					m.logEvent("Theme -> %s", name)
				}
			}
			m.ui.Separator()

			m.ui.Label("Select Box Border Style:")
			for i, name := range borderNames {
				if m.ui.RadioButtonInt(name, &m.borderIdx, i) {
					m.applySelectedTheme()
					m.logEvent("Border -> %s", name)
				}
			}
			m.ui.Separator()

			m.ui.Label("Select Render Backend (or press 'g'):")
			m.ui.BeginHorizontal()
			if m.ui.RadioButtonInt("Glyph (ntcanvas)", &m.backendMode, 0) {
				m.logEvent("Backend -> Glyph (ntcanvas)")
			}
			if m.ui.RadioButtonInt("Raster (ntpicture)", &m.backendMode, 1) {
				m.logEvent("Backend -> Raster (ntpicture)")
			}
			m.ui.EndHorizontal()
		}
	}
	m.ui.EndPanel()

	// Right Window 1: Event Log
	rightX := 55
	if m.ui.BeginPanel("Interaction Log", ntimgui.NewRect(rightX, 1, 35, 13)) {
		for _, ev := range m.events {
			m.ui.Label(ev)
		}
		m.ui.Separator()
		if m.ui.Button("Clear Log") {
			m.events = m.events[:0]
			m.logEvent("Log cleared.")
		}
	}
	m.ui.EndPanel()

	// Right Window 2: Live IMGUI State Inspector
	if m.ui.BeginPanel("IMGUI State Inspector", ntimgui.NewRect(rightX, 15, 35, 12)) {
		toggleLabel := "Switch to Raster (g)"
		if m.backendMode == 1 {
			toggleLabel = "Switch to Glyph (g)"
		}
		if m.ui.Button(toggleLabel + "###toggle_backend") {
			m.backendMode = 1 - m.backendMode
			m.logEvent("Backend toggled -> %d", m.backendMode)
		}
		backendStr := "Glyph (ntcanvas)"
		if m.backendMode == 1 {
			backendStr = "Raster (ntpicture)"
		}
		m.ui.KeyValue("Backend", backendStr)
		m.ui.KeyValue("Screen Cells", fmt.Sprintf("%dx%d", m.width, m.height))
		m.ui.KeyValue("Mouse Cell", fmt.Sprintf("(%d, %d)", m.ui.Input.MouseX, m.ui.Input.MouseY))
		m.ui.KeyValue("WantCaptureMouse", fmt.Sprintf("%v", m.ui.WantCaptureMouse()))
		m.ui.KeyValue("WantCaptureKeys", fmt.Sprintf("%v", m.ui.WantCaptureKeyboard()))
		m.ui.KeyValue("Hot / Active", fmt.Sprintf("%v / %v", m.ui.Hot != 0, m.ui.Active != 0))
		m.ui.KeyValue("DrawList Cmds", fmt.Sprintf("%d", m.ui.DrawList.Len()))
	}
	m.ui.EndPanel()

	m.ui.EndFrame()
}

func (m *catalogModel) submitRasterIfReady() tea.Cmd {
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

func (m *catalogModel) syncBackendVisibility(prevMode int) tea.Cmd {
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

func (m *catalogModel) Help() []key.Binding {
	return []key.Binding{
		demohelp.Key("Next tab", "]"),
		demohelp.Key("Cycle theme", "t"),
		demohelp.Key("Cycle border", "b"),
		demohelp.Key("Glyph / Raster", "g"),
		demohelp.Key("Next widget", "tab"),
		demohelp.Key("Quit", "q", "ctrl+c"),
	}
}

func (m *catalogModel) Summary() string {
	return fmt.Sprintf("Catalog: tab=%s theme=%d border=%d backend=%d", catalogTabs[m.activeTab], m.themeIdx, m.borderIdx, m.backendMode)
}

func (m *catalogModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
		if m.animateWave {
			m.phase += 0.18
			m.updatePlots()
			if m.activeTab == 1 {
				m.dirty = true
			}
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
			case "]":
				m.activeTab = (m.activeTab + 1) % len(catalogTabs)
				m.dirty = true
			case "[":
				m.activeTab = (m.activeTab - 1 + len(catalogTabs)) % len(catalogTabs)
				m.dirty = true
			case "t":
				m.themeIdx = (m.themeIdx + 1) % len(themeNames)
				m.applySelectedTheme()
				m.dirty = true
			case "b":
				m.borderIdx = (m.borderIdx + 1) % len(borderNames)
				m.applySelectedTheme()
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

func (m *catalogModel) renderRasterFrameFor(g picture.RasterGeometry) *image.RGBA {
	pw, ph := g.Pixels.X, g.Pixels.Y
	cw := pw / max(1, m.width)
	ch := ph / max(1, m.height)
	if cw < 7 || ch < 13 {
		cw, ch = 9, 18
		pw, ph = max(1, m.width)*cw, max(1, m.height)*ch
	}
	m.raster.SetCellPixelSize(cw, ch)

	img := image.NewRGBA(image.Rect(0, 0, pw, ph))
	for y := 0; y < ph; y++ {
		c := color.RGBA{
			R: 12,
			G: uint8(14 + (y*20)/max(1, ph)),
			B: uint8(26 + (y*16)/max(1, ph)),
			A: 255,
		}
		row := img.Pix[y*img.Stride : y*img.Stride+pw*4]
		for x := 0; x < len(row); x += 4 {
			row[x], row[x+1], row[x+2], row[x+3] = c.R, c.G, c.B, c.A
		}
	}
	m.raster.RenderOver(img, m.ui.DrawList)
	return img
}

func (m *catalogModel) View() tea.View {
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

func writeSnapshot(path string, m *catalogModel) error {
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
	snapshot := flag.String("snapshot", "", "render the catalog to a PNG file and exit")
	tab := flag.Int("tab", 0, "initial catalog tab (0..3)")
	theme := flag.Int("theme", 0, "initial theme preset (0..4)")
	flag.Parse()

	m := newCatalogModel()
	defer m.session.Close()
	defer func() { _ = m.pic.Close() }()

	if *tab >= 0 && *tab < len(catalogTabs) {
		m.activeTab = *tab
	}
	if *theme >= 0 && *theme < len(themeNames) {
		m.themeIdx = *theme
		m.applySelectedTheme()
	}
	m.buildFrame()

	if *snapshot != "" {
		if err := writeSnapshot(*snapshot, m); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	demohelp.Main(m)
}
