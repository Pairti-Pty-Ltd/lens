package ui

import (
	"context"
	"fmt"
	"image"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/diagram"
	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
)

func TestPageUsesTheOfferedWidth(t *testing.T) {
	src := strings.Repeat("alpha ", 80)
	wide := NewPage()
	wide.SetBlocks(doc.Blocks([]byte(src)))
	wide.Layout(nil, geometry.Constraints{MaxWidth: 1200, MaxHeight: geometry.Infinity})
	narrow := NewPage()
	narrow.SetBlocks(doc.Blocks([]byte(src)))
	narrow.Layout(nil, geometry.Constraints{MaxWidth: 360, MaxHeight: geometry.Infinity})
	if len(wide.lines) == 0 || len(wide.lines) >= len(narrow.lines) {
		t.Fatalf("wide lines %d narrow lines %d", len(wide.lines), len(narrow.lines))
	}
}

func TestDocumentScrollContentIsTheWholeNote(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, "line %d of the assay note\n\n", i)
	}
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte(b.String())))
	sv := scrollview.New(p, scrollview.DirectionOpt(scrollview.Vertical))
	uitest.LayoutWidget(sv, 900, 400)
	h := sv.ContentSize().Height
	if h <= 400 {
		t.Fatalf("content height %v, viewport 400", h)
	}
	if 400/h > 0.5 {
		t.Fatalf("thumb would cover most of the track, content %v", h)
	}
}

func TestPageLongTitleIsTheHeadingOnce(t *testing.T) {
	src, err := os.ReadFile("../../testdata/membrane-assay.md")
	if err != nil {
		t.Fatal(err)
	}
	p := NewPage()
	p.SetLineNumbers(true)
	p.SetBlocks(doc.Blocks(src))
	uitest.LayoutWidget(p, 860, 2400)
	p.SetBounds(geometry.NewRect(0, 0, 860, 2400))
	rows := glyphs(uitest.DrawWidget(p))
	title := "Write-up"
	nTitle, nH1 := 0, 0
	for _, tx := range rows {
		if strings.Contains(tx.text, title) {
			nTitle++
		}
		if tx.text == "h1" {
			nH1++
		}
		if strings.Contains(tx.text, ":") && len(tx.text) < 12 {
			t.Fatalf("coordinate drawn as text: %q", tx.text)
		}
	}
	if nTitle != 1 || nH1 != 1 {
		t.Fatalf("title %d h1 %d", nTitle, nH1)
	}
}

type fakeDiagram struct {
	n   int
	img image.Image
	err error
}

func (f *fakeDiagram) Languages() []string { return []string{"mermaid"} }

func (f *fakeDiagram) Render(context.Context, string) (image.Image, error) {
	f.n++
	return f.img, f.err
}

func TestPageMermaidFenceBecomesThePicture(t *testing.T) {
	diagram.Reset()
	t.Cleanup(diagram.Reset)
	img := image.NewNRGBA(image.Rect(0, 0, 12, 9))
	fake := &fakeDiagram{img: img}
	diagram.Register(fake)
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte("```mermaid\ngraph TD\nA-->B\n```\n\nAfter.\n")))
	canvas := drawPage(p, 640, 400)
	if len(canvas.Images) != 1 || canvas.Images[0].Image.Bounds().Dx() != 12 {
		t.Fatalf("images %d", len(canvas.Images))
	}
	for _, tx := range glyphs(canvas) {
		if strings.Contains(tx.text, "graph") || strings.Contains(tx.text, "A-->B") {
			t.Fatalf("source drawn under the diagram: %q", tx.text)
		}
	}
	if !hasGlyph(glyphs(canvas), "After.") {
		t.Fatal("paragraph after the diagram is missing")
	}
	uitest.LayoutWidget(p, 640, 400)
	if fake.n != 1 {
		t.Fatalf("rendered %d times", fake.n)
	}
}

