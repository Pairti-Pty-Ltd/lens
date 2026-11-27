// Pairti Lens build 0: several project folders, one rendered Markdown file.
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/state"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/agent"
	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/gitlog"
	"pairti-lense/internal/index"
	"pairti-lense/internal/library"
	"pairti-lense/internal/project"
	"pairti-lense/internal/report"
	"pairti-lense/internal/session"
	"pairti-lense/internal/theme"
	"pairti-lense/internal/ui"
	"pairti-lense/internal/watch"
)

func (s *shell) initWidgets() {
	look := s.look
	s.side = ui.NewSidebar(s.onItem)
	s.side.OnChevron(s.onChevron)
	s.side.OnTab(s.openTab)
	s.side.OnMenu(s.showTreeMenu)
	s.side.OnOrder(func(it ui.Item, dir int) {
		s.book.Move(it.Note, it.Rel, dir)
		s.saveReports()
		s.refresh()
	})
	s.side.SetTheme(look)
	s.menu = ui.NewTreeMenu(s.onTreeMenu)
	s.menu.SetTheme(look)
	s.tabMenu = ui.NewTabMenu(s.onTabMenu)
	s.tabMenu.SetTheme(look)
	s.palette = ui.NewPalette(s.askIndex, s.openHit)
	s.palette.SetTheme(look)
	s.actions = ui.NewProjectActions(s.onProjectView)
	s.actions.OnNames(s.onFileNames)
	s.actions.OnDocs(s.onDocs)
	s.actions.SetTheme(look)
	s.line1 = state.NewSignal(s.idleLine())
	s.line2 = state.NewSignal("")
	s.line3 = state.NewSignal("")
	s.page = ui.NewPage()
	s.page.SetTheme(look)
	s.page.SetFollow(func(raw string) {
		if s.focus != 0 {
			s.remember()
			s.focus = 0
			s.applyFocus()
		}
		s.follow(raw)
	})
	s.page.SetInline(func() { s.notePage(0) }, s.saveDoc, s.runCmd)
	s.editor = ui.NewEditor(s.noteEdit, s.saveDoc)
	s.editor.SetWindow(s.winCmd)
	s.page.SetWindow(s.winCmd)
	s.editor.SetTheme(look)
	s.editor.SetCommand(s.runCmd)
	s.editor.SetActivate(func() { s.notePage(0) })
	s.pageB = ui.NewPage()
	s.pageB.SetTheme(look)
	s.pageB.SetFollow(func(raw string) {
		if s.splitAt >= 0 && s.focus != 1 {
			s.remember()
			s.focus = 1
			s.applyFocus()
		}
		s.follow(raw)
	})
	s.pageB.SetInline(func() { s.notePage(1) }, s.saveDoc, s.runCmd)
	s.editorB = ui.NewEditor(s.noteEdit, s.saveDoc)
	s.editorB.SetWindow(s.winCmd)
	s.pageB.SetWindow(s.winCmd)
	s.editorB.SetTheme(look)
	s.editorB.SetCommand(s.runCmd)
	s.editorB.SetActivate(func() { s.notePage(1) })
	s.scroll = state.NewSignal(float32(0))
	s.scrollB = state.NewSignal(float32(0))
	s.page.OnScroll(func(y float32) { s.scroll.Set(y) })
	s.pageB.OnScroll(func(y float32) { s.scrollB.Set(y) })
	s.words = state.NewSignal("")
	s.chars = state.NewSignal("")
	s.caret = state.NewSignal("")
	s.foot = state.NewSignal("")
	s.holdSave = true
	s.restoreDocs(s.boot)
	s.holdSave = false
	s.act = ui.NewAction(s.pick)
	s.act.SetTheme(look)
	s.mode = ui.NewViewSegment(s.pick)
	s.mode.SetTheme(look)
	s.side.SetItems(s.items())
	s.syncActions()

}

type consolePanel interface {
	widget.Widget
	widget.Focusable
	Open() bool
	Start(string, func()) error
	SetOpen(bool)
	Paste(string)
	Close()
	Pump() bool
	SetTheme(theme.Theme)
}

type shell struct {
	remote *browserFiles

	finderOpens     []string // delivered on AppKit’s main thread, applied in the next platform update
	files           fileActions
	recent          []string
	menuContext     *widget.ContextImpl
	clipboard       widget.ClipboardProvider
	lastEditorStats struct {
		buffer     *edit.Buffer
		text, path string
		pane       int
		raw        bool
	}
	lib              *library.Library
	libPath          string
	projects         []doc.Project
	cache            map[string][]doc.DirEntry
	expanded         map[string]bool
	searchSaved      map[string]bool
	filter           string
	chosen           string
	editing          bool
	look             theme.Theme
	base             theme.Theme
	hideRawNumbers   bool
	formattedNumbers bool
	zoomIndex        int
	zoomCtl          *ui.Zoom
	themePath        string
	docPath          string
	docRel           string
	docProject       string
	buf              *edit.Buffer

	side     *ui.Sidebar
	pane     *ui.SidePane
	editor   *ui.Editor
	frame    *ui.Frame
	act      *ui.Action
	mode     *ui.Segment
	watch    *watch.Watch
	pending  string
	ignored  string
	line1    state.Signal[string]
	line2    state.Signal[string]
	line3    state.Signal[string]
	words    state.Signal[string]
	chars    state.Signal[string]
	caret    state.Signal[string]
	foot     state.Signal[string]
	transfer string
	split    *ui.Split
	page     *ui.Page
	actions  *ui.ProjectActions

	docs          []openDoc
	active        int
	splitAt       int
	focus         int
	hist          []ui.Item
	histAt        int
	nav           bool
	pageB         *ui.Page
	editorB       *ui.Editor
	frameB        *ui.Frame
	tabs          *ui.Tabs
	tabsB         *ui.Tabs
	pair          *ui.Pair
	rail          *ui.Rail
	railSplit     *ui.Split
	cmdLine       *ui.CommandLine
	console       consolePanel
	tools         *ui.ToolBar
	exportBtn     *ui.ToolBar
	sheet         *ui.ExportSheet
	menu          *ui.TreeMenu
	tabMenu       *ui.TabMenu
	palette       *ui.Palette
	idx           *index.Index
	replaced      replacedDoc
	scroll        state.Signal[float32]
	scrollB       state.Signal[float32]
	armFocus      bool
	wake          func()
	railPath      string
	agent         *agent.Hub
	mcpSrv        *http.Server
	mcpURL        string
	mcpToken      string
	mcpErr        string
	mcpHelp       *ui.MCPHelp
	mcpAsk        atomic.Bool
	quitAsked     atomic.Bool
	quit          func()
	refreshMenu   func()
	menusDirty    bool
	exportRoot    string
	exportName    string
	reportMode    bool
	diff          *historyComparison
	histMode      bool
	histRepo      bool
	histMore      bool
	histScopeBar  *ui.ModeBar
	reportPick    string
	reportRoot    string
	reportOpen    map[string]bool
	reports       *ui.ReportBar
	viewing       string
	book          project.Book
	commits       []gitlog.Commit
	histRoot      string
	histRel       string
	histErr       string
	histOpen      map[string]bool
	histFiles     map[string][]string
	branch        string
	branchFor     string
	rename        *ui.Rename
	renameID      string
	renameAsk     atomic.Bool
	sessionPath   string
	boot          session.State
	holdSave      bool
	openSavedHook func(path string) // tests: runs after a restored note opens
}

// replacedDoc is the tab a tree click displaced. A double click puts it back.
type replacedDoc struct {
	path  string
	index int
	doc   openDoc
	at    time.Time
}

// openDoc is one tab. The focused tab's fields also live on the shell.
type openDoc struct {
	path, rel, project string
	buf                *edit.Buffer
	editing            bool
	pending, ignored   string
}

func (s *shell) rescan() {
	s.invalidateMenus()
	s.projects = s.projects[:0]
	s.cache = map[string][]doc.DirEntry{}
	s.expanded = map[string]bool{}
	for _, p := range s.lib.Paths {
		proj := doc.Project{Name: filepath.Base(p), Root: p}
		if library.Absent(p) {
			proj.Absent = true
		} else if entries, err := doc.ReadDir(p, p, s.listOf(p)); err != nil {
			proj.Absent = true
		} else {
			s.cache[p] = entries
			s.expanded[p] = true
		}
		s.projects = append(s.projects, proj)
	}
	s.kickIndex()
	s.syncWatch()
}

func (s *shell) save() {
	if s.libPath == "" {
		return
	}
	if err := s.lib.Save(s.libPath); err != nil {
		s.showMessage(err.Error())
	}
}

func (s *shell) refresh() {
	if s.side != nil {
		s.side.SetItems(s.items())
	}
	s.syncReportBar()
	s.syncActions()
}

func (s *shell) setFilter(v string) {
	prev := strings.TrimSpace(s.filter)
	next := strings.TrimSpace(v)
	s.filter = v
	if next == prev {
		s.refresh()
		return
	}
	if prev == "" && next != "" {
		s.searchSaved = copyExpanded(s.expanded)
	}
	if next == "" {
		if s.searchSaved != nil {
			s.expanded = s.searchSaved
			s.searchSaved = nil
		}
		s.refresh()
		s.syncWatch()
		return
	}
	if s.searchSaved != nil {
		s.expanded = copyExpanded(s.searchSaved)
	}
	s.openMatches(strings.ToLower(next))
	s.refresh()
	s.syncWatch()
}

