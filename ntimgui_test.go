// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	"image"
	"image/color"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestParseLabelAndIDStack(t *testing.T) {
	disp, key := ParseLabel("Rotate##btn1")
	if disp != "Rotate" || key != "Rotate##btn1" {
		t.Fatalf("ParseLabel(##) = (%q, %q)", disp, key)
	}

	disp2, key2 := ParseLabel("FPS: 60###fps_counter")
	if disp2 != "FPS: 60" || key2 != "###fps_counter" {
		t.Fatalf("ParseLabel(###) = (%q, %q)", disp2, key2)
	}

	ctx := New()
	ctx.BeginFrame()
	_, idA := ctx.GetID("Button##1")
	_, idB := ctx.GetID("Button##2")
	if idA == idB || idA == 0 {
		t.Fatalf("expected distinct non-zero IDs, got %d and %d", idA, idB)
	}

	ctx.PushID("scope1")
	_, scoped1 := ctx.GetID("Item")
	ctx.PopID()

	ctx.PushID("scope2")
	_, scoped2 := ctx.GetID("Item")
	ctx.PopID()

	if scoped1 == scoped2 {
		t.Fatalf("expected PushID scopes to disambiguate identical labels")
	}

	// ### suffix should produce the same ID even when visible prefix changes.
	_, dyn1 := ctx.GetID("Frame 1###status")
	_, dyn2 := ctx.GetID("Frame 2###status")
	if dyn1 != dyn2 {
		t.Fatalf("expected ### stable ID to match across dynamic prefixes")
	}
	ctx.EndFrame()
}

func TestButtonLifecycleAndBubbleTeaInput(t *testing.T) {
	ctx := New(WithScreenSize(80, 24))

	// Frame 1: Move mouse over button at (2, 1) and press left mouse button.
	ctx.ProcessInput(tea.MouseClickMsg{X: 2, Y: 1, Button: tea.MouseLeft})
	ctx.BeginFrame()
	clicked := ctx.Button("Pan Camera")
	if clicked {
		t.Fatalf("button should not fire on press before release")
	}
	if !ctx.IsItemHovered() || !ctx.IsItemActive() {
		t.Fatalf("expected button to be both Hot and Active on mouse press")
	}
	if !ctx.WantCaptureMouse() {
		t.Fatalf("expected WantCaptureMouse() = true while button is Hot/Active")
	}
	ctx.EndFrame()

	// Frame 2: Release mouse button over the same widget -> triggers click!
	ctx.ProcessInput(tea.MouseReleaseMsg{X: 2, Y: 1, Button: tea.MouseLeft})
	ctx.BeginFrame()
	clicked = ctx.Button("Pan Camera")
	if !clicked {
		t.Fatalf("expected button to return true on release while Hot and Active")
	}
	ctx.EndFrame()
	if ctx.Active != 0 {
		t.Fatalf("expected Active to reset to 0 after release, got %d", ctx.Active)
	}
}

func TestLayoutHorizontalAndSameLine(t *testing.T) {
	ctx := New(WithScreenSize(80, 24), WithTheme(HUDTheme()))
	ctx.BeginFrame()

	ctx.BeginHorizontal()
	ctx.Button("Left")
	leftRect := ctx.LastWidgetRect
	ctx.Button("Right")
	rightRect := ctx.LastWidgetRect
	ctx.EndHorizontal()

	if rightRect.Y != leftRect.Y || rightRect.X <= leftRect.Right() {
		t.Fatalf("expected horizontal layout to place Right (%+v) beside Left (%+v)", rightRect, leftRect)
	}

	ctx.Button("First")
	firstRect := ctx.LastWidgetRect
	ctx.SameLine()
	ctx.Button("Second")
	secondRect := ctx.LastWidgetRect

	if secondRect.Y != firstRect.Y || secondRect.X <= firstRect.Right() {
		t.Fatalf("expected SameLine to place Second (%+v) on same row as First (%+v)", secondRect, firstRect)
	}
	ctx.EndFrame()
}

