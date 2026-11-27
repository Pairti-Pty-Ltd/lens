package ui

import (
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/theme"
)

func TestOutlineNumbersHeadingsAndScrolls(t *testing.T) {
	src := []byte("# Grove code quality standard\n\n## Purpose and authority\n\n## Required checks for every change\n\n### Authorization and tenant isolation\n")
	p := NewPage()
	p.SetBlocks(doc.Blocks(src))
	uitest.LayoutWidget(p, 800, 2000)
	heads := p.Outline()
	if len(heads) != 4 {
		t.Fatalf("headings %d", len(heads))
	}
	want := []string{"1", "1.1", "1.2", "1.2.1"}
	for i, n := range want {
		if heads[i].Number != n {
			t.Fatalf("heading %d number %q", i, heads[i].Number)
		}
	}
	if heads[3].Text != "Authorization and tenant isolation" {
		t.Fatalf("text %q", heads[3].Text)
	}
	y1 := p.LineY(heads[0].Line)
	y2 := p.LineY(heads[1].Line)
	if y2 <= y1 {
		t.Fatalf("line y %v then %v", y1, y2)
	}
}

func TestRailContentsJumpAndStayShort(t *testing.T) {
	var jumped int
	var opened string
	rail := NewRail(func(line int) { jumped = line }, func(url string) { opened = url })
	rail.SetOutline([]Heading{
		{Number: "1", Text: "Grove code quality standard", Level: 1, Line: 1},
		{Number: "1.2.1", Text: "Authorization and tenant isolation for a long wrapped heading in the contents", Level: 3, Line: 40},
	})
	rail.SetInfo([]string{"assay.md", "notes/assay.md", "1,470 words", "8,200 characters", "extra"})
	rail.SetLinks([]Ref{{Label: "authentication", URL: "authentication.md", Line: 12}})
	uitest.LayoutWidget(rail, 280, 500)
	rail.SetBounds(geometry.NewRect(0, 0, 280, 500))
	canvas := uitest.DrawWidget(rail)
	rows := glyphs(canvas)
	if !hasGlyph(rows, "Table of contents") || !hasGlyph(rows, "1.2.1") {
		t.Fatalf("rows %+v", rows)
	}
	for _, tx := range rows {
		if tx.text == "40" || tx.text == "extra" {
			t.Fatalf("rail drew %q", tx.text)
		}
	}
	if len(rail.rows) < 2 || rail.rows[1].h < 30 {
		t.Fatalf("wrapped row %+v", rail.rows)
	}
	if !uitest.SimulateClick(rail, 40, rail.rows[1].y+8) {
		t.Fatal("heading click was not consumed")
	}
	if jumped != 40 {
		t.Fatalf("jump %d", jumped)
	}
	if !uitest.SimulateClick(rail, 8+2*28+8, 16) {
		t.Fatal("links icon was not consumed")
	}
	uitest.LayoutWidget(rail, 280, 500)
	rail.SetBounds(geometry.NewRect(0, 0, 280, 500))
	if rail.Mode() != 2 {
		t.Fatalf("mode %d", rail.Mode())
	}
	if !uitest.SimulateClick(rail, 40, rail.rows[0].y+8) {
		t.Fatal("link click was not consumed")
	}
	if opened != "authentication.md" {
		t.Fatalf("opened %q", opened)
	}
	rail.mode = 1
	rail.Layout(nil, geometry.Constraints{MaxWidth: 280, MaxHeight: 500})
	if len(rail.rows) != 4 {
		t.Fatalf("info rows %d", len(rail.rows))
	}
	_ = widget.Hex(theme.Pairti().Accent)
}

func TestTabsSelectCloseAndSplit(t *testing.T) {
	var selected, closed, split = -1, -1, -1
	bar := NewTabs(
		func(i int) { selected = i },
		func(i int) { closed = i },
		func(i int) { split = i },
	)
	bar.Set([]Tab{{Label: "index.md"}, {Label: "authentication.md", Dirty: true}}, 0, -1)
	uitest.LayoutWidget(bar, 400, 28)
	bar.SetBounds(geometry.NewRect(0, 0, 400, 28))
	if !uitest.SimulateClick(bar, bar.widths[0]+20, 14) {
		t.Fatal("tab click was not consumed")
	}
	if selected != 1 {
		t.Fatalf("selected %d", selected)
	}
	ctx := uitest.NewMockContext()
	bar.Event(ctx, uitest.Click(24, 14))
	bar.Event(ctx, uitest.MouseDrag(300, 14))
	bar.Event(ctx, uitest.Release(300, 14))
	if split != 0 {
		t.Fatalf("split %d", split)
	}
	closeX := bar.widths[0] - 8
	if !uitest.SimulateClick(bar, closeX, 14) {
		t.Fatal("close was not consumed")
	}
	if closed != 0 {
		t.Fatalf("closed %d", closed)
	}
}

func TestSplitRightPutsTheRailOnTheRight(t *testing.T) {
	rail := NewPage()
	doc := NewPage()
	s := NewSplit(rail, doc, 280)
	s.SetRight(true)
	uitest.LayoutWidget(s, 1000, 400)
	if rail.Bounds().Min.X <= doc.Bounds().Max.X-1 {
		t.Fatalf("rail %v doc %v", rail.Bounds(), doc.Bounds())
	}
	s.Toggle()
	uitest.LayoutWidget(s, 1000, 400)
	if rail.Bounds().Width() != 0 {
		t.Fatalf("hidden rail width %v", rail.Bounds().Width())
	}
	if doc.Bounds().Width() < 900 {
		t.Fatalf("document width %v", doc.Bounds().Width())
	}
}

func TestPairOpensBeside(t *testing.T) {
	a := NewPage()
	b := NewPage()
	p := NewPair(a, b)
	uitest.LayoutWidget(p, 800, 400)
	if a.Bounds().Width() < 700 || b.Bounds().Width() != 0 {
		t.Fatalf("closed pair a %v b %v", a.Bounds(), b.Bounds())
	}
	p.SetSplit(true)
	uitest.LayoutWidget(p, 800, 400)
	if a.Bounds().Width() < 300 || b.Bounds().Min.X <= a.Bounds().Min.X {
		t.Fatalf("split a %v b %v", a.Bounds(), b.Bounds())
	}
}