func TestDiagramKeepsItsHeight(t *testing.T) {
	diagram.Reset()
	t.Cleanup(diagram.Reset)
	diagram.Register(&fakeDiagram{img: image.NewNRGBA(image.Rect(0, 0, 1000, 4000))})
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte("```mermaid\ngraph TD\nA-->B\n```\n")))
	canvas := drawPage(p, 400, 2000)
	if len(canvas.Images) != 1 {
		t.Fatal("diagram was not drawn")
	}
	h := canvas.Images[0].Image.Bounds().Dy()
	if h < 800 {
		t.Fatalf("diagram height %d was crushed", h)
	}
}

func TestPageMermaidFailureKeepsTheSource(t *testing.T) {
	diagram.Reset()
	t.Cleanup(diagram.Reset)
	diagram.Register(&fakeDiagram{err: &diagram.NotInstalledError{Name: "mmdc"}})
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte("```mermaid\ngraph TD\nA-->B\n```\n")))
	rows := glyphs(drawPage(p, 640, 300))
	if !hasGlyph(rows, "mmdc not installed") || !hasGlyph(rows, "A-->B") {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestPageGoFenceIsNotADiagram(t *testing.T) {
	diagram.Reset()
	t.Cleanup(diagram.Reset)
	diagram.Register(&fakeDiagram{img: image.NewNRGBA(image.Rect(0, 0, 4, 4))})
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte("```go\nfmt.Println\n```\n")))
	canvas := drawPage(p, 480, 200)
	if len(canvas.Images) != 0 {
		t.Fatal("go fence was rendered as a diagram")
	}
	rows := glyphs(canvas)
	if !hasGlyph(rows, "Println") && !hasGlyph(rows, "fmt.Println") {
		t.Fatal("go source missing")
	}
}

func TestPageHighlightsSource(t *testing.T) {
	src := "import { logger } from '@/lib/sentry';\n/**\n * Used by polling.\n */\nexport const n = 1;\n"
	p := NewPage()
	p.SetPath("poll-retry.ts")
	p.SetBlocks(doc.Render("poll-retry.ts", []byte(src)))
	canvas := drawPage(p, 860, 600)
	var sawKeyword, sawComment bool
	for _, tx := range canvas.StyledTexts {
		if tx.Text == "import" || tx.Text == "export" || tx.Text == "const" {
			if tx.Style.Color != widget.Hex(0x2979FF) {
				t.Fatalf("%q color %v", tx.Text, tx.Style.Color)
			}
			sawKeyword = true
		}
		if strings.Contains(tx.Text, "Used by polling") {
			if tx.Style.Color != widget.Hex(0xA3A3A3) {
				t.Fatalf("comment color %v", tx.Style.Color)
			}
			if !strings.Contains(tx.Text, "*") {
				t.Fatalf("comment lost its mark: %q", tx.Text)
			}
			sawComment = true
		}
	}
	if !sawKeyword || !sawComment {
		t.Fatalf("keyword %v comment %v", sawKeyword, sawComment)
	}
	for _, tx := range canvas.StyledTexts {
		if tx.Text == "▪" || tx.Text == "•" {
			t.Fatalf("comment drawn as a bullet %q", tx.Text)
		}
	}
}

func TestPageWordsUseMeasuredAdvances(t *testing.T) {
	p := NewPage()
	p.SetBlocks([]doc.Block{{
		Kind:  "p",
		Spans: []doc.Span{{Text: "Hello"}, {Text: "world"}},
	}})
	uitest.LayoutWidget(p, 480, 200)
	p.SetBounds(geometry.NewRect(0, 0, 480, 200))
	canvas := uitest.DrawWidget(p)
	rows := glyphs(canvas)
	var hello, world *glyph
	for i := range rows {
		if rows[i].text == "Hello" {
			hello = &rows[i]
		}
		if rows[i].text == "world" {
			world = &rows[i]
		}
	}
	if hello == nil || world == nil {
		t.Fatalf("rows = %+v", rows)
	}
	gap := world.y - hello.y
	if gap != 0 {
		t.Fatalf("words wrapped onto separate lines, y %v %v", hello.y, world.y)
	}
	want := helloX(canvas, "Hello") + canvas.MeasureStyledText("Hello ", widget.TextStyle{FontSize: p.theme.Body, FontFamily: p.theme.UIFont})
	if worldX(canvas, "world") != want {
		t.Fatalf("world x = %v, want %v", worldX(canvas, "world"), want)
	}
}

