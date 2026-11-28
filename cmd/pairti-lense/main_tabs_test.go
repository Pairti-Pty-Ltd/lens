package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/library"
	"pairti-lense/internal/ui"
)

func TestTreeClickReplacesTheOpenDocument(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	a := treeNote(t, dir, "a.md", "# Alpha\n")
	b := treeNote(t, dir, "b.md", "# Beta\n")
	s.onItem(a)
	s.onItem(b)
	if len(s.docs) != 1 || s.docPath != b.Path || s.buf.Text() != "# Beta\n" {
		t.Fatalf("docs %d path %q text %q", len(s.docs), s.docPath, s.buf.Text())
	}
}

func TestTreeClickShowsShortDocumentAtTop(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	s.initWidgets()
	long := treeNote(t, dir, "long.md", strings.Repeat("Paragraph.\n\n", 100))
	short := treeNote(t, dir, "short.md", "# Short\n\nVisible\n")

	s.onItem(long)
	s.scroll.Set(900)
	s.onItem(long)
	if got := s.scroll.Get(); got != 900 {
		t.Fatalf("reselecting the same document moved scroll to %v", got)
	}
	s.onItem(short)
	if got := s.scroll.Get(); got != 0 {
		t.Fatalf("short document opened with scroll offset %v", got)
	}
	size := uitest.LayoutWidget(s.page, 640, 160)
	s.page.SetBounds(geometry.NewRect(0, 0, size.Width, size.Height))
	if !drawn(uitest.DrawWidget(s.page), "Visible") {
		t.Fatal("short document body did not draw")
	}
}

func TestTreeDoubleClickKeepsThePreviousDocument(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	a := treeNote(t, dir, "a.md", "# Alpha\n")
	b := treeNote(t, dir, "b.md", "# Beta\n")
	s.onItem(a)
	s.onItem(b)
	s.openTab(b)
	if len(s.docs) != 2 || s.docs[0].path != a.Path || s.docs[1].path != b.Path || s.docPath != b.Path {
		t.Fatalf("docs %q %q focus %q", pathOf(s, 0), pathOf(s, 1), s.docPath)
	}
	if s.docs[0].buf.Text() != "# Alpha\n" {
		t.Fatalf("restored %q", s.docs[0].buf.Text())
	}
}

func TestTreeClickLeavesADirtyDocumentOpen(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	a := treeNote(t, dir, "a.md", "# Alpha\n")
	b := treeNote(t, dir, "b.md", "# Beta\n")
	s.onItem(a)
	s.buf.InsertText("x")
	s.onItem(b)
	if len(s.docs) != 2 || s.docs[0].path != a.Path || s.docs[1].path != b.Path {
		t.Fatalf("docs %q %q", pathOf(s, 0), pathOf(s, 1))
	}
	if s.docs[0].buf.Text() != "x# Alpha\n" {
		t.Fatalf("dirty %q", s.docs[0].buf.Text())
	}
}

func TestTabMenuClosesAroundTheClickedTab(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{}
	a := treeNote(t, dir, "a.md", "# A\n")
	b := treeNote(t, dir, "b.md", "# B\n")
	c := treeNote(t, dir, "c.md", "# C\n")
	s.open(a)
	s.open(b)
	s.open(c)
	s.onTabMenu(0, ui.TabCloseRight)
	if len(s.docs) != 1 || s.docPath != a.Path {
		t.Fatalf("right %d %q", len(s.docs), s.docPath)
	}
	s.docs = nil
	s.active = -1
	s.open(a)
	s.open(b)
	s.open(c)
	s.onTabMenu(2, ui.TabCloseLeft)
	if len(s.docs) != 1 || s.docPath != c.Path {
		t.Fatalf("left %d %q", len(s.docs), s.docPath)
	}
	s.docs = nil
	s.active = -1
	s.open(a)
	s.open(b)
	s.open(c)
	s.onTabMenu(1, ui.TabCloseOthers)
	if len(s.docs) != 1 || s.docs[0].path != b.Path {
		t.Fatalf("others %d %q", len(s.docs), pathOf(s, 0))
	}
}

func TestTabMenuKeepsADirtyDocument(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{}
	a := treeNote(t, dir, "a.md", "# A\n")
	b := treeNote(t, dir, "b.md", "# B\n")
	c := treeNote(t, dir, "c.md", "# C\n")
	s.open(a)
	s.open(b)
	s.open(c)
	s.docs[0].buf.InsertText("x")
	s.onTabMenu(1, ui.TabCloseOthers)
	if len(s.docs) != 2 || s.docs[0].path != a.Path || s.docs[1].path != b.Path {
		t.Fatalf("kept %q %q (%d)", pathOf(s, 0), pathOf(s, 1), len(s.docs))
	}
	if s.line3.Get() != "Unsaved" {
		t.Fatalf("line %q", s.line3.Get())
	}
}

func treeNote(t *testing.T, dir, name, body string) ui.Item {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return ui.Item{Path: path, Rel: name, ProjectRoot: dir, ProjectName: "notes", Label: name}
}

func pathOf(s *shell, i int) string {
	if i < 0 || i >= len(s.docs) {
		return ""
	}
	return s.docs[i].path
}
