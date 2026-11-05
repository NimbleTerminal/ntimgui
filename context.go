// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import "slices"

// Option configures a Context at construction time.
type Option func(*Context)

// WithScreenSize sets the initial terminal grid dimensions in cells.
func WithScreenSize(width, height int) Option {
	return func(ctx *Context) {
		ctx.SetScreenSize(width, height)
	}
}

// WithTheme configures the visual Theme for the Context.
func WithTheme(theme Theme) Option {
	return func(ctx *Context) {
		ctx.Theme = theme
	}
}

// Context manages immediate-mode UI state, layout evaluation, and DrawList generation.
type Context struct {
	Input    InputState
	Theme    Theme
	DrawList *DrawList
	Storage  *Storage

	// Hot is the ID of the widget under the cursor during the current frame.
	Hot ID
	// Active is the ID of the widget currently being interacted with across frames.
	Active ID
	// Focus is the ID of the widget currently receiving keyboard focus.
	Focus ID

	// LastWidgetID and LastWidgetRect describe the most recently evaluated widget.
	LastWidgetID   ID
	LastWidgetRect Rect

	screenW, screenH int
	inFrame          bool
	idStack          []ID
	layoutStack      []LayoutScope
	panelStack       []panelFrame
	hitRegions       []Rect
	focusOrder       []ID
	editingText      bool
}

// New constructs a new immediate-mode UI Context.
func New(opts ...Option) *Context {
	ctx := &Context{
		Theme:   DefaultTheme(),
		Storage: newStorage(),
		screenW: 80,
		screenH: 24,
	}
	ctx.DrawList = NewDrawList(ctx)
	for _, opt := range opts {
		if opt != nil {
			opt(ctx)
		}
	}
	return ctx
}

// Bounds returns the root screen rectangle in terminal cell units.
func (ctx *Context) Bounds() Rect {
	return Rect{X: 0, Y: 0, W: ctx.screenW, H: ctx.screenH}
}

// SetScreenSize updates the root screen dimensions in terminal cell units.
func (ctx *Context) SetScreenSize(width, height int) {
	if width > 0 {
		ctx.screenW = width
	}
	if height > 0 {
		ctx.screenH = height
	}
}

// ScreenSize returns the current screen dimensions in terminal cell units.
func (ctx *Context) ScreenSize() (width, height int) {
	return ctx.screenW, ctx.screenH
}

// BeginFrame resets per-frame layout, DrawList, and Hot state before widget declarations.
func (ctx *Context) BeginFrame() {
	ctx.inFrame = true
	ctx.Hot = 0
	ctx.LastWidgetID = 0
	ctx.LastWidgetRect = Rect{}
	ctx.editingText = false
	ctx.idStack = append(ctx.idStack[:0], ID(fnvOffset64))
	ctx.layoutStack = ctx.layoutStack[:0]
	ctx.panelStack = ctx.panelStack[:0]
	ctx.hitRegions = ctx.hitRegions[:0]
	ctx.focusOrder = ctx.focusOrder[:0]

	root := ctx.Bounds()
	ctx.DrawList.Reset(root)
	ctx.PushLayout(root, FlowVertical, 1, 0)
}

// EndFrame finalizes focus navigation, clears released active states, and resets
// one-shot input edge flags for the next frame.
func (ctx *Context) EndFrame() {
	if !ctx.inFrame {
		return
	}
	ctx.inFrame = false

	// Close any unclosed panels gracefully.
	for len(ctx.panelStack) > 0 {
		ctx.EndPanel()
	}

	// Handle Tab / Shift+Tab focus cycling across focusable widgets declared this frame.
	if len(ctx.focusOrder) > 0 {
		if ctx.Input.KeyPressed(KeyTab) {
			idx := slices.Index(ctx.focusOrder, ctx.Focus)
			if idx < 0 || idx+1 >= len(ctx.focusOrder) {
				ctx.Focus = ctx.focusOrder[0]
			} else {
				ctx.Focus = ctx.focusOrder[idx+1]
			}
		} else if ctx.Input.KeyPressed(KeyShiftTab) {
			idx := slices.Index(ctx.focusOrder, ctx.Focus)
			if idx <= 0 {
				ctx.Focus = ctx.focusOrder[len(ctx.focusOrder)-1]
			} else {
				ctx.Focus = ctx.focusOrder[idx-1]
			}
		}
	}
	if ctx.Input.KeyPressed(KeyEscape) || (ctx.Input.MousePressed && ctx.Hot == 0) {
		ctx.Focus = 0
		ctx.editingText = false
	}

	// If the mouse button is no longer held down after frame resolution, clear Active.
	if ctx.Input.MouseReleased || !ctx.Input.MouseDown {
		ctx.Active = 0
	}

	ctx.Input.ResetFrame()
}