func helloX(c *uitest.MockCanvas, text string) float32 { return wordX(c, text) }
func worldX(c *uitest.MockCanvas, text string) float32 { return wordX(c, text) }

func wordX(c *uitest.MockCanvas, text string) float32 {
	for _, t := range c.Texts {
		if t.Text == text {
			return t.Bounds.Min.X
		}
	}
	for _, t := range c.StyledTexts {
		if t.Text == text {
			return t.Bounds.Min.X
		}
	}
	return -1
}

func TestFrontMatterDrawsTheCard(t *testing.T) {
	src := []byte("---\ntitle: Fern\nclient: Alpha\n---\n\n# Write-up\n")
	p := NewPage()
	p.SetBlocks(doc.Blocks(src))
	uitest.LayoutWidget(p, 640, 800)
	p.SetBounds(geometry.NewRect(0, 0, 640, 800))
	canvas := uitest.DrawWidget(p)
	if wordX(canvas, "Document details") < 0 || wordX(canvas, "title:") >= 0 || wordX(canvas, "Write-up") < 0 {
		t.Fatal("expected only the disclosure and document heading")
	}
	click := &event.MouseEvent{MouseType: event.MousePress, Button: event.ButtonLeft, Position: geometry.Pt(90, 10)}
	if !p.Event(nil, click) {
		t.Fatal("disclosure did not expand")
	}
	uitest.LayoutWidget(p, 640, 800)
	canvas = uitest.DrawWidget(p)
	accent, muted := rgb(p.theme.Accent), rgb(p.theme.Muted)
	var label, key, value, title int
	for _, tx := range canvas.StyledTexts {
		switch tx.Text {
		case "Document details":
			label++
		case "title:":
			key++
			if tx.Style.Color != accent {
				t.Fatalf("key colour %v", tx.Style.Color)
			}
		case " Fern":
			value++
			if tx.Style.Color != muted {
				t.Fatalf("value colour %v", tx.Style.Color)
			}
		case "Write-up":
			title++
		}
	}
	if label != 1 || key != 1 || value != 1 || title != 1 {
		t.Fatalf("label %d key %d value %d title %d", label, key, value, title)
	}
	var card bool
	for _, r := range canvas.RoundRects {
		if r.Radius >= 6 && r.Bounds.Width() > 200 && r.Color == rgb(p.theme.Surface) {
			card = true
		}
	}
	if !card {
		t.Fatalf("no card %+v", canvas.RoundRects)
	}
}