func copyExpanded(m map[string]bool) map[string]bool {
	out := make(map[string]bool, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func (s *shell) openMatches(q string) {
	for _, p := range s.projects {
		if p.Absent || p.Root == "" {
			continue
		}
		if s.markMatch(p.Root, p.Root, q) {
			s.expanded[p.Root] = true
		}
	}
}

func (s *shell) markMatch(dir, root, q string) bool {
	hit := false
	for _, e := range s.ensureListed(dir, root) {
		if e.More {
			continue
		}
		if e.IsDir {
			if s.markMatch(e.Abs, root, q) {
				s.expanded[e.Abs] = true
				hit = true
			}
			continue
		}
		if s.fileMatch(e, q) {
			hit = true
		}
	}
	return hit
}

func (s *shell) ensureListed(dir, root string) []doc.DirEntry {
	if entries, ok := s.cache[dir]; ok {
		return entries
	}
	entries, err := doc.ReadDir(dir, root, s.listOf(root))
	if err != nil {
		return nil
	}
	s.cache[dir] = entries
	return entries
}

func (s *shell) fileMatch(e doc.DirEntry, q string) bool {
	if strings.HasPrefix(q, "#") {
		return s.tagged(e.Abs, strings.TrimPrefix(q, "#"))
	}
	blob := strings.ToLower(e.Name + "\n" + e.Title + "\n" + e.Rel)
	return strings.Contains(blob, q)
}

func (s *shell) add(dir string) {
	if dir == "" {
		return
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		s.showMessage(dir + " is not a folder")
		return
	}
	dir, err = filepath.Abs(dir)
	if err != nil {
		s.fileError(err)
		return
	}
	if err := s.lib.Add(dir); err != nil {
		s.showMessage(err.Error())
		return
	}
	s.chosen = filepath.Clean(dir)
	s.save()
	s.rescan()
	s.refresh()
	s.rememberPath(s.chosen)
}

func (s *shell) removeChosen() {
	root := s.chosen
	if root == "" {
		return
	}
	for _, d := range s.docs {
		if strings.HasPrefix(d.path, root+string(filepath.Separator)) && d.buf != nil && d.buf.Dirty() {
			s.fileError(fmt.Errorf("save %s before removing its project", filepath.Base(d.path)))
			return
		}
	}
	s.lib.Remove(root)
	s.chosen = ""
	s.save()
	s.rescan()
	s.refresh()
	var keep []openDoc
	for _, d := range s.docs {
		if d.path == root || strings.HasPrefix(d.path, root+string(filepath.Separator)) {
			continue
		}
		keep = append(keep, d)
	}
	s.docs = keep
	s.active = -1
	s.splitAt = -1
	s.focus = 0
	if s.pair != nil {
		s.pair.SetSplit(false)
	}
	if len(s.docs) > 0 {
		s.active = 0
		s.loadPane(0)
		s.applyFocus()
		return
	}
	s.pending = ""
	s.ignored = ""
	s.docPath = ""
	s.buf = nil
	s.editing = false
	if s.watch != nil {
		s.watch.Set("")
	}
	if s.line1 != nil {
		s.line1.Set(s.idleLine())
	}
	if s.line2 != nil {
		s.line2.Set("")
	}
	s.noteMCP()
	if s.words != nil {
		s.words.Set("")
	}
	if s.page != nil {
		s.page.SetBlocks(nil)
	}
	s.syncAction()
}

func (s *shell) idleLine() string {
	if s.lib == nil || len(s.lib.Paths) == 0 {
		return "Open a project folder"
	}
	return "Select a document"
}

func (s *shell) showMessage(msg string) {
	if s.line2 != nil {
		s.line2.Set("")
	}
	if s.line3 != nil {
		s.line3.Set(msg)
	}
	if s.words != nil {
		s.words.Set("")
	}
	if s.chars != nil {
		s.chars.Set("")
	}
	if s.caret != nil {
		s.caret.Set("")
	}
	s.writeFooter()
	if s.page != nil {
		s.page.SetBlocks(nil)
	}
}

func (s *shell) onItem(it ui.Item) {
	if s.histMode {
		s.onHistory(it)
		return
	}
	if s.reportMode {
		s.onReport(it)
		return
	}
	if it.More || it.Path == "" {
		return
	}
	if it.Dir {
		if it.Path != "" && it.Path == it.ProjectRoot {
			s.chosen = it.Path
		}
		s.toggle(it)
		return
	}
	s.openReplacing(it)
}

// openReplacing shows the file in the focused tab.
// A file that is already open is shown. A dirty tab is left in place and the file opens beside it.
func (s *shell) openReplacing(it ui.Item) {
	if s.awaitDocument(it, s.openReplacing) {
		return
	}
	s.chosen = it.ProjectRoot
	if s.side != nil {
		s.side.Select(it.Path)
	}
	s.replaced = replacedDoc{path: it.Path, index: -1, at: time.Now()}
	if i := s.indexOf(it.Path); i >= 0 {
		s.show(i)
		s.rememberPath(it.Path)
		return
	}
	i := s.focusedIndex()
	if i < 0 {
		s.open(it)
		return
	}
	s.remember()
	if s.docs[i].buf != nil && s.docs[i].buf.Dirty() {
		s.open(it)
		return
	}
	prev := s.docs[i]
	src, err := s.readDocument(it.Path)
	if err != nil {
		if s.line3 != nil {
			s.line3.Set(err.Error())
			s.writeFooter()
		}
		return
	}
	s.docs[i] = openDoc{
		path:    it.Path,
		rel:     it.Rel,
		project: it.ProjectName,
		buf:     edit.New(string(src)),
	}
	s.replaced = replacedDoc{path: it.Path, index: i, doc: prev, at: time.Now()}
	s.present(it)
}

// openTab opens the file in its own tab. A double click restores the document
// the first click replaced, then opens this one beside it.
func (s *shell) openTab(it ui.Item) {
	if s.histMode {
		s.onHistory(it)
		return
	}
	s.undoReplace(it)
	s.open(it)
}

func (s *shell) undoReplace(it ui.Item) {
	prev := s.replaced
	s.replaced = replacedDoc{}
	if prev.path != it.Path || prev.index < 0 || prev.doc.path == "" {
		return
	}
	if time.Since(prev.at) > 500*time.Millisecond {
		return
	}
	if prev.index >= len(s.docs) || s.docs[prev.index].path != it.Path {
		return
	}
	s.docs[prev.index] = prev.doc
	s.loadPane(0)
	if s.splitAt >= 0 {
		s.loadPane(1)
	}
	s.applyFocus()
}

func (s *shell) present(it ui.Item) {
	s.loadPane(0)
	if s.splitAt >= 0 {
		s.loadPane(1)
	}
	s.applyFocus()
	s.focusDoc()
	s.noteHist(it)
	s.rememberPath(it.Path)
	if s.lib != nil && s.lib.OpenRaw && doc.IsMarkdown(it.Path) {
		s.showRaw(true)
	}
	s.saveSession()
	s.redraw()
}

func (s *shell) toggle(it ui.Item) {
	if s.expanded[it.Path] {
		s.expanded[it.Path] = false
		s.refresh()
		s.syncWatch()
		return
	}
	s.expanded[it.Path] = true
	if _, ok := s.cache[it.Path]; !ok {
		entries, err := doc.ReadDir(it.Path, it.ProjectRoot, s.listOf(it.ProjectRoot))
		if err != nil {
			s.showMessage(err.Error())
			s.expanded[it.Path] = false
			return
		}
		s.cache[it.Path] = entries
	}
	s.refresh()
	s.saveSession()
	s.syncWatch()
}

func (s *shell) syncWatch() {
	if s.watch == nil {
		return
	}
	var paths []string
	for path, on := range s.expanded {
		if on && path != "" {
			paths = append(paths, path)
		}
	}
	s.watch.Tree(paths)
}

func (s *shell) open(it ui.Item) {
	if s.awaitDocument(it, s.open) {
		return
	}
	s.chosen = it.ProjectRoot
	if s.side != nil {
		s.side.Select(it.Path)
	}
	if i := s.indexOf(it.Path); i >= 0 {
		s.show(i)
		s.rememberPath(it.Path)
		return
	}
	src, err := s.readDocument(it.Path)
	if err != nil {
		if s.buf != nil {
			s.fileError(err)
			return
		}
		s.docPath = ""
		s.editing = false
		s.pending = ""
		s.ignored = ""
		if s.watch != nil {
			s.watch.Set("")
		}
		if s.frame != nil {
			s.frame.ShowB(false)
		}
		if s.line1 != nil {
			s.line1.Set(it.ProjectName)
		}
		if s.line2 != nil {
			s.line2.Set(doc.ShortPath(it.Rel))
		}
		if s.line3 != nil {
			s.line3.Set("Formatted")
		}
		if s.page != nil {
			s.page.SetBlocks([]doc.Block{{Kind: "p", Text: err.Error()}})
		}
		s.syncAction()
		return
	}
	s.remember()
	s.docs = append(s.docs, openDoc{
		path:    it.Path,
		rel:     it.Rel,
		project: it.ProjectName,
		buf:     edit.New(string(src)),
	})
	idx := len(s.docs) - 1
	if s.focus == 1 && s.splitAt >= 0 {
		s.splitAt = idx
	} else {
		s.focus = 0
		s.active = idx
	}
	s.loadPane(0)
	if s.splitAt >= 0 {
		s.loadPane(1)
	}
	s.applyFocus()
	s.focusDoc()
	s.noteHist(it)
	s.rememberPath(it.Path)
	if s.lib != nil && s.lib.OpenRaw && doc.IsMarkdown(it.Path) {
		s.showRaw(true)
	}
	s.saveSession()
	s.redraw()
}

func (s *shell) redraw() {
	if s.wake != nil {
		s.wake()
	}
}

func (s *shell) noteEdit() {
	s.focusEditor()
	if ed := s.currentEditor(); ed != nil && ed.Buffer() != nil {
		s.bindBuf(ed.Buffer())
	}
	if s.buf != nil {
		s.line3.Set(s.modeLine())
		s.editorStats()
	}
	s.syncAction()
	s.syncCommand()
}

func (s *shell) saveDoc() {
	s.focusEditor()
	if s.docPath == "" || s.buf == nil {
		return
	}
	text := s.buf.Text()
	if s.remote != nil {
		s.remote.save(s.docPath, s.buf)
		return
	}
	if err := edit.WriteFile(s.docPath, text); err != nil {
		s.showMessage(err.Error())
		return
	}
	s.documentSaved(s.docPath, s.buf, text)
}

// documentSaved updates the buffer which was written, even if another tab is
// now focused or the writer typed during a server save.
func (s *shell) documentSaved(path string, b *edit.Buffer, text string) {
	b.MarkSavedText(text)
	title := doc.Title([]byte(text))
	if !doc.IsMarkdown(path) || title == "" {
		title = filepath.Base(path)
	}
	s.retitle(path, title)
	if s.buf != b {
		s.syncTabs()
		s.refresh()
		return
	}
	s.pending = ""
	s.ignored = ""
	if s.idx != nil && doc.IsMarkdown(s.docPath) {
		s.idx.Put(s.docPath, s.projectRoot(), []byte(text))
	}
	doc.ForgetWiki(s.projectRoot())
	text = b.Text()
	blocks := doc.Render(path, []byte(text))
	if page := s.currentPage(); page != nil {
		reopen := page.OpenLine()
		page.SetPath(s.docPath)
		page.SetSource(filepath.Dir(s.docPath), blocks)
		if reopen > 0 {
			page.Reopen(reopen)
		}
	}
	s.line1.Set(fileTitle(s.docPath))
	s.stats(text)
	s.line3.Set(s.modeLine())
	s.syncAction()
	s.syncCommand()
	s.refresh()
	s.fillRail()
}

func (s *shell) discard() {
	s.focusEditor()
	if s.docPath == "" {
		return
	}
	var src []byte
	var err error
	if s.remote != nil && s.buf != nil {
		src = []byte(s.buf.Saved())
	} else {
		src, err = os.ReadFile(s.docPath)
	}
	if err != nil {
		s.showMessage(err.Error())
		return
	}
	b := edit.New(string(src))
	s.pending = ""
	s.ignored = ""
	s.editing = false
	s.bindBuf(b)
	if page := s.currentPage(); page != nil {
		page.UseBuffer(b)
	}
	if ed := s.currentEditor(); ed != nil {
		ed.UseBuffer(b)
		ed.SetFocused(false)
	}
	if fr := s.currentFrame(); fr != nil {
		fr.ShowB(false)
	}
	s.publish(string(src), "Markdown")
	s.syncCommand()
}

func (s *shell) syncAction() {
	s.syncNumberButton()
	if s.act == nil {
		return
	}
	if s.diff != nil {
		s.act.SetChoices(nil)
		s.act.SetLabel("Close diff")
		if s.mode != nil {
			s.mode.SetShown(false)
		}
		s.writeFooter()
		s.syncTabs()
		return
	}
	if s.pending != "" {
		s.act.SetChoices([]string{"Ours", "Theirs", "Merge"})
		if s.mode != nil {
			s.mode.SetShown(false)
		}
		s.syncTabs()
		return
	}
	s.act.SetChoices(nil)
	if s.docPath == "" || s.buf == nil {
		s.act.SetLabel("")
		if s.mode != nil {
			s.mode.SetShown(false)
		}
		s.writeFooter()
		s.syncTabs()
		return
	}
	if s.mode != nil {
		s.mode.SetShown(true)
		if s.editing {
			s.mode.SetActive("Raw")
		} else {
			s.mode.SetActive("Formatted")
		}
	}
	if s.buf.Dirty() {
		s.act.SetLabel("Save")
	} else {
		s.act.SetLabel("")
	}
	s.writeFooter()
	s.syncTabs()
}

func (s *shell) tick(ctx widget.Context) {
	if msg, changed := report.Transfer(); changed {
		s.transfer = msg
		s.writeFooter()
		if ctx != nil {
			ctx.Invalidate()
		}
	}
	if s.armFocus {
		s.armFocus = false
		if s.console != nil && s.console.Open() && s.console.IsFocused() {
			ctx.RequestFocus(s.console)
		} else if s.diff != nil {
			ctx.RequestFocus(s.diff.scroll)
		} else if s.editing {
			if ed := s.currentEditor(); ed != nil {
				ctx.RequestFocus(ed)
			}
		} else if p := s.currentPage(); p != nil && p.IsFocusable() {
			ctx.RequestFocus(p)
		}
	}
	edited := s.receiveDiff()
	if s.drainAgent() {
		edited = true
	}
	if s.watch != nil {
		if text, ok := s.watch.Take(); ok && s.onDisk(text) {
			edited = true
		}
		if dirs := s.watch.TakeDirs(); len(dirs) > 0 {
			for _, dir := range dirs {
				delete(s.cache, dir)
			}
			s.refresh()
			edited = true
		}
	}
	if s.mcpAsk.Swap(false) && s.mcpHelp != nil {
		s.mcpHelp.SetTheme(s.look)
		s.mcpHelp.Show(s.mcpURL, s.mcpToken, s.mcpErr)
		edited = true
	}
	if s.renameAsk.Swap(false) && s.rename != nil {
		name := ""
		if r := s.bookReport(s.renameID); r != nil {
			name = r.Name
		}
		s.rename.SetTheme(s.look)
		s.rename.Show(name)
		if ctx != nil {
			ctx.RequestFocus(s.rename.Field())
		}
		edited = true
	}
	if s.console != nil && s.console.Pump() {
		edited = true
	}
	if edited {
		ctx.Invalidate()
	}
}

func (s *shell) onDisk(text string) bool {
	if s.docPath == "" || s.buf == nil {
		return false
	}
	if text == s.buf.Text() {
		s.pending = ""
		s.ignored = ""
		if s.buf.Dirty() {
			s.buf.MarkSaved()
		}
		s.syncAction()
		s.publish(text, s.modeLine())
		return true
	}
	if !s.buf.Dirty() {
		s.adopt(text)
		return true
	}
	if text == s.buf.Saved() || text == s.ignored {
		return false
	}
	s.pending = text
	s.line3.Set("Changed on disk")
	s.syncAction()
	return true
}

func (s *shell) parkCaret(page *ui.Page, line int) {
	if page == nil || s.buf == nil {
		return
	}
	n := len(s.buf.Lines())
	if n < 1 {
		n = 1
	}
	if line < 1 || line > n {
		line = s.buf.CaretLine() + 1
	}
	if line < 1 {
		line = 1
	}
	if line > n {
		line = n
	}
	s.buf.Place(line-1, s.buf.CaretCol())
	page.Reopen(line)
	page.SetFocused(true)
}

func (s *shell) showRaw(on bool) {
	if s.docPath == "" || s.buf == nil {
		return
	}
	page, ed := s.currentPage(), s.currentEditor()
	if page != nil && ed != nil && (on != s.editing || page.Buffer() != s.buf) {
		if on {
			anchor := page.Passage()
			ed.UseBuffer(s.buf)
			ed.RestorePassage(anchor)
		} else {
			anchor := ed.Passage()
			page.UseBuffer(s.buf)
			page.SetPath(s.docPath)
			page.SetSource(filepath.Dir(s.docPath), doc.Render(s.docPath, []byte(s.buf.Text())))
			page.ResumeFormatted(anchor)
		}
	}
	s.editing = on
	if ed != nil {
		if on && ed.Buffer() != s.buf {
			ed.UseBuffer(s.buf)
		}
		ed.SetFile(s.docPath)
		ed.SetFocused(on)
	}
	if fr := s.currentFrame(); fr != nil {
		fr.ShowB(on)
	}
	if s.line3 != nil {
		s.line3.Set(s.modeLine())
	}
	s.stats(s.buf.Text())
	s.syncAction()
	s.focusDoc()
	s.saveSession()
}

func (s *shell) focusDoc() {
	if s.console != nil {
		s.console.SetFocused(false)
	}
	s.armFocus = true
}

func (s *shell) pick(label string) {
	switch label {
	case "Close diff":
		s.closeDiff()
		s.syncAction()
		s.syncCommand()
		s.focusDoc()
	case "Formatted":
		s.showRaw(false)
	case "Raw":
		s.showRaw(true)
	case "Save":
		s.saveDoc()
	case "Ours":
		if s.pending == "" {
			return
		}
		s.ignored = s.pending
		s.pending = ""
		s.line3.Set("Unsaved")
		s.syncAction()
	case "Theirs":
		if s.pending == "" {
			return
		}
		s.adopt(s.pending)
	case "Merge":
		if s.pending == "" || s.buf == nil {
			return
		}
		s.mergeDisk(s.pending)
	}
}

func (s *shell) adopt(text string) {
	b := edit.New(text)
	s.pending = ""
	s.ignored = ""
	s.bindBuf(b)
	if ed := s.currentEditor(); ed != nil {
		ed.UseBuffer(b)
	}
	s.line3.Set(s.modeLine())
	s.publish(text, s.line3.Get())
}

func (s *shell) mergeDisk(disk string) {
	merged, n := edit.Merge(s.buf.Saved(), s.buf.Text(), disk)
	b := edit.New(merged)
	b.SetSaved(disk)
	s.pending = ""
	s.ignored = ""
	s.bindBuf(b)
	if ed := s.currentEditor(); ed != nil {
		ed.UseBuffer(b)
	}
	status := "Markdown"
	if n > 0 {
		s.editing = true
		if ed := s.currentEditor(); ed != nil {
			ed.SetFocused(true)
		}
		if fr := s.currentFrame(); fr != nil {
			fr.ShowB(true)
		}
		status = conflictLine(n)
	} else if b.Dirty() {
		status = "Unsaved"
	}
	s.publish(merged, status)
}

func (s *shell) publish(text, status string) {
	blocks := doc.Render(s.docPath, []byte(text))
	if page := s.currentPage(); page != nil {
		line := page.OpenLine()
		page.SetPath(s.docPath)
		page.SetSource(filepath.Dir(s.docPath), blocks)
		s.parkCaret(page, line)
	}
	s.stats(text)
	title := ""
	if doc.IsMarkdown(s.docPath) {
		title = doc.Title([]byte(text))
	}
	if title == "" {
		title = filepath.Base(s.docPath)
	}
	s.retitle(s.docPath, title)
	if s.docPath != "" {
		s.line1.Set(fileTitle(s.docPath))
	}
	s.line2.Set(doc.ShortPath(s.docRel))
	s.line3.Set(status)
	s.syncAction()
	s.refresh()
	s.fillRail()
}

func conflictLine(n int) string {
	if n == 1 {
		return "1 conflict"
	}
	return strconv.Itoa(n) + " conflicts"
}

func (s *shell) retitle(path, title string) {
	for dir, entries := range s.cache {
		for i := range entries {
			if entries[i].Abs == path {
				entries[i].Title = title
				s.cache[dir] = entries
				return
			}
		}
	}
}

func (s *shell) items() []ui.Item {
	if len(s.projects) == 0 {
		return []ui.Item{{Label: "Open a project folder"}}
	}
	if s.reportMode {
		return s.reportItems()
	}
	if s.histMode {
		return s.historyItems()
	}
	q := strings.ToLower(strings.TrimSpace(s.filter))
	var out []ui.Item
	for _, p := range s.projects {
		label := p.Name
		if p.Absent {
			label += "  ·  absent"
		}
		if q != "" && !strings.Contains(strings.ToLower(label), q) && !s.branchHit(p.Root, q) {
			continue
		}
		out = append(out, ui.Item{
			Label:       label,
			Path:        p.Root,
			ProjectRoot: p.Root,
			ProjectName: p.Name,
			Dir:         !p.Absent,
			Expanded:    s.expanded[p.Root],
		})
		if p.Absent || !s.expanded[p.Root] {
			continue
		}
		out = append(out, s.walk(p.Root, p, 1, q)...)
	}
	disambiguateDirs(out)
	return out
}

// disambiguateDirs adds the parent folder in parentheses when two visible
// directories share a name, the way Zettlr labels a second docs folder.
func disambiguateDirs(items []ui.Item) {
	counts := map[string]int{}
	for _, it := range items {
		if !it.Dir || it.Path == "" {
			continue
		}
		counts[strings.ToLower(filepath.Base(it.Path))]++
	}
	for i, it := range items {
		if !it.Dir || it.Path == "" {
			continue
		}
		base := filepath.Base(it.Path)
		if counts[strings.ToLower(base)] < 2 {
			continue
		}
		parent := filepath.Base(filepath.Dir(it.Path))
		if parent == "" || parent == "." || parent == string(filepath.Separator) {
			continue
		}
		label := base + " (" + parent + ")"
		if strings.HasSuffix(it.Label, "  ·  absent") {
			label += "  ·  absent"
		}
		items[i].Label = label
	}
}

func (s *shell) walk(dir string, p doc.Project, depth int, q string) []ui.Item {
	entries := s.ensureListed(dir, p.Root)
	var out []ui.Item
	shown := 0
	for _, e := range entries {
		if e.More {
			out = append(out, ui.Item{Depth: depth, Label: e.Title, More: true})
			continue
		}
		if e.IsDir {
			if q != "" && !strings.Contains(strings.ToLower(e.Name), q) && !s.branchHit(e.Abs, q) {
				continue
			}
			out = append(out, ui.Item{
				Depth: depth, Label: e.Name, Path: e.Abs, Rel: e.Rel,
				ProjectRoot: p.Root, ProjectName: p.Name,
				Dir: true, Expanded: s.expanded[e.Abs],
			})
			shown++
			if s.expanded[e.Abs] {
				out = append(out, s.walk(e.Abs, p, depth+1, q)...)
			}
			continue
		}
		if strings.HasPrefix(q, "#") {
			if !s.tagged(e.Abs, strings.TrimPrefix(q, "#")) {
				continue
			}
		} else if q != "" && !strings.Contains(strings.ToLower(e.Title), q) && !strings.Contains(strings.ToLower(e.Name), q) && !strings.Contains(strings.ToLower(e.Rel), q) {
			continue
		}
		out = append(out, ui.Item{
			Depth: depth, Label: s.fileLabel(p.Root, e), Path: e.Abs, Rel: e.Rel,
			ProjectRoot: p.Root, ProjectName: p.Name,
		})
		shown++
	}
	if q == "" && shown == 0 {
		out = append(out, ui.Item{Depth: depth, Label: "No Markdown files"})
	}
	return out
}

func (s *shell) fileLabel(root string, e doc.DirEntry) string {
	if s.lib != nil && s.lib.View(root).Names {
		if e.Name != "" {
			return e.Name
		}
	}
	if e.Title != "" {
		return e.Title
	}
	return e.Name
}

func (s *shell) onDocs(docs bool) {
	if s.lib == nil {
		return
	}
	for _, root := range s.projectRoots() {
		v := s.lib.View(root)
		v.AllDirs = !docs
		s.lib.SetView(root, v)
		s.rereadProject(root)
	}
	s.save()
	s.refresh()
}

func (s *shell) onFileNames(names bool) {
	if s.lib == nil {
		return
	}
	for _, root := range s.projectRoots() {
		v := s.lib.View(root)
		v.Names = names
		s.lib.SetView(root, v)
	}
	s.save()
	s.refresh()
}

func (s *shell) projectRoots() []string {
	var roots []string
	seen := map[string]bool{}
	for _, p := range s.projects {
		if p.Root == "" || p.Absent || seen[p.Root] {
			continue
		}
		seen[p.Root] = true
		roots = append(roots, p.Root)
	}
	if len(roots) == 0 && s.lib != nil {
		for _, p := range s.lib.Paths {
			if p != "" && !seen[p] {
				seen[p] = true
				roots = append(roots, p)
			}
		}
	}
	return roots
}

func (s *shell) tagged(path, tag string) bool {
	tag = strings.TrimSpace(tag)
	if tag == "" {
		return false
	}
	if s.lib != nil && s.lib.HasTag(path, tag) {
		return true
	}
	if !doc.IsMarkdown(path) {
		return false
	}
	for _, have := range doc.TagsOf(path) {
		if strings.EqualFold(have, tag) {
			return true
		}
	}
	return false
}

func (s *shell) branchHit(dir, q string) bool {
	if strings.HasPrefix(q, "#") {
		tag := strings.TrimPrefix(q, "#")
		for _, e := range s.cache[dir] {
			if !e.IsDir && s.tagged(e.Abs, tag) {
				return true
			}
			if e.IsDir && s.branchHit(e.Abs, q) {
				return true
			}
		}
		return false
	}
	for _, e := range s.cache[dir] {
		if strings.Contains(strings.ToLower(e.Name), q) || strings.Contains(strings.ToLower(e.Title), q) || strings.Contains(strings.ToLower(e.Rel), q) {
			return true
		}
		if e.IsDir && s.expanded[e.Abs] && s.branchHit(e.Abs, q) {
			return true
		}
	}
	return false
}

func (s *shell) modeLine() string {
	if s.pending != "" {
		return "Changed on disk"
	}
	if s.buf != nil && s.buf.Dirty() {
		return "Unsaved"
	}
	if s.editing || s.pageOpen() {
		switch s.modeName() {
		case "Edit", "Visual", "Visual Line", "Command":
			return s.modeName()
		default:
			return "View"
		}
	}
	if s.docPath == "" {
		return ""
	}
	return "Formatted"
}

// editorStats refreshes document-wide work only when the text or pane changes.
func (s *shell) editorStats() bool {
	cache := &s.lastEditorStats
	text := s.buf.Text()
	if cache.buffer == s.buf && cache.path == s.docPath && cache.text == text && cache.pane == s.focus && cache.raw == s.editing {
		s.setCaretStatus()
		return false
	}
	blocks := doc.Render(s.docPath, []byte(text))
	s.setStats(text, blocks)
	cache.buffer, cache.text, cache.path = s.buf, text, s.docPath
	cache.pane, cache.raw = s.focus, s.editing
	if page := s.currentPage(); s.editing && page != nil && s.rail != nil && doc.IsMarkdown(s.docPath) {
		page.SetBlocks(blocks)
		s.rail.SetOutline(page.Outline())
	}
	return true
}

func (s *shell) stats(text string) {
	s.setStats(text, doc.Render(s.docPath, []byte(text)))
}

func (s *shell) setStats(text string, blocks []doc.Block) {
	s.lastEditorStats.buffer = nil
	n := doc.CountWords(blocks)
	if s.words != nil {
		s.words.Set(comma(n) + " words")
	}
	if s.chars != nil {
		s.chars.Set(comma(len([]rune(text))) + " characters")
	}
	s.setCaretStatus()
}

func (s *shell) setCaretStatus() {
	if s.caret == nil {
		return
	}
	if s.buf != nil && (s.editing || s.pageOpen()) {
		s.caret.Set(strconv.Itoa(s.buf.CaretLine()+1) + ":" + strconv.Itoa(s.buf.CaretCol()))
	} else {
		s.caret.Set("")
	}
}

func (s *shell) writeFooter() {
	if s.foot == nil {
		return
	}
	if s.diff != nil {
		s.foot.Set("Git diff")
		return
	}
	var parts []string
	if s.line3 != nil && s.line3.Get() != "" {
		status := s.line3.Get()
		if status == "Unsaved" || status == "Changed on disk" {
			if mode := s.modeName(); mode != "" {
				parts = append(parts, mode)
			}
		}
		parts = append(parts, status)
	}
	if s.words != nil && s.words.Get() != "" {
		parts = append(parts, s.words.Get())
	}
	if s.chars != nil && s.chars.Get() != "" {
		parts = append(parts, s.chars.Get())
	}
	if s.caret != nil && s.caret.Get() != "" {
		parts = append(parts, s.caret.Get())
	}
	if s.transfer != "" {
		parts = append(parts, s.transfer)
	}
	s.noteBranch()
	if s.branch != "" {
		parts = append(parts, s.branch)
	}
	s.foot.Set(strings.Join(parts, "   ·   "))
}

func (s *shell) noteBranch() {
	root := s.projectRoot()
	if root == "" {
		root = s.histRoot
	}
	if root == s.branchFor {
		return
	}
	s.branchFor = root
	name, err := gitlog.Branch(root)
	if err != nil {
		name = ""
	}
	s.branch = name
}

func (s *shell) noteHist(it ui.Item) {
	if s.nav || it.Path == "" {
		return
	}
	if s.histAt >= 0 && s.histAt < len(s.hist) && s.hist[s.histAt].Path == it.Path {
		return
	}
	cut := 0
	if s.histAt >= 0 {
		cut = s.histAt + 1
	}
	if cut > len(s.hist) {
		cut = len(s.hist)
	}
	s.hist = append(append([]ui.Item{}, s.hist[:cut]...), it)
	s.histAt = len(s.hist) - 1
}

func (s *shell) back() {
	if s.histAt <= 0 {
		return
	}
	s.histAt--
	s.nav = true
	s.open(s.hist[s.histAt])
	s.nav = false
}

func (s *shell) forward() {
	if s.histAt < 0 || s.histAt+1 >= len(s.hist) {
		return
	}
	s.histAt++
	s.nav = true
	s.open(s.hist[s.histAt])
	s.nav = false
}

func (s *shell) findInDoc() {
	q := strings.TrimSpace(s.filter)
	if q == "" || strings.HasPrefix(q, "#") {
		if s.line3 != nil {
			s.line3.Set("Type a search")
		}
		s.writeFooter()
		return
	}
	if s.buf == nil {
		return
	}
	line := edit.FindLine(s.buf.Text(), q, s.buf.CaretLine()+1)
	if line == 0 {
		if s.line3 != nil {
			s.line3.Set("No match")
		}
		s.writeFooter()
		return
	}
	s.buf.Place(line-1, 0)
	if p := s.currentPage(); p != nil {
		p.Reopen(line)
		p.SetFocused(true)
	}
	s.jump(line)
	if s.line3 != nil {
		s.line3.Set(q)
	}
	s.writeFooter()
	s.syncAction()
}

func (s *shell) newDoc() {
	s.newDocIn(s.projectRoot())
}

func (s *shell) newDocIn(dir string) {
	if dir == "" {
		if s.line3 != nil {
			s.line3.Set("Open a project")
		}
		s.writeFooter()
		return
	}
	name := "untitled.md"
	for n := 2; ; n++ {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			break
		}
		name = "untitled-" + strconv.Itoa(n) + ".md"
	}
	path := filepath.Join(dir, name)
	if err := createNote(path); err != nil {
		s.showMessage(err.Error())
		return
	}
	delete(s.cache, dir)
	s.rescan()
	s.refresh()
	rel, _ := filepath.Rel(dir, path)
	s.open(ui.Item{Path: path, Rel: rel, ProjectRoot: dir, ProjectName: filepath.Base(dir)})
}

func (s *shell) tagCurrent() {
	if s.lib == nil {
		return
	}
	tag := strings.TrimSpace(strings.TrimPrefix(s.filter, "#"))
	if tag == "" {
		if s.line3 != nil {
			s.line3.Set("Type a tag in Search")
		}
		s.writeFooter()
		return
	}
	if s.docPath == "" {
		if s.line3 != nil {
			s.line3.Set("Open a document")
		}
		s.writeFooter()
		return
	}
	s.lib.AddTag(s.docPath, tag)
	s.save()
	if s.line3 != nil {
		s.line3.Set("Tagged " + tag)
	}
	s.writeFooter()
	s.refresh()
}

func (s *shell) toggleOpenRaw() {
	if s.lib == nil {
		return
	}
	s.lib.OpenRaw = !s.lib.OpenRaw
	s.save()
	msg := "Opens in Formatted"
	if s.lib.OpenRaw {
		msg = "Opens in Raw"
	}
	if s.line3 != nil {
		s.line3.Set(msg)
	}
	s.writeFooter()
}

func (s *shell) insertMarkup(kind string) {
	if s.buf == nil {
		return
	}
	text := markup(kind)
	if text == "" {
		return
	}
	s.buf.InsertText(text)
	if p := s.currentPage(); p != nil {
		p.SetBlocks(doc.Render(s.docPath, []byte(s.buf.Text())))
		p.Reopen(s.buf.CaretLine() + 1)
	}
	s.syncAction()
	if s.line3 != nil {
		s.line3.Set(kind)
	}
	s.writeFooter()
}

func markup(kind string) string {
	switch kind {
	case "Code":
		return "\n```\n\n```\n"
	case "Link":
		return "[text](url)"
	case "Image":
		return "![](image)"
	case "Table":
		return "\n| Column | Column |\n| --- | --- |\n|  |  |\n"
	case "Footnote":
		return "[^1]\n\n[^1]: note\n"
	default:
		return ""
	}
}

func (s *shell) projectRoot() string {
	if s.docPath != "" {
		for _, p := range s.libPaths() {
			if s.docPath == p || strings.HasPrefix(s.docPath, p+string(os.PathSeparator)) {
				return p
			}
		}
		return filepath.Dir(s.docPath)
	}
	if s.chosen != "" {
		return s.chosen
	}
	paths := s.libPaths()
	if len(paths) > 0 {
		return paths[0]
	}
	return ""
}

func (s *shell) libPaths() []string {
	if s.lib == nil {
		return nil
	}
	return s.lib.Paths
}

func (s *shell) exportReport() {
	if s.docPath == "" || s.buf == nil {
		if s.line3 != nil {
			s.line3.Set("Open a document")
		}
		s.writeFooter()
		return
	}
	root := s.projectRoot()
	if root == "" {
		return
	}
	prof := report.LoadProfile(root)
	title := prof.Title
	if title == "" {
		title = doc.Title([]byte(s.buf.Text()))
	}
	date := prof.Date
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	res, err := report.CompileJob(report.Job{
		Root: root, Src: []byte(s.buf.Text()),
		Title: title, Author: prof.Author, Date: date, Template: prof.Template,
	})
	if err != nil {
		if s.line3 != nil {
			s.line3.Set(err.Error())
		}
		s.writeFooter()
		return
	}
	s.showReport(res)
}

func (s *shell) showReport(res report.Result) {
	if s.editing {
		s.showRaw(false)
	}
	var blocks []doc.Block
	for _, page := range res.Pages {
		blocks = append(blocks, doc.Block{
			Kind:  "p",
			Spans: []doc.Span{{Image: true, Link: page}},
		})
	}
	if page := s.currentPage(); page != nil && len(blocks) > 0 {
		page.UseBuffer(nil)
		page.SetSource(filepath.Dir(res.Pages[0]), blocks)
		if s.mode != nil {
			s.mode.SetActive("")
		}
	}
	msg := res.Template + " · out/report.pdf"
	if res.Note != "" {
		msg = res.Note
	}
	if len(blocks) > 0 {
		msg = "Preview · " + msg
	}
	if s.line3 != nil {
		s.line3.Set(msg)
	}
	s.writeFooter()
}

func (s *shell) zoom(dir int) {
	if s.base.Body < 1 {
		s.base = theme.WithPairtiType(s.look)
	}
	i := s.zoomIndex + dir
	if i < 0 {
		i = 0
	}
	if i >= theme.ZoomCount() {
		i = theme.ZoomCount() - 1
	}
	s.zoomIndex = i
	k, fit := theme.ZoomScale(i)
	if fit {
		k = s.fitScale()
	}
	s.look = s.base.Scale(k)
	s.applyType()
	label := theme.ZoomLabel(i)
	if s.zoomCtl != nil {
		s.zoomCtl.SetLabel(label)
	}
	if s.themePath != "" {
		if err := theme.WriteZoom(s.themePath, label); err != nil {
			if s.line3 != nil {
				s.line3.Set(err.Error())
				s.writeFooter()
			}
			return
		}
	}
	if s.line3 != nil {
		s.line3.Set(label)
		s.writeFooter()
	}
}

func (s *shell) fitScale() float32 {
	w := float32(720)
	if s.page != nil && s.page.Bounds().Width() > 280 {
		w = s.page.Bounds().Width() - 68
	}
	body := float32(int(w/36 + 0.5))
	if body < 12 {
		body = 12
	}
	if body > 32 {
		body = 32
	}
	base := s.base.Body
	if base < 1 {
		base = 16
	}
	return body / base
}

func (s *shell) applyType() {
	if s.diff != nil {
		s.diff.view.SetTheme(s.look)
	}
	if s.page != nil {
		s.page.SetTheme(s.look)
	}
	if s.pageB != nil {
		s.pageB.SetTheme(s.look)
	}
	if s.editor != nil {
		s.editor.SetTheme(s.look)
	}
	if s.editorB != nil {
		s.editorB.SetTheme(s.look)
	}
}

func (s *shell) applyLineNumbers() {
	for _, p := range []*ui.Page{s.page, s.pageB} {
		if p != nil {
			p.SetLineNumbers(s.formattedNumbers)
		}
	}
	for _, e := range []*ui.Editor{s.editor, s.editorB} {
		if e != nil {
			e.SetLineNumbers(!s.hideRawNumbers)
		}
	}
	s.syncNumberButton()
}

func (s *shell) syncNumberButton() {
	if s.tools == nil {
		return
	}
	active := ""
	if s.showingLineNumbers() {
		active = "Line numbers"
	}
	s.tools.SetActive(active)
}

func (s *shell) toolAvailable(name string) bool {
	if s.remote != nil {
		switch name {
		case "Shell", "New", "Open", "Remove", "Export", "Tag":
			return false
		}
	}
	switch name {
	case "Back":
		return s.histAt > 0
	case "Forward":
		return s.histAt >= 0 && s.histAt+1 < len(s.hist)
	case "Split":
		return len(s.docs) > 0
	case "Line numbers":
		return s.diff == nil && s.buf != nil && s.docPath != ""
	}
	if s.diff == nil {
		return true
	}
	switch name {
	case "Code", "Image", "Link", "Footnote", "Table", "Line numbers", "Tag", "Search", "Export", "Outline":
		return false
	}
	return true
}

func (s *shell) onTool(name string) {
	if !s.toolAvailable(name) {
		return
	}
	switch name {
	case "Files":
		s.toggleSide()
	case "Outline":
		s.toggleRail()
	case "Split":
		s.toggleSplit()
	case "Shell":
		s.toggleShell()
	case "Search":
		s.findInDoc()
	case "New":
		s.newDoc()
	case "Tag":
		s.tagCurrent()
	case "Settings":
		s.toggleOpenRaw()
	case "Line numbers":
		if s.buf == nil || s.docPath == "" {
			return
		}
		if s.editing {
			s.hideRawNumbers = !s.hideRawNumbers
		} else {
			s.formattedNumbers = !s.formattedNumbers
		}
		s.applyLineNumbers()
		s.saveSession()
	case "Back":
		s.back()
	case "Forward":
		s.forward()
	case "Code", "Link", "Image", "Table", "Footnote":
		s.insertMarkup(name)
	case "Export":
		s.openExport()
	case "Open":
		dir, err := pickFolder()
		if err != nil {
			return
		}
		s.add(dir)
	case "Remove":
		s.removeChosen()
	default:
		s.line3.Set(name)
		s.syncAction()
	}
}

func (s *shell) toggleSide() {
	if s.split == nil {
		return
	}
	s.split.Toggle()
	s.saveSession()
}

func (s *shell) follow(raw string) {
	if name, ok := strings.CutPrefix(raw, "wiki:"); ok {
		s.followWiki(name)
		return
	}
	if line, ok := strings.CutPrefix(raw, "#line:"); ok {
		n, err := strconv.Atoi(line)
		if err == nil && n > 0 {
			s.jump(n)
		}
		return
	}
	if path, ok := doc.DocFile(s.docRoot(), raw); ok {
		if isFile(path) {
			root := s.docRoot()
			rel := filepath.Base(path)
			if r, err := filepath.Rel(root, path); err == nil {
				rel = filepath.ToSlash(r)
			}
			name := s.docProject
			if name == "" {
				name = filepath.Base(root)
			}
			s.open(ui.Item{Path: path, Rel: rel, ProjectRoot: root, ProjectName: name})
			return
		}
		if s.line3 != nil {
			s.line3.Set("Missing link")
			s.writeFooter()
		}
		s.syncAction()
		return
	}
	kind, target := resolveLink(s.docPath, raw)
	switch kind {
	case "web":
		openExternal(target)
	case "file":
		root := s.docRoot()
		rel, err := filepath.Rel(root, target)
		if err != nil || rel == "" {
			rel = filepath.Base(target)
		}
		name := s.docProject
		if name == "" {
			name = filepath.Base(root)
		}
		s.open(ui.Item{
			Path:        target,
			Rel:         filepath.ToSlash(rel),
			ProjectRoot: root,
			ProjectName: name,
		})
	case "missing":
		s.line3.Set("Missing link")
		s.syncAction()
	}
}

func (s *shell) followWiki(raw string) {
	heading := ""
	name := raw
	if i := strings.Index(raw, "#"); i >= 0 {
		heading = strings.TrimSpace(raw[i+1:])
		name = strings.TrimSpace(raw[:i])
	}
	root := s.docRoot()
	if root == "" || name == "" {
		s.missLink()
		return
	}
	path, ok := doc.WikiFile(root, s.docPath, name)
	if !ok {
		dir := filepath.Dir(s.docPath)
		if s.docPath == "" || dir == "" || dir == "." {
			dir = root
		}
		made, err := doc.CreateWiki(dir, name)
		if err != nil {
			s.missLink()
			return
		}
		path = made
		doc.ForgetWiki(root)
		delete(s.cache, dir)
	}
	rel := filepath.Base(path)
	if r, err := filepath.Rel(root, path); err == nil {
		rel = filepath.ToSlash(r)
	}
	project := s.docProject
	if project == "" {
		project = filepath.Base(root)
	}
	s.open(ui.Item{Path: path, Rel: rel, ProjectRoot: root, ProjectName: project})
	if heading == "" {
		return
	}
	page := s.currentPage()
	if page == nil {
		return
	}
	if line := page.HeadingLine(heading); line > 1 {
		page.Layout(nil, geometry.Constraints{MaxWidth: 880, MaxHeight: geometry.Infinity})
		s.jump(line)
	}
}

func (s *shell) missLink() {
	if s.line3 != nil {
		s.line3.Set("Missing link")
		s.writeFooter()
	}
	s.syncAction()
}

func (s *shell) docRoot() string {
	for _, p := range s.projects {
		if s.docPath == p.Root || strings.HasPrefix(s.docPath, p.Root+string(filepath.Separator)) {
			return p.Root
		}
	}
	if s.docPath == "" {
		return ""
	}
	return filepath.Dir(s.docPath)
}

// resolveLink classifies a markdown link against the open document.
// kind is "web", "file", or "missing".
func resolveLink(docPath, raw string) (kind, target string) {
	raw = strings.TrimSpace(raw)
	if raw == "" || strings.HasPrefix(raw, "#") {
		return "", ""
	}
	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "mailto:") {
		return "web", raw
	}
	u := raw
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	if u == "" {
		return "", ""
	}
	if dec, err := url.PathUnescape(u); err == nil {
		u = dec
	}
	u = filepath.FromSlash(u)
	var abs string
	if filepath.IsAbs(u) {
		abs = u
	} else if docPath != "" {
		abs = filepath.Clean(filepath.Join(filepath.Dir(docPath), u))
	} else {
		return "missing", ""
	}
	if isFile(abs) {
		return "file", abs
	}
	low := strings.ToLower(abs)
	if !strings.HasSuffix(low, ".md") && !strings.HasSuffix(low, ".markdown") && isFile(abs+".md") {
		return "file", abs + ".md"
	}
	return "missing", abs
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func openExternal(target string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		cmd = exec.Command("xdg-open", target)
	}
	_ = cmd.Start()
}

