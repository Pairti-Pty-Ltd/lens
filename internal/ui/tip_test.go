package ui

import (
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestZoomSegmentShowsTheStop(t *testing.T) {
	clearTip()
	var dir int
	z := NewZoom(func(d int) { dir = d })
	z.SetLabel("100%")
	size := z.Layout(nil, geometry.Constraints{MaxWidth: 200, MaxHeight: 40})
	z.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	canvas := uitest.DrawWidget(z)
	if !glyphHas(glyphs(canvas), "100%") || !glyphHas(glyphs(canvas), "−") || !glyphHas(glyphs(canvas), "+") {
		t.Fatalf("drawn %+v", glyphs(canvas))
	}
	if !z.Event(nil, &event.MouseEvent{MouseType: event.MousePress, Button: event.ButtonLeft, Position: geometry.Pt(size.Width-8, 14)}) || dir != 1 {
		t.Fatalf("plus %d", dir)
	}
	if !z.Event(nil, &event.MouseEvent{MouseType: event.MousePress, Button: event.ButtonLeft, Position: geometry.Pt(8, 14)}) || dir != -1 {
		t.Fatalf("minus %d", dir)
	}
	move(z, 8, 10)
	if tipText() != "Zoom out" {
		t.Fatalf("tip %q", tipText())
	}
	move(z, toolSlot+20, 10)
	if tipText() != "100%" {
		t.Fatalf("middle %q", tipText())
	}
}

func TestToolbarNamesEveryIcon(t *testing.T) {
	clearTip()
	tools := DocumentTools()
	bar := NewToolBar(tools, nil)
	size := uitest.LayoutWidget(bar, 800, 40)
	bar.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	for _, sp := range bar.spans {
		move(bar, sp.x+8, 10)
		if tipText() != sp.name {
			t.Fatalf("%s tip %q", sp.name, tipText())
		}
	}
	bar.SetScreenOrigin(geometry.Pt(100, 40))
	var search toolSpan
	for _, sp := range bar.spans {
		if sp.name == "Search" {
			search = sp
		}
	}
	bar.Event(nil, &event.MouseEvent{
		MouseType: event.MouseEnter,
		Position:  geometry.Pt(100+search.x+8, 50),
	})
	if tipText() != "Search" {
		t.Fatalf("enter tip %q", tipText())
	}
	bar.Event(nil, &event.MouseEvent{MouseType: event.MouseLeave})
	if tipText() != "" {
		t.Fatalf("left tip %q", tipText())
	}
}

func TestSegmentAndFiltersNameTheirIcons(t *testing.T) {
	clearTip()
	seg := NewViewSegment(nil)
	seg.Layout(nil, geometry.Constraints{MaxWidth: 80, MaxHeight: 40})
	seg.SetBounds(geometry.NewRect(0, 0, toolSlot*2, toolSlot))
	move(seg, 8, 10)
	if tipText() != "Formatted" {
		t.Fatalf("preview %q", tipText())
	}
	move(seg, toolSlot+8, 10)
	if tipText() != "Raw" {
		t.Fatalf("raw %q", tipText())
	}

	pane := NewProjectActions(nil)
	pane.Set("notes", false, false, false, true)
	wide := toolSlot*3 + nameGap + toolSlot*2
	pane.Layout(nil, geometry.Constraints{MaxWidth: wide, MaxHeight: 40})
	pane.SetBounds(geometry.NewRect(0, 0, wide, toolSlot))
	move(pane, 8, 10)
	if tipText() != "Show hidden files" {
		t.Fatalf("eye %q", tipText())
	}
	move(pane, toolSlot+8, 10)
	if tipText() != "Show gitignored files" {
		t.Fatalf("slash %q", tipText())
	}
	move(pane, toolSlot*2+8, 10)
	if tipText() != "Documents" {
		t.Fatalf("documents %q", tipText())
	}
	pane.Set("notes", true, true, false, false)
	move(pane, 8, 10)
	if tipText() != "Hide hidden files" {
		t.Fatalf("eye on %q", tipText())
	}
	move(pane, toolSlot+8, 10)
	if tipText() != "Hide gitignored files" {
		t.Fatalf("slash on %q", tipText())
	}
	move(pane, toolSlot*2+8, 10)
	if tipText() != "All folders" {
		t.Fatalf("folders %q", tipText())
	}
	move(pane, toolSlot*3+nameGap+8, 10)
	if tipText() != "Titles" {
		t.Fatalf("titles %q", tipText())
	}
	move(pane, toolSlot*3+nameGap+toolSlot+8, 10)
	if tipText() != "Filenames" {
		t.Fatalf("filenames %q", tipText())
	}
}

func TestRailAndCloseNameTheirIcons(t *testing.T) {
	clearTip()
	rail := NewRail(nil, nil)
	rail.SetBounds(geometry.NewRect(0, 0, 280, 400))
	want := []string{"Table of contents", "Document", "Links", "Images"}
	for i, name := range want {
		move(rail, 8+float32(i)*28+10, 10)
		if tipText() != name {
			t.Fatalf("mode %d tip %q", i, tipText())
		}
	}
	move(rail, 20, 80)
	if tipText() != "" {
		t.Fatalf("list tip %q", tipText())
	}

	tabs := NewTabs(nil, nil, nil)
	tabs.Set([]Tab{{Label: "left.md"}, {Label: "right.md"}}, 0, -1)
	tabs.Layout(nil, geometry.Constraints{MaxWidth: 400, MaxHeight: 40})
	tabs.SetBounds(geometry.NewRect(0, 0, 400, tabH))
	move(tabs, 40, 10)
	if tipText() != "" {
		t.Fatalf("tab name tip %q", tipText())
	}
	if len(tabs.widths) == 0 || tabs.widths[0] <= 64 {
		t.Fatalf("widths %v", tabs.widths)
	}
	move(tabs, tabs.widths[0]-8, 10)
	if tipText() != "Close" {
		t.Fatalf("close %q", tipText())
	}
}

func TestTipDrawsBelowTheIconAndInsideTheWindow(t *testing.T) {
	clearTip()
	bar := NewToolBar([]Tool{RailTool()}, nil)
	bar.Layout(nil, geometry.Constraints{MaxWidth: 40, MaxHeight: 28})
	bar.SetBounds(geometry.NewRect(760, 4, 28, 28))
	move(bar, 768, 14)
	if tipText() != "Outline" {
		t.Fatalf("tip %q", tipText())
	}
	pulse := NewPulse(bar, nil)
	pulse.SetBounds(geometry.NewRect(0, 0, 800, 200))
	canvas := uitest.DrawWidget(pulse)
	var y float32
	found := false
	for _, g := range glyphs(canvas) {
		if g.text == "Outline" {
			found = true
			y = g.y
		}
	}
	if !found {
		t.Fatal("tip was not drawn")
	}
	if y < 32 {
		t.Fatalf("tip y %v covers the icon", y)
	}
	var box geometry.Rect
	for _, r := range canvas.RoundRects {
		if r.Bounds.Min.Y >= 32 && r.Bounds.Height() > 10 {
			box = r.Bounds
		}
	}
	if box.Max.X > 796 || box.Min.X < 4 {
		t.Fatalf("tip %v leaves the window", box)
	}
}

func move(w widget.Widget, x, y float32) {
	w.Event(nil, &event.MouseEvent{MouseType: event.MouseMove, Position: geometry.Pt(x, y)})
}
