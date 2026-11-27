package main

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"

	"pairti-lense/internal/library"
	"pairti-lense/internal/session"
	"pairti-lense/internal/ui"
)

func TestFileMenusRestoreRecentDocumentsAndFolders(t *testing.T) {
	s := testShell()
	cfg := t.TempDir()
	s.sessionPath = filepath.Join(cfg, "session.json")
	s.lib = &library.Library{}
	s.libPath = filepath.Join(cfg, "library.json")
	d1, d2 := t.TempDir(), t.TempDir()
	a := treeNote(t, d1, "shared.md", "# First")
	b := treeNote(t, d2, "shared.md", "# Second")
	s.files.pick = func(opts gogpu.FileDialogOptions) ([]string, error) {
		if opts.Directory {
			return []string{d1}, nil
		}
		return []string{a.Path, b.Path}, nil
	}
	menuByTitle(t, s.menuCommands(), "Open Project Folder…").invoke()
	menuByTitle(t, s.menuCommands(), "Open Document…").invoke()
	if s.docPath != b.Path || len(s.docs) != 2 || !slices.Equal(s.lib.Paths, []string{d1}) {
		t.Fatalf("folder/doc pickers opened %v, %v", s.lib.Paths, s.docs)
	}
	// The two entries have the same name and still address different files.
	menuByTitle(t, s.menuCommands(), a.Path).invoke()
	want := []string{a.Path, b.Path, d1}
	st, err := session.Load(s.sessionPath)
	if err != nil || !slices.Equal(st.Recent, want) {
		t.Fatalf("recent %v err %v", st.Recent, err)
	}
	restored := testShell()
	restored.lib = s.lib
	restored.sessionPath = s.sessionPath
	restored.holdSave = true
	restored.restoreDocs(st)
	restored.holdSave = false
	if restored.docPath != a.Path || !slices.Equal(restored.recent, want) {
		t.Fatalf("restored %q, %v", restored.docPath, restored.recent)
	}
	if err := os.Remove(b.Path); err != nil {
		t.Fatal(err)
	}
	menuByTitle(t, restored.menuCommands(), b.Path).invoke()
	if restored.docPath != a.Path || slices.Contains(restored.recent, b.Path) {
		t.Fatal("missing recent item changed the active tab or remained listed")
	}
	menuByTitle(t, restored.menuCommands(), "Clear Menu").invoke()
	st, err = session.Load(restored.sessionPath)
	if err != nil || len(st.Recent) != 0 || len(st.Docs) != 2 || menuByTitle(t, restored.menuCommands(), "Clear Menu").enabled() {
		t.Fatalf("clear menu: %+v, %v", st, err)
	}
}

func TestFileCommandsKeepFocusedDocumentAndUnsavedWork(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	s.libPath = filepath.Join(t.TempDir(), "library.json")
	s.rescan()
	s.chosen = dir
	command := func(title string) { menuByTitle(t, s.menuCommands(), title).invoke() }
	command("New")
	first := s.docPath
	command("New")
	path := s.docPath
	if first == path || filepath.Dir(path) != dir {
		t.Fatalf("New paths %q and %q", first, path)
	}
	s.buf.InsertText("draft")
	contents := s.buf.Text()
	s.files.confirmClose = func(got string) closeChoice {
		if got != path {
			t.Fatalf("prompted for %s", got)
		}
		return cancelClose
	}
	command("Close Tab")
	if s.canCloseWindow() || len(s.docs) != 2 || s.docPath != path || s.buf.Text() != contents {
		t.Fatal("cancel lost the focused dirty tab")
	}
	s.files.reveal = func(got string) error {
		if got != path {
			t.Fatalf("revealed %s", got)
		}
		return nil
	}
	command("Reveal in Finder")
	command("Remove Project from Library")
	if len(s.lib.Paths) != 1 || !s.buf.Dirty() {
		t.Fatal("removing project lost edits")
	}
	command("Save")
	got, err := os.ReadFile(path)
	if err != nil || string(got) != contents {
		t.Fatalf("saved bytes %q, err %v", got, err)
	}
	s.reportMode = true
	var choice ui.ExportChoice
	s.sheet = ui.NewExportSheet(func(c ui.ExportChoice) { choice = c })
	command("Export Document…")
	if !s.sheet.Shown() {
		t.Fatal("document export unavailable with Reports sidebar")
	}
	s.sheet.SetBounds(geometry.NewRect(0, 0, 800, 600))
	pt, ok := s.sheet.Spot("go")
	if !ok || !uitest.SimulateClick(s.sheet, pt.X, pt.Y) || choice.Project || s.sheet.Shown() {
		t.Fatal("File did not export document scope")
	}
	s.reportMode = false
	command("Close Tab")
	if len(s.docs) != 1 || s.docPath != first {
		t.Fatal("closed wrong document")
	}
	s.openPath(path)
	if s.buf.Text() != contents {
		t.Fatal("reopened file differs from saved buffer")
	}
	command("Remove Project from Library")
	if len(s.lib.Paths) != 0 || !isFile(first) || !isFile(path) {
		t.Fatal("removing project changed source files")
	}
}

func TestNewStandaloneFilePreservesExistingFile(t *testing.T) {
	s := testShell()
	path := filepath.Join(t.TempDir(), "standalone.md")
	s.files.newPath = func(gogpu.FileDialogOptions) (string, error) { return path, nil }
	menuByTitle(t, s.menuCommands(), "New").invoke()
	if s.docPath != path {
		t.Fatal("new document did not open")
	}
	if err := os.WriteFile(path, []byte("existing"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := createNote(path); !os.IsExist(err) {
		t.Fatalf("expected existing file refusal: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "existing" {
		t.Fatalf("existing document changed: %q %v", got, err)
	}
}
