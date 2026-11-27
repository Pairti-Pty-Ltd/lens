package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pairti-lense/internal/report"
	"pairti-lense/internal/ui"
)

func TestExportBodyKeepsTheSingleFile(t *testing.T) {
	root := t.TempDir()
	note := filepath.Join(root, "nested", "setting-up.md")
	src, name, current, err := exportBody(root, note, []byte("# One\n"), ui.ExportChoice{Dest: ui.DestCurrent, Open: true})
	if err != nil {
		t.Fatal(err)
	}
	if string(src) != "# One\n" || name != "setting-up" || current != filepath.Dir(note) {
		t.Fatalf("src %q name %q current %q", src, name, current)
	}
}

func TestExportBodyJoinsTheProject(t *testing.T) {
	root := t.TempDir()
	writeExport(t, root, "b.md", "# B\n")
	writeExport(t, root, "a.md", "# A\n")
	note := filepath.Join(root, "nested", "setting-up.md")
	src, name, current, err := exportBody(root, note, []byte("ignored"), ui.ExportChoice{
		Project: true,
		Files:   []string{"b.md", "a.md"},
		Dest:    ui.DestTemp,
		Open:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if name != filepath.Base(root) || current != root {
		t.Fatalf("name %q current %q root %q", name, current, root)
	}
	text := string(src)
	if strings.Index(text, "# B") > strings.Index(text, "# A") || strings.Contains(text, "ignored") {
		t.Fatalf("%s", text)
	}
}

func TestExportBodyReportsAnEmptyProject(t *testing.T) {
	root := t.TempDir()
	_, _, _, err := exportBody(root, filepath.Join(root, "setting-up.md"), []byte("x"), ui.ExportChoice{Project: true})
	if err == nil {
		t.Fatal("expected an error")
	}
}

func writeExport(t *testing.T, root, rel, body string) {
	t.Helper()
	abs := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFormattedReturnsFromCompiledPreview(t *testing.T) {
	s := testShell()
	path := openNote(t, s, "# Source note\n")
	s.showReport(report.Result{Pages: []string{filepath.Join(filepath.Dir(path), "page.png")}})
	if s.page.Buffer() != nil || !strings.HasPrefix(s.line3.Get(), "Preview") {
		t.Fatal("compiled preview must have its own read-only surface")
	}
	s.pick("Formatted")
	headings := s.page.Outline()
	if s.page.Buffer() != s.buf || len(headings) != 1 || headings[0].Text != "Source note" {
		t.Fatal("Formatted did not restore the editable source note")
	}
}
