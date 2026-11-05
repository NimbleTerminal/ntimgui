// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	"fmt"
	"math"
)

// TabBar renders a horizontal row of selectable tabs and returns true when the
// active tab index changes.
func (ctx *Context) TabBar(label string, active *int, tabs []string) bool {
	if len(tabs) == 0 {
		return false
	}
	cur := 0
	if active != nil {
		cur = max(0, min(len(tabs)-1, *active))
	}

	ctx.PushID(label)
	ctx.BeginHorizontal()
	changed := false
	for i, tab := range tabs {
		ctx.PushIDInt(i)
		display, id := ctx.GetID(tab)
		w := displayWidth(display) + 4
		bounds := ctx.AllocateRect(w, 1)
		ctx.registerFocusable(id)
		recordWidget(ctx, id, bounds)

		if ctx.isMouseInRect(bounds) {
			if ctx.Active == 0 || ctx.Active == id {
				ctx.Hot = id
				if ctx.Input.MousePressed {
					ctx.Active = id
					ctx.Focus = id
				}
			}
		}

		clicked := false
		if ctx.Input.MouseReleased && ctx.Hot == id && ctx.Active == id {
			clicked = true
			ctx.Active = 0
		}
		if ctx.Focus == id && (ctx.Input.KeyPressed(KeyEnter) || ctx.Input.KeyPressed(KeySpace)) {
			clicked = true
		}
		if clicked && cur != i {
			cur = i
			changed = true
		}

		var style WidgetStyle
		var text string
		switch {
		case i == cur:
			style = ctx.Theme.Active
			text = "▸ " + display + " "
		case ctx.Hot == id:
			style = ctx.Theme.Hover
			text = "  " + display + " "
		case ctx.Focus == id:
			style = ctx.Theme.Focus
			text = "  " + display + " "
		default:
			style = ctx.Theme.Normal
			text = "  " + display + " "
		}
		ctx.DrawList.AddRect(bounds, style)
		ctx.DrawList.AddText(bounds, text, AlignCenter, style)
		ctx.PopID()
	}
	ctx.EndHorizontal()
	ctx.PopID()

	if changed && active != nil {
		*active = cur
	}
	return changed
}

// Combo evaluates a cycle/selector widget (< Item ▾ >) and returns true when
// the selected index changes via mouse click or Left/Right arrow keys.
func (ctx *Context) Combo(label string, selected *int, items []string) bool {
	display, id := ctx.GetID(label)
	if len(items) == 0 {
		return false
	}
	idx := 0
	if selected != nil {
		idx = max(0, min(len(items)-1, *selected))
	}

	maxItemW := 8
	for _, it := range items {
		if w := displayWidth(it); w > maxItemW {
			maxItemW = w
		}
	}
	boxW := maxItemW + 6
	labelW := displayWidth(display)
	totalW := boxW
	if labelW > 0 {
		totalW += 1 + labelW
	}

	bounds := ctx.AllocateRect(totalW, 1)
	boxRect := Rect{X: bounds.X, Y: bounds.Y, W: boxW, H: 1}
	ctx.registerFocusable(id)
	recordWidget(ctx, id, bounds)

	if ctx.isMouseInRect(boxRect) {
		if ctx.Active == 0 || ctx.Active == id {
			ctx.Hot = id
			if ctx.Input.MousePressed {
				ctx.Active = id
				ctx.Focus = id
			}
		}
	}

	changed := false
	if ctx.Input.MouseReleased && ctx.Hot == id && ctx.Active == id {
		if ctx.Input.MouseX < boxRect.X+3 {
			idx = (idx - 1 + len(items)) % len(items)
		} else {
			idx = (idx + 1) % len(items)
		}
		changed = true
		ctx.Active = 0
	}
	if ctx.Focus == id {
		if ctx.Input.KeyPressed(KeyLeft) {
			idx = (idx - 1 + len(items)) % len(items)
			changed = true
		} else if ctx.Input.KeyPressed(KeyRight) || ctx.Input.KeyPressed(KeySpace) || ctx.Input.KeyPressed(KeyEnter) {
			idx = (idx + 1) % len(items)
			changed = true
		}
	}

	if changed && selected != nil {
		*selected = idx
	}

	style := ctx.widgetStyleFor(id)
	ctx.DrawList.AddRect(boxRect, style)
	ctx.DrawList.AddText(boxRect, fmt.Sprintf("◂ %s ▸", items[idx]), AlignCenter, style)

	if labelW > 0 {
		lblRect := Rect{X: boxRect.Right() + 1, Y: bounds.Y, W: labelW, H: 1}
		ctx.DrawList.AddText(lblRect, display, AlignLeft, ctx.Theme.Text)
	}
	return changed
}

