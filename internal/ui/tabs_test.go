package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestTabLabelStopsBeforeTheClose(t *testing.T) {
	tabs := NewTabs(nil, nil, nil)
	name := "orchard-survey-coverage.md"
	tabs.Set([]Tab{{Label: name}}, 0, -1)
	tabs.Layout(nil, geometry.Constraints{MaxWidth: 180, MaxHeight: 40})
	w := tabs.widths[0]
	tabs.SetBounds(geometry.NewRect(0, 0, w, tabH))
	canvas := uitest.DrawWidget(tabs)
	var label, close geometry.Rect
	var labelText string
	for _, tx := range canvas.StyledTexts {
		if tx.Text == "×" {
			close = tx.Bounds
			continue
		}
		label = tx.Bounds
		labelText = tx.Text
	}
	if close.Width() < 1 {
		t.Fatal("close mark was not drawn")
	}
	if label.Max.X > close.Min.X {
		t.Fatalf("label ends at %v, close starts at %v", label.Max.X, close.Min.X)
	}
	if labelText == name || !strings.Contains(labelText, "…") {
		t.Fatalf("label %q", labelText)
	}
}

func TestTabRightClickNamesTheTab(t *testing.T) {
	got := -1
	bar := NewTabs(nil, nil, nil)
	bar.OnMenu(func(i int, _ geometry.Point) { got = i })
	bar.Set([]Tab{{Label: "a.md"}, {Label: "b.md"}, {Label: "c.md"}}, 0, -1)
	uitest.LayoutWidget(bar, 400, 28)
	bar.SetBounds(geometry.NewRect(0, 0, 400, 28))
	x := bar.widths[0] + 20
	if !bar.Event(nil, uitest.RightClick(x, 14)) {
		t.Fatal("right click was not consumed")
	}
	if got != 1 {
		t.Fatalf("tab %d", got)
	}
}

func TestTabMenuListsTheCloseCommands(t *testing.T) {
	const name = "orchard-survey-coverage-and-the-seasonal-planting-review-for-the-west-orchard.md"
	var index int
	var action string
	menu := NewTabMenu(func(i int, id string) {
		index = i
		action = id
	})
	menu.Show(name, 1, 3, geometry.Pt(40, 40))
	menu.SetBounds(geometry.NewRect(0, 0, 800, 600))
	text := painted(uitest.DrawWidget(menu))
	for _, want := range []string{"Close", "Close Others", "Close to the Right", "Close to the Left", "Close All"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s\n%s", want, text)
		}
	}
	if strings.Contains(text, name) || strings.Contains(text, "/notes/") {
		t.Fatalf("menu painted the whole name or a path:\n%s", text)
	}
	if !strings.Contains(text, "…") || strings.Count(text, "…") != 1 {
		t.Fatalf("name:\n%s", text)
	}
	p, ok := menu.Spot(TabCloseOthers)
	if !ok || !uitest.SimulateClick(menu, p.X, p.Y) {
		t.Fatal("close others was not hit")
	}
	if index != 1 || action != TabCloseOthers || menu.Shown() {
		t.Fatalf("index %d action %s open %v", index, action, menu.Shown())
	}
}

func TestTabMenuDropsCommandsWithNothingToClose(t *testing.T) {
	menu := NewTabMenu(nil)
	menu.SetBounds(geometry.NewRect(0, 0, 800, 600))
	menu.Show("a.md", 0, 1, geometry.Pt(40, 40))
	text := painted(uitest.DrawWidget(menu))
	if !strings.Contains(text, "Close") || strings.Contains(text, "Close Others") || strings.Contains(text, "Close to the") || strings.Contains(text, "Close All") {
		t.Fatalf("one tab:\n%s", text)
	}
	menu.Show("c.md", 2, 3, geometry.Pt(40, 40))
	text = painted(uitest.DrawWidget(menu))
	if strings.Contains(text, "Close to the Right") || !strings.Contains(text, "Close to the Left") {
		t.Fatalf("last tab:\n%s", text)
	}
	if !menu.Event(nil, event.NewKeyEvent(event.KeyPress, event.KeyEscape, 0, 0)) || menu.Shown() {
		t.Fatal("escape left the menu open")
	}
}