func TestDocumentDetailsCanEditAndRefold(t *testing.T) {
	src := "---\ntitle: Fern\n---\n\n# Contents"
	p := NewPage()
	p.UseBuffer(edit.New(src))
	p.SetBlocks(doc.Blocks([]byte(src)))
	p.Reopen(1)
	uitest.LayoutWidget(p, 640, 800)
	p.SetBounds(geometry.NewRect(0, 0, 640, 800))
	click := func(y float32) {
		t.Helper()
		if !p.Event(nil, &event.MouseEvent{MouseType: event.MousePress, Button: event.ButtonLeft, Position: geometry.Pt(100, y)}) {
			t.Fatal("click ignored")
		}
		uitest.LayoutWidget(p, 640, 800)
	}
	if p.frontExpanded {
		t.Fatal("initial caret expanded details")
	}
	click(10)
	click(p.LineTop(2) + 1)
	if p.OpenLine() != 2 || p.buf.CaretLine() != 1 {
		t.Fatalf("open %d caret %d", p.OpenLine(), p.buf.CaretLine())
	}
	if !p.Event(nil, uitest.KeyType(event.KeyI, 'i', 0)) || !p.Event(nil, uitest.KeyType(event.KeyX, 'X', 0)) {
		t.Fatal("metadata editor did not consume typing")
	}
	edited := p.buf.Text()
	if strings.Replace(edited, "X", "", 1) != src || !strings.Contains(strings.Split(edited, "\n")[1], "X") {
		t.Fatalf("unexpected source after editing details: %q", edited)
	}
	click(10)
	if p.frontExpanded || p.buf.Text() != edited {
		t.Fatal("fold changed source or stayed open")
	}
	p.Reopen(2)
	if !p.frontExpanded {
		t.Fatal("metadata navigation did not reveal target")
	}
}

func TestPageDrawsStyledMarkdown(t *testing.T) {
	src := []byte("A **bold** word.\n\n> quoted\n\n| A | B |\n| - | - |\n| 1 | 2 |\n")
	p := NewPage()
	p.SetBlocks(doc.Blocks(src))
	uitest.LayoutWidget(p, 480, 800)
	p.SetBounds(geometry.NewRect(0, 0, 480, 800))
	rows := glyphs(uitest.DrawWidget(p))

	var sawBold, sawQuote, sawCell bool
	for _, tx := range rows {
		if strings.Contains(tx.text, "│") || strings.Contains(tx.text, "─") {
			t.Fatalf("box drawing glyph %q", tx.text)
		}
		if tx.bold && tx.text == "bold" {
			sawBold = true
		}
		if tx.text == "quoted" {
			sawQuote = true
		}
		if tx.text == "2" {
			sawCell = true
		}
	}
	if !sawBold || !sawQuote || !sawCell {
		t.Fatalf("bold=%v quote=%v cell=%v texts=%+v", sawBold, sawQuote, sawCell, rows)
	}
	canvas := uitest.DrawWidget(p)
	if !verticalBar(canvas) {
		t.Fatal("quote has no bar")
	}
}

func TestTableUsesOneGrid(t *testing.T) {
	src := []byte(strings.TrimSpace(`
| Command | Notes |
| --- | --- |
| bun dev | Run API with hot reload. Don't kill the dev server between iterations — it reloads on file save. |
| bun check | TypeScript check. May OOM — fall back to a larger heap. |
| bun generate | Regenerate Prisma client after schema changes. |
| bun test:rate-limit / bun test:load / bun test:soak | Targeted suites for the load harness and the rate limit checks that run before release. |
`) + "\n")
	p := NewPage()
	p.SetBlocks(doc.Blocks(src))
	uitest.LayoutWidget(p, 860, 900)
	p.SetBounds(geometry.NewRect(0, 0, 860, 900))
	canvas := uitest.DrawWidget(p)
	rows := glyphs(canvas)
	for _, want := range []string{"bun generate", "bun check", "save.", "release.", "soak"} {
		if !glyphHas(rows, want) {
			t.Fatalf("missing %q in %+v", want, rows)
		}
	}
	if !hasGlyph(rows, "bun check") || !hasGlyph(rows, "bun generate") {
		t.Fatalf("command clipped: %+v", rows)
	}
	xs := map[int]map[int]bool{}
	var ruleW float32
	rules := 0
	for _, ln := range canvas.Lines {
		if ln.From.X == ln.To.X && ln.To.Y > ln.From.Y {
			y := int(ln.From.Y)
			if xs[y] == nil {
				xs[y] = map[int]bool{}
			}
			xs[y][int(ln.From.X+0.5)] = true
		}
		if ln.From.Y == ln.To.Y && ln.To.X > ln.From.X {
			w := ln.To.X - ln.From.X
			if rules == 0 {
				ruleW = w
			} else if w < ruleW-1 || w > ruleW+1 {
				t.Fatalf("row width %v, first row %v", w, ruleW)
			}
			rules++
		}
	}
	if len(xs) != 0 {
		t.Fatalf("table drew vertical rules %v", xs)
	}
	measure := 860 - p.gutterWidth()
	if ruleW < measure-1 || ruleW > measure+1 {
		t.Fatalf("table width %v, want %v", ruleW, measure)
	}
	var card bool
	for _, r := range canvas.RoundRects {
		if r.Bounds.Width() > measure-2 && r.Bounds.Width() < measure+2 && r.Radius >= 8 {
			card = true
		}
	}
	if !card {
		t.Fatal("table has no card")
	}
}

