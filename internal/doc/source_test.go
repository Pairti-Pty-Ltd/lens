package doc

import (
	"strings"
	"testing"
)

func TestSourceIsNotMarkdown(t *testing.T) {
	src := "import { logger } from '@/lib/sentry';\n/**\n * Used by polling.\n */\nexport const n = 1;\n"
	blocks := Render("poll-retry.ts", []byte(src))
	if len(blocks) != len(strings.Split(src, "\n")) {
		t.Fatalf("lines %d", len(blocks))
	}
	var sawKeyword, sawString, sawComment bool
	for _, b := range blocks {
		if b.Kind != "code" {
			t.Fatalf("kind %s text %q", b.Kind, b.Text)
		}
		if strings.Contains(b.Text, "*") && !sawComment {
			for _, sp := range b.Spans {
				if strings.Contains(sp.Text, "Used by polling") && sp.Tone == "comment" && strings.Contains(sp.Text, "*") {
					sawComment = true
				}
			}
		}
		for _, sp := range b.Spans {
			if (sp.Text == "import" || sp.Text == "export" || sp.Text == "const") && sp.Tone == "keyword" {
				sawKeyword = true
			}
			if sp.Tone == "string" && strings.Contains(sp.Text, "sentry") {
				sawString = true
			}
		}
	}
	if !sawKeyword || !sawString || !sawComment {
		t.Fatalf("keyword %v string %v comment %v", sawKeyword, sawString, sawComment)
	}
	if !IsSource("main.go") || !IsSource("Makefile") || IsSource("shot.png") || IsSource("note.md") {
		t.Fatal("source detection")
	}
	md := Render("note.md", []byte("# Title\n"))
	if len(md) == 0 || md[0].Kind != "h1" {
		t.Fatalf("markdown %+v", md)
	}
}

func TestMarkdownSourceIsColoured(t *testing.T) {
	src := "---\ntitle: Fern\n---\n\n# Title\n\nA **bold** word and [link](a.md).\n\n| A | B |\n"
	rows := MarkdownLines(src)
	var heading, strong, key, value, pipes int
	for _, row := range rows {
		for _, sp := range row {
			if sp.Text == "# Title" && sp.Bold && sp.Tone == "heading" {
				heading++
			}
			if sp.Text == "**bold**" && sp.Bold && sp.Tone == "strong" {
				strong++
			}
			if sp.Text == "title" && sp.Tone == "keyword" {
				key++
			}
			if sp.Text == "Fern" && sp.Tone == "comment" {
				value++
			}
			if sp.Text == "|" && sp.Tone == "punct" {
				pipes++
			}
		}
	}
	if heading != 1 || strong != 1 || key != 1 || value != 1 || pipes < 2 {
		t.Fatalf("heading %d strong %d key %d value %d pipes %d rows %+v", heading, strong, key, value, pipes, rows)
	}
}

func TestFenceLanguageHighlights(t *testing.T) {
	rows := TokenLines("", "go", "func main() {}\n")
	var saw bool
	for _, row := range rows {
		for _, sp := range row {
			if sp.Text == "func" && sp.Tone == "keyword" {
				saw = true
			}
		}
	}
	if !saw {
		t.Fatalf("rows %+v", rows)
	}
}
