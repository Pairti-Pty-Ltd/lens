package doc

import (
	"strings"
	"testing"
)

func TestFrontMatterIsOneBlock(t *testing.T) {
	src := []byte("---\ntitle: Orchard questions\nsubtitle: gaps\n---\n\n# Write-up\n")
	blocks := Blocks(src)
	if blocks[0].Kind != "front" || blocks[0].Line != 1 || blocks[0].Lines != 4 {
		t.Fatalf("front %+v", blocks[0])
	}
	if blocks[0].Text != "title: Orchard questions\nsubtitle: gaps" {
		t.Fatalf("body %q", blocks[0].Text)
	}
	var h1 Block
	for _, b := range blocks {
		if b.Kind == "h1" {
			h1 = b
		}
		if b.Kind == "hr" || b.Kind == "p" {
			t.Fatalf("front matter also drawn as %s %+v", b.Kind, b)
		}
	}
	if h1.Line != 6 || h1.Text != "Write-up" {
		t.Fatalf("heading %+v", h1)
	}
}

func TestARuleIsNotFrontMatter(t *testing.T) {
	blocks := Blocks([]byte("---\n\nHello\n\n---\n"))
	for _, b := range blocks {
		if b.Kind == "front" {
			t.Fatalf("prose between rules %+v", blocks)
		}
	}
	if len(blocks) != 3 || blocks[0].Kind != "hr" || blocks[0].Line != 1 || blocks[2].Line != 5 {
		t.Fatalf("rule source positions %+v", blocks)
	}
}

func TestSourceLinesCoverFencesAndTables(t *testing.T) {
	src := []byte("# T\n\n```\na\nb\n```\n\nafter\n\n| A | B |\n| - | - |\n| 1 | 2 |\n")
	blocks := Blocks(src)
	byKind := map[string]Block{}
	for _, b := range blocks {
		if _, ok := byKind[b.Kind]; !ok {
			byKind[b.Kind] = b
		}
	}
	if byKind["h1"].Line != 1 || byKind["h1"].Text != "T" {
		t.Fatalf("h1 %+v", byKind["h1"])
	}
	if byKind["code"].Line != 4 || byKind["code"].Lines != 2 || byKind["code"].Text != "a\nb" {
		t.Fatalf("code %+v", byKind["code"])
	}
	if byKind["p"].Line != 8 || byKind["p"].Text != "after" {
		t.Fatalf("paragraph %+v", byKind["p"])
	}
	if byKind["th"].Line != 10 || byKind["th"].Lines < 3 {
		t.Fatalf("header %+v", byKind["th"])
	}
	if byKind["tr"].Line != 0 {
		t.Fatalf("body row should stay inside the table span: %+v", byKind["tr"])
	}
}

func TestImagesKeepAltAndDestination(t *testing.T) {
	blocks := Blocks([]byte("![](pic.png)\n\n![plate][shot]\n\n[shot]: figures/shot.png\n\n<img src=\"a.png\" alt=\"plate\">\n"))
	if len(blocks) < 3 {
		t.Fatalf("blocks = %+v", blocks)
	}
	empty := blocks[0].Spans[0]
	if !empty.Image || empty.Link != "pic.png" || empty.Text != "" {
		t.Fatalf("empty alt = %+v", empty)
	}
	ref := blocks[1].Spans[0]
	if !ref.Image || ref.Text != "plate" || ref.Link != "figures/shot.png" {
		t.Fatalf("reference = %+v", ref)
	}
	var html Span
	for _, b := range blocks {
		for _, sp := range b.Spans {
			if sp.Image && sp.Link == "a.png" {
				html = sp
			}
		}
	}
	if !html.Image || html.Text != "plate" {
		t.Fatalf("html img = %+v blocks %+v", html, blocks)
	}
}

func TestAlertDropsTheTag(t *testing.T) {
	src := []byte("> [!WARNING]\n> **Ship Basic.**\n>\n> Hold the rest.\n")
	blocks := Blocks(src)
	if len(blocks) != 2 {
		t.Fatalf("blocks %+v", blocks)
	}
	if blocks[0].Kind != "alert" || blocks[0].Marker != "warning" || blocks[0].Text != "Ship Basic." {
		t.Fatalf("title %+v", blocks[0])
	}
	if blocks[1].Kind != "alert" || blocks[1].Marker != "" || blocks[1].Text != "Hold the rest." {
		t.Fatalf("body %+v", blocks[1])
	}
	for _, b := range blocks {
		if strings.Contains(b.Text, "WARNING") {
			t.Fatalf("tag leaked %q", b.Text)
		}
	}
}

