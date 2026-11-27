package ui

import (
	"strings"
	"testing"

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
