package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestViewSegmentIsTwoIcons(t *testing.T) {
	var got string
	seg := NewViewSegment(func(label string) { got = label })
	size := seg.Layout(nil, geometry.Constraints{MaxWidth: 200, MaxHeight: 40})
	if size.Width != toolSlot*2 || size.Height != toolSlot {
		t.Fatalf("size %v", size)
	}
	seg.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	rows := glyphs(uitest.DrawWidget(seg))
	joined := ""
	for _, row := range rows {
		joined += row.text
	}
	if strings.Contains(joined, "Formatted") || strings.Contains(joined, "Raw") {
		t.Fatalf("text labels %q", joined)
	}
	if !uitest.SimulateClick(seg, 8, 14) {
		t.Fatal("preview half was not consumed")
	}
	if got != "Formatted" {
		t.Fatalf("got %q", got)
	}
	if !uitest.SimulateClick(seg, toolSlot+8, 14) {
		t.Fatal("raw half was not consumed")
	}
	if got != "Raw" {
		t.Fatalf("got %q", got)
	}
}
