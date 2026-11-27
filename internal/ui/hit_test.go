package ui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"
)

func TestRowIndexUsesListTop(t *testing.T) {
	const row = float32(28)
	const top = float32(120)
	if got := RowIndex(top, top, 0, row, 5); got != 0 {
		t.Fatalf("first row = %d", got)
	}
	if got := RowIndex(top+row*2+1, top, 0, row, 5); got != 2 {
		t.Fatalf("third row = %d", got)
	}
	if got := RowIndex(top+row*2+1, 0, 0, row, 5); got == 2 {
		t.Fatal("ignoring the list top selected the painted row")
	}
	if got := RowIndex(top+10, top, row, row, 5); got != 1 {
		t.Fatalf("scrolled = %d", got)
	}
}

func TestWheelScrollNaturalOnMac(t *testing.T) {
	if got := WheelScroll(4, true); got <= 0 {
		t.Fatalf("natural scroll = %v, want the list to follow a positive delta", got)
	}
	if got := WheelScroll(4, false); got >= 0 {
		t.Fatalf("classic scroll = %v, want the previous sign", got)
	}
}

func TestSplitToggleHidesTheSidebar(t *testing.T) {
	left := NewPage()
	right := NewPage()
	s := NewSplit(left, right, 300)
	uitest.LayoutWidget(s, 1000, 400)
	if left.Bounds().Width() < 200 {
		t.Fatalf("sidebar width %v", left.Bounds().Width())
	}
	s.Toggle()
	uitest.LayoutWidget(s, 1000, 400)
	if left.Bounds().Width() != 0 {
		t.Fatalf("hidden sidebar width %v", left.Bounds().Width())
	}
	if right.Bounds().Width() < 900 {
		t.Fatalf("document width %v", right.Bounds().Width())
	}
}

func TestDividerHoverUsesTheResizeCursor(t *testing.T) {
	ctx := uitest.NewMockContext()
	split := NewSplit(NewPage(), NewPage(), 300)
	uitest.LayoutWidget(split, 1000, 400)
	split.SetBounds(geometry.NewRect(0, 0, 1000, 400))
	split.Event(ctx, uitest.MouseMove(302, 40))
	if ctx.Cursor() != widget.CursorResizeEW {
		t.Fatalf("sidebar cursor %v", ctx.Cursor())
	}
	ctx.SetCursor(widget.CursorDefault)
	split.Event(ctx, uitest.MouseMove(40, 40))
	if ctx.Cursor() != widget.CursorDefault {
		t.Fatalf("pane cursor %v", ctx.Cursor())
	}

	rail := NewSplit(NewPage(), NewPage(), 280)
	rail.SetRight(true)
	uitest.LayoutWidget(rail, 1000, 400)
	rail.SetBounds(geometry.NewRect(0, 0, 1000, 400))
	rail.Event(ctx, uitest.MouseMove(716, 40))
	if ctx.Cursor() != widget.CursorResizeEW {
		t.Fatalf("rail cursor %v", ctx.Cursor())
	}

	ctx.SetCursor(widget.CursorDefault)
	left := NewPage()
	pair := NewPair(left, NewPage())
	pair.SetSplit(true)
	uitest.LayoutWidget(pair, 800, 400)
	pair.SetBounds(geometry.NewRect(0, 0, 800, 400))
	pair.Event(ctx, uitest.MouseMove(left.Bounds().Max.X+2, 40))
	if ctx.Cursor() != widget.CursorResizeEW {
		t.Fatalf("document cursor %v", ctx.Cursor())
	}
}

func TestSidebarWidthStaysPut(t *testing.T) {
	if got := SidebarWidth(300, 1400); got != 300 {
		t.Fatalf("wider window moved the sidebar to %v", got)
	}
	if got := SidebarWidth(300, 800); got != 300 {
		t.Fatalf("same window = %v", got)
	}
	got := SidebarWidth(300, 400)
	if got >= 300 {
		t.Fatalf("narrow window did not clamp: %v", got)
	}
	if got < 160 {
		t.Fatalf("clamped below the minimum: %v", got)
	}
}