func fileTitle(path string) string {
	base := filepath.Base(path)
	r := []rune(base)
	if len(r) <= 42 {
		return base
	}
	return string(r[:20]) + "…" + string(r[len(r)-18:])
}

func comma(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		s = s[1:]
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	if n < 0 {
		return "-" + string(out)
	}
	return string(out)
}

func (s *shell) build() widget.Widget {
	field := ui.FieldPainter(s.look)
	filter := ui.NewNarrow(textfield.New(
		textfield.Placeholder("Search"),
		textfield.PainterOpt(field),
		textfield.OnChange(func(v string) {
			s.setFilter(v)
		}),
	))
	left := ui.NewSidePane(filter, s.actions, s.side)
	s.pane = left
	bar := ui.NewModeBar(s.setSide)
	if s.remote != nil {
		bar.SetAvailable(func(mode int) bool { return mode == ui.SideFiles })
	}
	bar.SetTheme(s.look)
	left.SetBar(bar)
	s.histScopeBar = ui.NewHistoryScopeBar(func(mode int) {
		s.histRepo = mode == 1
		s.loadHistory()
		s.refresh()
	})
	s.histScopeBar.SetTheme(s.look)
	left.SetHistoryBar(s.histScopeBar)
	plus := ui.NewReportBar(s.newReport, s.pickReportProject)
	plus.SetTheme(s.look)
	s.reports = plus
	left.SetReportBar(plus)
	left.SetTheme(s.look)

	tools := ui.NewToolBar(ui.DocumentTools(), s.onTool)
	tools.SetAvailable(s.toolAvailable)
	tools.SetTheme(s.look)
	s.tools = tools
	s.syncNumberButton()
	railBtn := ui.NewToolBar([]ui.Tool{ui.RailTool()}, s.onTool)
	railBtn.SetAvailable(s.toolAvailable)
	railBtn.SetTheme(s.look)
	exportBtn := ui.NewToolBar([]ui.Tool{ui.ExportTool()}, s.onTool)
	exportBtn.SetAvailable(s.toolAvailable)
	exportBtn.SetTheme(s.look)
	s.exportBtn = exportBtn
	s.zoomCtl = ui.NewZoom(s.zoom)
	s.zoomCtl.SetTheme(s.look)
	s.zoomCtl.SetLabel(theme.ZoomLabel(s.zoomIndex))
	if s.mode == nil {
		s.mode = ui.NewViewSegment(s.pick)
		s.mode.SetTheme(s.look)
	}
	if s.docPath == "" {
		s.mode.SetShown(false)
	}
	toolbar := primitives.Box(
		tools,
		primitives.Expanded(primitives.Box()),
		s.zoomCtl,
		s.mode,
		s.act,
		exportBtn,
		railBtn,
	).SetDirection(primitives.DirectionHorizontal).Padding(4).Gap(12).Background(widget.Hex(s.look.Surface))

	// The toolkit default step is 40. Trackpad deltas are already in points,
	// so 40 makes the page jump. 12 matches the file list and the editor.
	sideW := float32(280)
	if s.boot.Sidebar >= 120 && s.boot.Sidebar <= 2000 {
		sideW = s.boot.Sidebar
	}
	split := ui.NewSplit(left, s.documentColumn(), sideW)
	split.SetTheme(s.look)
	s.split = split

	// An empty string measures as no height, and the text signal only redraws.
	// Pin one meta line so the bar is right before a document is open and
	// stays that tall when the status arrives, without a window resize.
	const footPad = float32(8)
	line := s.look.Meta * 1.2
	foot := primitives.Text("").FontSize(s.look.Meta).LineHeight(1.2).MaxLines(1).Color(widget.Hex(s.look.Muted)).ContentSignal(s.foot)
	footer := primitives.Box(
		primitives.Expanded(foot),
	).SetDirection(primitives.DirectionHorizontal).Padding(footPad).Height(line + footPad*2).Background(widget.Hex(s.look.Surface))

	if s.sheet == nil {
		s.sheet = ui.NewExportSheet(s.commitExport)
	}
	s.sheet.SetTheme(s.look)
	if s.boot.HideSide && s.split != nil {
		s.split.SetShown(false)
	}
	if s.boot.Rail && s.railSplit != nil {
		s.railSplit.SetShown(true)
	}
	if s.splitAt >= 0 && s.pair != nil {
		s.pair.SetSplit(true)
	}
	cover := ui.NewCover(primitives.Box(
		toolbar,
		primitives.Expanded(split),
		footer,
	).Background(widget.Hex(s.look.Canvas)), s.sheet)
	if s.menu != nil {
		s.menu.SetTheme(s.look)
		cover.SetMenu(s.menu)
	}
	if s.tabMenu != nil {
		s.tabMenu.SetTheme(s.look)
		cover.SetTabMenu(s.tabMenu)
	}
	if s.palette != nil {
		s.palette.SetTheme(s.look)
		cover.SetPalette(s.palette)
	}
	s.mcpHelp = ui.NewMCPHelp()
	s.mcpHelp.SetTheme(s.look)
	cover.SetHelp(s.mcpHelp)
	s.rename = ui.NewRename(s.renameReport)
	s.rename.SetTheme(s.look)
	cover.SetRename(s.rename)
	cover.OnChord(s.chord)
	return cover
}

