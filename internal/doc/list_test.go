package doc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadDirOneLevel(t *testing.T) {
	root := t.TempDir()
	notes := filepath.Join(root, "notes")
	if err := os.Mkdir(notes, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".beads"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "readme.md"), "# Hello\n")
	write(filepath.Join(notes, "keep.md"), "# Nested\n")
	write(filepath.Join(root, "node_modules", "pkg", "a.md"), "# Skip\n")
	write(filepath.Join(root, "main.go"), "package main\n")
	write(filepath.Join(root, "shot.png"), "not really")

	got, err := ReadDir(root, root, List{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("entries = %+v", got)
	}
	if !got[0].IsDir || got[0].Name != "notes" {
		t.Fatalf("first = %+v", got[0])
	}
	if got[1].IsDir || got[1].Name != "main.go" || got[1].Title != "main.go" {
		t.Fatalf("source = %+v", got[1])
	}
	if got[2].IsDir || got[2].Title != "Hello" {
		t.Fatalf("markdown = %+v", got[2])
	}
	for _, e := range got {
		if strings.Contains(e.Rel, "keep") || strings.Contains(e.Name, "node_modules") {
			t.Fatalf("listed nested or skipped dir: %+v", e)
		}
	}
}

func TestReadDirSkipsSniffWhenCrowded(t *testing.T) {
	root := t.TempDir()
	for i := 0; i < 3; i++ {
		name := string(rune('a'+i)) + ".md"
		body := "# Real title\n"
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := readDir(root, root, List{}, 10, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d", len(got))
	}
	for _, e := range got {
		if e.Title != e.Name {
			t.Fatalf("sniffed a crowded directory: %+v", e)
		}
	}
}

func TestReadDirFiltersDotAndIgnored(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, ".beads"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, ".gitignore"), "skip.md\n")
	write(filepath.Join(root, "visible.md"), "# Visible\n")
	write(filepath.Join(root, "skip.md"), "# Skip\n")
	write(filepath.Join(root, ".secret.md"), "# Secret\n")
	write(filepath.Join(root, ".beads", "note.md"), "# Bead\n")

	hidden, err := ReadDir(root, root, List{})
	if err != nil {
		t.Fatal(err)
	}
	if names := entryNames(hidden); len(names) != 1 || names[0] != "visible.md" {
		t.Fatalf("hidden = %v", names)
	}
	dots, err := ReadDir(root, root, List{ShowDot: true})
	if err != nil {
		t.Fatal(err)
	}
	if names := entryNames(dots); !hasName(names, ".beads") || !hasName(names, ".secret.md") || hasName(names, "skip.md") || hasName(names, ".git") {
		t.Fatalf("dots = %v", names)
	}
	both, err := ReadDir(root, root, List{ShowDot: true, ShowIgnored: true})
	if err != nil {
		t.Fatal(err)
	}
	if names := entryNames(both); !hasName(names, "skip.md") || !hasName(names, ".secret.md") || hasName(names, ".git") {
		t.Fatalf("shown = %v", names)
	}
}

func TestReadDirKeepsDocumentDirectories(t *testing.T) {
	root := t.TempDir()
	write := func(path, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(root, "notes", "a.md"), "# Notes\n")
	write(filepath.Join(root, "brief", "chapter", "a.typ"), "= Brief\n")
	write(filepath.Join(root, "src", "pkg", "sub", "main.go"), "package sub\n")
	write(filepath.Join(root, "web", "main.go"), "package main\n")
	write(filepath.Join(root, "web", "node_modules", "pkg", "a.md"), "# Skip\n")

	got, err := ReadDir(root, root, List{})
	if err != nil {
		t.Fatal(err)
	}
	names := entryNames(got)
	if !hasName(names, "notes") || !hasName(names, "brief") || hasName(names, "src") || hasName(names, "web") {
		t.Fatalf("filtered = %v", names)
	}
	chapter := filepath.Join(root, "brief", "chapter")
	inside, err := ReadDir(chapter, root, List{})
	if err != nil {
		t.Fatal(err)
	}
	if names := entryNames(inside); len(names) != 1 || names[0] != "a.typ" {
		t.Fatalf("typst = %v", names)
	}

	write(filepath.Join(root, "src", "pkg", "sub", "note.md"), "# Note\n")
	again, err := ReadDir(root, root, List{})
	if err != nil {
		t.Fatal(err)
	}
	if names := entryNames(again); !hasName(names, "src") || hasName(names, "web") {
		t.Fatalf("after a nested note = %v", names)
	}

	all, err := ReadDir(root, root, List{AllDirs: true})
	if err != nil {
		t.Fatal(err)
	}
	if names := entryNames(all); !hasName(names, "web") || !hasName(names, "src") {
		t.Fatalf("all folders = %v", names)
	}
}

func entryNames(entries []DirEntry) []string {
	var out []string
	for _, e := range entries {
		out = append(out, e.Name)
	}
	return out
}

func hasName(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

func TestReadDirCaps(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"a", "b", "c"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	got, err := readDir(root, root, List{AllDirs: true}, 2, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || !got[2].More {
		t.Fatalf("capped = %+v", got)
	}
}
