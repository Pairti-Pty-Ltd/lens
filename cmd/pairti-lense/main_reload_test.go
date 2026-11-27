package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/uitest"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/library"
	"pairti-lense/internal/theme"
	"pairti-lense/internal/ui"
)

func testShell() *shell {
	s := &shell{
		look:     theme.Pairti(),
		cache:    map[string][]doc.DirEntry{},
		expanded: map[string]bool{},
		active:   -1,
		splitAt:  -1,
	}
	s.line1 = state.NewSignal("")
	s.line2 = state.NewSignal("")
	s.line3 = state.NewSignal("")
	s.words = state.NewSignal("")
	s.side = ui.NewSidebar(func(ui.Item) {})
	s.page = ui.NewPage()
	s.editor = ui.NewEditor(s.noteEdit, s.saveDoc)
	s.actions = ui.NewProjectActions(s.onProjectView)
	s.frame = ui.NewFrame(s.page, s.editor)
	s.act = ui.NewAction(s.pick)
	return s
}

func openNote(t *testing.T, s *shell, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "assay.md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	s.open(ui.Item{Path: path, Rel: "assay.md", ProjectName: "notes", ProjectRoot: dir})
	return path
}

func TestRawOutlineJumpMovesTheCaret(t *testing.T) {
	s := testShell()
	body := "# Top\n\n" + strings.Repeat("word\n", 30) + "## Later\n"
	s.buf = edit.New(body)
	s.editor.UseBuffer(s.buf)
	s.editing = true
	s.editor.SetBounds(geometry.NewRect(0, 0, 480, 120))
	s.scroll = state.NewSignal(float32(99))
	s.caret = state.NewSignal("")
	s.words = state.NewSignal("")
	s.chars = state.NewSignal("")
	line := 1 + strings.Count(body[:strings.Index(body, "## Later")], "\n")
	s.jump(line)
	if got := s.buf.CaretLine() + 1; got != line {
		t.Fatalf("caret %d want %d", got, line)
	}
	if s.scroll.Get() != 99 {
		t.Fatalf("preview scroll %v", s.scroll.Get())
	}
	if s.caret.Get() == "" || s.caret.Get() == "1:0" {
		t.Fatalf("footer caret %q", s.caret.Get())
	}
}

func TestZoomGrowsTheReadingSize(t *testing.T) {
	s := testShell()
	s.page.SetTheme(s.look)
	s.editor.SetTheme(s.look)
	s.page.SetBlocks(doc.Blocks([]byte("# Title\n\nHello there.\n")))
	s.editor.SetBuffer("Hello\n")
	before := uitest.LayoutWidget(s.page, 480, 2000).Height
	raw := uitest.LayoutWidget(s.editor, 480, 2000).Height
	s.base = theme.WithPairtiType(s.look)
	s.zoomIndex = theme.ZoomIndex("100%")
	s.zoomCtl = ui.NewZoom(s.zoom)
	s.zoom(1)
	if s.look.Body != 20 || s.zoomCtl.Label() != "125%" || s.line3.Get() != "125%" {
		t.Fatalf("body %v label %q line %q", s.look.Body, s.zoomCtl.Label(), s.line3.Get())
	}
	if s.look.H1 != float32(int(s.look.H1)) || s.look.Code != float32(int(s.look.Code)) {
		t.Fatalf("fractional type %+v", s.look)
	}
	if h := uitest.LayoutWidget(s.page, 480, 2000).Height; h <= before {
		t.Fatalf("page %v then %v", before, h)
	}
	if h := uitest.LayoutWidget(s.editor, 480, 2000).Height; h <= raw {
		t.Fatalf("raw %v then %v", raw, h)
	}
	s.zoom(-1)
	if s.look.Body != 16 || s.zoomCtl.Label() != "100%" {
		t.Fatalf("back %v %q", s.look.Body, s.zoomCtl.Label())
	}
	for range 10 {
		s.zoom(-1)
	}
	if s.zoomCtl.Label() != "50%" || s.look.Body != 8 {
		t.Fatalf("floor %v %q", s.look.Body, s.zoomCtl.Label())
	}
}