func (s *shell) chord(ke *event.KeyEvent) bool {
	if ke.Key != event.KeyK || ke.IsShift() || (!ke.IsSuper() && !ke.IsCtrl()) {
		return false
	}
	if s.palette == nil {
		return false
	}
	if s.palette.Shown() {
		s.palette.Hide()
	} else {
		s.palette.Show()
	}
	s.redraw()
	return true
}

func (s *shell) askIndex(q string) {
	if s.palette == nil || s.idx == nil {
		return
	}
	found := s.idx.Search(q, 8)
	rows := make([]ui.PaletteHit, len(found))
	for i, h := range found {
		rows[i] = ui.PaletteHit{
			Title: h.Title, Rel: h.Rel, Snippet: h.Snippet,
			Path: h.Path, Root: h.Root,
		}
	}
	s.palette.SetHits(rows)
}

func (s *shell) openHit(h ui.PaletteHit) {
	name := filepath.Base(h.Root)
	s.open(ui.Item{
		Path: h.Path, Rel: h.Rel, ProjectRoot: h.Root, ProjectName: name,
	})
}

func (s *shell) kickIndex() {
	if s.idx == nil {
		return
	}
	roots := append([]string{}, s.projectRoots()...)
	go func() {
		for _, root := range roots {
			s.idx.Sync(root)
		}
		s.idx.Keep(roots)
	}()
}

