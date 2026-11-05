// Copyright (c) 2026 Neomantra Corp.

package ntimgui

// PanelConfig configures a floating or docked panel created with BeginPanel.
type PanelConfig struct {
	Collapsible bool
	Draggable   bool
	Scrollable  bool
	PaddingX    int
	PaddingY    int
	SpacingY    int
}

// PanelOption modifies a PanelConfig.
type PanelOption func(*PanelConfig)

// WithCollapsible enables or disables collapsing the panel via its title bar.
func WithCollapsible(collapsible bool) PanelOption {
	return func(cfg *PanelConfig) {
		cfg.Collapsible = collapsible
	}
}

// WithDraggable enables or disables dragging the panel by its title bar.
func WithDraggable(draggable bool) PanelOption {
	return func(cfg *PanelConfig) {
		cfg.Draggable = draggable
	}
}

// WithScrollable enables or disables mouse-wheel vertical scrolling inside the panel.
func WithScrollable(scrollable bool) PanelOption {
	return func(cfg *PanelConfig) {
		cfg.Scrollable = scrollable
	}
}

// WithPanelPadding sets the horizontal and vertical inner padding in cells.
func WithPanelPadding(padX, padY int) PanelOption {
	return func(cfg *PanelConfig) {
		cfg.PaddingX = max(0, padX)
		cfg.PaddingY = max(0, padY)
	}
}

// WithPanelSpacing sets the vertical spacing between widgets inside the panel.
func WithPanelSpacing(spacingY int) PanelOption {
	return func(cfg *PanelConfig) {
		cfg.SpacingY = max(0, spacingY)
	}
}

type panelFrame struct {
	id         ID
	bounds     Rect
	inner      Rect
	open       bool
	scrollable bool
	scrollY    int
	prevLayer  Layer
}

