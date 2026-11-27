package ui

import (
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/theme"
)

func TestOpenLineWrapsToTheColumn(t *testing.T) {
	text := "**Status on 24 November 2026.** The three replacement homepages returned HTTP 200 at their Firebase addresses. The existing www domains still served the previous websites, pending DNS access."
	p := NewPage()
	p.measure = func(text string, size float32, bold bool, family string) float32 {
		return float32(len([]rune(text))) * size * 0.5
	}
	p.UseBuffer(edit.New(text + "\n"))
	p.SetBlocks(doc.Blocks([]byte(text + "\n")))
	p.openLine = 1
	const col = float32(420)
	var rows []rendered
	var joined strings.Builder
	for _, ln := range p.layoutBlocks(col) {
		if ln.src != 1 {
			continue
		}
		rows = append(rows, ln)
		for _, pc := range ln.pieces {
			if w := p.pieceWidth(pc); w > col+0.5 {
				t.Fatalf("piece width %v over %v: %q", w, col, pc.text)
			}
			joined.WriteString(pc.text)
		}
	}
	if len(rows) < 2 {
		t.Fatalf("rows %d", len(rows))
	}
	if joined.String() != text {
		t.Fatalf("joined %q", joined.String())
	}
	if rows[0].num != 1 || rows[1].num != 0 {
		t.Fatalf("numbers %d %d", rows[0].num, rows[1].num)
	}
	if strings.HasPrefix(rows[1].pieces[0].text, " ") {
		t.Fatalf("second row %q", rows[1].pieces[0].text)
	}
	caret := rows[1].col + 1
	if _, ok := caretInRow(rows[0], caret, len([]rune(text))); ok {
		t.Fatal("caret on the first row")
	}
	local, ok := caretInRow(rows[1], caret, len([]rune(text)))
	if !ok || local != 1 {
		t.Fatalf("caret local %d ok %v", local, ok)
	}
}

func TestClickRevealsHeadingMarksAndTypes(t *testing.T) {
	src := "status in beads, not in this document.\n\n## Required checks for every change\n"
	p := NewPage()
	buf := edit.New(src)
	p.UseBuffer(buf)
	p.SetBlocks(doc.Blocks([]byte(src)))
	uitest.LayoutWidget(p, 800, 400)
	p.SetBounds(geometry.NewRect(0, 0, 800, 400))
	for _, g := range glyphs(uitest.DrawWidget(p)) {
		if strings.Contains(g.text, "##") {
			t.Fatalf("marks visible before the click: %q", g.text)
		}
	}
	ctx := uitest.NewMockContext()
	if !p.Event(ctx, uitest.Click(p.gutterWidth()+10, p.LineTop(3)+4)) {
		t.Fatal("click was not consumed")
	}
	uitest.LayoutWidget(p, 800, 400)
	p.SetBounds(geometry.NewRect(0, 0, 800, 400))
	canvas := uitest.DrawWidget(p)
	accent := rgb(theme.Pairti().Accent)
	found := false
	for _, tx := range canvas.StyledTexts {
		if tx.Text != "##" {
			continue
		}
		found = true
		if tx.Style.Color != accent {
			t.Fatalf("mark color %v", tx.Style.Color)
		}
		if tx.Style.FontSize != theme.Pairti().H2 {
			t.Fatalf("mark size %v", tx.Style.FontSize)
		}
	}
	if !found {
		t.Fatalf("marks missing: %+v", glyphs(canvas))
	}
	body := false
	for _, g := range glyphs(canvas) {
		if strings.Contains(g.text, "Required checks for every change") {
			body = true
		}
	}
	if !body {
		t.Fatalf("heading text missing: %+v", glyphs(canvas))
	}
	caret := false
	for _, r := range canvas.Rects {
		if r.Bounds.Width() >= 6 && r.Color == accent {
			caret = true
		}
	}
	if !caret {
		t.Fatal("no view caret")
	}
	if p.ModeName() != "View" {
		t.Fatalf("mode %s", p.ModeName())
	}
	if !p.Event(ctx, uitest.KeyType(event.KeyI, 'i', 0)) {
		t.Fatal("i was not consumed")
	}
	if p.ModeName() != "Edit" {
		t.Fatalf("mode %s", p.ModeName())
	}
	if !p.Event(ctx, uitest.KeyType(event.KeyX, 'X', 0)) {
		t.Fatal("key was not consumed")
	}
	if !strings.Contains(buf.Text(), "X") || !strings.Contains(buf.Text(), "##") {
		t.Fatalf("buffer %q", buf.Text())
	}
	if !p.Event(ctx, uitest.KeyPress(event.KeyEscape, 0)) {
		t.Fatal("escape was not consumed")
	}
	if p.ModeName() != "View" {
		t.Fatalf("mode %s", p.ModeName())
	}
	uitest.LayoutWidget(p, 800, 400)
	p.SetBounds(geometry.NewRect(0, 0, 800, 400))
	if !hasGlyph(glyphs(uitest.DrawWidget(p)), "##") && !containsGlyph(glyphs(uitest.DrawWidget(p)), "##") {
		t.Fatal("marks hid after the first escape")
	}
	if !p.Event(ctx, uitest.KeyPress(event.KeyEscape, 0)) {
		t.Fatal("second escape was not consumed")
	}
	uitest.LayoutWidget(p, 800, 400)
	p.SetBounds(geometry.NewRect(0, 0, 800, 400))
	if !p.Event(ctx, uitest.KeyType(event.KeyK, 'k', 0)) || buf.CaretLine() != 1 {
		t.Fatal("second Escape interrupted keyboard navigation")
	}
}

func TestClickRevealsRawMarkdownInFixedPitch(t *testing.T) {
	src := "see **secret** here\n"
	p := NewPage()
	buf := edit.New(src)
	p.UseBuffer(buf)
	p.SetBlocks(doc.Blocks([]byte(src)))
	uitest.LayoutWidget(p, 800, 200)
	p.SetBounds(geometry.NewRect(0, 0, 800, 200))
	for _, g := range glyphs(uitest.DrawWidget(p)) {
		if strings.Contains(g.text, "**") {
			t.Fatalf("stars visible before the click: %q", g.text)
		}
	}
	ctx := uitest.NewMockContext()
	if !p.Event(ctx, uitest.Click(p.gutterWidth()+8, p.LineTop(1)+4)) {
		t.Fatal("click was not consumed")
	}
	uitest.LayoutWidget(p, 800, 200)
	p.SetBounds(geometry.NewRect(0, 0, 800, 200))
	canvas := uitest.DrawWidget(p)
	raw := false
	for _, tx := range canvas.StyledTexts {
		if !strings.Contains(tx.Text, "**secret**") {
			continue
		}
		raw = true
		if tx.Style.FontFamily != theme.Pairti().CodeFont {
			t.Fatalf("family %q", tx.Style.FontFamily)
		}
	}
	if !raw {
		t.Fatalf("raw line missing: %+v", glyphs(canvas))
	}
}
