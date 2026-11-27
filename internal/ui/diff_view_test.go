package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/gitlog"
)

func TestDiffWrapPreservesSourceAndWordHighlights(t *testing.T) {
	line := &gitlog.DiffLine{Runs: []gitlog.DiffRun{
		{Text: "\tbefore "},
		{Text: "résumé", Bold: true, Changed: true},
		{Text: " after", Tone: "comment"},
	}}
	rows := wrapDiffLine(line, 7)
	var all, emphasis strings.Builder
	for _, row := range rows {
		width := 0
		for _, run := range row {
			width += len([]rune(run.Text))
			all.WriteString(run.Text)
			if run.Changed {
				if !run.Bold {
					t.Fatal("lost emphasis across wrap")
				}
				emphasis.WriteString(run.Text)
			}
		}
		if width > 7 {
			t.Fatalf("row is %d cells", width)
		}
	}
	if all.String() != "    before résumé after" || emphasis.String() != "résumé" {
		t.Fatal(all.String(), emphasis.String())
	}
}

func TestDiffViewAlignedAndUnified(t *testing.T) {
	line := func(n int, text string) *gitlog.DiffLine { return &gitlog.DiffLine{Number: n, Text: text} }
	old := line(42, "old")
	old.Runs = []gitlog.DiffRun{{Text: "old", Tone: "keyword", Changed: true}}
	old.NoNewline = true
	model := gitlog.Comparison{Files: []gitlog.FileDiff{{
		OldPath: "a.md", NewPath: "a.md", Status: "Modified",
		Hunks: []gitlog.Hunk{{Header: "@@ -42,2 +42,2 @@", Rows: []gitlog.DiffRow{
			{Old: old, New: line(42, "new "+strings.Repeat("word ", 90))},
			{Old: line(43, "second"), New: line(43, "other")},
			{Old: line(44, "context"), New: line(44, "context"), Context: true},
		}}},
	}}}
	view := NewDiffView()
	view.SetComparison("Revise document", "26 Nov", model)
	draw := func(width float32) *uitest.MockCanvas {
		size := view.Layout(nil, geometry.Constraints{MaxWidth: width, MaxHeight: geometry.Infinity})
		view.SetBounds(geometry.NewRect(0, 0, width, size.Height))
		return uitest.DrawWidget(view)
	}
	canvas := draw(1100)
	var oldX, newX, oldY, newY, secondY, otherY float32
	var eof, context int
	for _, run := range canvas.StyledTexts {
		switch {
		case run.Text == "old":
			oldX, oldY = run.Bounds.Min.X, run.Bounds.Min.Y
			if run.Style.Color != rgb(view.theme.Accent) {
				t.Fatal("missing syntax colour")
			}
		case strings.HasPrefix(run.Text, "new "):
			newX, newY = run.Bounds.Min.X, run.Bounds.Min.Y
		case run.Text == "second":
			secondY = run.Bounds.Min.Y
		case run.Text == "other":
			otherY = run.Bounds.Min.Y
		case run.Text == "No newline at end of file":
			eof++
		case run.Text == "context":
			context++
		}
	}
	if !view.sideBySide || newX-oldX < 500 || oldY != newY || secondY != otherY || secondY < oldY+2*view.lineH() || eof != 1 || context != 2 {
		t.Fatalf("positions %.0f,%.0f / %.0f,%.0f; next %.0f,%.0f eof %d context %d", oldX, oldY, newX, newY, secondY, otherY, eof, context)
	}
	highlighted := false
	for _, r := range canvas.Rects {
		if r.Color == mixHex(view.theme.Canvas, 0xc74952, 0.46) {
			highlighted = true
		}
	}
	if !highlighted {
		t.Fatal("missing inline change background")
	}
	canvas = draw(520)
	var source []string
	for _, run := range canvas.StyledTexts {
		switch {
		case strings.HasPrefix(run.Text, "new "):
			source = append(source, "new")
		case run.Text == "old", run.Text == "second", run.Text == "other", run.Text == "context":
			source = append(source, run.Text)
		}
	}
	if view.sideBySide || strings.Join(source, ",") != "old,second,new,other,context" {
		t.Fatalf("unified order %v", source)
	}
	if view.Event(nil, event.NewKeyEvent(event.KeyPress, event.KeyX, 'x', 0)) {
		t.Fatal("read-only diff consumed edit key")
	}
}
