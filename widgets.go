// Copyright (c) 2026 Neomantra Corp.

package ntimgui

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func displayWidth(s string) int {
	return runewidth.StringWidth(s)
}

func recordWidget(ctx *Context, id ID, bounds Rect) {
	ctx.LastWidgetID = id
	ctx.LastWidgetRect = bounds
}

// Label renders static left-aligned text at the current layout cursor.
func (ctx *Context) Label(text string) {
	ctx.LabelStyled(text, ctx.Theme.Text)
}

// LabelMuted renders dimmed static text at the current layout cursor.
func (ctx *Context) LabelMuted(text string) {
	ctx.LabelStyled(text, ctx.Theme.Muted)
}

// LabelStyled renders single-line text with an explicit WidgetStyle.
func (ctx *Context) LabelStyled(text string, style WidgetStyle) {
	w := max(1, displayWidth(text))
	bounds := ctx.AllocateRect(w, 1)
	ctx.DrawList.AddText(bounds, text, AlignLeft, style)
}

// Text formats and renders static text at the current layout cursor.
func (ctx *Context) Text(format string, args ...any) {
	ctx.Label(fmt.Sprintf(format, args...))
}

// Separator draws a horizontal dividing line across the available layout width.
func (ctx *Context) Separator() {
	w := ctx.AvailableWidth()
	bounds := ctx.AllocateRect(w, 1)
	ctx.DrawList.AddLine(bounds, ctx.Theme.Muted)
}

// Button evaluates an immediate-mode push button and returns true on the frame
// it is activated (mouse click-and-release while Hot, or Space/Enter when focused).
func (ctx *Context) Button(label string) bool {
	display, _ := ParseLabel(label)
	w := displayWidth(display) + 4
	h := 3
	if ctx.Theme.CompactButtons {
		h = 1
	}
	return ctx.ButtonSized(label, w, h)
}

// ButtonSized evaluates an immediate-mode push button with explicit cell dimensions.
func (ctx *Context) ButtonSized(label string, width, height int) bool {
	display, id := ctx.GetID(label)
	if width <= 0 {
		width = displayWidth(display) + 4
	}
	if height <= 0 {
		height = 1
	}
	bounds := ctx.AllocateRect(width, height)
	ctx.registerFocusable(id)
	recordWidget(ctx, id, bounds)

	// State resolution (Hot / Active / Focus).
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

	style := ctx.widgetStyleFor(id)
	if height >= 3 {
		ctx.DrawList.AddBox(bounds, ctx.Theme.Border, "", style)
		textRect := Rect{X: bounds.X + 1, Y: bounds.Y + height/2, W: max(1, bounds.W-2), H: 1}
		ctx.DrawList.AddText(textRect, display, AlignCenter, style)
	} else {
		ctx.DrawList.AddRect(bounds, style)
		ctx.DrawList.AddText(bounds, "[ "+display+" ]", AlignCenter, style)
	}

	return clicked
}

// Checkbox evaluates a boolean checkbox widget and returns true on the frame its value changes.
func (ctx *Context) Checkbox(label string, checked *bool) bool {
	display, id := ctx.GetID(label)
	val := false
	if checked != nil {
		val = *checked
	}

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

	changed := false
	if ctx.Input.MouseReleased && ctx.Hot == id && ctx.Active == id {
		val = !val
		changed = true
		ctx.Active = 0
	}
	if ctx.Focus == id && (ctx.Input.KeyPressed(KeyEnter) || ctx.Input.KeyPressed(KeySpace)) {
		val = !val
		changed = true
	}
	if changed && checked != nil {
		*checked = val
	}

	mark := "[ ] "
	if val {
		mark = "[x] "
	}
	style := ctx.Theme.Text
	if ctx.Active == id {
		style = ctx.Theme.Active
	} else if ctx.Hot == id {
		style = ctx.Theme.Hover
	} else if ctx.Focus == id {
		style = ctx.Theme.Focus
	}
	if style.BG.A > 0 {
		ctx.DrawList.AddRect(bounds, style)
	}
	ctx.DrawList.AddText(bounds, mark+display, AlignLeft, style)
	return changed
}

// Toggle evaluates a pill/switch toggle widget and returns true when flipped.
func (ctx *Context) Toggle(label string, value *bool) bool {
	display, id := ctx.GetID(label)
	val := false
	if value != nil {
		val = *value
	}

	w := displayWidth(display) + 6
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

	changed := false
	if ctx.Input.MouseReleased && ctx.Hot == id && ctx.Active == id {
		val = !val
		changed = true
		ctx.Active = 0
	}
	if ctx.Focus == id && (ctx.Input.KeyPressed(KeyEnter) || ctx.Input.KeyPressed(KeySpace)) {
		val = !val
		changed = true
	}
	if changed && value != nil {
		*value = val
	}

	pill := "(○─) "
	if val {
		pill = "(─●) "
	}
	style := ctx.Theme.Text
	if ctx.Active == id {
		style = ctx.Theme.Active
	} else if ctx.Hot == id {
		style = ctx.Theme.Hover
	} else if ctx.Focus == id {
		style = ctx.Theme.Focus
	}
	if style.BG.A > 0 {
		ctx.DrawList.AddRect(bounds, style)
	}
	ctx.DrawList.AddText(bounds, pill+display, AlignLeft, style)
	return changed
}