func TestFollowOpensARelativeNote(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	index := filepath.Join(dir, "index.md")
	other := filepath.Join(dir, "authentication.md")
	if err := os.WriteFile(index, []byte("See [auth](authentication.md).\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("# Auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.open(ui.Item{Path: index, Rel: "index.md", ProjectName: "notes", ProjectRoot: dir})
	s.follow("authentication.md")
	if s.docPath != other {
		t.Fatalf("path %q", s.docPath)
	}
	if s.buf.Text() != "# Auth\n" {
		t.Fatalf("text %q", s.buf.Text())
	}
	kind, target := resolveLink(index, "https://example.com/docs")
	if kind != "web" || target != "https://example.com/docs" {
		t.Fatalf("web %s %s", kind, target)
	}
	kind, _ = resolveLink(index, "missing-note")
	if kind != "missing" {
		t.Fatalf("kind %s", kind)
	}
}

func TestSecondDocumentStaysOpen(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	first := filepath.Join(dir, "index.md")
	second := filepath.Join(dir, "authentication.md")
	if err := os.WriteFile(first, []byte("# Index\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte("# Auth\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.open(ui.Item{Path: first, Rel: "index.md", ProjectName: "notes", ProjectRoot: dir})
	dirty := edit.New("# Index\nx\n")
	dirty.SetSaved("# Index\n")
	s.bindBuf(dirty)
	s.open(ui.Item{Path: second, Rel: "authentication.md", ProjectName: "notes", ProjectRoot: dir})
	if s.docPath != second || s.buf.Text() != "# Auth\n" {
		t.Fatalf("showing %q %q", s.docPath, s.buf.Text())
	}
	if len(s.docs) != 2 || !strings.Contains(s.docs[0].buf.Text(), "x") {
		t.Fatalf("first tab lost: %+v", s.docs[0].buf.Text())
	}
	s.show(0)
	if s.docPath != first || !strings.Contains(s.buf.Text(), "x") {
		t.Fatalf("back on %q %q", s.docPath, s.buf.Text())
	}
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(nil, nil)
	s.frameB = ui.NewFrame(s.pageB, s.editorB)
	s.splitDoc(1)
	if s.docPath != second || s.docs[0].path != first {
		t.Fatalf("split focus %q main %q", s.docPath, s.docs[0].path)
	}
	if heads := s.page.Outline(); len(heads) != 1 || heads[0].Text != "Index" {
		t.Fatalf("main outline %+v", heads)
	}
	if heads := s.pageB.Outline(); len(heads) != 1 || heads[0].Text != "Auth" {
		t.Fatalf("side outline %+v", heads)
	}
	s.closeDoc(0)
	if len(s.docs) != 2 || s.line3.Get() != "Unsaved" {
		t.Fatalf("docs %d line %q", len(s.docs), s.line3.Get())
	}
}

func TestShellShowsATabBarAndAHiddenRail(t *testing.T) {
	s := testShell()
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(nil, nil)
	s.scroll = state.NewSignal(float32(0))
	s.scrollB = state.NewSignal(float32(0))
	s.words = state.NewSignal("")
	s.chars = state.NewSignal("")
	s.caret = state.NewSignal("")
	s.foot = state.NewSignal("")
	root := s.build()
	uitest.LayoutWidget(root, 1200, 800)
	if s.tabs == nil || s.tabs.Bounds().Height() < 20 {
		t.Fatalf("tab bar %v", s.tabs.Bounds())
	}
	if s.rail == nil || s.rail.Bounds().Width() != 0 {
		t.Fatalf("rail starts open: %v", s.rail.Bounds())
	}
	s.onTool("Outline")
	uitest.LayoutWidget(root, 1200, 800)
	if s.rail.Bounds().Width() < 200 || s.rail.Bounds().Min.X < 600 {
		t.Fatalf("rail %v", s.rail.Bounds())
	}
}

func TestEachSplitHasItsOwnTabs(t *testing.T) {
	s := testShell()
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(nil, nil)
	s.scroll = state.NewSignal(float32(0))
	s.scrollB = state.NewSignal(float32(0))
	s.words = state.NewSignal("")
	s.chars = state.NewSignal("")
	s.caret = state.NewSignal("")
	s.foot = state.NewSignal("")
	root := s.build()
	dir := t.TempDir()
	first := filepath.Join(dir, "index.md")
	second := filepath.Join(dir, "auth.md")
	os.WriteFile(first, []byte("# Index\n"), 0o644)
	os.WriteFile(second, []byte("# Auth\n"), 0o644)
	s.open(ui.Item{Path: first, Rel: "index.md", ProjectRoot: dir, ProjectName: "notes"})
	s.open(ui.Item{Path: second, Rel: "auth.md", ProjectRoot: dir, ProjectName: "notes"})
	s.splitDoc(1)
	uitest.LayoutWidget(root, 1200, 800)
	if s.tabs == nil || s.tabsB == nil {
		t.Fatal("missing a tab bar")
	}
	if s.tabs.Bounds().Height() < 20 || s.tabsB.Bounds().Height() < 20 {
		t.Fatalf("tabs %v tabsB %v", s.tabs.Bounds(), s.tabsB.Bounds())
	}
	left := s.tabs.Parent().(interface{ Bounds() geometry.Rect }).Bounds()
	right := s.tabsB.Parent().(interface{ Bounds() geometry.Rect }).Bounds()
	if right.Min.X <= left.Min.X || left.Width() < 200 || right.Width() < 200 {
		t.Fatalf("panes left %v right %v", left, right)
	}
	s.showPane(0, 1)
	s.showPane(1, 0)
	if s.active != 1 || s.splitAt != 0 {
		t.Fatalf("active %d split %d", s.active, s.splitAt)
	}
	if s.page.OpenLine() == 0 && s.pageB.OpenLine() == 0 {
		t.Fatal("panes did not load")
	}
}

func TestSidebarNamesUsesTheFilename(t *testing.T) {
	s := testShell()
	dir := t.TempDir()
	s.lib = &library.Library{}
	if err := s.lib.Add(dir); err != nil {
		t.Fatal(err)
	}
	s.libPath = filepath.Join(dir, "library.json")
	s.projects = []doc.Project{{Name: "notes", Root: dir}}
	s.expanded[dir] = true
	s.cache[dir] = []doc.DirEntry{{
		Name: "keep.md", Abs: filepath.Join(dir, "keep.md"), Rel: "keep.md", Title: "Keep",
	}}
	label := func() string {
		for _, it := range s.items() {
			if it.Rel == "keep.md" {
				return it.Label
			}
		}
		return ""
	}
	if label() != "Keep" {
		t.Fatalf("titles %q", label())
	}
	s.onFileNames(true)
	if label() != "keep.md" {
		t.Fatalf("filenames %q", label())
	}
	s.onFileNames(false)
	if label() != "Keep" {
		t.Fatalf("back to titles %q", label())
	}
}

func TestViewControlsApplyToEveryProject(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	write := func(dir, name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(a, "keep.md", "# Keep\n")
	write(a, ".secret.md", "# Secret\n")
	write(b, "note.md", "# Note\n")
	write(b, ".hidden.md", "# Hidden\n")
	s := testShell()
	s.lib = &library.Library{}
	if err := s.lib.Add(a); err != nil {
		t.Fatal(err)
	}
	if err := s.lib.Add(b); err != nil {
		t.Fatal(err)
	}
	s.libPath = filepath.Join(t.TempDir(), "library.json")
	s.rescan()
	s.expanded[a] = true
	s.expanded[b] = true
	s.cache[a] = []doc.DirEntry{{
		Name: "keep.md", Abs: filepath.Join(a, "keep.md"), Rel: "keep.md", Title: "Keep",
	}}
	s.cache[b] = []doc.DirEntry{{
		Name: "note.md", Abs: filepath.Join(b, "note.md"), Rel: "note.md", Title: "Note",
	}}
	s.onFileNames(true)
	if !s.lib.View(a).Names || !s.lib.View(b).Names {
		t.Fatalf("names a %v b %v", s.lib.View(a).Names, s.lib.View(b).Names)
	}
	got := map[string]bool{}
	for _, it := range s.items() {
		if it.Rel == "keep.md" || it.Rel == "note.md" {
			got[it.Label] = true
		}
	}
	if !got["keep.md"] || !got["note.md"] {
		t.Fatalf("labels %v", got)
	}
	s.onProjectView(true, false)
	if !has(cachedNames(s, a), ".secret.md") || !has(cachedNames(s, b), ".hidden.md") {
		t.Fatalf("a %v b %v", cachedNames(s, a), cachedNames(s, b))
	}
}

func TestProjectTogglesChangeTheTree(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".gitignore", "skip.md\n")
	write("visible.md", "# Visible\n")
	write("skip.md", "# Skip\n")
	write(".secret.md", "# Secret\n")
	s := testShell()
	s.lib = &library.Library{}
	if err := s.lib.Add(root); err != nil {
		t.Fatal(err)
	}
	s.libPath = filepath.Join(t.TempDir(), "library.json")
	s.rescan()
	if names := cachedNames(s, root); len(names) != 1 || names[0] != "visible.md" {
		t.Fatalf("hidden = %v", names)
	}
	s.onProjectView(true, false)
	names := cachedNames(s, root)
	if !has(names, ".secret.md") || has(names, "skip.md") {
		t.Fatalf("dots = %v", names)
	}
	s.onProjectView(true, true)
	names = cachedNames(s, root)
	if !has(names, "skip.md") || !has(names, ".secret.md") || has(names, ".git") {
		t.Fatalf("shown = %v", names)
	}
}

func cachedNames(s *shell, root string) []string {
	var out []string
	for _, e := range s.cache[root] {
		out = append(out, e.Name)
	}
	return out
}

func has(names []string, want string) bool {
	for _, n := range names {
		if n == want {
			return true
		}
	}
	return false
}

func TestTagAsksForANameWhenSearchIsEmpty(t *testing.T) {
	s := testShell()
	s.lib = &library.Library{}
	s.onTool("Tag")
	if s.line3.Get() != "Type a tag in Search" {
		t.Fatalf("line3 %q", s.line3.Get())
	}
}

func TestCleanDiskWriteReloads(t *testing.T) {
	s := testShell()
	openNote(t, s, "# A\n")
	if !s.onDisk("# B\n") {
		t.Fatal("clean write was ignored")
	}
	if s.buf.Text() != "# B\n" || s.buf.Dirty() || s.pending != "" {
		t.Fatalf("text %q dirty %v pending %q", s.buf.Text(), s.buf.Dirty(), s.pending)
	}
	if s.line3.Get() != "Formatted" {
		t.Fatalf("line3 %q", s.line3.Get())
	}
}

func TestDirtyDiskWriteAsks(t *testing.T) {
	s := testShell()
	openNote(t, s, "# A\n")
	s.buf.Insert('x')
	if !s.onDisk("# B\n") {
		t.Fatal("did not ask")
	}
	if !strings.HasPrefix(s.buf.Text(), "x") {
		t.Fatalf("ours lost: %q", s.buf.Text())
	}
	got := s.act.Choices()
	if len(got) != 3 || got[0] != "Ours" || got[1] != "Theirs" || got[2] != "Merge" {
		t.Fatalf("choices %v", got)
	}
	if s.line3.Get() != "Changed on disk" {
		t.Fatalf("line3 %q", s.line3.Get())
	}
	s.pick("Ours")
	if s.pending != "" || !strings.HasPrefix(s.buf.Text(), "x") || s.line3.Get() != "Unsaved" {
		t.Fatalf("after ours text %q line %q", s.buf.Text(), s.line3.Get())
	}
	if s.onDisk("# B\n") {
		t.Fatal("dismissed disk text asked again")
	}
	if !s.onDisk("# C\n") {
		t.Fatal("a new disk write did not ask")
	}
	s.pick("Theirs")
	if s.buf.Text() != "# C\n" || s.buf.Dirty() || s.line3.Get() != "Formatted" {
		t.Fatalf("theirs text %q dirty %v line %q", s.buf.Text(), s.buf.Dirty(), s.line3.Get())
	}
}

func TestMergeMarksConflicts(t *testing.T) {
	s := testShell()
	openNote(t, s, "a\nb\nc\n")
	s.editing = true
	s.buf = edit.New(strings.Replace(s.buf.Text(), "b", "B", 1))
	s.buf.SetSaved("a\nb\nc\n")
	if !s.onDisk("a\nX\nc\n") {
		t.Fatal("did not ask")
	}
	s.pick("Merge")
	if s.line3.Get() != "1 conflict" || !strings.Contains(s.buf.Text(), "<<<<<<< ours") || !strings.Contains(s.buf.Text(), ">>>>>>> theirs") {
		t.Fatalf("line %q text %q", s.line3.Get(), s.buf.Text())
	}
	if !s.editing || !s.buf.Dirty() {
		t.Fatal("conflict should stay in the editor, unsaved")
	}
}

func TestOutlineSitsAtTheRightEdge(t *testing.T) {
	s := testShell()
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(nil, nil)
	s.scroll = state.NewSignal(float32(0))
	s.scrollB = state.NewSignal(float32(0))
	s.words = state.NewSignal("")
	s.chars = state.NewSignal("")
	s.caret = state.NewSignal("")
	s.foot = state.NewSignal("")
	root := s.build()
	uitest.LayoutWidget(root, 1200, 800)
	column := root
	if kids := root.Children(); len(kids) > 0 {
		if _, ok := root.(*primitives.BoxWidget); !ok {
			column = kids[0]
		}
	}
	bar, ok := column.(*primitives.BoxWidget).Children()[0].(*primitives.BoxWidget)
	if !ok {
		t.Fatal("toolbar")
	}
	kids := bar.Children()
	rail, ok := kids[len(kids)-1].(interface{ Bounds() geometry.Rect })
	if !ok {
		t.Fatalf("rail %T", kids[len(kids)-1])
	}
	b := rail.Bounds()
	if b.Min.X < 1000 || b.Max.X < 1150 {
		t.Fatalf("outline %v", b)
	}
	tools, ok := kids[0].(interface{ Bounds() geometry.Rect })
	if !ok {
		t.Fatal("tools")
	}
	if tools.Bounds().Max.X > b.Min.X-200 {
		t.Fatalf("tools %v outline %v", tools.Bounds(), b)
	}
}

func TestSidePanelsFollowTheActiveSplit(t *testing.T) {
	s := testShell()
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(nil, nil)
	s.rail = ui.NewRail(nil, nil)
	dir := t.TempDir()
	left := filepath.Join(dir, "left.md")
	right := filepath.Join(dir, "right.md")
	if err := os.WriteFile(left, []byte("# Left note\n\nalpha\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(right, []byte("# Right note\n\n## Detail\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	s.side.SetItems([]ui.Item{{Label: "left.md", Path: left}, {Label: "right.md", Path: right}})
	s.open(ui.Item{Path: left, Rel: "left.md", ProjectName: "notes", ProjectRoot: dir})
	s.open(ui.Item{Path: right, Rel: "right.md", ProjectName: "notes", ProjectRoot: dir})
	s.splitDoc(1)
	if text := headingText(s.rail.Outline()); text != "Right note" || s.side.Selected() != right {
		t.Fatalf("right outline %q selected %q", text, s.side.Selected())
	}
	s.notePage(0)
	if text := headingText(s.rail.Outline()); text != "Left note" || s.side.Selected() != left {
		t.Fatalf("left outline %q selected %q", text, s.side.Selected())
	}
	s.editorB.SetActivate(func() { s.notePage(1) })
	s.editorB.SetBounds(geometry.NewRect(0, 0, 400, 80))
	if !s.editorB.Event(uitest.NewMockContext(), uitest.Click(editGutter()+4, 8)) {
		t.Fatal("click missed the right editor")
	}
	if text := headingText(s.rail.Outline()); text != "Right note" || s.side.Selected() != right {
		t.Fatalf("after click outline %q selected %q", text, s.side.Selected())
	}
	if len(s.rail.Outline()) < 2 || s.rail.Outline()[1].Text != "Detail" {
		t.Fatalf("outline %+v", s.rail.Outline())
	}
}

func TestFocusRevealsTheFileInTheTree(t *testing.T) {
	s := testShell()
	s.lib = &library.Library{}
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	note := filepath.Join(sub, "note.md")
	if err := os.WriteFile(note, []byte("# Nested\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "top.md"), []byte("# Top\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.lib.Add(root); err != nil {
		t.Fatal(err)
	}
	s.rescan()
	s.open(ui.Item{Path: note, Rel: "sub/note.md", ProjectName: "notes", ProjectRoot: root})
	if s.side.Selected() != note {
		t.Fatalf("selected %q", s.side.Selected())
	}
	for _, it := range s.items() {
		if it.Path == note {
			return
		}
	}
	t.Fatalf("tree hid %s: %+v", note, s.items())
}

func headingText(heads []ui.Heading) string {
	if len(heads) == 0 {
		return ""
	}
	return heads[0].Text
}

func editGutter() float32 { return 48 }

func TestToolbarOmitsTheDocumentTitle(t *testing.T) {
	s := testShell()
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(nil, nil)
	s.scroll = state.NewSignal(float32(0))
	s.scrollB = state.NewSignal(float32(0))
	s.words = state.NewSignal("")
	s.chars = state.NewSignal("")
	s.caret = state.NewSignal("")
	s.foot = state.NewSignal("")
	root := s.build()
	s.line1.Set("AGENTS.md")
	uitest.LayoutWidget(root, 1200, 800)
	canvas := uitest.DrawWidget(root)
	if drawn(canvas, "AGENTS.md") {
		t.Fatal("menu bar still shows the document title")
	}
}

func TestFooterStaysOneLine(t *testing.T) {
	s := testShell()
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(nil, nil)
	s.scroll = state.NewSignal(float32(0))
	s.scrollB = state.NewSignal(float32(0))
	s.words = state.NewSignal("")
	s.chars = state.NewSignal("")
	s.caret = state.NewSignal("")
	s.foot = state.NewSignal("")
	root := s.build()
	ctx := uitest.NewMockContext()
	layout := func(w, h float32) {
		t.Helper()
		widget.LayoutChild(root, ctx, geometry.Tight(geometry.Sz(w, h)))
	}
	layout(1200, 800)
	foot := lastBox(t, root)
	want := s.look.Meta*1.2 + 16
	if h := foot.Bounds().Height(); h < want-0.5 || h > want+0.5 {
		t.Fatalf("empty footer %v want %v", h, want)
	}
	if y := foot.Bounds().Min.Y; y < 700 {
		t.Fatalf("footer y %v", y)
	}
	if h := footerTextHeight(t, foot); h < s.look.Meta {
		t.Fatalf("text box %v", h)
	}
	status := "Formatted   ·   493 words   ·   3,375 characters"
	s.foot.Set(status)
	// Same window size. The text signal must not leave the short cached bar.
	layout(1200, 800)
	if h := foot.Bounds().Height(); h < want-0.5 || h > want+0.5 {
		t.Fatalf("status footer %v want %v", h, want)
	}
	canvas := uitest.DrawWidget(root)
	if !drawn(canvas, status) {
		t.Fatal("status did not draw")
	}
	layout(1100, 640)
	if h := foot.Bounds().Height(); h < want-0.5 || h > want+0.5 {
		t.Fatalf("resized footer %v want %v", h, want)
	}
}

func drawn(canvas *uitest.MockCanvas, text string) bool {
	for _, call := range canvas.Texts {
		if call.Text == text {
			return true
		}
	}
	for _, call := range canvas.StyledTexts {
		if call.Text == text {
			return true
		}
	}
	return false
}

func lastBox(t *testing.T, root widget.Widget) *primitives.BoxWidget {
	t.Helper()
	if kids := root.Children(); len(kids) > 0 {
		if _, ok := root.(*primitives.BoxWidget); !ok {
			root = kids[0]
		}
	}
	box, ok := root.(*primitives.BoxWidget)
	if !ok {
		t.Fatalf("root %T", root)
	}
	kids := box.Children()
	foot, ok := kids[len(kids)-1].(*primitives.BoxWidget)
	if !ok {
		t.Fatalf("footer %T", kids[len(kids)-1])
	}
	return foot
}

func footerTextHeight(t *testing.T, foot *primitives.BoxWidget) float32 {
	t.Helper()
	exp, ok := foot.Children()[0].(*primitives.ExpandedWidget)
	if !ok {
		t.Fatalf("footer child %T", foot.Children()[0])
	}
	b, ok := exp.Child().(interface{ Bounds() geometry.Rect })
	if !ok {
		t.Fatalf("text %T", exp.Child())
	}
	return b.Bounds().Height()
}

func TestSwitchToFormattedRendersUnsavedTextAndPreservesPassage(t *testing.T) {
	s := testShell()
	path := openNote(t, s, strings.Repeat("A paragraph.\n\n", 80)+"## Draft\n")
	s.editor.SetBounds(geometry.NewRect(0, 0, 640, 160))
	s.pick("Raw")
	s.editor.ShowLine(161)
	s.buf.Place(160, 8)
	s.buf.Insert('!')
	cursor := s.buf.Cursor()
	var scrolled float32
	s.page.OnScroll(func(y float32) { scrolled = y })
	s.pick("Formatted")
	uitest.LayoutWidget(s.page, 640, 10000)
	headings := s.page.Outline()
	if len(headings) != 1 || headings[0].Text != "Draft!" || scrolled <= 0 {
		t.Fatalf("outline %v, scroll %v", headings, scrolled)
	}
	if s.buf.Cursor() != cursor || s.page.Open() || s.page.Buffer() != s.buf {
		t.Fatal("switch lost the caret or left the active block in source form")
	}
	s.page.SetFocused(true)
	if !s.page.Event(nil, uitest.KeyType(event.KeyI, 'i', 0)) || s.page.OpenLine() != 161 {
		t.Fatal("could not resume editing at the same passage")
	}
	s.pick("Raw")
	if s.buf.Cursor() != cursor || !s.buf.Dirty() {
		t.Fatal("round trip lost unsaved editing position")
	}
	onDisk, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(onDisk), "Draft!") {
		t.Fatalf("switch unexpectedly saved edits, read error %v", err)
	}
}

func TestDocumentTabsLeaveHeadingNavigationToRail(t *testing.T) {
	s := testShell()
	s.pageB = ui.NewPage()
	s.editorB = ui.NewEditor(nil, nil)
	s.scroll = state.NewSignal(float32(0))
	s.scrollB = state.NewSignal(float32(0))
	s.documentColumn()
	openNote(t, s, "# Report\n\n## Start\n\n### Next\n")
	s.buf.Place(4, 0)
	uitest.LayoutWidget(s.page, 700, 900)
	s.page.SetBounds(geometry.NewRect(0, 0, 700, 900))
	if heads := s.rail.Outline(); len(heads) != 3 || heads[2].Text != "Next" {
		t.Fatalf("outline after opening note: %v", heads)
	}
	uitest.LayoutWidget(s.tabs, 600, 28)
	s.tabs.SetBounds(geometry.NewRect(0, 0, 600, 28))
	canvas := uitest.DrawWidget(s.tabs)
	for _, row := range canvas.StyledTexts {
		if row.Text != "assay.md" && row.Text != "×" {
			t.Errorf("unexpected text in tab strip: %q", row.Text)
		}
	}
	if len(canvas.Lines) != 0 || uitest.SimulateClick(s.tabs, 182, 14) {
		t.Fatal("spare tab space has a disclosure icon or accepts clicks")
	}
}