func TestWidgetsCheckboxRadioSliderTextInput(t *testing.T) {
	ctx := New(WithScreenSize(60, 20), WithTheme(HUDTheme()))
	wireframe := false
	mode := 0
	fov := 60.0
	name := "spot"

	// Frame 1: Render widgets at known locations.
	ctx.BeginFrame()
	ctx.Checkbox("Wireframe", &wireframe)
	chkRect := ctx.LastWidgetRect

	ctx.RadioButtonInt("GPU", &mode, 0)
	ctx.RadioButtonInt("Software", &mode, 1)
	radioSoftRect := ctx.LastWidgetRect

	ctx.SliderFloat("FOV", &fov, 30, 120, "%.0f")
	sliderRect := ctx.LastWidgetRect

	ctx.TextInput("Model", &name, 14)
	inputRect := ctx.LastWidgetRect
	ctx.EndFrame()

	// Click Checkbox (press + release).
	ctx.ProcessInput(tea.MouseClickMsg{X: chkRect.X + 1, Y: chkRect.Y, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.Checkbox("Wireframe", &wireframe)
	ctx.EndFrame()

	ctx.ProcessInput(tea.MouseReleaseMsg{X: chkRect.X + 1, Y: chkRect.Y, Button: tea.MouseLeft})
	ctx.BeginFrame()
	if !ctx.Checkbox("Wireframe", &wireframe) || !wireframe {
		t.Fatalf("expected Checkbox to toggle wireframe to true")
	}
	ctx.EndFrame()

	// Click RadioButton "Software".
	ctx.ProcessInput(tea.MouseClickMsg{X: radioSoftRect.X + 1, Y: radioSoftRect.Y, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.Checkbox("Wireframe", &wireframe)
	ctx.RadioButtonInt("GPU", &mode, 0)
	ctx.RadioButtonInt("Software", &mode, 1)
	ctx.EndFrame()

	ctx.ProcessInput(tea.MouseReleaseMsg{X: radioSoftRect.X + 1, Y: radioSoftRect.Y, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.Checkbox("Wireframe", &wireframe)
	ctx.RadioButtonInt("GPU", &mode, 0)
	if !ctx.RadioButtonInt("Software", &mode, 1) || mode != 1 {
		t.Fatalf("expected RadioButtonInt to select mode 1, got %d", mode)
	}
	ctx.EndFrame()

	// Drag SliderFloat to ~75%.
	ctx.ProcessInput(tea.MouseClickMsg{X: sliderRect.X + 10, Y: sliderRect.Y, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.Checkbox("Wireframe", &wireframe)
	ctx.RadioButtonInt("GPU", &mode, 0)
	ctx.RadioButtonInt("Software", &mode, 1)
	if !ctx.SliderFloat("FOV", &fov, 30, 120, "%.0f") || fov <= 30 {
		t.Fatalf("expected SliderFloat value to update on click/drag, got %v", fov)
	}
	ctx.EndFrame()
	ctx.ProcessInput(tea.MouseReleaseMsg{X: sliderRect.X + 10, Y: sliderRect.Y, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.EndFrame()

	// Focus TextInput and type "-cow".
	ctx.ProcessInput(tea.MouseClickMsg{X: inputRect.X + 1, Y: inputRect.Y, Button: tea.MouseLeft})
	ctx.ProcessInput(tea.MouseReleaseMsg{X: inputRect.X + 1, Y: inputRect.Y, Button: tea.MouseLeft})
	ctx.ProcessInput(tea.KeyPressMsg{Code: '-', Text: "-"})
	ctx.ProcessInput(tea.KeyPressMsg{Code: 'c', Text: "c"})
	ctx.BeginFrame()
	ctx.Checkbox("Wireframe", &wireframe)
	ctx.RadioButtonInt("GPU", &mode, 0)
	ctx.RadioButtonInt("Software", &mode, 1)
	ctx.SliderFloat("FOV", &fov, 30, 120, "%.0f")
	if !ctx.TextInput("Model", &name, 14) || name != "spot-c" {
		t.Fatalf("expected TextInput to append typed characters, got %q", name)
	}
	if !ctx.WantCaptureKeyboard() {
		t.Fatalf("expected WantCaptureKeyboard() = true when TextInput is focused")
	}
	ctx.EndFrame()
}

func TestPanelDragCollapseAndCapture(t *testing.T) {
	ctx := New(WithScreenSize(80, 24), WithTheme(HUDTheme()))
	panelBounds := NewRect(4, 2, 26, 10)

	// Hover inside panel -> WantCaptureMouse should be true even on empty panel area.
	ctx.ProcessInput(tea.MouseMotionMsg{X: 10, Y: 6})
	ctx.BeginFrame()
	if !ctx.BeginPanel("Scene HUD", panelBounds) {
		t.Fatalf("expected panel to start open")
	}
	ctx.Label("Triangles: 12,480")
	ctx.EndPanel()
	ctx.EndFrame()

	if !ctx.WantCaptureMouse() {
		t.Fatalf("expected WantCaptureMouse() = true when cursor is inside open panel")
	}

	// Drag panel by its title bar from (12, 2) to (20, 5).
	ctx.ProcessInput(tea.MouseClickMsg{X: 12, Y: 2, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.BeginPanel("Scene HUD", panelBounds)
	ctx.EndPanel()
	ctx.EndFrame()

	ctx.ProcessInput(tea.MouseMotionMsg{X: 20, Y: 5, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.BeginPanel("Scene HUD", panelBounds)
	ctx.EndPanel()
	ctx.EndFrame()

	ctx.ProcessInput(tea.MouseReleaseMsg{X: 20, Y: 5, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.BeginPanel("Scene HUD", panelBounds)
	ctx.EndPanel()
	ctx.EndFrame()

	// Panel should now have moved by (+8, +3) to (12, 5).
	// Clicking the collapse toggle at (12+1, 5) should collapse it.
	ctx.ProcessInput(tea.MouseClickMsg{X: 13, Y: 5, Button: tea.MouseLeft})
	ctx.BeginFrame()
	ctx.BeginPanel("Scene HUD", panelBounds)
	ctx.EndPanel()
	ctx.EndFrame()

	ctx.ProcessInput(tea.MouseReleaseMsg{X: 13, Y: 5, Button: tea.MouseLeft})
	ctx.BeginFrame()
	open := ctx.BeginPanel("Scene HUD", panelBounds)
	ctx.EndPanel()
	ctx.EndFrame()
	if open {
		t.Fatalf("expected panel to collapse after clicking collapse toggle")
	}
}

func TestGlyphAndRasterBackendsRenderSameDrawList(t *testing.T) {
	ctx := New(WithScreenSize(40, 12), WithTheme(HUDTheme()))
	fov := 75.0
	shadows := true

	ctx.BeginFrame()
	if ctx.BeginPanel("Controls", NewRect(2, 1, 30, 9)) {
		ctx.Label("Renderer: WebGPU")
		ctx.Checkbox("Shadows", &shadows)
		ctx.SliderFloat("FOV", &fov, 30, 120, "%.0f°")
		ctx.ProgressBar(0.65, 20, "65%")
		ctx.Separator()
		ctx.Button("Reset Camera")
	}
	ctx.EndPanel()
	ctx.EndFrame()

	// 1. Render with GlyphBackend.
	glyph := NewGlyphBackend(40, 12)
	out := glyph.RenderToString(ctx.DrawList)
	if !strings.Contains(out, "Controls") || !strings.Contains(out, "Renderer: WebGPU") || !strings.Contains(out, "Reset Camera") {
		t.Fatalf("GlyphBackend output missing expected widget text:\n%s", out)
	}

	// 2. Render with RasterBackend over a solid background image.
	raster := NewRasterBackend(9, 18)
	bg := image.NewRGBA(image.Rect(0, 0, 40*9, 12*18))
	baseCol := color.RGBA{R: 10, G: 80, B: 40, A: 255}
	for y := 0; y < bg.Bounds().Dy(); y++ {
		for x := 0; x < bg.Bounds().Dx(); x++ {
			bg.SetRGBA(x, y, baseCol)
		}
	}
	raster.RenderOver(bg, ctx.DrawList)

	// Verify that pixels inside the panel (e.g. cell 5,3 -> pixel 45,54) were alpha-blended
	// and differ from the raw background color, while pixels outside (cell 0,0 -> pixel 2,2)
	// remain untouched.
	if got := bg.RGBAAt(2, 2); got != baseCol {
		t.Fatalf("expected outside pixel (2,2) to remain %+v, got %+v", baseCol, got)
	}
	if got := bg.RGBAAt(45, 54); got == baseCol {
		t.Fatalf("expected inside-panel pixel (45,54) to be alpha-blended over background, still got %+v", got)
	}
}

func TestCatalogUXElements(t *testing.T) {
	ctx := New(WithScreenSize(80, 30), WithTheme(CyberpunkTheme()))
	activeTab := 0
	comboIdx := 0
	listIdx := 0
	col := RGB(100, 150, 200)
	wave := []float64{0.1, 0.4, 0.8, 0.5, 0.9, 0.2}

	var comboRect Rect
	ctx.BeginFrame()
	if ctx.BeginPanel("Catalog", NewRect(1, 1, 60, 26)) {
		ctx.TabBar("tabs", &activeTab, []string{"Controls", "Plots", "Trees"})
		ctx.Combo("Quality", &comboIdx, []string{"Low", "Medium", "High"})
		comboRect = ctx.LastWidgetRect

		ctx.ListBox("Shaders", &listIdx, []string{"pbr.wgsl", "fur.wgsl", "fog.wgsl"}, 5)
		if ctx.TreeNode("Root", true) {
			ctx.KeyValue("Triangles", "12,480")
			ctx.TreePop()
		}
		ctx.PlotLines("Wave", wave, 30, 2, "60 FPS")
		ctx.PlotHistogram("Hist", wave, 30, 2, "Pass ms")
		ctx.ColorEdit("Tint", &col)
		ctx.TableRow([]string{"Pass", "Cost"}, []int{12, 10}, true)
		ctx.TableRow([]string{"Shadow", "1.2ms"}, []int{12, 10}, false)
	}
	ctx.EndPanel()
	ctx.EndFrame()

	// Click Combo to advance selection
	ctx.ProcessInput(tea.MouseClickMsg{X: comboRect.X + 4, Y: comboRect.Y, Button: tea.MouseLeft})
	ctx.BeginFrame()
	if ctx.BeginPanel("Catalog", NewRect(1, 1, 60, 26)) {
		ctx.TabBar("tabs", &activeTab, []string{"Controls", "Plots", "Trees"})
		ctx.Combo("Quality", &comboIdx, []string{"Low", "Medium", "High"})
	}
	ctx.EndPanel()
	ctx.EndFrame()

	ctx.ProcessInput(tea.MouseReleaseMsg{X: comboRect.X + 4, Y: comboRect.Y, Button: tea.MouseLeft})
	ctx.BeginFrame()
	if ctx.BeginPanel("Catalog", NewRect(1, 1, 60, 26)) {
		ctx.TabBar("tabs", &activeTab, []string{"Controls", "Plots", "Trees"})
		if !ctx.Combo("Quality", &comboIdx, []string{"Low", "Medium", "High"}) || comboIdx != 1 {
			t.Fatalf("expected Combo click to advance comboIdx to 1, got %d", comboIdx)
		}
	}
	ctx.EndPanel()
	ctx.EndFrame()

	// Render full catalog frame through both GlyphBackend and RasterBackend
	ctx.BeginFrame()
	if ctx.BeginPanel("Catalog", NewRect(1, 1, 60, 26)) {
		ctx.TabBar("tabs", &activeTab, []string{"Controls", "Plots", "Trees"})
		ctx.PlotLines("Wave", wave, 30, 2, "60 FPS")
		ctx.PlotHistogram("Hist", wave, 30, 2, "Pass ms")
		ctx.ColorEdit("Tint", &col)
	}
	ctx.EndPanel()
	ctx.EndFrame()

	glyphOut := NewGlyphBackend(80, 30).RenderToString(ctx.DrawList)
	if !strings.Contains(glyphOut, "Controls") || !strings.Contains(glyphOut, "#6496C8") {
		t.Fatalf("expected GlyphBackend output to contain tab and ColorEdit hex, got:\n%s", glyphOut)
	}
	img := NewRasterBackend(9, 18).RenderToImage(80, 30, ctx.DrawList)
	if img.Bounds().Dx() != 720 || img.Bounds().Dy() != 540 {
		t.Fatalf("unexpected raster image dimensions: %v", img.Bounds())
	}
}
