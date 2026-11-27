package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/ui/state"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/gitlog"
	"pairti-lense/internal/library"
	"pairti-lense/internal/session"
	"pairti-lense/internal/ui"
)

func TestFindJumpsToABodyWord(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	os.WriteFile(path, []byte("# Note\n\nThe membrane assay sits here.\n"), 0o644)
	s.lib = &library.Library{Paths: []string{dir}}
	s.open(ui.Item{Path: path, Rel: "note.md", ProjectRoot: dir, ProjectName: "notes"})
	s.filter = "membrane"
	s.findInDoc()
	if s.buf.CaretLine() != 2 {
		t.Fatalf("line %d", s.buf.CaretLine())
	}
	if s.page.OpenLine() != 3 {
		t.Fatalf("open %d", s.page.OpenLine())
	}
}

func TestNewDocumentOpensATab(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	s.chosen = dir
	s.cache = map[string][]doc.DirEntry{}
	s.newDoc()
	path := filepath.Join(dir, "untitled.md")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(b), "# ") {
		t.Fatalf("body %q", b)
	}
	if s.docPath != path {
		t.Fatalf("open %q", s.docPath)
	}
}

func TestBackAndForward(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{Paths: []string{dir}}
	var paths []string
	for _, name := range []string{"a.md", "b.md", "c.md"} {
		p := filepath.Join(dir, name)
		os.WriteFile(p, []byte("# "+name+"\n"), 0o644)
		paths = append(paths, p)
		s.open(ui.Item{Path: p, Rel: name, ProjectRoot: dir, ProjectName: "notes"})
	}
	s.back()
	s.back()
	if s.docPath != paths[0] {
		t.Fatalf("back %q", s.docPath)
	}
	s.forward()
	if s.docPath != paths[1] {
		t.Fatalf("forward %q", s.docPath)
	}
}

func TestTagFiltersTheTree(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep.md")
	drop := filepath.Join(dir, "drop.md")
	os.WriteFile(keep, []byte("# Keep\n"), 0o644)
	os.WriteFile(drop, []byte("# Drop\n"), 0o644)
	s.lib = &library.Library{Paths: []string{dir}}
	s.libPath = filepath.Join(dir, "library.json")
	s.projects = []doc.Project{{Name: "notes", Root: dir}}
	s.expanded[dir] = true
	s.cache[dir] = []doc.DirEntry{
		{Name: "keep.md", Abs: keep, Rel: "keep.md", Title: "Keep"},
		{Name: "drop.md", Abs: drop, Rel: "drop.md", Title: "Drop"},
	}
	s.open(ui.Item{Path: keep, Rel: "keep.md", ProjectRoot: dir, ProjectName: "notes"})
	s.filter = "lab"
	s.tagCurrent()
	s.filter = "#lab"
	var sawKeep, sawDrop bool
	for _, it := range s.items() {
		if it.Path == keep {
			sawKeep = true
		}
		if it.Path == drop {
			sawDrop = true
		}
	}
	if !sawKeep || sawDrop {
		t.Fatalf("keep %v drop %v", sawKeep, sawDrop)
	}
}

