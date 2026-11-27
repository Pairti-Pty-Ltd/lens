package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrderSkipsMissingAndAppendsNew(t *testing.T) {
	root := t.TempDir()
	write(t, root, "b.md", "# B\n")
	write(t, root, "a.md", "# A\n")
	write(t, root, "nest/c.md", "# C\n")
	got, err := Order(root, []string{"missing.md", "nest/c.md", "b.md"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got, ",") != "nest/c.md,b.md,a.md" {
		t.Fatalf("%v", got)
	}
}

func TestJoinRetargetsAnImageToTheReportRoot(t *testing.T) {
	root := t.TempDir()
	write(t, root, "wiki/topics/note.md", "# Note\n\n![plate](figures/shot.png)\n")
	got, err := Join(root, []string{"wiki/topics/note.md"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "](wiki/topics/figures/shot.png)") {
		t.Fatalf("%s", got)
	}
}

func TestJoinKeepsTheFirstFrontMatter(t *testing.T) {
	root := t.TempDir()
	write(t, root, "one.md", "---\ntitle: Coverage\n---\n\n# One\n\nAlpha.\n")
	write(t, root, "two.md", "---\ntitle: Other\n---\n\n# Two\n\nBeta.\n")
	got, err := Join(root, []string{"gone.md", "one.md", "two.md"})
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.Contains(text, "title: Coverage") || strings.Count(text, "title:") != 1 {
		t.Fatalf("%s", text)
	}
	if !strings.Contains(text, "# One") || !strings.Contains(text, "# Two") || !strings.Contains(text, "Beta.") {
		t.Fatalf("%s", text)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	root := t.TempDir()
	in := Settings{Files: []string{"b.md", "../gone.md", "a.md"}, Template: "cedar"}
	if err := Save(root, in); err != nil {
		t.Fatal(err)
	}
	got, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Template != "cedar" || strings.Join(got.Files, ",") != "b.md,a.md" {
		t.Fatalf("%+v", got)
	}
}

func TestCorruptProjectFile(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".pairti-project.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected an error")
	}
}

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