func TestFigureRowAndAlert(t *testing.T) {
	src := []byte(strings.TrimSpace(`
> [!WARNING]
> **Ship Basic.**
>
> Hold the rest.

| 4 | 5 | 1 | 11 |
| --- | --- | --- | --- |
| Holds before one number | Guides that misdirect | Scorer on the analytics path | Catalogue readers, of 270 names |

| Hold | Where |
| --- | --- |
| • Advanced analytics | overview.ts |
`) + "\n")
	p := NewPage()
	p.SetBlocks(doc.Blocks(src))
	uitest.LayoutWidget(p, 860, 900)
	p.SetBounds(geometry.NewRect(0, 0, 860, 900))
	canvas := uitest.DrawWidget(p)
	rows := glyphs(canvas)
	if glyphHas(rows, "WARNING") || glyphHas(rows, "[!") {
		t.Fatalf("alert tag drawn %+v", rows)
	}
	if !glyphHas(rows, "Ship") || !glyphHas(rows, "⚠") || !glyphHas(rows, "Hold") {
		t.Fatalf("alert %+v", rows)
	}
	var numSize float32
	for _, tx := range canvas.StyledTexts {
		if tx.Text == "4" && tx.Style.FontSize > numSize {
			numSize = tx.Style.FontSize
		}
	}
	for _, tx := range canvas.Texts {
		if tx.Text == "4" && tx.FontSize > numSize {
			numSize = tx.FontSize
		}
	}
	if numSize < 28 {
		t.Fatalf("figure size %v", numSize)
	}
	if !glyphHas(rows, "Holds before one number") || !glyphHas(rows, "Advanced analytics") {
		t.Fatalf("captions %+v", rows)
	}
	if len(canvas.Circles) == 0 {
		t.Fatal("table bullet was a glyph")
	}
}

func glyphHas(rows []glyph, sub string) bool {
	for _, r := range rows {
		if strings.Contains(r.text, sub) {
			return true
		}
	}
	return false
}

func keysOf(set map[int]bool) string {
	var ns []int
	for n := range set {
		ns = append(ns, n)
	}
	sortInts(ns)
	s := ""
	for _, n := range ns {
		if s != "" {
			s += ","
		}
		s += strconv.Itoa(n)
	}
	return s
}

func sortInts(ns []int) {
	for i := 1; i < len(ns); i++ {
		j := i
		for j > 0 && ns[j] < ns[j-1] {
			ns[j], ns[j-1] = ns[j-1], ns[j]
			j--
		}
	}
}

