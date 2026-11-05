// Copyright (c) 2026 Neomantra Corp.

// A standalone Ebitengine window: no terminal, ntebitengine, or GPU readback.
package main

import (
	"image/color"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"nimbleterminal.dev/ntimgui"
	ntimguiebiten "nimbleterminal.dev/ntimgui/ebiten"
)

// A fixed logical grid keeps hit testing and rendering in agreement as the
// window resizes. The resulting pixels-per-cell scales can be fractional.
const columns, rows = 80, 30

type game struct {
	ui            *ntimgui.Context
	backend       *ntimguiebiten.Backend
	commands      []ntimgui.DrawCmd
	width, height int
	speed, phase  float64
	paused, light bool
}

func (g *game) Update() error {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		if g.backend != nil {
			g.backend.Dispose()
		}
		return ebiten.Termination
	}
	// Map Ebitengine input directly to ntimgui's logical cells. No Bubble Tea loop
	// or ProcessInput adapter is necessary in an ordinary Ebitengine game.
	x, y := ebiten.CursorPosition()
	in := &g.ui.Input
	in.MouseX = int(math.Floor(float64(x) * columns / float64(g.width)))
	in.MouseY = int(math.Floor(float64(y) * rows / float64(g.height)))
	in.MouseDown = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	in.MousePressed = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	in.MouseReleased = inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft)
	_, wheel := ebiten.Wheel()
	in.WheelY = -int(wheel)
	for _, key := range []struct {
		ebiten  ebiten.Key
		ntimgui ntimgui.KeyCode
	}{
		{ebiten.KeyTab, ntimgui.KeyTab}, {ebiten.KeyEnter, ntimgui.KeyEnter},
		{ebiten.KeySpace, ntimgui.KeySpace}, {ebiten.KeyArrowLeft, ntimgui.KeyLeft},
		{ebiten.KeyArrowRight, ntimgui.KeyRight}, {ebiten.KeyArrowUp, ntimgui.KeyUp},
		{ebiten.KeyArrowDown, ntimgui.KeyDown},
	} {
		if inpututil.IsKeyJustPressed(key.ebiten) {
			code := key.ntimgui
			if code == ntimgui.KeyTab && ebiten.IsKeyPressed(ebiten.KeyShift) {
				code = ntimgui.KeyShiftTab
			}
			in.Keys = append(in.Keys, code)
		}
	}
	clicked := in.MousePressed // EndFrame consumes input edges.
	if !g.paused {
		g.phase += g.speed / float64(ebiten.TPS())
	}
	theme := ntimgui.HUDTheme()
	if g.light {
		theme = ntimgui.LightTheme()
	}
	g.ui.Theme = theme
	g.ui.BeginFrame()
	if g.ui.BeginPanel("Ebitengine + ntimgui", ntimgui.Rect{X: 2, Y: 2, W: 34, H: 15}) {
		g.ui.Label("Drag this panel by its title")
		g.ui.SliderFloat("Speed", &g.speed, .2, 3, "%.1fx")
		g.ui.Checkbox("Pause animation", &g.paused)
		g.ui.Checkbox("Light theme", &g.light)
		if g.ui.Button("Reset") {
			g.speed = 1
			g.phase = 0
			g.paused = false
		}
		g.ui.Separator()
		g.ui.ProgressBar((math.Sin(g.phase)+1)/2, 30, "Motion")
		g.ui.Label("Click the scene to pause")
		g.ui.LabelMuted("Tab / arrows / Space: widgets")
		g.ui.LabelMuted("Esc: close window")
	}
	g.ui.EndPanel()
	g.ui.EndFrame()
	if clicked && !g.ui.WantCaptureMouse() {
		g.paused = !g.paused
	}
	// Update and Draw are serialized by Ebitengine. Snapshot is optional here,
	// but demonstrates the same owned command handoff used by threaded hosts.
	g.commands = g.ui.DrawList.Snapshot()
	return nil
}

func (g *game) Draw(screen *ebiten.Image) {
	// Font and renderer resources are created and reused on the callback side.
	if g.backend == nil {
		g.backend = ntimguiebiten.New()
	}
	screen.Fill(color.RGBA{R: 18, G: 24, B: 38, A: 255})
	w, h := float64(g.width), float64(g.height)
	for i := 0; i < 6; i++ {
		angle := g.phase + float64(i)*math.Pi/3
		x := w*.64 + math.Cos(angle)*w*.23
		y := h*.53 + math.Sin(angle)*h*.31
		vector.FillCircle(screen, float32(x), float32(y), float32(18+i*6), color.NRGBA{R: uint8(75 + i*25), G: 160, B: 235, A: 180}, true)
	}
	g.backend.Render(screen, g.commands, ntimguiebiten.RenderOptions{
		ScaleX: w / columns, ScaleY: h / rows,
	})
}

func (g *game) Layout(width, height int) (int, int) {
	g.width, g.height = max(1, width), max(1, height)
	return g.width, g.height
}

func main() {
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowSizeLimits(640, 480, -1, -1)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowTitle("ntimgui — standalone Ebitengine")
	g := &game{ui: ntimgui.New(ntimgui.WithScreenSize(columns, rows)), width: 800, height: 600, speed: 1}
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
