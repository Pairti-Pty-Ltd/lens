package main

import (
	"path/filepath"
	"testing"

	"pairti-lense/internal/library"
	"pairti-lense/internal/ui"
)

func menuByTitle(t *testing.T, commands []menuCommand, title string) menuCommand {
	t.Helper()
	for _, c := range commands {
		if c.title == title {
			return c
		}
	}
	t.Fatalf("missing menu item %q", title)
	return menuCommand{}
}

func TestMenuSharesToolbarStateAndHistory(t *testing.T) {
	s := testShell()
	s.histAt = -1
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	s.libPath = filepath.Join(dir, "library.json")
	commands := s.menuCommands()
	item := func(title string) menuCommand { return menuByTitle(t, commands, title) }

	if item("Back").enabled() || item("Forward").enabled() || item("Raw").enabled() {
		t.Fatal("navigation or document commands enabled in empty window")
	}
	a := treeNote(t, dir, "a.md", "# Alpha")
	b := treeNote(t, dir, "b.md", "# Beta")
	s.open(a)
	s.open(b)
	if !item("Back").enabled() || item("Forward").enabled() {
		t.Fatal("history should allow back only")
	}
	item("Back").invoke()
	if s.docPath != a.Path || item("Back").enabled() || !item("Forward").enabled() {
		t.Fatal("Back must open Alpha at the start of history")
	}
	s.onTool("Forward")
	if s.docPath != b.Path || item("Forward").enabled() {
		t.Fatal("toolbar Forward must update native history state")
	}
	s.pick("Raw")
	if !item("Raw").checked() || item("Formatted").checked() || !item("Show Line Numbers").checked() {
		t.Fatal("Raw toolbar selection must be reflected in the menu")
	}
	s.onTool("Line numbers")
	if item("Show Line Numbers").checked() {
		t.Fatal("toolbar line number change omitted from native menu")
	}
	item("Open Documents in Raw").invoke()
	lib, err := library.Load(s.libPath)
	if err != nil || !lib.OpenRaw || !item("Open Documents in Raw").checked() {
		t.Fatalf("raw preference not persisted: %v, %v", lib, err)
	}
}

func TestMenuRoutesToFocusedPaneAndProtectsDirtyTab(t *testing.T) {
	s := testShell()
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(s.noteEdit, s.saveDoc)
	s.frameB = ui.NewFrame(s.pageB, s.editorB)
	s.pair = ui.NewPair(s.frame, s.frameB)
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	s.sessionPath = filepath.Join(dir, "session.json")
	a := treeNote(t, dir, "a.md", "# Alpha")
	b := treeNote(t, dir, "b.md", "# Beta")
	s.open(a)
	s.open(b)
	commands := s.menuCommands()
	item := func(title string) menuCommand { return menuByTitle(t, commands, title) }
	item("Split Pane").invoke()
	item("Raw").invoke()
	if s.focus != 1 || !s.docs[s.splitAt].editing || s.docs[s.active].editing {
		t.Fatal("Raw did not target the second pane")
	}
	s.buf.InsertText("draft")
	item("Focus Other Pane").invoke()
	if s.docPath != a.Path || !item("Formatted").checked() {
		t.Fatal("Format checks did not follow pane focus")
	}
	s.rename = ui.NewRename(nil)
	s.rename.Show("Report")
	item("Next Tab").invoke()
	if item("Next Tab").enabled() || s.docPath != a.Path {
		t.Fatal("underlying tab changed while rename was active")
	}
	s.rename.Hide()
	item("Next Tab").invoke()
	if s.docPath != b.Path || !item("Raw").checked() {
		t.Fatal("Next Tab did not focus Beta")
	}
	item("Previous Tab").invoke()
	if s.canCloseWindow() || s.docPath != b.Path || s.buf.Text() != "draft# Beta" {
		t.Fatal("closing a window must keep its dirty tab and edits")
	}
	s.saveDoc()
	if !s.canCloseWindow() {
		t.Fatal("clean window should close")
	}
}
