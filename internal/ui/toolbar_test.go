package ui

import (
	"testing"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/theme"
)

func TestToolBarHitsAPlaceholder(t *testing.T) {
	var got string
	bar := NewToolBar(DocumentTools(), func(name string) { got = name })
	uitest.LayoutWidget(bar, 800, 28)
	bar.SetBounds(geometry.NewRect(0, 0, 800, 28))
	var search toolSpan
	for _, sp := range bar.spans {
		if sp.name == "Search" {
			search = sp
		}
	}
	if search.name == "" {
		t.Fatal("no search slot")
	}
	if !uitest.SimulateClick(bar, search.x+8, 14) {
		t.Fatal("click was not consumed")
	}
	if got != "Search" {
		t.Fatalf("got %q", got)
	}
}

func TestNarrowClickReachesTheField(t *testing.T) {
	field := textfield.New(textfield.Placeholder("Search"))
	n := NewNarrow(field)
	uitest.LayoutWidget(n, 180, 26)
	n.SetBounds(geometry.NewRect(30, 40, 180, 26))
	if !uitest.SimulateClick(n, 40, 48) {
		t.Fatal("click missed the field")
	}
}

func TestSearchFieldPaintsTheTypedText(t *testing.T) {
	field := textfield.New(textfield.Placeholder("Search"))
	field.SetText("assay")
	n := NewNarrow(field)
	uitest.LayoutWidget(n, 180, 26)
	n.SetBounds(geometry.NewRect(0, 0, 180, 26))
	canvas := uitest.DrawWidget(n)
	var found bool
	for _, tx := range canvas.Texts {
		if tx.Text != "assay" {
			continue
		}
		found = true
		if tx.Bounds.Min.Y < 0 || tx.Bounds.Max.Y > 26 {
			t.Fatalf("text clipped %v", tx.Bounds)
		}
	}
	if !found {
		t.Fatal("typed text was not drawn")
	}
}

func TestNarrowFieldIsOneLine(t *testing.T) {
	f := NewNarrow(nil)
	size := f.Layout(nil, geometry.Constraints{MaxWidth: 220, MaxHeight: 200})
	if size.Height != 26 {
		t.Fatalf("height %v, want 26", size.Height)
	}
	_ = theme.Pairti()
}
