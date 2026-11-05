// Copyright (c) 2026 Neomantra Corp.

package ntimgui

// FlowDir controls how sequential widgets advance the layout cursor.
type FlowDir uint8

const (
	FlowVertical FlowDir = iota
	FlowHorizontal
)

// LayoutScope stores cursor state for the current container or layout group.
type LayoutScope struct {
	Bounds      Rect
	Cursor      Point
	Flow        FlowDir
	SpacingX    int
	SpacingY    int
	Indent      int
	LineHeight  int
	MaxX, MaxY  int
	LastRect    Rect
	SameLineReq bool
	ScrollY     int
}

// PushLayout pushes a new layout scope onto the context's layout stack.
func (ctx *Context) PushLayout(bounds Rect, flow FlowDir, spacingX, spacingY int) {
	scope := LayoutScope{
		Bounds:   bounds,
		Cursor:   bounds.Min(),
		Flow:     flow,
		SpacingX: spacingX,
		SpacingY: spacingY,
		MaxX:     bounds.X,
		MaxY:     bounds.Y,
	}
	ctx.layoutStack = append(ctx.layoutStack, scope)
}

// PopLayout pops the current layout scope and advances the parent scope by the
// bounding box consumed by the child scope.
func (ctx *Context) PopLayout() LayoutScope {
	if len(ctx.layoutStack) <= 1 {
		return ctx.layoutStack[0]
	}
	child := ctx.layoutStack[len(ctx.layoutStack)-1]
	ctx.layoutStack = ctx.layoutStack[:len(ctx.layoutStack)-1]
	return child
}

// currentLayout returns a pointer to the active LayoutScope.
func (ctx *Context) currentLayout() *LayoutScope {
	return &ctx.layoutStack[len(ctx.layoutStack)-1]
}

// AllocateRect claims a bounding rectangle of (width, height) terminal cells
// from the current layout cursor and advances the cursor according to flow.
func (ctx *Context) AllocateRect(width, height int) Rect {
	l := ctx.currentLayout()
	if width < 0 {
		width = 0
	}
	if height < 0 {
		height = 0
	}

	var x, y int
	if l.SameLineReq && !l.LastRect.Empty() {
		x = l.LastRect.Right() + max(1, l.SpacingX)
		y = l.LastRect.Y
		l.SameLineReq = false
	} else if l.Flow == FlowHorizontal {
		x = l.Cursor.X
		y = l.Cursor.Y
	} else {
		x = l.Bounds.X + l.Indent
		y = l.Cursor.Y
	}

	r := Rect{X: x, Y: y, W: width, H: height}
	l.LastRect = r
	if r.Right() > l.MaxX {
		l.MaxX = r.Right()
	}
	if r.Bottom() > l.MaxY {
		l.MaxY = r.Bottom()
	}

	if l.Flow == FlowHorizontal {
		l.Cursor.X = r.Right() + l.SpacingX
		if height > l.LineHeight {
			l.LineHeight = height
		}
	} else {
		rowBottom := r.Bottom()
		if l.LineHeight > 0 && l.Cursor.Y+l.LineHeight > rowBottom {
			rowBottom = l.Cursor.Y + l.LineHeight
		}
		l.Cursor.X = l.Bounds.X + l.Indent
		l.Cursor.Y = rowBottom + l.SpacingY
		l.LineHeight = height
	}
	return r
}

// BeginHorizontal starts a horizontal layout row (HBox) at the current cursor position.
func (ctx *Context) BeginHorizontal() {
	l := ctx.currentLayout()
	start := Point{X: l.Bounds.X + l.Indent, Y: l.Cursor.Y}
	availW := max(0, l.Bounds.Right()-start.X)
	availH := max(0, l.Bounds.Bottom()-start.Y)
	ctx.PushLayout(Rect{X: start.X, Y: start.Y, W: availW, H: availH}, FlowHorizontal, max(1, l.SpacingX), l.SpacingY)
}

// EndHorizontal closes the current horizontal layout row and advances the parent cursor.
func (ctx *Context) EndHorizontal() {
	if len(ctx.layoutStack) <= 1 {
		return
	}
	child := ctx.PopLayout()
	w := max(0, child.MaxX-child.Bounds.X)
	h := max(child.LineHeight, child.MaxY-child.Bounds.Y)
	ctx.AllocateRect(w, h)
}

// SameLine places the next widget immediately to the right of the previous widget.
func (ctx *Context) SameLine() {
	l := ctx.currentLayout()
	l.SameLineReq = true
}

// Indent increases the left indentation of the current layout scope by cols cells.
func (ctx *Context) Indent(cols int) {
	if cols <= 0 {
		cols = 2
	}
	l := ctx.currentLayout()
	l.Indent += cols
	if l.Flow == FlowVertical {
		l.Cursor.X = l.Bounds.X + l.Indent
	}
}

// Unindent decreases the left indentation of the current layout scope by cols cells.
func (ctx *Context) Unindent(cols int) {
	if cols <= 0 {
		cols = 2
	}
	l := ctx.currentLayout()
	l.Indent = max(0, l.Indent-cols)
	if l.Flow == FlowVertical {
		l.Cursor.X = l.Bounds.X + l.Indent
	}
}

// Spacing advances the vertical cursor by rows terminal lines.
func (ctx *Context) Spacing(rows int) {
	if rows <= 0 {
		rows = 1
	}
	l := ctx.currentLayout()
	l.Cursor.Y += rows
}

// Dummy allocates an invisible spacer widget of (width, height) cells.
func (ctx *Context) Dummy(width, height int) Rect {
	return ctx.AllocateRect(width, height)
}

// CursorPos returns the current layout cursor position in terminal cells.
func (ctx *Context) CursorPos() Point {
	return ctx.currentLayout().Cursor
}

// SetCursorPos sets the current layout cursor position in terminal cells.
func (ctx *Context) SetCursorPos(p Point) {
	l := ctx.currentLayout()
	l.Cursor = p
	l.SameLineReq = false
}

// AvailableWidth returns the remaining horizontal width in the current layout scope.
func (ctx *Context) AvailableWidth() int {
	l := ctx.currentLayout()
	if l.Flow == FlowHorizontal {
		return max(1, l.Bounds.Right()-l.Cursor.X)
	}
	return max(1, l.Bounds.W-l.Indent)
}

// AvailableHeight returns the remaining vertical height in the current layout scope.
func (ctx *Context) AvailableHeight() int {
	l := ctx.currentLayout()
	return max(1, l.Bounds.Bottom()-l.Cursor.Y)
}
