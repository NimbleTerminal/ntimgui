// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	tea "charm.land/bubbletea/v2"
)

// KeyCode represents abstract navigation and editing keys handled by widgets.
type KeyCode uint8

const (
	KeyNone KeyCode = iota
	KeyTab
	KeyShiftTab
	KeyEnter
	KeySpace
	KeyEscape
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyBackspace
	KeyDelete
)

// InputState holds both continuous cursor state and per-frame edge events.
type InputState struct {
	// MouseX and MouseY are the current cursor coordinates in terminal cells.
	MouseX, MouseY int
	// MousePixelX and MousePixelY are optional sub-cell pixel coordinates.
	// When HasPixelPos is false, they are derived from MouseX/MouseY during hit-testing.
	MousePixelX, MousePixelY int
	HasPixelPos              bool

	// MouseDown is true while the primary mouse button is held down.
	MouseDown bool
	// MousePressed is true during the frame the primary mouse button was pressed.
	MousePressed bool
	// MouseReleased is true during the frame the primary mouse button was released.
	MouseReleased bool

	// WheelX and WheelY record scroll wheel ticks accumulated for this frame.
	WheelX, WheelY int

	// Keys contains navigation/action keys pressed since the last EndFrame.
	Keys []KeyCode
	// Text contains printable characters typed since the last EndFrame.
	Text string

	// OffsetX and OffsetY subtract a host viewport origin from incoming mouse coordinates.
	OffsetX, OffsetY int
}

// ResetFrame clears one-shot per-frame edge transitions while preserving
// continuous cursor position and button-down state.
func (in *InputState) ResetFrame() {
	in.MousePressed = false
	in.MouseReleased = false
	in.WheelX = 0
	in.WheelY = 0
	in.Keys = in.Keys[:0]
	in.Text = ""
}

// KeyPressed reports whether k was pressed during the current frame.
func (in *InputState) KeyPressed(k KeyCode) bool {
	for _, pressed := range in.Keys {
		if pressed == k {
			return true
		}
	}
	return false
}

// SetMousePos updates the cursor position in terminal cell coordinates.
func (in *InputState) SetMousePos(x, y int) {
	in.MouseX = x - in.OffsetX
	in.MouseY = y - in.OffsetY
}

// SetMousePixelPos updates optional pixel-level mouse coordinates alongside cell coordinates.
func (in *InputState) SetMousePixelPos(px, py, cellW, cellH int) {
	in.MousePixelX = px
	in.MousePixelY = py
	in.HasPixelPos = true
	if cellW > 0 && cellH > 0 {
		in.MouseX = px/cellW - in.OffsetX
		in.MouseY = py/cellH - in.OffsetY
	}
}

// ProcessInput translates a Bubble Tea v2 message into InputState mutations.
// It returns true if msg was a recognized input or window-size event.
func (ctx *Context) ProcessInput(msg tea.Msg) bool {
	switch m := msg.(type) {
	case tea.WindowSizeMsg:
		if m.Width > 0 && m.Height > 0 {
			ctx.SetScreenSize(m.Width, m.Height)
		}
		return true

	case tea.MouseClickMsg:
		ev := m.Mouse()
		ctx.Input.SetMousePos(ev.X, ev.Y)
		if ev.Button == tea.MouseLeft {
			ctx.Input.MousePressed = true
			ctx.Input.MouseDown = true
		}
		return true

	case tea.MouseMotionMsg:
		ev := m.Mouse()
		ctx.Input.SetMousePos(ev.X, ev.Y)
		if ev.Button == tea.MouseLeft {
			ctx.Input.MouseDown = true
		}
		return true

	case tea.MouseReleaseMsg:
		ev := m.Mouse()
		ctx.Input.SetMousePos(ev.X, ev.Y)
		if ev.Button == tea.MouseLeft || ev.Button == tea.MouseNone || ctx.Input.MouseDown {
			ctx.Input.MouseReleased = true
			ctx.Input.MouseDown = false
		}
		return true

	case tea.MouseWheelMsg:
		ev := m.Mouse()
		ctx.Input.SetMousePos(ev.X, ev.Y)
		switch ev.Button {
		case tea.MouseWheelUp:
			ctx.Input.WheelY--
		case tea.MouseWheelDown:
			ctx.Input.WheelY++
		case tea.MouseWheelLeft:
			ctx.Input.WheelX--
		case tea.MouseWheelRight:
			ctx.Input.WheelX++
		}
		return true

	case tea.KeyPressMsg:
		switch m.Code {
		case tea.KeyTab:
			if m.Mod&tea.ModShift != 0 {
				ctx.Input.Keys = append(ctx.Input.Keys, KeyShiftTab)
			} else {
				ctx.Input.Keys = append(ctx.Input.Keys, KeyTab)
			}
		case tea.KeyEnter:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyEnter)
		case tea.KeySpace:
			ctx.Input.Keys = append(ctx.Input.Keys, KeySpace)
			ctx.Input.Text += " "
		case tea.KeyEscape:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyEscape)
		case tea.KeyUp:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyUp)
		case tea.KeyDown:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyDown)
		case tea.KeyLeft:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyLeft)
		case tea.KeyRight:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyRight)
		case tea.KeyHome:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyHome)
		case tea.KeyEnd:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyEnd)
		case tea.KeyBackspace:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyBackspace)
		case tea.KeyDelete:
			ctx.Input.Keys = append(ctx.Input.Keys, KeyDelete)
		default:
			if m.Text != "" {
				ctx.Input.Text += m.Text
			} else if m.Code >= 32 && m.Code != 127 && m.Mod == 0 {
				ctx.Input.Text += string(m.Code)
			}
		}
		return true
	}
	return false
}
