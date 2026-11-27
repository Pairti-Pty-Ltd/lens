package ui

import (
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func sampleItems() []Item {
	return []Item{
		{Label: "notes", Path: "/proj/notes", Dir: true},
		{Label: "Membrane assay", Path: "/proj/notes/membrane-assay.md", Rel: "notes/membrane-assay.md"},
		{Label: "Plate map", Path: "/proj/notes/plate-map.md", Rel: "notes/plate-map.md"},
	}
}

func placeSidebar(s *Sidebar) {
	uitest.LayoutWidget(s, 280, 120)
	s.SetBounds(geometry.NewRect(16, 120, 280, 120))
}

func TestSidebarDrawsRowsAtItsOwnOrigin(t *testing.T) {
	s := NewSidebar(func(Item) {})
	s.SetItems(sampleItems())
	placeSidebar(s)

	rows := glyphs(uitest.DrawWidget(s))
	if len(rows) != 3 {
		t.Fatalf("drew %d rows, want 3", len(rows))
	}
	if rows[0].text != "notes" {
		t.Fatalf("first row = %q", rows[0].text)
	}
	if rows[1].text != "Membrane assay" {
		t.Fatalf("second row = %q", rows[1].text)
	}
	top := s.Bounds().Min.Y
	if rows[0].y != top {
		t.Fatalf("first row y = %v, want %v", rows[0].y, top)
	}
	if rows[2].y != top+2*sidebarRow {
		t.Fatalf("third row y = %v, want %v", rows[2].y, top+2*sidebarRow)
	}
}

func TestReportNameOpensAndChevronToggles(t *testing.T) {
	var picked, chev string
	s := NewSidebar(func(it Item) { picked = it.Label })
	s.OnChevron(func(it Item) { chev = it.Label })
	s.SetItems([]Item{{
		Label: "Field Guide - Orchard Survey Manual",
		Path:  "report:report-1", Rel: "report-1", Dir: true,
	}})
	s.SetBounds(geometry.NewRect(0, 0, 280, 80))
	nameX := s.Bounds().Min.X + 40
	if !uitest.SimulateClick(s, nameX, s.Bounds().Min.Y+8) {
		t.Fatal("name click was not used")
	}
	if picked != "Field Guide - Orchard Survey Manual" || chev != "" {
		t.Fatalf("picked %q chevron %q", picked, chev)
	}
	picked = ""
	if !uitest.SimulateClick(s, s.Bounds().Min.X+6, s.Bounds().Min.Y+8) {
		t.Fatal("chevron click was not used")
	}
	if chev != "Field Guide - Orchard Survey Manual" || picked != "" {
		t.Fatalf("picked %q chevron %q", picked, chev)
	}
}

func TestReportRowArrowsReorderWithoutOpening(t *testing.T) {
	var picked string
	var moved int
	s := NewSidebar(func(it Item) { picked = it.Label })
	s.OnOrder(func(_ Item, dir int) { moved = dir })
	s.SetItems([]Item{
		{Label: "Field Guide - Orchard Survey Manual", Rel: "report-1", Dir: true, Expanded: true},
		{Label: "index.md", Path: "/manual/index.md", Rel: "index.md", Note: "report-1", Depth: 1},
		{Label: "setting-up.md", Path: "/manual/setting-up.md", Rel: "setting-up.md", Note: "report-1", Depth: 1},
	})
	s.SetBounds(geometry.NewRect(0, 0, 280, 120))
	down := s.orderBox(s.Bounds(), sidebarRow, 1)
	if !uitest.SimulateClick(s, down.Min.X+4, down.Min.Y+4) {
		t.Fatal("arrow was not consumed")
	}
	if moved != 1 || picked != "" {
		t.Fatalf("moved %d picked %q", moved, picked)
	}
}

func TestShortListIsDrawnAfterALongScroll(t *testing.T) {
	s := NewSidebar(nil)
	var long []Item
	for i := 0; i < 80; i++ {
		long = append(long, Item{Label: fmt.Sprintf("file-%d.md", i), Path: fmt.Sprintf("/n/%d.md", i)})
	}
	s.SetItems(long)
	s.SetBounds(geometry.NewRect(0, 0, 280, 200))
	s.scroll = 2000
	s.SetItems([]Item{
		{Label: "Field Guide - Orchard Survey Manual", Rel: "report-1", Dir: true, Expanded: true},
		{Label: "index.md", Path: "/manual/index.md", Rel: "index.md", Note: "report-1", Depth: 1},
	})
	if s.scroll != 0 {
		t.Fatalf("scroll %v", s.scroll)
	}
	var saw bool
	for _, row := range glyphs(uitest.DrawWidget(s)) {
		if strings.HasPrefix(row.text, "Field Guide") {
			saw = true
		}
	}
	if !saw {
		t.Fatal("report name was not drawn")
	}
}

func TestSidebarClickUsesListTop(t *testing.T) {
	var got Item
	s := NewSidebar(func(it Item) { got = it })
	s.SetItems(sampleItems())
	placeSidebar(s)

	y := s.Bounds().Min.Y + sidebarRow + 1
	if !uitest.SimulateClick(s, 40, y) {
		t.Fatal("click was not consumed")
	}
	if got.Path != "/proj/notes/membrane-assay.md" {
		t.Fatalf("selected %q", got.Path)
	}
	if got.Label != "Membrane assay" {
		t.Fatalf("selected label %q", got.Label)
	}
}

func TestSidebarDoubleClickAsksForATab(t *testing.T) {
	var got []string
	s := NewSidebar(func(Item) { got = append(got, "pick") })
	s.OnTab(func(Item) { got = append(got, "tab") })
	s.SetItems([]Item{{Label: "Morning", Path: "/notes/morning.md"}})
	s.SetBounds(geometry.NewRect(0, 0, 240, 80))
	if !uitest.SimulateClick(s, 20, 8) || !uitest.SimulateClick(s, 20, 8) {
		t.Fatal("click was not consumed")
	}
	if strings.Join(got, ",") != "pick,tab" {
		t.Fatalf("%v", got)
	}
	got = nil
	ev := uitest.Click(20, 8)
	ev.ClickCount = 2
	if !s.Event(nil, ev) {
		t.Fatal("counted click was not consumed")
	}
	if strings.Join(got, ",") != "tab" {
		t.Fatalf("%v", got)
	}
	got = nil
	if !s.Event(nil, uitest.DoubleClick(20, 8)) {
		t.Fatal("double click was not consumed")
	}
	if strings.Join(got, ",") != "tab" {
		t.Fatalf("%v", got)
	}
}

func TestSidebarWheelOnMac(t *testing.T) {
	s := NewSidebar(func(Item) {})
	items := make([]Item, 20)
	for i := range items {
		items[i] = Item{Label: fmt.Sprintf("row-%02d", i), Path: fmt.Sprintf("/%02d", i)}
	}
	s.SetItems(items)
	placeSidebar(s)

	ctx := uitest.NewMockContext()
	delta := float32(4)
	if !s.Event(ctx, uitest.WheelScroll(40, s.Bounds().Min.Y+8, delta)) {
		t.Fatal("wheel was not consumed")
	}

	rows := glyphs(uitest.DrawWidget(s))
	if len(rows) == 0 {
		t.Fatal("scrolled list drew nothing")
	}
	natural := runtime.GOOS == "darwin"
	shift := WheelScroll(delta, natural)
	first := int(shift / sidebarRow)
	wantY := s.Bounds().Min.Y - shift + float32(first)*sidebarRow
	if rows[0].text != items[first].Label {
		t.Fatalf("first visible = %q, want %q", rows[0].text, items[first].Label)
	}
	if rows[0].y != wantY {
		t.Fatalf("first visible y = %v, want %v", rows[0].y, wantY)
	}
	for _, tx := range rows {
		if tx.text == items[0].Label {
			t.Fatal("the row that scrolled away is still drawn")
		}
	}
}