// ListBox renders a bordered scrollable single-selection list box of heightRows
// rows and returns true when the selected item changes.
func (ctx *Context) ListBox(label string, selected *int, items []string, heightRows int) bool {
	display, id := ctx.GetID(label)
	if heightRows < 3 {
		heightRows = 3
	}
	cur := 0
	if selected != nil && len(items) > 0 {
		cur = max(0, min(len(items)-1, *selected))
	}

	w := max(16, ctx.AvailableWidth())
	bounds := ctx.AllocateRect(w, heightRows)
	ctx.registerFocusable(id)
	recordWidget(ctx, id, bounds)

	scrollID := hashString(id, "##scroll")
	innerH := max(1, bounds.H-2)
	scroll := ctx.Storage.Int(scrollID, 0)

	if bounds.Contains(ctx.Input.MouseX, ctx.Input.MouseY) && ctx.Input.WheelY != 0 {
		scroll += ctx.Input.WheelY
	}
	if ctx.Focus == id && len(items) > 0 {
		if ctx.Input.KeyPressed(KeyUp) && cur > 0 {
			cur--
			if selected != nil {
				*selected = cur
			}
			if cur < scroll {
				scroll = cur
			}
			ctx.Storage.SetInt(scrollID, scroll)
			ctx.drawListBoxContents(bounds, display, items, cur, scroll, id)
			return true
		}
		if ctx.Input.KeyPressed(KeyDown) && cur+1 < len(items) {
			cur++
			if selected != nil {
				*selected = cur
			}
			if cur >= scroll+innerH {
				scroll = cur - innerH + 1
			}
			ctx.Storage.SetInt(scrollID, scroll)
			ctx.drawListBoxContents(bounds, display, items, cur, scroll, id)
			return true
		}
	}

	maxScroll := max(0, len(items)-innerH)
	scroll = max(0, min(maxScroll, scroll))
	ctx.Storage.SetInt(scrollID, scroll)

	changed := false
	inner := bounds.Inset(1, 1)
	for row := 0; row < innerH; row++ {
		idx := scroll + row
		if idx >= len(items) {
			break
		}
		rowRect := Rect{X: inner.X, Y: inner.Y + row, W: inner.W, H: 1}
		itemID := hashInt(id, idx)
		if ctx.isMouseInRect(rowRect) {
			if ctx.Active == 0 || ctx.Active == itemID {
				ctx.Hot = itemID
				if ctx.Input.MousePressed {
					ctx.Active = itemID
					ctx.Focus = id
				}
			}
		}
		if ctx.Input.MouseReleased && ctx.Hot == itemID && ctx.Active == itemID {
			if cur != idx {
				cur = idx
				changed = true
			}
			ctx.Active = 0
		}
	}

	if changed && selected != nil {
		*selected = cur
	}
	ctx.drawListBoxContents(bounds, display, items, cur, scroll, id)
	return changed
}

func (ctx *Context) drawListBoxContents(bounds Rect, title string, items []string, selected, scroll int, id ID) {
	boxStyle := ctx.Theme.Input
	if ctx.Focus == id {
		boxStyle.Border = ctx.Theme.Focus.Border
	}
	ctx.DrawList.AddBox(bounds, ctx.Theme.Border, title, boxStyle)

	inner := bounds.Inset(1, 1)
	ctx.DrawList.PushClipRect(inner)
	for row := 0; row < inner.H; row++ {
		idx := scroll + row
		if idx >= len(items) {
			break
		}
		rowRect := Rect{X: inner.X, Y: inner.Y + row, W: inner.W, H: 1}
		itemID := hashInt(id, idx)
		prefix := "  "
		st := ctx.Theme.Text
		if idx == selected {
			st = ctx.Theme.Active
			prefix = "▸ "
			ctx.DrawList.AddRect(rowRect, st)
		} else if ctx.Hot == itemID {
			st = ctx.Theme.Hover
			ctx.DrawList.AddRect(rowRect, st)
		}
		ctx.DrawList.AddText(rowRect, prefix+items[idx], AlignLeft, st)
	}
	ctx.DrawList.PopClipRect()
}

// TreeNode renders an expandable hierarchical tree item. When it returns true,
// the caller must render its children and then call TreePop().
func (ctx *Context) TreeNode(label string, defaultOpen bool) bool {
	display, id := ctx.GetID(label)
	open := ctx.Storage.Bool(id, defaultOpen)

	w := ctx.AvailableWidth()
	bounds := ctx.AllocateRect(w, 1)
	ctx.registerFocusable(id)
	recordWidget(ctx, id, bounds)

	if ctx.isMouseInRect(bounds) {
		if ctx.Active == 0 || ctx.Active == id {
			ctx.Hot = id
			if ctx.Input.MousePressed {
				ctx.Active = id
				ctx.Focus = id
			}
		}
	}

	toggled := false
	if ctx.Input.MouseReleased && ctx.Hot == id && ctx.Active == id {
		toggled = true
		ctx.Active = 0
	}
	if ctx.Focus == id && (ctx.Input.KeyPressed(KeyEnter) || ctx.Input.KeyPressed(KeySpace)) {
		toggled = true
	}
	if toggled {
		open = !open
		ctx.Storage.SetBool(id, open)
	}

	arrow := "▸ "
	if open {
		arrow = "▾ "
	}
	st := ctx.Theme.Text
	if ctx.Hot == id {
		st = ctx.Theme.Hover
		ctx.DrawList.AddRect(bounds, st)
	} else if ctx.Focus == id {
		st = ctx.Theme.Focus
		ctx.DrawList.AddRect(bounds, st)
	}
	ctx.DrawList.AddText(bounds, arrow+display, AlignLeft, st)

	if open {
		ctx.PushID(label)
		ctx.Indent(2)
	}
	return open
}

