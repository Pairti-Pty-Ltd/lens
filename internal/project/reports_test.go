package project

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReportStartsEmptyAndKeepsOnlyAddedFiles(t *testing.T) {
	root := t.TempDir()
	kept := filepath.Join(root, "wiki", "topics")
	if err := os.MkdirAll(kept, 0o755); err != nil {
		t.Fatal(err)
	}
	note := "wiki/topics/orchard-survey-and-sampling-plan.md"
	other := filepath.Join(root, "parent.md")
	if err := os.WriteFile(filepath.Join(root, note), []byte("# Compliance\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("# Parent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	book := Book{}
	book.Add()
	if !book.AddFile(note) {
		t.Fatal("add")
	}
	if book.AddFile(note) {
		t.Fatal("added twice")
	}
	if err := SaveBook(root, book); err != nil {
		t.Fatal(err)
	}
	got, err := LoadBook(root)
	if err != nil {
		t.Fatal(err)
	}
	rep, ok := got.Current()
	if !ok || len(rep.Files) != 1 || rep.Files[0] != note {
		t.Fatalf("%+v", rep)
	}
	for _, f := range rep.Files {
		if f == "parent.md" {
			t.Fatal("parent file was included")
		}
	}
}
