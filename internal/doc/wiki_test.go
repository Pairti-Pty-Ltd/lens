package doc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWikiLinkResolvesTitleAndFileName(t *testing.T) {
	root := t.TempDir()
	grove := filepath.Join(root, "wiki", "entities", "grove-f.md")
	road := filepath.Join(root, "wiki", "topics", "orchard-planting-calendar.md")
	iso := filepath.Join(root, "wiki", "entities", "iso-42001.md")
	for _, pair := range []struct{ path, body string }{
		{grove, "---\ntitle: GroveF\n---\n\n# GroveF\n"},
		{road, "---\ntitle: Orchard Planting Calendar (Now, Next, Later)\n---\n\n# Orchard Planting Calendar (Now, Next, Later)\n"},
		{iso, "---\ntitle: ISO 42001\n---\n\n# ISO 42001\n"},
	} {
		if err := os.MkdirAll(filepath.Dir(pair.path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(pair.path, []byte(pair.body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	from := filepath.Join(root, "wiki", "topics", "orchard-survey-and-sampling-plan.md")
	got, ok := WikiFile(root, from, "Grove-F")
	if !ok || got != grove {
		t.Fatalf("Grove-F %q %v", got, ok)
	}
	got, ok = WikiFile(root, from, "Orchard Planting Calendar")
	if !ok || got != road {
		t.Fatalf("short title %q %v", got, ok)
	}
	got, ok = WikiFile(root, from, "Orchard Planting Calendar (Now, Next, Later)")
	if !ok || got != road {
		t.Fatalf("full title %q %v", got, ok)
	}
	got, ok = WikiFile(root, from, "ISO 42001")
	if !ok || got != iso {
		t.Fatalf("ISO %q %v", got, ok)
	}
	if _, ok := WikiFile(root, from, "../library"); ok {
		t.Fatal("a wiki path climbed out of the project")
	}

	blocks := Blocks([]byte("See [[Grove-F]] and [[Orchard Planting Calendar|Orchard Planting Calendar (Now, Next, Later)]].\n\n`[[Grove-F]]`\n\n| **[[ISO 42001]]** |\n| --- |\n| keep |\n"))
	var sawGrove, sawLabel, sawCode, sawCell bool
	for _, b := range blocks {
		for _, sp := range b.Spans {
			if !sp.Code && strings.Contains(sp.Text, "[[") {
				t.Fatalf("brackets in %s: %q", b.Kind, sp.Text)
			}
		}
		for _, sp := range b.Spans {
			if sp.Text == "Grove-F" && sp.Link == "wiki:Grove-F" {
				sawGrove = true
			}
			if sp.Text == "Orchard Planting Calendar (Now, Next, Later)" && sp.Link == "wiki:Orchard Planting Calendar" {
				sawLabel = true
			}
			if sp.Code && sp.Text == "[[Grove-F]]" && sp.Link == "" {
				sawCode = true
			}
		}
		for i, cell := range b.Cells {
			if cell == "ISO 42001" && i < len(b.CellLinks) && b.CellLinks[i] == "wiki:ISO 42001" {
				sawCell = true
			}
		}
	}
	if !sawGrove || !sawLabel || !sawCode || !sawCell {
		t.Fatalf("grove %v label %v code %v cell %v", sawGrove, sawLabel, sawCode, sawCell)
	}
}

func TestCreateWikiWritesTheTitle(t *testing.T) {
	dir := t.TempDir()
	path, err := CreateWiki(dir, "Brand New Note")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "brand-new-note.md" {
		t.Fatalf("name %s", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, "title: Brand New Note") || !strings.Contains(text, "# Brand New Note") || strings.Contains(text, "[[") {
		t.Fatalf("%s", text)
	}
	if _, err := CreateWiki(dir, "../secret"); err == nil {
		t.Fatal("created a path outside the folder")
	}
}