func TestLinkPunctuationStaysTightAndClicks(t *testing.T) {
	var got string
	p := NewPage()
	p.SetFollow(func(u string) { got = u })
	p.SetBlocks(doc.Blocks([]byte("Sources: [authentication](authentication.md), [mail integrations](mail.md).\n")))
	canvas := drawPage(p, 860, 200)
	ax := wordX(canvas, "authentication")
	cx := wordX(canvas, ",")
	aw := canvas.MeasureStyledText("authentication", widget.TextStyle{FontSize: p.theme.Body, FontFamily: p.theme.UIFont})
	if ax < 0 || cx < 0 || cx != ax+aw {
		t.Fatalf("comma x %v, authentication x %v width %v", cx, ax, aw)
	}
	for _, tx := range canvas.StyledTexts {
		if tx.Text == "," && tx.Style.Color == widget.Hex(0x2979FF) {
			t.Fatal("comma took the link colour")
		}
	}
	if len(p.hits) < 2 || p.hits[0].url != "authentication.md" {
		t.Fatalf("hits %+v", p.hits)
	}
	h := p.hits[0].rect
	if !uitest.SimulateClick(p, h.Min.X+2, h.Min.Y+2) {
		t.Fatal("click was not consumed")
	}
	if got != "authentication.md" {
		t.Fatalf("opened %q", got)
	}
}

func TestLineNumberSitsOnTheHeadingBaseline(t *testing.T) {
	p := NewPage()
	p.SetLineNumbers(true)
	p.SetBlocks(doc.Blocks([]byte("# Title\n")))
	uitest.LayoutWidget(p, 640, 200)
	p.SetBounds(geometry.NewRect(0, 0, 640, 200))
	rows := glyphs(uitest.DrawWidget(p))
	num, title := glyphYOf(rows, "1"), glyphYOf(rows, "Title")
	shift := baselineShift(p.theme.H1, p.theme.Meta)
	if num < 0 || title < 0 {
		t.Fatalf("num %v title %v", num, title)
	}
	if num < title+shift-0.6 || num > title+shift+0.6 {
		t.Fatalf("number y %v title y %v shift %v", num, title, shift)
	}
}

func TestFormattedParagraphSeparatorUsesLessThanOneLine(t *testing.T) {
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte("one two three four five six seven eight nine ten eleven twelve\n\nNext.\n")))
	uitest.LayoutWidget(p, 180, 400)
	var wrap, gap float32
	for i, ln := range p.lines {
		if ln.gap {
			gap = p.rowHeight(ln)
			continue
		}
		if i > 0 && ln.num == 0 && len(ln.pieces) > 0 && wrap == 0 {
			wrap = p.rowHeight(ln)
		}
	}
	lead := p.theme.Body * proseLead
	if wrap != lead {
		t.Fatalf("wrap height %v want %v", wrap, lead)
	}
	if gap <= 0 || gap >= wrap {
		t.Fatalf("paragraph separator %v should be positive and shorter than a full line %v", gap, wrap)
	}
}

func glyphYOf(rows []glyph, text string) float32 {
	for _, r := range rows {
		if r.text == text {
			return r.y
		}
	}
	return -1
}

func TestPageNumbersSourceLines(t *testing.T) {
	p := NewPage()
	p.SetLineNumbers(true)
	p.SetBlocks(doc.Blocks([]byte("# Title\n\nHello\n")))
	uitest.LayoutWidget(p, 640, 400)
	p.SetBounds(geometry.NewRect(0, 0, 640, 400))
	rows := glyphs(uitest.DrawWidget(p))
	need := []string{"1", "h1", "Title", "2", "3", "Hello"}
	for _, want := range need {
		if !hasGlyph(rows, want) {
			t.Fatalf("missing %q in %+v", want, rows)
		}
	}
}

