package index

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSearchRanksTheNoteWithBothWords(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "other.md", "# Other\n\nAn assay in the yard.\n")
	writeNote(t, root, "grove/notes/orchard-survey-coverage.md", "# Orchard survey coverage\n\nThe cohort calls we will not repeat the assay coverage plan.\n")
	x := New()
	x.Sync(root)
	got := x.Search("assay coverage", 5)
	if len(got) != 1 || !strings.Contains(got[0].Title, "Orchard") {
		t.Fatalf("%+v", got)
	}
	if !strings.Contains(got[0].Snippet, "cohort") {
		t.Fatalf("snippet %q", got[0].Snippet)
	}
	if got[0].Rel != "grove/notes/orchard-survey-coverage.md" {
		t.Fatalf("rel %q", got[0].Rel)
	}
}

func TestSearchPrefixAndSkipsNodeModules(t *testing.T) {
	root := t.TempDir()
	writeNote(t, root, "notes/coverage.md", "# Coverage\n\nWest orchard.\n")
	writeNote(t, root, "node_modules/pkg/secret.md", "# Secret\n\nzzskipnode should not be found.\n")
	x := New()
	x.Sync(root)
	if got := x.Search("cov", 5); len(got) != 1 || got[0].Title != "Coverage" {
		t.Fatalf("prefix %+v", got)
	}
	if got := x.Search("zzskipnode", 5); len(got) != 0 {
		t.Fatalf("cache was indexed %+v", got)
	}
}

func writeNote(t *testing.T, root, rel, body string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