// RadioButton evaluates a single radio option and returns true when selected.
func (ctx *Context) RadioButton(label string, active bool) bool {
	display, id := ctx.GetID(label)
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

	prefix := "( ) "
	if active {
		prefix = "(◉) "
	}
	style := ctx.Theme.Text
	if ctx.Active == id {
		style = ctx.Theme.Active
	} else if ctx.Hot == id {
		style = ctx.Theme.Hover
	} else if ctx.Focus == id {
		style = ctx.Theme.Focus
	}
	if style.BG.A > 0 {
		ctx.DrawList.AddRect(bounds, style)
	}
	ctx.DrawList.AddText(bounds, prefix+display, AlignLeft, style)
	return clicked
}

// RadioButtonInt binds a RadioButton to an integer state variable.
func (ctx *Context) RadioButtonInt(label string, current *int, buttonValue int) bool {
	active := current != nil && *current == buttonValue
	if ctx.RadioButton(label, active) {
		if current != nil {
			*current = buttonValue
		}
		return true
	}
	return false
}

// SliderFloat evaluates a horizontal floating-point slider and returns true when modified.
func (ctx *Context) SliderFloat(label string, value *float64, minVal, maxVal float64, format string) bool {
	display, id := ctx.GetID(label)
	if format == "" {
		format = "%.2f"
	}
	val := minVal
	if value != nil {
		val = *value
	}
	if maxVal <= minVal {
		maxVal = minVal + 1
	}

	availW := max(14, ctx.AvailableWidth())
	bounds := ctx.AllocateRect(availW, 1)
	ctx.registerFocusable(id)
	recordWidget(ctx, id, bounds)

	labelW := displayWidth(display)
	trackW := availW
	if labelW > 0 && availW > labelW+8 {
		trackW = availW - labelW - 1
	}
	trackRect := Rect{X: bounds.X, Y: bounds.Y, W: trackW, H: 1}

	if ctx.isMouseInRect(bounds) {
		if ctx.Active == 0 || ctx.Active == id {
			ctx.Hot = id
			if ctx.Input.MousePressed {
				ctx.Active = id
				ctx.Focus = id
			}
		}
	}

	changed := false
	if ctx.Active == id && (ctx.Input.MouseDown || ctx.Input.MousePressed) {
		var t float64
		if trackRect.W <= 1 {
			t = 0
		} else {
			t = float64(ctx.Input.MouseX-trackRect.X) / float64(trackRect.W-1)
		}
		t = max(0, min(1, t))
		newVal := minVal + t*(maxVal-minVal)
		if math.Abs(newVal-val) > 1e-9 {
			val = newVal
			changed = true
		}
	}

	if ctx.Focus == id {
		step := (maxVal - minVal) / 20.0
		if ctx.Input.KeyPressed(KeyLeft) {
			val = max(minVal, val-step)
			changed = true
		}
		if ctx.Input.KeyPressed(KeyRight) {
			val = min(maxVal, val+step)
			changed = true
		}
	}

	val = max(minVal, min(maxVal, val))
	if changed && value != nil {
		*value = val
	}

	frac := (val - minVal) / (maxVal - minVal)
	valText := fmt.Sprintf(format, val)
	style := ctx.widgetStyleFor(id)
	ctx.DrawList.AddGauge(trackRect, frac, true, valText, style)

	if trackW < availW && display != "" {
		lblRect := Rect{X: trackRect.Right() + 1, Y: bounds.Y, W: availW - trackW - 1, H: 1}
		ctx.DrawList.AddText(lblRect, display, AlignLeft, ctx.Theme.Text)
	}
	return changed
}

// SliderInt evaluates a horizontal integer slider and returns true when modified.
func (ctx *Context) SliderInt(label string, value *int, minVal, maxVal int) bool {
	val := minVal
	if value != nil {
		val = *value
	}
	fval := float64(val)
	if ctx.SliderFloat(label, &fval, float64(minVal), float64(maxVal), "%.0f") {
		newInt := int(math.Round(fval))
		newInt = max(minVal, min(maxVal, newInt))
		if value != nil && *value != newInt {
			*value = newInt
			return true
		}
	}
	return false
}