// PushID hashes s with the current ID seed and pushes it onto the ID stack.
func (ctx *Context) PushID(s string) {
	seed := ctx.currentSeed()
	ctx.idStack = append(ctx.idStack, hashString(seed, s))
}

// PushIDInt hashes i with the current ID seed and pushes it onto the ID stack.
func (ctx *Context) PushIDInt(i int) {
	seed := ctx.currentSeed()
	ctx.idStack = append(ctx.idStack, hashInt(seed, i))
}

// PopID removes the top seed from the ID stack.
func (ctx *Context) PopID() {
	if len(ctx.idStack) > 1 {
		ctx.idStack = ctx.idStack[:len(ctx.idStack)-1]
	}
}

func (ctx *Context) currentSeed() ID {
	if len(ctx.idStack) == 0 {
		return ID(fnvOffset64)
	}
	return ctx.idStack[len(ctx.idStack)-1]
}

// GetID computes a widget ID for label within the current ID stack scope and
// returns both the stripped display label and the hashed ID.
func (ctx *Context) GetID(label string) (display string, id ID) {
	display, hashKey := ParseLabel(label)
	return display, hashString(ctx.currentSeed(), hashKey)
}

// SetFocus explicitly assigns keyboard focus to id.
func (ctx *Context) SetFocus(id ID) {
	ctx.Focus = id
}

// ClearFocus removes keyboard focus from all widgets.
func (ctx *Context) ClearFocus() {
	ctx.Focus = 0
}

// IsItemHovered reports whether the most recently declared widget is Hot.
func (ctx *Context) IsItemHovered() bool {
	return ctx.LastWidgetID != 0 && ctx.Hot == ctx.LastWidgetID
}

// IsItemActive reports whether the most recently declared widget is Active.
func (ctx *Context) IsItemActive() bool {
	return ctx.LastWidgetID != 0 && ctx.Active == ctx.LastWidgetID
}

// IsItemFocused reports whether the most recently declared widget has keyboard focus.
func (ctx *Context) IsItemFocused() bool {
	return ctx.LastWidgetID != 0 && ctx.Focus == ctx.LastWidgetID
}

// WantCaptureMouse reports whether the UI is currently interacting with or
// hovering over a widget or panel. Host applications (such as nt3d viewports)
// can check this to avoid orbiting or picking when clicking on a UI overlay.
func (ctx *Context) WantCaptureMouse() bool {
	if ctx.Hot != 0 || ctx.Active != 0 {
		return true
	}
	mx, my := ctx.Input.MouseX, ctx.Input.MouseY
	for _, r := range ctx.hitRegions {
		if r.Contains(mx, my) {
			return true
		}
	}
	return false
}

// WantCaptureKeyboard reports whether a text input widget is actively editing text.
// Non-text widgets only use navigation keys (Tab, Space/Enter, Arrows) and do not
// block single-letter application shortcuts (like 'q' or 'g').
func (ctx *Context) WantCaptureKeyboard() bool {
	return ctx.editingText
}

func (ctx *Context) registerFocusable(id ID) {
	ctx.focusOrder = append(ctx.focusOrder, id)
}

func (ctx *Context) isMouseInRect(bounds Rect) bool {
	clipped := bounds.Intersect(ctx.DrawList.CurrentClip())
	return clipped.Contains(ctx.Input.MouseX, ctx.Input.MouseY)
}

func (ctx *Context) widgetStyleFor(id ID) WidgetStyle {
	switch {
	case ctx.Active == id:
		return ctx.Theme.Active
	case ctx.Hot == id:
		return ctx.Theme.Hover
	case ctx.Focus == id:
		return ctx.Theme.Focus
	default:
		return ctx.Theme.Normal
	}
}