// TreePop closes an open TreeNode scope started when TreeNode returned true.
func (ctx *Context) TreePop() {
	ctx.Unindent(2)
	ctx.PopID()
}

// PlotLines renders an inline sparkline chart of values normalized to [min..max].
func (ctx *Context) PlotLines(label string, values []float64, width, height int, overlay string) {
	ctx.plotSeries(label, values, width, height, false, overlay)
}

// PlotHistogram renders an inline bar histogram of values normalized to [min..max].
func (ctx *Context) PlotHistogram(label string, values []float64, width, height int, overlay string) {
	ctx.plotSeries(label, values, width, height, true, overlay)
}

func (ctx *Context) plotSeries(label string, values []float64, width, height int, histogram bool, overlay string) {
	display, _ := ParseLabel(label)
	if width <= 0 {
		width = ctx.AvailableWidth()
	}
	if height <= 0 {
		height = 2
	}
	if display != "" && overlay == "" {
		overlay = display
	}

	norm := make([]float64, len(values))
	if len(values) > 0 {
		minV, maxV := values[0], values[0]
		for _, v := range values[1:] {
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}
		span := maxV - minV
		if span <= 1e-9 {
			span = 1
		}
		for i, v := range values {
			norm[i] = (v - minV) / span
		}
	}

	bounds := ctx.AllocateRect(width, height)
	ctx.DrawList.AddSparkline(bounds, norm, histogram, overlay, ctx.Theme.Normal)
}

// ColorEdit renders an inline RGB color editor with a live color swatch and
// returns true when R, G, or B is modified.
func (ctx *Context) ColorEdit(label string, col *RGBA) bool {
	display, _ := ParseLabel(label)
	c := RGB(128, 128, 128)
	if col != nil {
		c = *col
	}

	ctx.PushID(label)
	ctx.BeginHorizontal()
	swatchStyle := WidgetStyle{
		FG: c,
		BG: c,
	}
	swatchBounds := ctx.AllocateRect(4, 1)
	ctx.DrawList.AddRect(swatchBounds, swatchStyle)
	ctx.DrawList.AddText(swatchBounds, "████", AlignCenter, swatchStyle)
	ctx.Label(fmt.Sprintf("%s #%02X%02X%02X", display, c.R, c.G, c.B))
	ctx.EndHorizontal()

	r, g, b := int(c.R), int(c.G), int(c.B)
	changed := false
	if ctx.SliderInt("R", &r, 0, 255) {
		changed = true
	}
	if ctx.SliderInt("G", &g, 0, 255) {
		changed = true
	}
	if ctx.SliderInt("B", &b, 0, 255) {
		changed = true
	}
	ctx.PopID()

	if changed && col != nil {
		col.R = uint8(max(0, min(255, r)))
		col.G = uint8(max(0, min(255, g)))
		col.B = uint8(max(0, min(255, b)))
		if col.A == 0 {
			col.A = 255
		}
	}
	_ = math.Round
	return changed
}

// KeyValue renders a two-column key/value inspector row across the available width.
func (ctx *Context) KeyValue(key, value string) {
	w := ctx.AvailableWidth()
	bounds := ctx.AllocateRect(w, 1)
	keyW := min(w/2, max(8, displayWidth(key)+2))
	ctx.DrawList.AddText(Rect{X: bounds.X, Y: bounds.Y, W: keyW, H: 1}, key, AlignLeft, ctx.Theme.Muted)
	ctx.DrawList.AddText(Rect{X: bounds.X + keyW, Y: bounds.Y, W: max(1, w-keyW), H: 1}, value, AlignRight, ctx.Theme.Text)
}

// TableRow renders a multi-column row with explicit column widths. When header
// is true, cells are highlighted with the theme's PanelTitle style.
func (ctx *Context) TableRow(cols []string, widths []int, header bool) {
	availW := ctx.AvailableWidth()
	bounds := ctx.AllocateRect(availW, 1)
	style := ctx.Theme.Text
	if header {
		style = ctx.Theme.PanelTitle
		ctx.DrawList.AddRect(bounds, style)
	}
	x := bounds.X
	for i, col := range cols {
		cw := 10
		if i < len(widths) && widths[i] > 0 {
			cw = widths[i]
		}
		if x >= bounds.Right() {
			break
		}
		cw = min(cw, bounds.Right()-x)
		ctx.DrawList.AddText(Rect{X: x, Y: bounds.Y, W: cw, H: 1}, col, AlignLeft, style)
		x += cw + 1
	}
}