func TestDocLinkAndFrontMatterTag(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	note := filepath.Join(dir, "wiki", "orchard-survey-and-sampling-plan.md")
	proposal := filepath.Join(dir, "doc", "research", "sample-orchard-field-survey", "PROPOSAL.md")
	other := filepath.Join(dir, "banner.md")
	if err := os.MkdirAll(filepath.Dir(note), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(proposal), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "---\ntitle: Orchard Survey and Sampling Plan\ntype: topic\n---\n\n# Orchard Survey and Sampling Plan\n"
	if err := os.WriteFile(note, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(proposal, []byte("# Proposal\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("---\ntype: entity\n---\n\n# Powered by Grove-P Banner\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.lib = &library.Library{}
	s.projects = []doc.Project{{Name: "grove", Root: dir}}
	s.expanded[dir] = true
	s.cache[dir] = []doc.DirEntry{
		{Name: "orchard-survey-and-sampling-plan.md", Abs: note, Rel: "wiki/orchard-survey-and-sampling-plan.md", Title: "Orchard Survey and Sampling Plan"},
		{Name: "banner.md", Abs: other, Rel: "banner.md", Title: "Powered by Grove-P Banner"},
	}
	s.open(ui.Item{Path: note, Rel: "wiki/orchard-survey-and-sampling-plan.md", ProjectRoot: dir, ProjectName: "grove"})
	s.follow("doc:///research/sample-orchard-field-survey/PROPOSAL.md")
	if s.docPath != proposal {
		t.Fatalf("opened %s", s.docPath)
	}
	s.follow("doc:///missing-proposal.md")
	if s.docPath != proposal || s.line3.Get() != "Missing link" {
		t.Fatalf("path %s line %s", s.docPath, s.line3.Get())
	}
	s.filter = "#topic"
	var sawNote, sawBanner bool
	for _, it := range s.items() {
		if it.Path == note {
			sawNote = true
		}
		if it.Path == other || it.Label == "Powered by Grove-P Banner" {
			sawBanner = true
		}
	}
	if !sawNote || sawBanner {
		t.Fatalf("topic %v banner %v", sawNote, sawBanner)
	}
}

func TestRestoredOpenFolderListsItsNotes(t *testing.T) {
	s := testShell()
	root := t.TempDir()
	sub := filepath.Join(root, "wiki")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(sub, "orchard-survey-and-sampling-plan.md")
	if err := os.WriteFile(note, []byte("# Orchard Survey and Sampling Plan\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(root, "empty")
	if err := os.Mkdir(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	s.projects = []doc.Project{{Name: "grove", Root: root}}
	s.cache[root] = []doc.DirEntry{
		{Name: "wiki", Abs: sub, Rel: "wiki", IsDir: true},
		{Name: "empty", Abs: empty, Rel: "empty", IsDir: true},
	}
	s.expanded[root] = true
	s.expanded[sub] = true
	s.expanded[empty] = true
	var sawNote, sawEmpty bool
	for _, it := range s.items() {
		if it.Path == note {
			sawNote = true
		}
		if it.Label == "No Markdown files" {
			sawEmpty = true
		}
	}
	if !sawNote || !sawEmpty {
		t.Fatalf("note %v empty %v items %+v", sawNote, sawEmpty, s.items())
	}
}

func TestWikiLinkOpensTheNoteOrCreatesIt(t *testing.T) {
	s := testShell()
	root := t.TempDir()
	from := filepath.Join(root, "wiki", "topics", "orchard-survey-and-sampling-plan.md")
	grove := filepath.Join(root, "wiki", "entities", "grove-f.md")
	if err := os.MkdirAll(filepath.Dir(from), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(grove), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(from, []byte("# Orchard Survey and Sampling Plan\n\nSee [[Grove-F]].\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(grove, []byte("---\ntitle: GroveF\n---\n\n# GroveF\n\n## Key Facts\n\n"+strings.Repeat("word\n", 40)), 0o644); err != nil {
		t.Fatal(err)
	}
	s.projects = []doc.Project{{Name: "grove", Root: root}}
	s.open(ui.Item{Path: from, Rel: "wiki/topics/orchard-survey-and-sampling-plan.md", ProjectRoot: root, ProjectName: "grove"})
	s.follow("wiki:Grove-F#Key Facts")
	if s.docPath != grove {
		t.Fatalf("opened %s", s.docPath)
	}
	s.open(ui.Item{Path: from, Rel: "wiki/topics/orchard-survey-and-sampling-plan.md", ProjectRoot: root, ProjectName: "grove"})
	s.follow("wiki:Brand New Note")
	made := filepath.Join(filepath.Dir(from), "brand-new-note.md")
	if s.docPath != made {
		t.Fatalf("created %s", s.docPath)
	}
	b, err := os.ReadFile(made)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "title: Brand New Note") || strings.Contains(string(b), "[[") {
		t.Fatalf("%s", b)
	}
}

func TestSameFolderNameShowsTheParent(t *testing.T) {
	s := testShell()
	root := t.TempDir()
	back := filepath.Join(root, "grove-p-backend", "docs")
	front := filepath.Join(root, "grove-p-frontend", "docs")
	wiki := filepath.Join(root, "wiki")
	s.projects = []doc.Project{{Name: "grove", Root: root}}
	s.cache[root] = []doc.DirEntry{
		{Name: "grove-p-backend", Abs: filepath.Join(root, "grove-p-backend"), Rel: "grove-p-backend", IsDir: true},
		{Name: "grove-p-frontend", Abs: filepath.Join(root, "grove-p-frontend"), Rel: "grove-p-frontend", IsDir: true},
		{Name: "wiki", Abs: wiki, Rel: "wiki", IsDir: true},
	}
	s.cache[filepath.Join(root, "grove-p-backend")] = []doc.DirEntry{{Name: "docs", Abs: back, Rel: "grove-p-backend/docs", IsDir: true}}
	s.cache[filepath.Join(root, "grove-p-frontend")] = []doc.DirEntry{{Name: "docs", Abs: front, Rel: "grove-p-frontend/docs", IsDir: true}}
	s.cache[back] = []doc.DirEntry{{Name: "note.md", Abs: filepath.Join(back, "note.md"), Rel: "grove-p-backend/docs/note.md", Title: "Backend note"}}
	s.cache[front] = []doc.DirEntry{{Name: "note.md", Abs: filepath.Join(front, "note.md"), Rel: "grove-p-frontend/docs/note.md", Title: "Frontend note"}}
	s.cache[wiki] = []doc.DirEntry{{Name: "index.md", Abs: filepath.Join(wiki, "index.md"), Rel: "wiki/index.md", Title: "Index"}}
	for _, dir := range []string{root, filepath.Join(root, "grove-p-backend"), filepath.Join(root, "grove-p-frontend"), back, front, wiki} {
		s.expanded[dir] = true
	}
	var sawBack, sawFront, sawWiki bool
	for _, it := range s.items() {
		switch it.Label {
		case "docs (grove-p-backend)":
			sawBack = it.Path == back
		case "docs (grove-p-frontend)":
			sawFront = it.Path == front
		case "wiki":
			sawWiki = true
		case "docs":
			t.Fatalf("bare docs while both are open: %+v", s.items())
		}
	}
	if !sawBack || !sawFront || !sawWiki {
		t.Fatalf("back %v front %v wiki %v items %+v", sawBack, sawFront, sawWiki, s.items())
	}
	s.expanded[filepath.Join(root, "grove-p-frontend")] = false
	for _, it := range s.items() {
		if it.Path == back && it.Label != "docs" {
			t.Fatalf("single docs labeled %q", it.Label)
		}
		if it.Label == "docs (grove-p-frontend)" {
			t.Fatal("closed folder still labeled")
		}
	}
}

func TestHistoryRowsStayShort(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.projects = []doc.Project{{Name: "field-guide", Root: dir}}
	s.chosen = dir
	s.histMode = true
	s.histRoot = dir
	s.commits = []gitlog.Commit{{
		Hash:    "9df6a5deb0123456789abcdef",
		Date:    "22 Nov",
		Subject: "Field guide, observations and plot notes: the north trail markers, the spring sampling dates and the annotated map for the orchard team (field-guide-ab12.3.4.5)",
	}}
	s.histFiles = map[string][]string{"9df6a5deb0123456789abcdef": {"doc/architecture.md"}}
	items := s.historyItems()
	if len(items) != 1 {
		t.Fatalf("rows %d %+v", len(items), items)
	}
	if strings.Contains(items[0].Label, "9df6a5deb") || strings.Contains(items[0].Label, "field-guide-ab12.3.4.5") {
		t.Fatalf("label %q", items[0].Label)
	}
	if !strings.Contains(items[0].Label, "22 Nov") || !items[0].Dir || items[0].Expanded {
		t.Fatalf("row %+v", items[0])
	}
}

func TestFooterNamesTheBranch(t *testing.T) {
	s := testShell()
	s.foot = state.NewSignal("")
	s.branch = "main"
	s.line3.Set("Formatted")
	s.words.Set("9,579 words")
	s.writeFooter()
	got := s.foot.Get()
	if !strings.Contains(got, "main") || strings.Contains(got, "9df6a5") {
		t.Fatalf("footer %q", got)
	}
}

func TestReportListStaysOnTheChosenProject(t *testing.T) {
	s := testShell()
	refer := t.TempDir()
	grove := t.TempDir()
	s.projects = []doc.Project{
		{Name: "field-guide", Root: refer},
		{Name: "grove-p-workspace", Root: grove},
	}
	s.reportMode = true
	s.reportPick = refer
	s.reportRoot = refer
	s.book.Add()
	s.book.Reports[0].Name = "Orchard Survey Manual"
	s.docPath = filepath.Join(grove, "note.md")
	s.chosen = grove
	var saw bool
	for _, it := range s.items() {
		if it.Label == "Orchard Survey Manual" {
			saw = true
		}
	}
	if !saw {
		t.Fatalf("chosen project lost its reports: %+v", s.items())
	}
	if s.reportPick != refer {
		t.Fatalf("open file moved the reports to %s", s.reportPick)
	}
	s.pickReportProject(grove)
	for _, it := range s.items() {
		if it.Label == "Orchard Survey Manual" {
			t.Fatal("previous project's report still listed")
		}
	}
	if s.reportRoot != grove {
		t.Fatalf("root %s", s.reportRoot)
	}
}

func TestReportListShowsShortNames(t *testing.T) {
	s := testShell()
	root := t.TempDir()
	rel := "wiki/topics/orchard-survey-and-sampling-plan.md"
	s.projects = []doc.Project{{Name: "grove", Root: root}}
	s.chosen = root
	s.reportMode = true
	s.reportRoot = root
	s.book.Add()
	s.book.Add()
	s.book.AddFile(rel)
	s.reportOpen = map[string]bool{"report-2": true}
	var sawFile, sawFirst, sawSecond, sawPath bool
	for _, it := range s.items() {
		switch it.Label {
		case "orchard-survey-and-sampling-plan.md":
			sawFile = it.Note == "report-2"
		case "Report":
			sawFirst = it.Dir && !it.Expanded
		case "Report 2":
			sawSecond = it.Dir && it.Expanded
		case "New report":
			t.Fatal("new report is still a row")
		}
		if strings.Contains(it.Label, "wiki/topics") {
			sawPath = true
		}
	}
	if !sawFile || !sawFirst || !sawSecond || sawPath {
		t.Fatalf("file %v first %v second %v path %v items %+v", sawFile, sawFirst, sawSecond, sawPath, s.items())
	}
}

func TestSessionSkipsAMissingFile(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	keep := filepath.Join(dir, "keep.md")
	if err := os.WriteFile(keep, []byte("# Keep\n\nHello.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.projects = []doc.Project{{Name: "notes", Root: dir}}
	s.expanded[dir] = true
	s.holdSave = true
	s.restoreDocs(session.State{
		Docs: []session.Doc{
			{Path: filepath.Join(dir, "gone.md")},
			{Path: keep, Raw: true},
		},
		Active: 1,
		Split:  -1,
	})
	if len(s.docs) != 1 || s.docs[0].path != keep {
		t.Fatalf("docs %+v", s.docs)
	}
	if s.active != 0 || !s.editing || s.docPath != keep {
		t.Fatalf("active %d raw %v path %s", s.active, s.editing, s.docPath)
	}
}

func TestRestoreRunsBeforeTheFooterExists(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("# Hello\n\nWords here.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.projects = []doc.Project{{Name: "notes", Root: dir}}
	s.words = nil
	s.chars = nil
	s.caret = nil
	s.foot = nil
	s.holdSave = true
	s.restoreDocs(session.State{
		Docs:   []session.Doc{{Path: path}},
		Active: 0,
		Split:  -1,
	})
	if s.docPath != path || len(s.docs) != 1 {
		t.Fatalf("path %s docs %d", s.docPath, len(s.docs))
	}
}

func TestSearchOpensMatchesAndRestores(t *testing.T) {
	s := testShell()
	root := t.TempDir()
	nest := filepath.Join(root, "nest")
	if err := os.Mkdir(nest, 0o755); err != nil {
		t.Fatal(err)
	}
	hit := filepath.Join(nest, "note-004.md")
	stay := filepath.Join(root, "stay.md")
	if err := os.WriteFile(hit, []byte("# Hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stay, []byte("# Stay\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	open := filepath.Join(root, "open")
	if err := os.Mkdir(open, 0o755); err != nil {
		t.Fatal(err)
	}
	s.projects = []doc.Project{{Name: "notes", Root: root}}
	s.expanded[root] = true
	s.expanded[open] = true
	s.setFilter("004")
	var sawHit bool
	for _, it := range s.items() {
		if it.Path == hit {
			sawHit = true
		}
	}
	if !sawHit || !s.expanded[nest] {
		t.Fatalf("hit %v nest %v items %+v", sawHit, s.expanded[nest], s.items())
	}
	s.setFilter("")
	if s.expanded[nest] {
		t.Fatal("closed folder stayed open")
	}
	if !s.expanded[open] || !s.expanded[root] {
		t.Fatalf("original folders collapsed %v", s.expanded)
	}
	for _, it := range s.items() {
		if it.Path == hit {
			t.Fatal("cleared search still shows the nested file")
		}
	}
}

func TestInsertsChangeTheBuffer(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	os.WriteFile(path, []byte("# Note\n"), 0o644)
	s.lib = &library.Library{Paths: []string{dir}}
	s.open(ui.Item{Path: path, Rel: "note.md", ProjectRoot: dir, ProjectName: "notes"})
	for _, kind := range []string{"Code", "Link", "Image", "Table", "Footnote"} {
		before := s.buf.Text()
		s.insertMarkup(kind)
		if s.buf.Text() == before || !strings.Contains(s.buf.Text(), map[string]string{
			"Code": "```", "Link": "](url)", "Image": "![](", "Table": "| Column |", "Footnote": "[^1]",
		}[kind]) {
			t.Fatalf("%s buffer %q", kind, s.buf.Text())
		}
	}
}

func TestOpenRawSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "library.json")
	lib := &library.Library{}
	lib.OpenRaw = true
	if err := lib.Save(path); err != nil {
		t.Fatal(err)
	}
	got, err := library.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.OpenRaw {
		t.Fatal("open raw was not stored")
	}
	s := testShell()
	s.lib = got
	s.libPath = path
	note := filepath.Join(dir, "note.md")
	os.WriteFile(note, []byte("# Note\n"), 0o644)
	s.open(ui.Item{Path: note, Rel: "note.md", ProjectRoot: dir, ProjectName: "notes"})
	if !s.editing {
		t.Fatal("note did not open raw")
	}
}

func TestGearFlipsOpenRaw(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{}
	s.libPath = filepath.Join(dir, "library.json")
	s.onTool("Settings")
	if !s.lib.OpenRaw || s.line3.Get() != "Opens in Raw" {
		t.Fatalf("raw %v line %q", s.lib.OpenRaw, s.line3.Get())
	}
	got, err := library.Load(s.libPath)
	if err != nil || !got.OpenRaw {
		t.Fatalf("saved %v %v", got, err)
	}
}

func TestSelectReportReplacesChapterContentsWithWholeReport(t *testing.T) {
	s := testShell()
	root := t.TempDir()
	s.projects = []doc.Project{{Name: "manual", Root: root}}
	s.reportMode = true
	s.reportRoot = root
	s.book.Add()
	for _, file := range []struct{ name, text string }{
		{"index.md", "# Manual\n\n## Context\n"},
		{"roles.md", "---\ntitle: Roles\n---\n\n# Roles\n\n## Permissions\n"},
	} {
		if err := os.WriteFile(filepath.Join(root, file.name), []byte(file.text), 0644); err != nil {
			t.Fatal(err)
		}
		s.book.AddFile(file.name)
	}
	s.rail = ui.NewRail(s.jump, s.follow)
	s.rail.SetOutline([]ui.Heading{{Text: "Previous file"}})
	s.onReport(ui.Item{Dir: true, Rel: s.book.Reports[0].ID})
	heads := s.rail.Outline()
	if len(heads) != 4 {
		t.Fatalf("report contents %+v", heads)
	}
	lines := strings.Split(s.buf.Text(), "\n")
	for i, title := range []string{"Manual", "Context", "Roles", "Permissions"} {
		h := heads[i]
		if h.Text != title || strings.TrimLeft(lines[h.Line-1], "# ") != title {
			t.Fatalf("heading %d got %+v", i, h)
		}
	}
}