func TestBlocksKeepCommonAndGFM(t *testing.T) {
	src := []byte(strings.TrimSpace(`
# Title

A **bold** and *em* word, plus ` + "`code`" + ` and ~~gone~~.

[name](https://example.com)

![plate](pic.png)

> quoted line

---

1. one
   - two

- [x] done
- [ ] open

` + "```" + `
plate
` + "```" + `

| A | B |
| - | - |
| 1 | 2 |

<div>raw</div>
`))
	blocks := Blocks(src)
	got := map[string][]Block{}
	for _, b := range blocks {
		got[b.Kind] = append(got[b.Kind], b)
	}
	if len(got["h1"]) != 1 || got["h1"][0].Text != "Title" || got["h1"][0].Line != 1 {
		t.Fatalf("h1 = %+v", got["h1"])
	}
	p := got["p"]
	if len(p) == 0 {
		t.Fatal("no paragraph")
	}
	var bold, italic, code, strike bool
	for _, sp := range p[0].Spans {
		if sp.Bold && strings.Contains(sp.Text, "bold") {
			bold = true
		}
		if sp.Italic && strings.Contains(sp.Text, "em") {
			italic = true
		}
		if sp.Code && sp.Text == "code" {
			code = true
		}
		if sp.Strike && strings.Contains(sp.Text, "gone") {
			strike = true
		}
	}
	if !bold || !italic || !code || !strike {
		t.Fatalf("inline spans = %+v", p[0].Spans)
	}
	var sawLink, sawImage bool
	for _, b := range blocks {
		if strings.Contains(b.Text, "https://example.com") || strings.Contains(b.Text, "pic.png") {
			t.Fatalf("url dumped into the line: %q", b.Text)
		}
		for _, sp := range b.Spans {
			if sp.Link == "https://example.com" && sp.Text == "name" {
				sawLink = true
			}
			if sp.Text == "plate" && sp.Link == "pic.png" && sp.Image {
				sawImage = true
			}
		}
	}
	if !sawLink || !sawImage {
		t.Fatalf("link=%v image=%v", sawLink, sawImage)
	}
	if len(got["quote"]) != 1 || got["quote"][0].Text != "quoted line" {
		t.Fatalf("quote = %+v", got["quote"])
	}
	if len(got["hr"]) == 0 {
		t.Fatal("missing rule")
	}
	for _, b := range got["hr"] {
		if strings.ContainsAny(b.Text, "─━") {
			t.Fatalf("rule is a glyph: %q", b.Text)
		}
	}
	var ordered, nested, done, open bool
	for _, b := range got["li"] {
		if b.Marker == "1." && b.Text == "one" && b.Depth == 0 {
			ordered = true
		}
		if b.Marker == "·" && b.Text == "two" && b.Depth == 1 {
			nested = true
		}
		if b.Marker == "[x]" && b.Text == "done" {
			done = true
		}
		if b.Marker == "[ ]" && b.Text == "open" {
			open = true
		}
	}
	if !ordered || !nested || !done || !open {
		t.Fatalf("lists = %+v", got["li"])
	}
	if len(got["code"]) == 0 || !strings.Contains(got["code"][0].Text, "plate") {
		t.Fatalf("code = %+v", got["code"])
	}
	if len(got["th"]) != 1 || len(got["th"][0].Cells) != 2 || got["th"][0].Cells[0] != "A" {
		t.Fatalf("header = %+v", got["th"])
	}
	if len(got["tr"]) != 1 || len(got["tr"][0].Cells) != 2 || got["tr"][0].Cells[1] != "2" {
		t.Fatalf("row = %+v", got["tr"])
	}
	if len(got["html"]) != 1 || !strings.Contains(got["html"][0].Text, "raw") {
		t.Fatalf("html = %+v", got["html"])
	}
}