func pickFolder() (string, error) {
	return chooseFolder("Open a project folder")
}

func chooseFolder(prompt string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("use the path field")
	}
	prompt = strings.ReplaceAll(prompt, `"`, "")
	out, err := exec.Command("osascript", "-e", `POSIX path of (choose folder with prompt "`+prompt+`")`).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func (s *shell) showTreeMenu(it ui.Item, at geometry.Point) {
	if s.menu == nil || s.histMode {
		return
	}
	if it.Path == "" && !(s.reportMode && it.Dir && it.Rel != "" && it.Rel != "new") {
		return
	}
	reportDir := s.reportMode && it.Dir && it.Rel != "" && it.Rel != "new"
	reportFile := s.reportMode && !it.Dir && it.Note != ""
	s.menu.SetReport(!s.reportMode && !it.Dir && it.Path != "", reportFile, reportDir)
	if s.tabMenu != nil {
		s.tabMenu.Hide()
	}
	s.menu.SetTheme(s.look)
	s.menu.Show(it, at)
	s.redraw()
}

func (s *shell) showTabMenu(i int, at geometry.Point) {
	if s.tabMenu == nil || i < 0 || i >= len(s.docs) {
		return
	}
	if s.menu != nil {
		s.menu.Hide()
	}
	s.tabMenu.SetTheme(s.look)
	s.tabMenu.Show(fileTitle(s.docs[i].path), i, len(s.docs), at)
	s.redraw()
}

