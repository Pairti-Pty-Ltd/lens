package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestModeBarSelectsHistory(t *testing.T) {
	var got int
	bar := NewModeBar(func(mode int) { got = mode })
	uitest.LayoutWidget(bar, 300, 26)
	bar.SetBounds(geometry.NewRect(0, 0, 300, 26))
	if !uitest.SimulateClick(bar, 250, 13) || got != SideHistory {
		t.Fatalf("mode %d", got)
	}
}

func TestSidePanePutsTheModeSwitchAboveTheControls(t *testing.T) {
	field := NewNarrow(textfield.New(textfield.Placeholder("Search")))
	actions := NewProjectActions(nil)
	actions.Set("notes", false, false, false, true)
	pane := NewSidePane(field, actions, NewSidebar(nil))
	bar := NewModeBar(nil)
	var added bool
	plus := NewReportBar(func() { added = true }, nil)
	plus.SetChoices([]ReportChoice{{Name: "field-guide", Root: "/refer"}}, "/refer")
	pane.SetBar(bar)
	pane.SetReportBar(plus)
	uitest.LayoutWidget(pane, 320, 500)
	pane.SetBounds(geometry.NewRect(0, 0, 320, 500))
	if bar.Bounds().Min.Y >= field.Bounds().Min.Y {
		t.Fatalf("switch %v search %v", bar.Bounds(), field.Bounds())
	}
	if field.Bounds().Min.Y >= actions.Bounds().Min.Y {
		t.Fatalf("search %v actions %v", field.Bounds(), actions.Bounds())
	}
	pane.SetReports(true)
	uitest.LayoutWidget(pane, 320, 500)
	pane.SetBounds(geometry.NewRect(0, 0, 320, 500))
	if plus.Bounds().Min.Y <= bar.Bounds().Min.Y {
		t.Fatalf("plus %v switch %v", plus.Bounds(), bar.Bounds())
	}
	for _, child := range pane.Children() {
		if child == field || child == actions {
			t.Fatal("file controls stay visible in reports")
		}
	}
	if !uitest.SimulateClick(plus, plus.Bounds().Max.X-10, plus.Bounds().Min.Y+8) || !added {
		t.Fatal("plus did not create a report")
	}
}

func TestReportBarPicksAProject(t *testing.T) {
	var picked string
	var added bool
	bar := NewReportBar(func() { added = true }, func(root string) { picked = root })
	bar.SetChoices([]ReportChoice{
		{Name: "field-guide", Root: "/refer"},
		{Name: "grove-p-workspace", Root: "/grove"},
	}, "/refer")
	uitest.LayoutWidget(bar, 280, 200)
	bar.SetBounds(geometry.NewRect(0, 0, 280, 200))
	if !uitest.SimulateClick(bar, 270, 14) || !added {
		t.Fatal("plus did not create a report")
	}
	if bar.MenuOpen() {
		t.Fatal("plus opened the project list")
	}
	if !uitest.SimulateClick(bar, 24, 14) || !bar.MenuOpen() {
		t.Fatal("the project name did not open the list")
	}
	canvas := uitest.DrawWidget(bar)
	var sawRefer, sawGrove, sawPath bool
	for _, tx := range canvas.StyledTexts {
		if tx.Text == "field-guide" {
			sawRefer = true
		}
		if tx.Text == "grove-p-workspace" {
			sawGrove = true
		}
		if strings.Contains(tx.Text, "/refer") || strings.Contains(tx.Text, "/grove") {
			sawPath = true
		}
	}
	if !sawRefer || !sawGrove || sawPath {
		t.Fatalf("refer %v grove %v path %v texts %+v", sawRefer, sawGrove, sawPath, canvas.StyledTexts)
	}
	if !uitest.SimulateClick(bar, 24, reportRow+reportRow+10) || picked != "/grove" || bar.MenuOpen() {
		t.Fatalf("picked %q open %v", picked, bar.MenuOpen())
	}
}

func TestSidePaneCentersTheSearch(t *testing.T) {
	field := NewNarrow(textfield.New(textfield.Placeholder("Search")))
	pane := NewSidePane(field, NewProjectActions(nil), NewSidebar(nil))
	uitest.LayoutWidget(pane, 320, 500)
	pane.SetBounds(geometry.NewRect(0, 0, 320, 500))
	b := field.Bounds()
	left := b.Min.X
	right := float32(320) - b.Max.X
	if left < sidePad-0.5 || right < sidePad-0.5 || left-right > 0.5 || right-left > 0.5 {
		t.Fatalf("search insets left %v right %v bounds %v", left, right, b)
	}
	if b.Width() < 280 {
		t.Fatalf("search width %v", b.Width())
	}
}
