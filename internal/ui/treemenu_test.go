package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestTreeMenuListsTheFileActions(t *testing.T) {
	var got string
	menu := NewTreeMenu(func(_ Item, action string) { got = action })
	menu.Show(Item{Label: "Morning, 23 Nov", Path: "/notes/morning.md", Rel: "morning.md"}, geometry.Pt(40, 40))
	menu.SetBounds(geometry.NewRect(0, 0, 800, 600))
	text := painted(uitest.DrawWidget(menu))
	for _, want := range []string{"Morning, 23 Nov", "Copy Path", "Copy Relative Path", "Reveal in Finder", "Open in App", "Move to Bin", "Delete"} {
		if !strings.Contains(text, want) {
			t.Fatalf("missing %s\n%s", want, text)
		}
	}
	if strings.Contains(text, "Delete forever") || strings.Contains(text, "/notes/morning.md") {
		t.Fatalf("menu repeated the path or armed delete:\n%s", text)
	}
	clickMenu(t, menu, MenuCopyPath)
	if got != MenuCopyPath {
		t.Fatalf("action %s", got)
	}
	if menu.Shown() {
		t.Fatal("menu stayed open")
	}
}

func TestTreeMenuDeleteAsksAgain(t *testing.T) {
	var got string
	menu := NewTreeMenu(func(_ Item, action string) { got = action })
	menu.Show(Item{Label: "note.md", Path: "/tmp/note.md"}, geometry.Pt(40, 40))
	menu.SetBounds(geometry.NewRect(0, 0, 800, 600))
	clickMenu(t, menu, MenuDelete)
	if got != "" || !menu.Shown() {
		t.Fatalf("first delete fired %q open %v", got, menu.Shown())
	}
	text := painted(uitest.DrawWidget(menu))
	if !strings.Contains(text, "Delete forever") {
		t.Fatalf("no confirmation\n%s", text)
	}
	clickMenu(t, menu, MenuDelete)
	if got != MenuDelete {
		t.Fatalf("second delete %q", got)
	}
}

func TestTreeMenuCutsALongName(t *testing.T) {
	const name = "Autumn field notes: trees already measured and plots still to visit on the north orchard survey"
	menu := NewTreeMenu(nil)
	menu.Show(Item{Label: name, Path: "/tmp/morning.md"}, geometry.Pt(20, 20))
	menu.SetBounds(geometry.NewRect(0, 0, 800, 600))
	text := painted(uitest.DrawWidget(menu))
	if strings.Contains(text, name) {
		t.Fatal("long name was painted whole")
	}
	if !strings.Contains(text, "…") {
		t.Fatalf("name was not cut\n%s", text)
	}
}

func TestSidebarRightClickOpensTheMenu(t *testing.T) {
	var got Item
	s := NewSidebar(nil)
	s.OnMenu(func(it Item, _ geometry.Point) { got = it })
	s.SetItems([]Item{{Label: "Morning", Path: "/notes/morning.md", Rel: "morning.md"}})
	s.SetBounds(geometry.NewRect(0, 0, 240, 200))
	if !s.Event(nil, uitest.RightClick(20, 8)) {
		t.Fatal("right click was not consumed")
	}
	if got.Path != "/notes/morning.md" || got.Rel != "morning.md" {
		t.Fatalf("%+v", got)
	}
}

func clickMenu(t *testing.T, menu *TreeMenu, id string) {
	t.Helper()
	p, ok := menu.Spot(id)
	if !ok {
		t.Fatalf("no row %s", id)
	}
	if !uitest.SimulateClick(menu, p.X, p.Y) {
		t.Fatalf("click missed %s", id)
	}
}
