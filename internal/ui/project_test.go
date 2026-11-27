package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
)

func TestProjectActionsIsTheNameAndTwoFilters(t *testing.T) {
	var dot, ignored bool
	pane := NewProjectActions(func(showDot, showIgnored bool) {
		dot, ignored = showDot, showIgnored
	})
	const name = "pairti-lense-documentation-archive-west"
	pane.Set(name, false, false, false, true)
	size := pane.Layout(nil, geometry.Constraints{MaxWidth: 280, MaxHeight: 400})
	if size.Height != toolSlot || size.Width != toolSlot*3+nameGap+toolSlot*2 {
		t.Fatalf("size %v", size)
	}
	pane.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	rows := glyphs(uitest.DrawWidget(pane))
	joined := ""
	for _, row := range rows {
		joined += row.text + "\n"
	}
	if strings.Contains(joined, name) || strings.Contains(joined, "Hidden") || strings.Contains(joined, "Ignored") {
		t.Fatalf("labels were drawn:\n%s", joined)
	}
	if !uitest.SimulateClick(pane, 10, 14) {
		t.Fatal("hidden click was not consumed")
	}
	if !dot || ignored {
		t.Fatalf("dot %v ignored %v", dot, ignored)
	}
	if !uitest.SimulateClick(pane, toolSlot+10, 14) {
		t.Fatal("ignored click was not consumed")
	}
	if !dot || !ignored {
		t.Fatalf("after ignored: dot %v ignored %v", dot, ignored)
	}
	canvas := uitest.DrawWidget(pane)
	var track bool
	for _, r := range canvas.RoundRects {
		if r.Radius >= 6 && r.Color == rgb(pane.theme.CodeFill) {
			track = true
		}
	}
	if !track {
		t.Fatal("name switch has no track")
	}
	var docs = true
	pane.OnDocs(func(on bool) { docs = on })
	if !uitest.SimulateClick(pane, toolSlot*2+10, 14) {
		t.Fatal("documents click was not consumed")
	}
	if docs {
		t.Fatal("documents filter stayed on")
	}
	var names bool
	pane.OnNames(func(on bool) { names = on })
	seg := toolSlot*3 + nameGap
	if !uitest.SimulateClick(pane, seg+toolSlot+8, 14) {
		t.Fatal("filename half was not consumed")
	}
	if !names {
		t.Fatal("filenames were not selected")
	}
	if !uitest.SimulateClick(pane, seg+8, 14) {
		t.Fatal("title half was not consumed")
	}
	if names {
		t.Fatal("titles were not selected")
	}
	empty := NewProjectActions(nil)
	size = empty.Layout(nil, geometry.Constraints{MaxWidth: 220, MaxHeight: 400})
	if size.Height != 0 {
		t.Fatalf("empty pane height %v", size.Height)
	}
}