// BeginPanel starts a floating HUD panel or window at bounds.
// It returns true when the panel is open and its child widgets should be evaluated.
// Callers must always pair BeginPanel with a matching EndPanel call.
func (ctx *Context) BeginPanel(title string, bounds Rect, opts ...PanelOption) bool {
	cfg := PanelConfig{
		Collapsible: true,
		Draggable:   true,
		Scrollable:  true,
		PaddingX:    1,
		PaddingY:    1,
		SpacingY:    0,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	displayTitle, id := ctx.GetID(title)
	openID := hashString(id, "##open")
	posID := hashString(id, "##pos")
	dragID := hashString(id, "##drag")
	dragOffID := hashString(id, "##drag_off")
	scrollID := hashString(id, "##scroll")

	pos := ctx.Storage.Point(posID, bounds.Min())
	if !cfg.Draggable {
		pos = bounds.Min()
	}
	panelRect := Rect{X: pos.X, Y: pos.Y, W: bounds.W, H: bounds.H}
	open := ctx.Storage.Bool(openID, true)
	if !cfg.Collapsible {
		open = true
	}

	titleBarRect := Rect{X: panelRect.X, Y: panelRect.Y, W: panelRect.W, H: 1}
	collapseRect := Rect{X: panelRect.X + 1, Y: panelRect.Y, W: 3, H: 1}

	// Handle collapse toggle click on the left side of the title bar.
	if cfg.Collapsible && collapseRect.Contains(ctx.Input.MouseX, ctx.Input.MouseY) {
		if ctx.Active == 0 || ctx.Active == openID {
			ctx.Hot = openID
			if ctx.Input.MousePressed {
				ctx.Active = openID
			}
		}
		if ctx.Input.MouseReleased && ctx.Hot == openID && ctx.Active == openID {
			open = !open
			ctx.Storage.SetBool(openID, open)
			ctx.Active = 0
		}
	} else if cfg.Draggable && titleBarRect.Contains(ctx.Input.MouseX, ctx.Input.MouseY) {
		if ctx.Active == 0 || ctx.Active == dragID {
			ctx.Hot = dragID
			if ctx.Input.MousePressed {
				ctx.Active = dragID
				ctx.Storage.SetPoint(dragOffID, Point{
					X: ctx.Input.MouseX - panelRect.X,
					Y: ctx.Input.MouseY - panelRect.Y,
				})
			}
		}
	}

	if cfg.Draggable && ctx.Active == dragID && ctx.Input.MouseDown {
		off := ctx.Storage.Point(dragOffID, Point{})
		newX := ctx.Input.MouseX - off.X
		newY := ctx.Input.MouseY - off.Y
		newX = max(0, min(newX, max(0, ctx.screenW-panelRect.W)))
		newY = max(0, min(newY, max(0, ctx.screenH-1)))
		pos = Point{X: newX, Y: newY}
		ctx.Storage.SetPoint(posID, pos)
		panelRect.X = pos.X
		panelRect.Y = pos.Y
		titleBarRect.X = pos.X
		titleBarRect.Y = pos.Y
	}

	prevLayer := ctx.DrawList.Layer()
	ctx.DrawList.SetLayer(LayerWindow)
	ctx.PushID(title)

	headerLabel := displayTitle
	if cfg.Collapsible {
		if open {
			headerLabel = "▾ " + displayTitle
		} else {
			headerLabel = "▸ " + displayTitle
		}
	}

	if !open {
		collapsedRect := Rect{X: panelRect.X, Y: panelRect.Y, W: panelRect.W, H: 1}
		ctx.hitRegions = append(ctx.hitRegions, collapsedRect)
		ctx.DrawList.PushClipRectUnclipped(ctx.Bounds())
		ctx.DrawList.AddRect(collapsedRect, ctx.Theme.PanelTitle)
		ctx.DrawList.AddText(collapsedRect.Inset(1, 0), headerLabel, AlignLeft, ctx.Theme.PanelTitle)
		ctx.PushLayout(Rect{X: panelRect.X, Y: panelRect.Y + 1, W: 0, H: 0}, FlowVertical, 1, 0)
		ctx.panelStack = append(ctx.panelStack, panelFrame{
			id:        id,
			bounds:    collapsedRect,
			inner:     Rect{},
			open:      false,
			prevLayer: prevLayer,
		})
		return false
	}

	ctx.hitRegions = append(ctx.hitRegions, panelRect)

	scrollY := ctx.Storage.Int(scrollID, 0)
	if cfg.Scrollable && panelRect.Contains(ctx.Input.MouseX, ctx.Input.MouseY) && ctx.Input.WheelY != 0 {
		scrollY = max(0, scrollY+ctx.Input.WheelY)
		ctx.Storage.SetInt(scrollID, scrollY)
	}

	ctx.DrawList.PushClipRectUnclipped(ctx.Bounds().Intersect(panelRect))
	ctx.DrawList.AddBox(panelRect, ctx.Theme.PanelBorder, headerLabel, ctx.Theme.Panel)

	inner := panelRect.Inset(max(1, cfg.PaddingX), max(1, cfg.PaddingY))
	ctx.DrawList.PushClipRect(inner)

	layoutBounds := inner
	layoutBounds.Y -= scrollY
	layoutBounds.H += scrollY
	ctx.PushLayout(layoutBounds, FlowVertical, 1, cfg.SpacingY)

	ctx.panelStack = append(ctx.panelStack, panelFrame{
		id:         id,
		bounds:     panelRect,
		inner:      inner,
		open:       true,
		scrollable: cfg.Scrollable,
		scrollY:    scrollY,
		prevLayer:  prevLayer,
	})
	return true
}

// EndPanel closes the current panel started by BeginPanel.
func (ctx *Context) EndPanel() {
	if len(ctx.panelStack) == 0 {
		return
	}
	pf := ctx.panelStack[len(ctx.panelStack)-1]
	ctx.panelStack = ctx.panelStack[:len(ctx.panelStack)-1]

	child := ctx.PopLayout()
	if pf.open {
		contentH := (child.MaxY - child.Bounds.Y)
		maxScroll := max(0, contentH-pf.inner.H)
		if pf.scrollY > maxScroll {
			scrollID := hashString(pf.id, "##scroll")
			ctx.Storage.SetInt(scrollID, maxScroll)
		}
		ctx.DrawList.PopClipRect() // inner clip
	}
	ctx.DrawList.PopClipRect() // panel clip
	ctx.PopID()
	ctx.DrawList.SetLayer(pf.prevLayer)
}