func TestFootnoteAndDocLinkRender(t *testing.T) {
	const (
		uri  = "doc:///research/sample-orchard-field-survey/PROPOSAL.md"
		note = "Count the trees before recording the plot area."
	)
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte(complianceNote)))
	p.toggleDetails()
	uitest.LayoutWidget(p, 720, 2000)
	p.SetBounds(geometry.NewRect(0, 0, 720, 2000))
	canvas := uitest.DrawWidget(p)
	var flat strings.Builder
	for _, tx := range canvas.StyledTexts {
		flat.WriteString(tx.Text)
	}
	text := flat.String()
	if strings.Contains(text, "[^") {
		t.Fatalf("raw footnote:\n%s", text)
	}
	if strings.Count(text, note) != 1 {
		t.Fatalf("footnote sentence:\n%s", text)
	}
	var linked, coded bool
	for _, tx := range canvas.StyledTexts {
		if tx.Text != uri {
			continue
		}
		if tx.Style.FontFamily == "Courier New" {
			coded = tx.Style.Color != widget.Hex(0x2979FF)
		}
		if tx.Style.Color == widget.Hex(0x2979FF) && tx.Style.FontFamily != "Courier New" {
			linked = true
		}
	}
	if !linked || !coded {
		t.Fatalf("linked %v coded %v", linked, coded)
	}
}

const complianceNote = "" +
	"---\n" +
	"title: Orchard Survey and Sampling Plan\n" +
	"type: topic\n" +
	"sources: [doc:///research/sample-orchard-field-survey/PROPOSAL.md]\n" +
	"---\n" +
	"\n" +
	"The orchard survey covers six sampling sites.[^1]\n" +
	"\n" +
	"A path in a code span stays a path: `doc:///research/sample-orchard-field-survey/PROPOSAL.md`\n" +
	"\n" +
	"[^1]: Count the trees before recording the plot area.\n"

func TestWikiLinkShowsTheLabelOnce(t *testing.T) {
	const label = "Orchard Planting Calendar (Now, Next, Later)"
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte("Unifies Grove-P, [[Grove-F]], and regional planting plots.\n\n- [[" + label + "|" + label + "]]\n\n`[[Grove-F]]`\n")))
	uitest.LayoutWidget(p, 720, 800)
	p.SetBounds(geometry.NewRect(0, 0, 720, 800))
	canvas := uitest.DrawWidget(p)
	var flat strings.Builder
	var linked bool
	for _, tx := range canvas.StyledTexts {
		flat.WriteString(tx.Text)
		if tx.Text == "Grove-F" && tx.Style.Color == widget.Hex(0x2979FF) {
			linked = true
		}
	}
	text := flat.String()
	if strings.Count(text, label) != 1 || strings.Count(text, "[[Grove-F]]") != 1 {
		t.Fatalf("label or code:\n%s", text)
	}
	if strings.Contains(strings.ReplaceAll(text, "[[Grove-F]]", ""), "[[") {
		t.Fatalf("a wiki link kept its brackets:\n%s", text)
	}
	if !linked {
		t.Fatal("Grove-F was not a link")
	}
}

func TestNumberedWrapHangsUnderTheText(t *testing.T) {
	p := NewPage()
	p.measure = func(text string, size float32, bold bool, family string) float32 {
		em := float32(0.5)
		if family == p.theme.CodeFont {
			em = 0.62
		}
		return float32(len([]rune(text))) * size * em
	}
	src := "1. Event-invite RSVP links fail at our API. Production sends the caller address in a `Forwarded` header. The limiter never reads it, so every pixel and every RSVP link shares one counter called `unknown`.\n"
	var li doc.Block
	for _, b := range doc.Blocks([]byte(src)) {
		if b.Kind == "li" {
			li = b
			break
		}
	}
	if li.Kind != "li" {
		t.Fatal("no list item")
	}
	const col = float32(480)
	lines := p.layoutFlow(li, li.Spans, col, 16, 0, rgb(0xFFFFFF), false, widget.Color{}, p.theme.UIFont)
	if len(lines) < 2 {
		t.Fatalf("lines %d", len(lines))
	}
	if lines[0].pieces[0].text != "1.  " {
		t.Fatalf("marker %q", lines[0].pieces[0].text)
	}
	hang := p.pieceWidth(lines[0].pieces[0])
	var body strings.Builder
	for i, ln := range lines {
		if w := p.piecesWidth(ln.pieces); w > col+0.5 {
			t.Fatalf("line %d width %v over %v", i, w, col)
		}
		if i > 0 {
			if ln.pieces[0].pad <= 0 || abs32(ln.pieces[0].pad-hang) > 0.5 {
				t.Fatalf("line %d hang %v want %v", i, ln.pieces[0].pad, hang)
			}
		}
		for _, pc := range ln.pieces {
			if pc.pad > 0 || pc.text == "1.  " {
				continue
			}
			body.WriteString(pc.text)
		}
	}
	if strings.Contains(body.String(), "  ") {
		t.Fatalf("double space:\n%s", body.String())
	}
	if !strings.Contains(body.String(), "fail at") || !strings.Contains(body.String(), "Forwarded") {
		t.Fatalf("text:\n%s", body.String())
	}
}