func (s *shell) onTabMenu(i int, action string) {
	switch action {
	case ui.TabClose:
		s.closeDoc(i)
	case ui.TabCloseOthers:
		s.closeWhere(func(j int) bool { return j != i })
	case ui.TabCloseRight:
		s.closeWhere(func(j int) bool { return j > i })
	case ui.TabCloseLeft:
		s.closeWhere(func(j int) bool { return j < i })
	case ui.TabCloseAll:
		s.closeWhere(func(int) bool { return true })
	default:
		return
	}
	s.redraw()
}

func (s *shell) onTreeMenu(it ui.Item, action string) {
	if s.remote != nil {
		s.showMessage("Project changes require the desktop app.")
		return
	}
	var err error
	switch action {
	case ui.MenuCopyPath:
		err = copyText(it.Path)
	case ui.MenuCopyRel:
		rel := it.Rel
		if rel == "" {
			rel = "."
		}
		err = copyText(rel)
	case ui.MenuReveal:
		err = revealPath(it.Path)
	case ui.MenuOpen:
		err = report.Open(it.Path)
	case ui.MenuTrash:
		err = moveToBin(it.Path)
		if err == nil {
			s.afterRemove(it)
		}
	case ui.MenuReportAdd:
		s.addToReport(it)
		return
	case ui.MenuReportDrop:
		s.book.RemoveFile(it.Note, it.Rel)
		s.saveReports()
		s.refresh()
		return
	case ui.MenuReportUp:
		s.book.Move(it.Note, it.Rel, -1)
		s.saveReports()
		s.refresh()
		return
	case ui.MenuReportDown:
		s.book.Move(it.Note, it.Rel, 1)
		s.saveReports()
		s.refresh()
		return
	case ui.MenuReportRename:
		s.renameID = it.Rel
		s.renameAsk.Store(true)
		if s.wake != nil {
			s.wake()
		}
		return
	case ui.MenuReportDelete:
		delete(s.reportOpen, it.Rel)
		s.book.Delete(it.Rel)
		s.saveReports()
		s.refresh()
		return
	case ui.MenuDelete:
		err = os.RemoveAll(it.Path)
		if err == nil {
			s.afterRemove(it)
		}
	}
	if err != nil {
		s.showMessage(err.Error())
		return
	}
	if action == ui.MenuCopyPath || action == ui.MenuCopyRel {
		s.showMessage("Copied")
	}
}

