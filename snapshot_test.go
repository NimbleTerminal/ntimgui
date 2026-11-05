package ntimgui

import "testing"

func TestSnapshotOwnershipAndOrder(t *testing.T) {
	dl := NewDrawList(nil)
	dl.SetLayer(LayerOverlay)
	dl.AddText(Rect{W: 8, H: 1}, "overlay", AlignLeft, WidgetStyle{})
	dl.SetLayer(LayerBackground)
	values := []float64{.2, .8}
	dl.AddSparkline(Rect{W: 8, H: 2}, values, false, "plot", WidgetStyle{})
	snap := dl.Snapshot()
	source := dl.Commands()
	values[0] = .9
	source[0].Values[0] = .7
	source[0].Text = "changed"
	dl.Reset(Rect{W: 20, H: 20})
	dl.AddRect(Rect{W: 1, H: 1}, WidgetStyle{})
	if len(snap) != 2 || snap[0].Kind != CmdSparkline || snap[1].Text != "overlay" || snap[0].Values[0] != .2 {
		t.Fatalf("snapshot changed: %+v", snap)
	}
	ctx := New()
	ctx.BeginFrame()
	ctx.DrawList.AddSparkline(Rect{W: 2, H: 2}, []float64{.4}, false, "", WidgetStyle{})
	ctx.EndFrame()
	frame := ctx.DrawList.Snapshot()
	ctx.BeginFrame()
	ctx.EndFrame()
	if frame[0].Values[0] != .4 {
		t.Fatal("BeginFrame changed snapshot")
	}
}
func TestSnapshotCommandsOwnsAllMutableData(t *testing.T) {
	source := []DrawCmd{{Text: "immutable", Values: []float64{.3}}}
	snap := SnapshotCommands(source)
	source[0].Values[0] = 1
	source[0].Text = "mutated"
	if snap[0].Text != "immutable" || snap[0].Values[0] != .3 {
		t.Fatal(snap)
	}
	snap[0].Values[0] = .6
	if source[0].Values[0] != 1 {
		t.Fatal("source aliases clone")
	}
	if SnapshotCommands(nil) != nil {
		t.Fatal("nil snapshot")
	}
}
