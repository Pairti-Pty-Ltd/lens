package gitlog

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogAndDiffStayReadOnly(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	run(t, dir, "init", "-b", "main")
	run(t, dir, "config", "user.email", "dev@example.com")
	run(t, dir, "config", "user.name", "Dev")
	note := filepath.Join(dir, "doc", "architecture.md")
	if err := os.MkdirAll(filepath.Dir(note), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(note, []byte("# Arch\n\none\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", "doc/architecture.md")
	run(t, dir, "commit", "-m", "first")
	if err := os.WriteFile(note, []byte("# Arch\n\ntwo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run(t, dir, "add", "doc/architecture.md")
	subject := "Field guide, observations and plot notes: the north trail markers, the spring sampling dates and the annotated map for the orchard team (field-guide-ab12.3.4.5)"
	run(t, dir, "commit", "-m", subject)
	branch, err := Branch(dir)
	if err != nil || branch != "main" {
		t.Fatal(branch, err)
	}
	commits, err := Log(dir, "doc/architecture.md", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("commits %d", len(commits))
	}
	if commits[0].Subject != subject {
		t.Fatalf("subject %q", commits[0].Subject)
	}
	files, err := Files(dir, commits[0].Hash)
	if err != nil || len(files) != 1 || files[0] != "doc/architecture.md" {
		t.Fatal(files, err)
	}
	patch, err := Diff(dir, commits[0].Hash, "doc/architecture.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(patch, "+two") || !strings.Contains(patch, "-one") {
		t.Fatalf("patch %s", patch)
	}
	page := Page(subject, "doc/architecture.md", commits[0].Date, patch)
	if strings.Contains(page, commits[0].Hash) || strings.Contains(page, "field-guide-ab12.3.4.5") || strings.Contains(page, "dev@example.com") {
		t.Fatalf("page repeats the hash, the ticket, or the address:\n%s", page)
	}
	if strings.Count(page, "Field guide, observations and plot notes") != 1 {
		t.Fatalf("subject count\n%s", page)
	}
	if !strings.Contains(page, "doc/architecture.md") || !strings.Contains(page, "+two") {
		t.Fatalf("page\n%s", page)
	}
}

func TestPlainDirectoryHasNoBranch(t *testing.T) {
	if _, err := Branch(t.TempDir()); err != ErrNone {
		t.Fatal(err)
	}
}

func run(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v %s", strings.Join(args, " "), err, out)
	}
}