func (s *shell) onReport(it ui.Item) {
	s.useReports()
	if s.reportOpen == nil {
		s.reportOpen = map[string]bool{}
	}
	if it.Rel == "new" {
		s.newReport()
		return
	}
	if it.Dir {
		s.openReport(it.Rel)
		return
	}
	if it.Path == "" {
		return
	}
	s.openReplacing(it)
}

func (s *shell) reportItems() []ui.Item {
	s.useReports()
	if s.reportOpen == nil {
		s.reportOpen = map[string]bool{}
	}
	root := s.reportRoot
	name := filepath.Base(root)
	var out []ui.Item
	for _, r := range s.book.Reports {
		open := s.reportOpen[r.ID]
		out = append(out, ui.Item{
			Label: r.Name, Path: "report:" + r.ID, Rel: r.ID, Dir: true, Expanded: open,
			ProjectRoot: root, ProjectName: name,
		})
		if !open {
			continue
		}
		for _, rel := range r.Files {
			out = append(out, ui.Item{
				Depth: 1, Label: filepath.Base(rel), Note: r.ID, Rel: rel,
				Path:        filepath.Join(root, filepath.FromSlash(rel)),
				ProjectRoot: root, ProjectName: name,
			})
		}
	}
	return out
}

func (s *shell) bookReport(id string) *project.Report {
	for i := range s.book.Reports {
		if s.book.Reports[i].ID == id {
			return &s.book.Reports[i]
		}
	}
	return nil
}

func (s *shell) useReports() {
	root := s.currentReportRoot()
	if root == "" {
		return
	}
	if root == s.reportRoot && s.reportRoot != "" {
		return
	}
	book, err := project.LoadBook(root)
	if err != nil {
		log.Printf("reports: %s: %v", root, err)
		book = project.Book{}
	}
	s.reportRoot = root
	s.invalidateMenus()
	s.reportOpen = nil
	s.book = book
}

func (s *shell) currentReportRoot() string {
	if s.projectKnown(s.reportPick) {
		return s.reportPick
	}
	s.reportPick = ""
	if root := s.docRoot(); s.projectKnown(root) {
		s.reportPick = root
		return root
	}
	for _, p := range s.projects {
		if !p.Absent && p.Root != "" {
			s.reportPick = p.Root
			return p.Root
		}
	}
	return ""
}

func (s *shell) projectKnown(root string) bool {
	if root == "" {
		return false
	}
	for _, p := range s.projects {
		if p.Root == root && !p.Absent {
			return true
		}
	}
	return false
}

func (s *shell) pickReportProject(root string) {
	if !s.projectKnown(root) || root == s.reportPick {
		return
	}
	s.reportPick = root
	s.reportRoot = ""
	s.useReports()
	s.refresh()
}

func (s *shell) syncReportBar() {
	if s.reports == nil {
		return
	}
	var present []doc.Project
	count := map[string]int{}
	for _, p := range s.projects {
		if p.Absent || p.Root == "" {
			continue
		}
		present = append(present, p)
		count[strings.ToLower(p.Name)]++
	}
	choices := make([]ui.ReportChoice, 0, len(present))
	for _, p := range present {
		name := p.Name
		if count[strings.ToLower(name)] > 1 {
			parent := filepath.Base(filepath.Dir(p.Root))
			if parent != "" && parent != "." && parent != string(filepath.Separator) {
				name += " (" + parent + ")"
			}
		}
		choices = append(choices, ui.ReportChoice{Name: name, Root: p.Root})
	}
	s.reports.SetChoices(choices, s.reportPick)
}

func (s *shell) addToReport(it ui.Item) {
	if it.Dir || it.Path == "" {
		return
	}
	s.useReports()
	if s.reportRoot == "" {
		return
	}
	if s.book.Open == "" {
		made := s.book.Add()
		if s.reportOpen == nil {
			s.reportOpen = map[string]bool{}
		}
		s.reportOpen[made.ID] = true
	}
	if s.reportOpen == nil {
		s.reportOpen = map[string]bool{}
	}
	s.reportOpen[s.book.Open] = true
	rel := it.Rel
	if rel == "" {
		r, err := filepath.Rel(s.reportRoot, it.Path)
		if err != nil {
			return
		}
		rel = filepath.ToSlash(r)
	}
	s.book.AddFile(rel)
	s.saveReports()
	s.refresh()
}