// ProgressBar renders a read-only fractional progress bar in [0..1].
func (ctx *Context) ProgressBar(fraction float64, width int, overlay string) {
	if width <= 0 {
		width = ctx.AvailableWidth()
	}
	bounds := ctx.AllocateRect(width, 1)
	if overlay == "" {
		overlay = fmt.Sprintf("%d%%", int(math.Round(max(0, min(1, fraction))*100)))
	}
	ctx.DrawList.AddGauge(bounds, fraction, false, overlay, ctx.Theme.Normal)
}

// CollapsingHeader renders an expandable/collapsible section header and returns
// true when the section is currently expanded.
func (ctx *Context) CollapsingHeader(label string, defaultOpen bool) bool {
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
	style := ctx.widgetStyleFor(id)
	ctx.DrawList.AddRect(bounds, style)
	ctx.DrawList.AddText(bounds, arrow+display, AlignLeft, style)
	return open
}

// TextInput evaluates a single-line editable text box and returns true when its
// contents are modified.
func (ctx *Context) TextInput(label string, value *string, width int) bool {
	display, id := ctx.GetID(label)
	text := ""
	if value != nil {
		text = *value
	}
	if width <= 0 {
		width = max(12, ctx.AvailableWidth())
	}

	labelW := displayWidth(display)
	fieldW := width
	totalW := width
	if labelW > 0 {
		totalW = fieldW + 1 + labelW
	}
	bounds := ctx.AllocateRect(totalW, 1)
	fieldRect := Rect{X: bounds.X, Y: bounds.Y, W: fieldW, H: 1}

	ctx.registerFocusable(id)
	recordWidget(ctx, id, bounds)

	if ctx.isMouseInRect(fieldRect) {
		if ctx.Active == 0 || ctx.Active == id {
			ctx.Hot = id
			if ctx.Input.MousePressed {
				ctx.Active = id
				ctx.Focus = id
			}
		}
	}

	changed := false
	if ctx.Focus == id {
		if ctx.Input.KeyPressed(KeyEnter) || ctx.Input.KeyPressed(KeyEscape) {
			ctx.Focus = 0
			ctx.editingText = false
		} else {
			ctx.editingText = true
			if ctx.Input.KeyPressed(KeyBackspace) && len(text) > 0 {
				_, size := utf8.DecodeLastRuneInString(text)
				text = text[:len(text)-size]
				changed = true
			}
			if ctx.Input.Text != "" {
				text += ctx.Input.Text
				changed = true
			}
		}
	}

	if changed && value != nil {
		*value = text
	}

	style := ctx.Theme.Input
	if ctx.Focus == id {
		style = ctx.Theme.InputFocus
	} else if ctx.Hot == id {
		style = ctx.Theme.Hover
	}

	ctx.DrawList.AddRect(fieldRect, style)
	rendered := text
	if ctx.Focus == id {
		rendered += "▏"
	}
	maxChars := max(1, fieldRect.W-2)
	for displayWidth(rendered) > maxChars && len(rendered) > 0 {
		_, size := utf8.DecodeRuneInString(rendered)
		rendered = rendered[size:]
	}
	ctx.DrawList.AddText(fieldRect.Inset(1, 0), rendered, AlignLeft, style)

	if labelW > 0 {
		lblRect := Rect{X: fieldRect.Right() + 1, Y: bounds.Y, W: labelW, H: 1}
		ctx.DrawList.AddText(lblRect, display, AlignLeft, ctx.Theme.Text)
	}
	return changed
}

// Tooltip renders a floating overlay tooltip near the cursor when the immediately
// preceding widget is Hot (hovered).
func (ctx *Context) Tooltip(text string) {
	if text == "" || !ctx.IsItemHovered() {
		return
	}
	w := displayWidth(text) + 4
	h := 3
	x := ctx.Input.MouseX + 2
	y := ctx.Input.MouseY + 1
	if x+w > ctx.screenW {
		x = max(0, ctx.screenW-w)
	}
	if y+h > ctx.screenH {
		y = max(0, ctx.Input.MouseY-h)
	}
	rect := Rect{X: x, Y: y, W: w, H: h}

	prevLayer := ctx.DrawList.Layer()
	ctx.DrawList.SetLayer(LayerOverlay)
	ctx.DrawList.PushClipRectUnclipped(ctx.Bounds())
	ctx.DrawList.AddBox(rect, BorderRounded, "", ctx.Theme.Tooltip)
	ctx.DrawList.AddText(Rect{X: rect.X + 2, Y: rect.Y + 1, W: w - 4, H: 1}, text, AlignLeft, ctx.Theme.Tooltip)
	ctx.DrawList.PopClipRect()
	ctx.DrawList.SetLayer(prevLayer)
}
