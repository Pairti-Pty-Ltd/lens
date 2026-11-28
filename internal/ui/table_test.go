package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
)

func TestFormattedTableKeepsStyledLabelsAndSourceRows(t *testing.T) {
	src := "| Label | Meaning |\n| --- | --- |\n| **Roadmap opportunity** | An *early* idea with " + strings.Repeat("more detail ", 20) + "|\n\nAfter."
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte(src)))
	rows := p.layoutBlocks(560)
	if len(rows) < 2 || len(rows[1].cellLines) != 2 {
		t.Fatalf("missing table body")
	}
	label := rows[1].cellLines[0]
	if len(label) != 1 || len(label[0]) != 1 || label[0][0].text != "Roadmap opportunity" || !label[0][0].bold {
		t.Fatalf("label wrapped or lost bold: %+v", label)
	}
	var italic bool
	for _, line := range rows[1].cellLines[1] {
		for _, run := range line {
			italic = italic || (run.text == "early" && run.italic)
		}
	}
	if !italic || rows[1].src != 3 {
		t.Fatalf("italic %v body source %d", italic, rows[1].src)
	}
	for _, row := range rows {
		if row.gap && row.src <= 3 {
			t.Fatalf("phantom blank at %d", row.src)
		}
	}

	p.buf = edit.New(src)
	p.openLine = 3
	rows = p.layoutBlocks(560)
	for _, row := range rows {
		if len(row.cols) > 0 {
			t.Fatal("table body selection did not expose Markdown")
		}
	}
}

func TestTableWrapKeepsLongUnicodeWords(t *testing.T) {
	p := NewPage()
	text := strings.Repeat("日本語", 12)
	runs := p.tableRuns([]doc.Span{{Text: text, Bold: true}}, false)
	lines := p.wrapTableRuns(runs, p.pieceWidth(runs[0])/9)
	var got strings.Builder
	for _, line := range lines {
		for _, run := range line {
			if !run.bold {
				t.Fatal("lost bold after wrapping")
			}
			got.WriteString(run.text)
		}
	}
	if len(lines) < 2 || got.String() != text {
		t.Fatalf("wrapped into %d lines: %q", len(lines), got.String())
	}
}

func TestPandocGridResizeAndReveal(t *testing.T) {
	src, err := os.ReadFile("../../testdata/orchard-grid.md")
	if err != nil {
		t.Fatal(err)
	}
	p := NewPage()
	p.SetBlocks(doc.Blocks(src))
	p.UseBuffer(edit.New(string(src)))
	for _, width := range []float32{640, 210} {
		c := drawPage(p, width, 1200)
		if len(c.StrokeRoundRects) != 2 {
			t.Fatalf("width %v: %d cards", width, len(c.StrokeRoundRects))
		}
		for _, word := range []string{"North", "South", "•", "gate", "map", "Field", "date"} {
			if !hasGlyph(glyphs(c), word) {
				t.Fatalf("missing %s at %v: %+v", word, width, glyphs(c))
			}
		}
		for _, g := range glyphs(c) {
			if strings.ContainsAny(g.text, "|{}+") || ((g.text == "North" || g.text == "South") && !g.bold) {
				t.Fatalf("unexpected glyph %+v", g)
			}
		}
		if glyphY(c, "•")-glyphY(c, "North") < 2*p.theme.Body*proseLead-1 {
			t.Fatal("missing paragraph gap")
		}
		mark := mixHex(p.theme.Surface, p.theme.Number, 0.42)
		for _, word := range []string{"field", "note", "Follow", "shed", "date", ","} {
			box := glyphBox(c, word)
			point := geometry.Pt(box.Min.X+1, box.Min.Y+1)
			covered := false
			for _, r := range c.Rects {
				if r.Color == mark && r.Bounds.Contains(point) {
					covered = true
					break
				}
			}
			if covered != (word != ",") {
				t.Fatalf("width %v glyph %q highlight %v", width, word, covered)
			}
		}
		if width == 210 && glyphY(c, "shed") <= glyphY(c, "Follow") {
			t.Fatal("marked passage did not wrap")
		}
		for _, hit := range p.hits {
			if hit.rect.Max.X > width {
				t.Fatalf("clipped link: %+v", hit)
			}
		}
		if len(p.hits) != 2 {
			t.Fatalf("links: %+v", p.hits)
		}
	}
	box := glyphBox(drawPage(p, 210, 1200), "date")
	if !p.openAt(nil, geometry.Pt(box.Min.X+1, box.Min.Y+1)) || p.openLine != 16 {
		t.Fatalf("source line %d", p.openLine)
	}
	drawPage(p, 210, 1200)
	if p.Buffer().Text() != string(src) {
		t.Fatal("revealing source changed Markdown")
	}
}

func TestPandocDirectoryGrid(t *testing.T) {
	src, err := os.ReadFile("../../testdata/directory-grid.md")
	if err != nil {
		t.Fatal(err)
	}
	p := NewPage()
	p.SetBlocks(doc.Blocks(src))
	p.UseBuffer(edit.New(string(src)))
	for _, width := range []float32{720, 250} {
		c := drawPage(p, width, 1200)
		if len(c.StrokeRoundRects) != 1 {
			t.Fatalf("width %v: got %d tables", width, len(c.StrokeRoundRects))
		}
		if len(p.hits) != 2 || p.hits[0].url != "guide.md" || p.hits[1].url != "desk.md" {
			t.Fatalf("links at %v: %+v", width, p.hits)
		}
		alder, birch := glyphY(c, "Alder"), glyphY(c, "Birch")
		if alder < 0 || p.hits[0].rect.Min.Y <= alder || p.hits[0].rect.Max.Y > birch || p.hits[1].rect.Min.Y < birch || glyphY(c, "Directory") <= birch {
			t.Fatalf("row placement at %v: %+v", width, glyphs(c))
		}
		for _, hit := range p.hits {
			if hit.rect.Min.X <= glyphBox(c, "Alder").Max.X || hit.rect.Max.X > width {
				t.Fatalf("link outside its column: %+v", hit)
			}
		}
		for _, g := range glyphs(c) {
			if strings.ContainsAny(g.text, "|+=") {
				t.Fatalf("border drawn: %+v", g)
			}
		}
	}
	if p.Buffer().Text() != string(src) {
		t.Fatal("formatted view changed the source")
	}
}