func (s *shell) saveReports() {
	s.invalidateMenus()
	if s.reportRoot == "" {
		return
	}
	if err := project.SaveBook(s.reportRoot, s.book); err != nil {
		log.Printf("reports: %s: %v", s.reportRoot, err)
		if s.line3 != nil {
			s.line3.Set(err.Error())
			s.writeFooter()
		}
	}
}

func (s *shell) onChevron(it ui.Item) {
	if s.histMode && strings.HasPrefix(it.Path, "commit:") {
		s.toggleHist(it.Rel)
		return
	}
	if s.reportMode && it.Dir && strings.HasPrefix(it.Path, "report:") {
		s.toggleReportList(it.Rel)
		return
	}
	if it.Dir && it.Path != "" {
		s.toggle(it)
	}
}

func (s *shell) toggleReportList(id string) {
	if s.reportOpen == nil {
		s.reportOpen = map[string]bool{}
	}
	s.reportOpen[id] = !s.reportOpen[id]
	s.refresh()
}

func (s *shell) openReport(id string) {
	s.useReports()
	rep := s.bookReport(id)
	if rep == nil {
		return
	}
	s.remember()
	s.book.Open = id
	s.saveReports()
	src, err := project.Join(s.reportRoot, rep.Files)
	if err != nil {
		src = []byte("# " + rep.Name + "\n")
		if s.line3 != nil {
			s.line3.Set("Add a file to the report")
			s.writeFooter()
		}
	}
	s.viewing = id
	s.docPath = ""
	s.docRel = ""
	s.docProject = rep.Name
	s.editing = false
	s.pending = ""
	s.ignored = ""
	s.buf = edit.New(string(src))
	s.buf.MarkSaved()
	if s.watch != nil {
		s.watch.Set("")
	}
	if frame := s.currentFrame(); frame != nil {
		frame.ShowB(false)
	}
	blocks := doc.Render("", src)
	if page := s.currentPage(); page != nil {
		page.UseBuffer(nil) // Joined report pages have no writable Markdown file.
		page.SetPath("")
		page.SetSource(s.reportRoot, blocks)
	}
	if s.focus == 1 && s.scrollB != nil {
		s.scrollB.Set(0)
	} else if s.scroll != nil {
		s.scroll.Set(0)
	}
	if s.line1 != nil {
		s.line1.Set(rep.Name)
	}
	if s.line2 != nil {
		s.line2.Set("Report")
	}
	if s.line3 != nil && err == nil {
		s.line3.Set("Report")
		s.writeFooter()
	}
	s.stats(string(src))
	s.fillRail()
	if s.rail != nil {
		s.rail.ShowContents()
	}
	if s.side != nil {
		s.side.Select("report:" + id)
	}
	s.syncAction()
	s.refresh()
}

func (s *shell) newReport() {
	s.useReports()
	if s.reportRoot == "" {
		return
	}
	if s.reportOpen == nil {
		s.reportOpen = map[string]bool{}
	}
	made := s.book.Add()
	s.reportOpen[made.ID] = true
	s.saveReports()
	s.refresh()
}

func (s *shell) setSide(mode int) {
	if s.remote != nil && mode != ui.SideFiles {
		s.showMessage("This library serves Markdown files. Reports and Git history require server adapters.")
		return
	}
	wasDiff := s.diff != nil
	if mode != ui.SideHistory {
		s.closeDiff()
	}
	s.reportMode = mode == ui.SideReports
	s.histMode = mode == ui.SideHistory
	if s.pane != nil {
		s.pane.SetMode(mode)
	}
	if s.histMode {
		s.loadHistory()
	} else if wasDiff {
		s.syncAction()
		s.syncCommand()
		s.focusDoc()
	} else {
		s.restoreDoc()
	}
	s.refresh()
}

func (s *shell) restoreDoc() {
	if s.viewing != "" {
		s.applyFocus()
	}
	if s.focus == 1 {
		s.loadPane(1)
	} else {
		s.loadPane(0)
	}
	if s.line3 != nil {
		s.line3.Set(s.modeLine())
		s.writeFooter()
	}
}

func (s *shell) afterRemove(it ui.Item) {
	s.dropDocs(it.Path)
	delete(s.cache, it.Path)
	delete(s.expanded, it.Path)
	if it.Path == it.ProjectRoot {
		for i := range s.projects {
			if s.projects[i].Root == it.Path {
				s.projects[i].Absent = true
			}
		}
	} else if parent := filepath.Dir(it.Path); parent != "" && it.ProjectRoot != "" {
		if entries, err := doc.ReadDir(parent, it.ProjectRoot, s.listOf(it.ProjectRoot)); err == nil {
			s.cache[parent] = entries
		}
	}
	s.refresh()
	s.saveSession()
	if s.idx != nil {
		s.idx.Drop(it.Path)
	}
}

func (s *shell) dropDocs(path string) {
	if s.buf != nil && underPath(s.docPath, path) {
		s.buf.MarkSaved()
	}
	for i := len(s.docs) - 1; i >= 0; i-- {
		if !underPath(s.docs[i].path, path) {
			continue
		}
		if s.docs[i].buf != nil {
			s.docs[i].buf.MarkSaved()
		}
		s.closeDoc(i)
	}
}

func underPath(path, root string) bool {
	if path == "" || root == "" {
		return false
	}
	return path == root || strings.HasPrefix(path, root+string(filepath.Separator))
}

func (s *shell) openExport() { s.openExportScope(s.reportMode) }

func (s *shell) openExportScope(reportScope bool) {
	if s.sheet == nil {
		return
	}
	if reportScope {
		s.useReports()
		if _, ok := s.book.Current(); !ok {
			if s.line3 != nil {
				s.line3.Set("Open a report")
			}
			s.writeFooter()
			return
		}
	} else if s.docPath == "" || s.buf == nil {
		if s.line3 != nil {
			s.line3.Set("Open a document")
		}
		s.writeFooter()
		return
	}
	var list []ui.ExportFormat
	for _, b := range report.Outputs() {
		list = append(list, ui.ExportFormat{ID: b.ID, Label: b.Label})
	}
	s.sheet.SetFormats(list)
	anchor := geometry.Rect{}
	if s.exportBtn != nil {
		anchor = s.exportBtn.Slot("Export")
	}
	s.exportRoot, s.exportName = "", ""
	if !reportScope {
		s.loadProject(s.sheet)
	}
	s.sheet.SetTheme(s.look)
	title := filepath.Base(s.docPath)
	if reportScope {
		if rep, ok := s.book.Current(); ok {
			title = rep.Name
			s.exportRoot, s.exportName = s.reportRoot, rep.Name
			s.sheet.SetProject(rep.Name, rep.Files, rep.Template)
		}
	}
	s.sheet.Show(title, anchor)
	s.sheet.FixScope(reportScope)
	if s.wake != nil {
		s.wake()
	}
}

func (s *shell) loadProject(sheet *ui.ExportSheet) {
	root := s.projectRoot()
	name := s.docProject
	if name == "" {
		name = filepath.Base(root)
	}
	var st project.Settings
	if root != "" {
		loaded, err := project.Load(root)
		if err != nil {
			log.Printf("project: %s: %v", root, err)
		} else {
			st = loaded
		}
		files, err := project.Order(root, st.Files)
		if err != nil {
			log.Printf("project: %s: %v", root, err)
		} else {
			st.Files = files
		}
	}
	sheet.SetProject(name, st.Files, st.Template)
	sheet.OnProjectSave(func(files []string, template string) {
		if root == "" {
			return
		}
		if err := project.Save(root, project.Settings{Files: files, Template: template}); err != nil {
			log.Printf("project: %s: %v", root, err)
			if s.line3 != nil {
				s.line3.Set(err.Error())
			}
			s.writeFooter()
		}
	})
}

func (s *shell) commitExport(choice ui.ExportChoice) {
	if !choice.Project && (s.buf == nil || s.docPath == "") {
		return
	}
	root := s.projectRoot()
	var note []byte
	if s.buf != nil {
		note = []byte(s.buf.Text())
	}
	if choice.Project && s.exportRoot != "" {
		root = s.exportRoot
	}
	src, name, current, err := exportBody(root, s.docPath, note, choice)
	if err != nil {
		if s.line3 != nil {
			s.line3.Set(err.Error())
		}
		s.writeFooter()
		return
	}
	if choice.Project && s.exportName != "" {
		name = s.exportName
	}
	dir := ""
	switch choice.Dest {
	case ui.DestCurrent:
		dir = current
	case ui.DestPick:
		picked, err := chooseFolder("Choose a folder for the export")
		if err != nil || picked == "" {
			if s.line3 != nil {
				s.line3.Set("Export cancelled")
			}
			s.writeFooter()
			return
		}
		dir = picked
	}
	res, err := report.Export(report.ExportRequest{
		Root:  root,
		Src:   src,
		Name:  name,
		Brand: choice.Brand,
		Dir:   dir,
		Base:  current,
	})
	if err != nil {
		if s.line3 != nil {
			s.line3.Set(err.Error())
		}
		s.writeFooter()
		return
	}
	if choice.Open {
		if err := report.Open(res.PDF); err != nil {
			if s.line3 != nil {
				s.line3.Set(err.Error())
			}
			s.writeFooter()
			return
		}
	}
	if s.line3 != nil {
		s.line3.Set(res.PDF)
	}
	s.writeFooter()
}

// exportBody is the Markdown a confirmed export writes.
// A project joins its files, names the result after the project folder,
// and treats that folder as the current directory.
// An empty Dir still means a temporary directory. Pick is unchanged.
func exportBody(root, docPath string, note []byte, choice ui.ExportChoice) (src []byte, name, current string, err error) {
	name = strings.TrimSuffix(filepath.Base(docPath), filepath.Ext(docPath))
	current = filepath.Dir(docPath)
	if !choice.Project {
		return note, name, current, nil
	}
	if root == "" {
		root = current
	}
	src, err = project.Join(root, choice.Files)
	if err != nil {
		return nil, "", "", err
	}
	return src, filepath.Base(root), root, nil
}