func TestBulletWrapHangsUnderTheText(t *testing.T) {
	p := NewPage()
	p.measure = func(text string, size float32, bold bool, family string) float32 {
		return float32(len([]rune(text))) * size * 0.5
	}
	var li doc.Block
	for _, b := range doc.Blocks([]byte("- Offices that cannot open the sites at all match Trend Micro's rating on every host and still need a business category.\n")) {
		if b.Kind == "li" {
			li = b
		}
	}
	lines := p.layoutFlow(li, li.Spans, 280, 16, 0, rgb(0xFFFFFF), false, widget.Color{}, p.theme.UIFont)
	if len(lines) < 2 {
		t.Fatalf("lines %d", len(lines))
	}
	if !lines[0].pieces[0].bullet {
		t.Fatal("first line has no bullet")
	}
	hang := p.pieceWidth(lines[0].pieces[0])
	for i, ln := range lines {
		if i == 0 {
			continue
		}
		if abs32(ln.pieces[0].pad-hang) > 0.5 {
			t.Fatalf("line %d hang %v want %v", i, ln.pieces[0].pad, hang)
		}
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func TestWrappedLineDoesNotLeadWithASpace(t *testing.T) {
	p := NewPage()
	text := "When the Source is Phone and you start typing, a chip shows On a call with a running timer. Its elapsed time is with the lead as the length of the first call attempt, shown on the timeline as, for example, Reached, 4 min."
	lines := p.layoutFlow(doc.Block{Kind: "p"}, []doc.Span{{Text: text}}, 320, 16, 0, rgb(0xFFFFFF), false, widget.Color{}, "Inter")
	if len(lines) < 2 {
		t.Fatalf("lines %d", len(lines))
	}
	for i, ln := range lines {
		if len(ln.pieces) == 0 {
			t.Fatalf("line %d empty", i)
		}
		if strings.TrimSpace(ln.pieces[0].text) == "" {
			t.Fatalf("line %d leads with %q", i, ln.pieces[0].text)
		}
	}
}

func TestPageLinkIsTheLabel(t *testing.T) {
	p := NewPage()
	p.SetBlocks(doc.Blocks([]byte("[name](https://example.com)\n")))
	uitest.LayoutWidget(p, 640, 200)
	p.SetBounds(geometry.NewRect(0, 0, 640, 200))
	canvas := uitest.DrawWidget(p)
	var saw bool
	for _, tx := range canvas.StyledTexts {
		if strings.Contains(tx.Text, "example.com") {
			t.Fatalf("url drawn: %q", tx.Text)
		}
		if tx.Text == "name" {
			saw = true
			if tx.Style.Color != widget.Hex(0x2979FF) {
				t.Fatalf("link color %v", tx.Style.Color)
			}
		}
	}
	if !saw {
		t.Fatal("link label was not drawn")
	}
}

func hasGlyph(rows []glyph, text string) bool {
	for _, r := range rows {
		if r.text == text {
			return true
		}
	}
	return false
}

func verticalBar(c *uitest.MockCanvas) bool {
	for _, ln := range c.Lines {
		if ln.From.X == ln.To.X && ln.To.Y-ln.From.Y > 8 {
			return true
		}
	}
	return false
}
