package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/uitest"
	"pairti-lense/internal/doc"
	"pairti-lense/internal/library"
	"pairti-lense/internal/project"
	"pairti-lense/internal/ui"
)

func reportCommand(t *testing.T, s *shell, group, title string) menuCommand {
	t.Helper()
	for _, cmd := range s.menuCommands() {
		if cmd.group == group && cmd.title == title {
			return cmd
		}
	}
	t.Fatalf("missing %q / %s", group, title)
	return menuCommand{}
}

func TestReportMenuTargetsProjectAndPreservesDirtyTab(t *testing.T) {
	s := testShell()
	a, b := t.TempDir(), t.TempDir()
	s.projects = []doc.Project{{Name: "notes", Root: a}, {Name: "notes", Root: b}}
	s.lib = &library.Library{Paths: []string{a, b}}
	first := treeNote(t, a, "note.md", "# Alpha\n")
	second := treeNote(t, b, "note.md", "# Beta\n")
	for _, root := range []string{a, b} {
		book := project.Book{}
		book.Add()
		book.Rename(book.Open, "Quarterly")
		book.AddFile("note.md")
		if err := project.SaveBook(root, book); err != nil {
			t.Fatal(err)
		}
	}
	s.open(first)
	dirty := s.buf
	dirty.InsertText("unsaved ")
	openA := menuGroup("Reports", "Open Report", a)
	openB := menuGroup("Reports", "Open Report", b)
	reportCommand(t, s, openA, "Quarterly").invoke()
	if s.buf.Text() != "# Alpha\n" && !strings.Contains(s.buf.Text(), "# Alpha") {
		t.Fatalf("Alpha: %q", s.buf.Text())
	}
	reportCommand(t, s, openB, "Quarterly").invoke()
	if !strings.Contains(s.buf.Text(), "# Beta") || s.reportRoot != b || !reportCommand(t, s, openB, "Quarterly").checked() || reportCommand(t, s, openA, "Quarterly").checked() {
		t.Fatalf("wrong report selected: %q %s", s.buf.Text(), s.reportRoot)
	}
	s.open(second)
	s.show(s.indexOf(first.Path))
	if s.buf != dirty || !s.buf.Dirty() || s.buf.Text() != "unsaved # Alpha\n" {
		t.Fatal("switching out of a report replaced the draft")
	}
	if reportCommand(t, s, "Reports", "Add Current Document").enabled() {
		t.Fatal("cannot add another project's active tab")
	}
}

func TestReportMenuBookChanges(t *testing.T) {
	s := testShell()
	root := t.TempDir()
	s.projects = []doc.Project{{Name: "Work", Root: root}}
	s.lib = &library.Library{Paths: []string{root}}
	one := treeNote(t, root, "one.md", "# One")
	two := treeNote(t, root, "two.md", "# Two")
	open := menuGroup("Reports", "Open Report", "Work")
	command := func(group, title string) menuCommand { return reportCommand(t, s, group, title) }
	if command(open, "No Reports").enabled() || command("Reports", "Export Report…").enabled() {
		t.Fatal("empty entries enabled")
	}
	command("Reports", "New Report").invoke()
	command("Reports", "Rename Report…").invoke()
	if s.renameID != s.book.Open || !s.renameAsk.Load() {
		t.Fatal("rename targeted wrong report")
	}
	s.renameReport("Manual")
	if !command(open, "Manual").checked() {
		t.Fatal("renamed report is not active")
	}
	for _, it := range []ui.Item{one, two} {
		s.open(it)
		add := command("Reports", "Add Current Document")
		if !add.enabled() {
			t.Fatal("add unavailable")
		}
		add.invoke()
		if command("Reports", "Add Current Document").enabled() {
			t.Fatal("duplicate add enabled")
		}
	}
	up := menuGroup("Reports", "Move Document Up")
	down := menuGroup("Reports", "Move Document Down")
	if command(up, "one.md").enabled() || command(down, "two.md").enabled() {
		t.Fatal("out-of-range moves enabled")
	}
	check := func(want ...string) {
		t.Helper()
		book, err := project.LoadBook(root)
		rep, ok := book.Current()
		if err != nil || !ok || !slices.Equal(rep.Files, want) {
			t.Fatalf("files %v, want %v: %v", rep.Files, want, err)
		}
	}
	command(up, "two.md").invoke()
	check("two.md", "one.md")
	command(down, "two.md").invoke()
	check("one.md", "two.md")
	command(menuGroup("Reports", "Remove Document"), "one.md").invoke()
	check("two.md")
	command("Reports", "Delete Report").invoke()
	if command(open, "No Reports").enabled() || command("Reports", "Rename Report…").enabled() {
		t.Fatal("deleted report still available")
	}
	for _, it := range []ui.Item{one, two} {
		if _, err := os.Stat(it.Path); err != nil {
			t.Fatal(err)
		}
	}
}

func TestReportMenuExportSelectedBook(t *testing.T) {
	if os.Getenv("LENS_REPORT_EXPORT_TEST") != "1" {
		t.Skip("requires Typst and pdftotext")
	}
	config := t.TempDir()
	for _, key := range []string{"HOME", "XDG_CONFIG_HOME", "AppData"} {
		t.Setenv(key, config)
	}
	userDir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.CopyFS(filepath.Join(userDir, "pairti-lense", "brands", "cedar"), os.DirFS("../../testdata/brands/cedar")); err != nil {
		t.Fatal(err)
	}
	s := testShell()
	a, b := t.TempDir(), t.TempDir()
	s.projects = []doc.Project{{Name: "Alpha", Root: a}, {Name: "Beta", Root: b}}
	s.lib = &library.Library{Paths: []string{a, b}}
	other := treeNote(t, a, "one.md", "# WrongProjectText")
	for _, it := range []struct{ name, text string }{{"one.md", "# Chapter\n\nPineChapterBody."}, {"two.md", "# Chapter\n\nCedarChapterBody."}} {
		treeNote(t, b, it.name, it.text)
	}
	book := project.Book{}
	book.Add()
	book.Reports[0].Name = "Quarterly"
	book.Reports[0].Template = "cedar"
	book.AddFile("two.md")
	book.AddFile("one.md")
	if err := project.SaveBook(b, book); err != nil {
		t.Fatal(err)
	}
	s.open(other)
	reportCommand(t, s, menuGroup("Reports", "Open Report", "Beta"), "Quarterly").invoke()
	s.open(other)
	s.sheet = ui.NewExportSheet(s.commitExport)

	s.sheet.SetBounds(geometry.NewRect(0, 0, 1000, 800))
	reportCommand(t, s, "Reports", "Export Report…").invoke()
	for _, id := range []string{"current", "open", "go"} {
		p, ok := s.sheet.Spot(id)
		if !ok || !uitest.SimulateClick(s.sheet, p.X, p.Y) {
			t.Fatalf("could not click %s", id)
		}
	}
	path := s.line3.Get()
	if filepath.Dir(path) != b || filepath.Ext(path) != ".pdf" {
		t.Fatalf("export target: %s", path)
	}
	data, err := exec.Command("pdftotext", path, "-").Output()
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)
	if !strings.Contains(body, "Cedar Studio") || strings.Contains(body, "WrongProjectText") || strings.Index(body, "CedarChapterBody") < 0 || strings.Index(body, "PineChapterBody") <= strings.Index(body, "CedarChapterBody") {
		t.Fatalf("wrong template, documents or order:\n%s", body)
	}
}
