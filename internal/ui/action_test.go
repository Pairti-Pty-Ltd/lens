package ui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestActionClickChoosesTheirs(t *testing.T) {
	var got string
	a := NewAction(func(label string) { got = label })
	a.SetChoices([]string{"Ours", "Theirs", "Merge"})
	uitest.LayoutWidget(a, 400, 26)
	a.SetBounds(geometry.NewRect(0, 0, 400, 26))
	theirs := a.spans[1]
	if !uitest.SimulateClick(a, theirs.x+4, 8) {
		t.Fatal("click was not consumed")
	}
	if got != "Theirs" {
		t.Fatalf("got %q", got)
	}
}

func TestActionDrawsTheDecision(t *testing.T) {
	a := NewAction(nil)
	a.SetChoices([]string{"Ours", "Theirs", "Merge"})
	uitest.LayoutWidget(a, 400, 26)
	a.SetBounds(geometry.NewRect(0, 0, 400, 26))
	rows := glyphs(uitest.DrawWidget(a))
	for _, want := range []string{"Ours", "Theirs", "Merge"} {
		if !hasGlyph(rows, want) {
			t.Fatalf("missing %q in %+v", want, rows)
		}
	}
	if hasGlyph(rows, "Edit") {
		t.Fatal("Edit is on the decision row")
	}
}
