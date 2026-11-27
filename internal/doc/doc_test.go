package doc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTitleFixture(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "testdata", "membrane-assay.md"))
	if err != nil {
		t.Fatal(err)
	}
	const want = "Write-up of the cohort run, with the plate map and the calls we will not repeat"
	got := Title(src)
	if got != want {
		t.Fatalf("title = %q", got)
	}
	if got == "membrane-assay.md" {
		t.Fatal("title fell back to the filename")
	}
	if len([]rune(got)) <= 40 {
		t.Fatal("fixture title should be longer than a path segment")
	}
	blocks := Blocks(src)
	if len(blocks) == 0 || blocks[0].Kind != "h1" || blocks[0].Text != want {
		t.Fatalf("first block = %+v", blocks[0])
	}
	var sawList, sawCode bool
	for _, b := range blocks {
		if b.Kind == "li" {
			sawList = true
		}
		if b.Kind == "code" && strings.Contains(b.Text, "plate") {
			sawCode = true
		}
	}
	if !sawList || !sawCode {
		t.Fatalf("list=%v code=%v", sawList, sawCode)
	}
}

func TestTitleFilenameWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bare.md")
	if err := os.WriteFile(path, []byte("no heading here\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := SniffTitle(path, "bare.md"); got != "bare.md" {
		t.Fatalf("title = %q", got)
	}
	p := Walk(dir)
	if p.Absent || len(p.Docs) != 1 || p.Docs[0].Title != "bare.md" {
		t.Fatalf("walk = %+v", p)
	}
}

func TestShortPathCutsLongSegment(t *testing.T) {
	in := "notes/this-segment-is-definitely-longer-than-forty-characters-yes/a.md"
	got := ShortPath(in)
	seg := strings.Split(got, "/")[1]
	if len([]rune(seg)) > 40 {
		t.Fatalf("segment len = %d (%q)", len([]rune(seg)), seg)
	}
	if !strings.Contains(seg, "…") {
		t.Fatalf("segment = %q", seg)
	}
	if strings.Contains(got, "definitely-longer-than-forty-characters-yes") {
		t.Fatalf("segment was not cut: %s", got)
	}
}

func TestWalkSkipsGitAndOutAndTypst(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "out"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "notes"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(dir, ".git", "COMMIT_EDITMSG.md"), "# secret\n")
	write(filepath.Join(dir, "out", "report.md"), "# out\n")
	write(filepath.Join(dir, "notes", "a.typ"), "= Typst\n")
	write(filepath.Join(dir, "notes", "keep.md"), "# Keep me\n")
	p := Walk(dir)
	if len(p.Docs) != 1 || p.Docs[0].Title != "Keep me" {
		t.Fatalf("docs = %+v", p.Docs)
	}
}

func TestAbsentProject(t *testing.T) {
	p := Walk(filepath.Join(t.TempDir(), "missing"))
	if !p.Absent {
		t.Fatal("expected absent")
	}
	if len(p.Docs) != 0 {
		t.Fatalf("docs = %+v", p.Docs)
	}
}

func TestReadTooLarge(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "big.md")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxBytes + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	_, err = Read(path)
	if _, ok := err.(*TooLargeError); !ok {
		t.Fatalf("err = %v", err)
	}
}

func TestWrap(t *testing.T) {
	got := Wrap("one two three four", 10)
	if len(got) < 2 {
		t.Fatalf("lines = %q", got)
	}
	for _, line := range got {
		if len([]rune(line)) > 10 {
			t.Fatalf("line too long: %q", line)
		}
	}
}
