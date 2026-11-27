package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/library"
	"pairti-lense/internal/report"
	"pairti-lense/internal/session"
	"pairti-lense/internal/ui"
)

func (s *shell) focusedIndex() int {
	if s.focus == 1 && s.splitAt >= 0 && s.splitAt < len(s.docs) {
		return s.splitAt
	}
	if s.active >= 0 && s.active < len(s.docs) {
		return s.active
	}
	return -1
}

func (s *shell) currentPage() *ui.Page {
	if s.focus == 1 && s.pageB != nil {
		return s.pageB
	}
	return s.page
}

func (s *shell) currentEditor() *ui.Editor {
	if s.focus == 1 && s.editorB != nil {
		return s.editorB
	}
	return s.editor
}

func (s *shell) currentFrame() *ui.Frame {
	if s.focus == 1 && s.frameB != nil {
		return s.frameB
	}
	return s.frame
}

func (s *shell) focusEditor() {
	if s.editorB != nil && s.editorB.IsFocused() {
		s.focus = 1
		return
	}
	if s.editor != nil && s.editor.IsFocused() {
		s.focus = 0
	}
}

func (s *shell) bindBuf(b *edit.Buffer) {
	s.buf = b
	if s.viewing != "" {
		return
	}
	i := s.focusedIndex()
	if i >= 0 {
		s.docs[i].buf = b
	}
}

func (s *shell) indexOf(path string) int {
	if path == "" {
		return -1
	}
	for i, d := range s.docs {
		if d.path == path {
			return i
		}
	}
	return -1
}

func (s *shell) remember() {
	if s.viewing != "" {
		return
	}
	i := s.focusedIndex()
	if i < 0 {
		return
	}
	if ed := s.currentEditor(); ed != nil && ed.Buffer() != nil && s.editing {
		s.buf = ed.Buffer()
	}
	d := &s.docs[i]
	d.buf = s.buf
	d.editing = s.editing
	d.pending = s.pending
	d.ignored = s.ignored
	if s.docPath != "" {
		d.path = s.docPath
		d.rel = s.docRel
		d.project = s.docProject
	}
}

func (s *shell) loadPane(pane int) {
	if s.diff != nil && s.diff.pane == pane {
		s.closeDiff()
	}
	idx := s.active
	page, editor, frame := s.page, s.editor, s.frame
	if pane == 1 {
		idx = s.splitAt
		page, editor, frame = s.pageB, s.editorB, s.frameB
	}
	if idx < 0 || idx >= len(s.docs) {
		if page != nil {
			page.SetBlocks(nil)
		}
		if frame != nil {
			frame.ShowB(false)
		}
		return
	}
	d := s.docs[idx]
	if editor != nil {
		editor.UseBuffer(d.buf)
		editor.SetFile(d.path)
		editor.SetFocused(d.editing && pane == s.focus)
	}
	if frame != nil {
		frame.ShowB(d.editing)
	}
	if page != nil && d.buf != nil {
		page.UseBuffer(d.buf)
		page.SetPath(d.path)
		page.SetSource(filepath.Dir(d.path), doc.Render(d.path, []byte(d.buf.Text())))
		s.parkCaret(page, 0)
	}
}

func (s *shell) notePage(pane int) {
	if s.diff != nil && s.diff.pane != pane {
		s.closeDiff()
	}
	if pane != s.focus {
		s.remember()
		s.focus = pane
		i := s.active
		if pane == 1 {
			i = s.splitAt
		}
		if i >= 0 && i < len(s.docs) {
			d := s.docs[i]
			s.docPath = d.path
			s.docRel = d.rel
			s.docProject = d.project
			s.editing = d.editing
			s.pending = d.pending
			s.ignored = d.ignored
			s.buf = d.buf
			if s.watch != nil {
				s.watch.Set(d.path)
			}
		}
	}
	if page := s.currentPage(); page != nil && page.Buffer() != nil {
		s.bindBuf(page.Buffer())
	}
	if s.buf != nil {
		s.line3.Set(s.modeLine())
		if s.editorStats() {
			s.panels()
		}
	}
	s.syncAction()
	s.syncCommand()
}

func (s *shell) modeName() string {
	if s.editing {
		if ed := s.currentEditor(); ed != nil {
			return ed.ModeName()
		}
	}
	if p := s.currentPage(); p != nil && p.Open() {
		return p.ModeName()
	}
	return ""
}

func (s *shell) syncCommand() {
	if s.cmdLine == nil {
		return
	}
	if s.diff != nil {
		s.cmdLine.Set(false, "")
		return
	}
	on, text := false, ""
	if s.editing {
		if ed := s.currentEditor(); ed != nil && ed.CommandOn() {
			on, text = true, ed.Command()
		}
	} else if p := s.currentPage(); p != nil && p.CommandOn() {
		on, text = true, p.Command()
	}
	if on && s.buf != nil {
		s.cmdLine.SetPrompt(s.buf.Modal.Prompt())
	}
	s.cmdLine.Set(on, text)
}

func (s *shell) runCmd(cmd string) {
	switch strings.TrimSpace(cmd) {
	case "w":
		s.saveDoc()
	case "wq", "x":
		s.saveDoc()
		s.leaveModal()
	case "q":
		if s.buf != nil && s.buf.Dirty() {
			if s.line3 != nil {
				s.line3.Set("Unsaved")
			}
			s.writeFooter()
			s.syncCommand()
			return
		}
		s.leaveModal()
	case "q!":
		s.discard()
	case "sp", "split", "vsp", "vsplit":
		s.ensureSplit()
	default:
		trimmed := strings.TrimSpace(cmd)
		if trimmed == "template" || strings.HasPrefix(trimmed, "template ") {
			s.useTemplate(strings.TrimSpace(strings.TrimPrefix(trimmed, "template")))
			return
		}
		if strings.TrimSpace(cmd) != "" && s.line3 != nil {
			s.line3.Set(strings.TrimSpace(cmd))
		}
		s.syncAction()
		s.syncCommand()
	}
}

func (s *shell) useTemplate(name string) {
	root := s.projectRoot()
	if root == "" {
		if s.line3 != nil {
			s.line3.Set("Open a project")
		}
		s.writeFooter()
		return
	}
	if name == "" {
		if s.line3 != nil {
			s.line3.Set(strings.Join(report.Templates(root), " "))
		}
		s.writeFooter()
		s.syncCommand()
		return
	}
	if err := report.SetTemplate(root, name); err != nil {
		if s.line3 != nil {
			s.line3.Set(err.Error())
		}
		s.writeFooter()
		s.syncCommand()
		return
	}
	s.exportReport()
}

func (s *shell) leaveModal() {
	if s.editing {
		s.editing = false
		if ed := s.currentEditor(); ed != nil {
			ed.SetFocused(false)
		}
		if fr := s.currentFrame(); fr != nil {
			fr.ShowB(false)
		}
	}
	if p := s.currentPage(); p != nil {
		p.Close()
	}
	if s.line3 != nil {
		s.line3.Set(s.modeLine())
	}
	s.syncAction()
	s.syncCommand()
}

func (s *shell) pageOpen() bool {
	if s.focus == 1 && s.pageB != nil {
		return s.pageB.Open()
	}
	return s.page != nil && s.page.Open()
}

func (s *shell) applyFocus() {
	s.viewing = ""
	if s.diff != nil && s.diff.pane != s.focus {
		s.closeDiff()
	}
	i := s.focusedIndex()
	if i < 0 {
		s.docPath = ""
		s.docRel = ""
		s.docProject = ""
		s.buf = nil
		s.editing = false
		s.pending = ""
		s.ignored = ""
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
		s.syncAction()
		s.fillRail()
		return
	}
	d := s.docs[i]
	s.docPath = d.path
	s.docRel = d.rel
	s.docProject = d.project
	s.buf = d.buf
	s.editing = d.editing
	s.pending = d.pending
	s.ignored = d.ignored
	if s.watch != nil {
		s.watch.Set(d.path)
	}
	text := ""
	if s.buf != nil {
		text = s.buf.Text()
	}
	if s.line1 != nil {
		s.line1.Set(fileTitle(d.path))
	}
	if s.line2 != nil {
		s.line2.Set(doc.ShortPath(d.rel))
	}
	s.stats(text)
	if s.line3 != nil {
		s.line3.Set(s.modeLine())
	}
	s.syncAction()
	s.panels()
}

// panels points the file tree and the right rail at the focused pane.
func (s *shell) panels() {
	if s.rail != nil && s.railPath != s.docPath {
		s.rail.Home()
		s.railPath = s.docPath
	}
	s.fillRail()
	s.showFile(s.docPath)
}

// showFile marks path in the tree and opens the folders that hide it.
func (s *shell) showFile(path string) {
	if s.side == nil || path == "" {
		return
	}
	if s.reveal(path) {
		s.refresh()
	}
	if s.side.Selected() != path {
		s.side.Select(path)
	}
}

// reveal expands each folder from the project root down to path.
// It reports whether the tree has to be rebuilt.
func (s *shell) reveal(path string) bool {
	root := ""
	for _, p := range s.projects {
		if path == p.Root || strings.HasPrefix(path, p.Root+string(filepath.Separator)) {
			root = p.Root
			break
		}
	}
	if root == "" {
		return false
	}
	changed := false
	if !s.expanded[root] {
		s.expanded[root] = true
		changed = true
	}
	dir := filepath.Dir(path)
	var chain []string
	for dir != root && strings.HasPrefix(dir, root) {
		chain = append(chain, dir)
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	for i := len(chain) - 1; i >= 0; i-- {
		d := chain[i]
		if _, ok := s.cache[d]; !ok {
			entries, err := doc.ReadDir(d, root, s.listOf(root))
			if err == nil {
				s.cache[d] = entries
				changed = true
			}
		}
		if !s.expanded[d] {
			s.expanded[d] = true
			changed = true
		}
	}
	return changed
}

// showPane displays document i in one pane. Pane 0 is the left, pane 1 the right.
func (s *shell) showPane(pane, i int) {
	if i < 0 || i >= len(s.docs) {
		return
	}
	s.closeDiff()
	s.remember()
	if pane == 1 {
		s.splitAt = i
		s.focus = 1
		if s.pair != nil && !s.pair.Split() {
			s.pair.SetSplit(true)
		}
		s.loadPane(1)
	} else {
		s.active = i
		s.focus = 0
		s.loadPane(0)
	}
	s.applyFocus()
	s.focusDoc()
	d := s.docs[i]
	s.noteHist(ui.Item{Path: d.path, Rel: d.rel, ProjectName: d.project, ProjectRoot: s.projectRoot()})
	s.saveSession()
	s.redraw()
}

func (s *shell) show(i int) {
	if i < 0 || i >= len(s.docs) {
		return
	}
	if i == s.focusedIndex() && s.viewing == "" {
		s.syncTabs()
		return
	}
	s.remember()
	if s.splitAt == i {
		s.focus = 1
	} else if s.focus == 1 && s.active == i {
		s.focus = 0
	} else if s.focus == 1 && s.splitAt >= 0 {
		s.splitAt = i
	} else {
		s.focus = 0
		s.active = i
	}
	s.loadPane(0)
	if s.splitAt >= 0 {
		s.loadPane(1)
	}
	s.applyFocus()
	s.focusDoc()
	s.saveSession()
	s.redraw()
	if i := s.focusedIndex(); i >= 0 {
		d := s.docs[i]
		s.noteHist(ui.Item{Path: d.path, Rel: d.rel, ProjectName: d.project, ProjectRoot: s.projectRoot()})
	}
}

func (s *shell) closeDoc(i int) {
	if i < 0 || i >= len(s.docs) {
		return
	}
	s.remember()
	if s.docs[i].buf != nil && s.docs[i].buf.Dirty() {
		if s.line3 != nil {
			s.line3.Set("Unsaved")
		}
		s.syncAction()
		return
	}
	s.dropDoc(i)
}

func (s *shell) dropDoc(i int) {
	s.docs = append(s.docs[:i], s.docs[i+1:]...)
	if s.splitAt == i {
		s.splitAt = -1
		if s.pair != nil {
			s.pair.SetSplit(false)
		}
		if s.focus == 1 {
			s.focus = 0
		}
	} else if s.splitAt > i {
		s.splitAt--
	}
	if s.active == i {
		s.active = i - 1
		if s.active < 0 && len(s.docs) > 0 {
			s.active = 0
		}
	} else if s.active > i {
		s.active--
	}
	if s.active >= len(s.docs) {
		s.active = len(s.docs) - 1
	}
	if s.splitAt == s.active {
		s.splitAt = -1
		if s.pair != nil {
			s.pair.SetSplit(false)
		}
		if s.focus == 1 {
			s.focus = 0
		}
	}
	s.loadPane(0)
	if s.splitAt >= 0 {
		s.loadPane(1)
	} else if s.pageB != nil {
		s.pageB.SetBlocks(nil)
	}
	s.applyFocus()
	s.saveSession()
}

// closeWhere closes every tab for which drop reports true, from the right
// so earlier indexes stay put. A dirty tab stays open.
func (s *shell) closeWhere(drop func(int) bool) {
	refused := false
	for i := len(s.docs) - 1; i >= 0; i-- {
		if !drop(i) {
			continue
		}
		n := len(s.docs)
		s.closeDoc(i)
		if len(s.docs) == n {
			refused = true
		}
	}
	if refused && s.line3 != nil {
		s.line3.Set("Unsaved")
		s.writeFooter()
	}
}

func (s *shell) ensureSplit() {
	if len(s.docs) == 0 {
		return
	}
	if s.pair != nil && s.pair.Split() {
		s.focus = 1
		s.applyFocus()
		return
	}
	s.toggleSplit()
}

func (s *shell) winCmd(dir rune) {
	switch dir {
	case 'l', 'w':
		if s.pair == nil || !s.pair.Split() {
			return
		}
		if dir == 'w' {
			s.focus = 1 - s.focus
		} else {
			s.focus = 1
		}
		s.applyFocus()
		s.focusDoc()
	case 'h':
		s.focus = 0
		s.applyFocus()
		s.focusDoc()
	}
}

func (s *shell) toggleSplit() {
	if len(s.docs) == 0 {
		return
	}
	if s.pair != nil && s.pair.Split() {
		s.remember()
		s.pair.SetSplit(false)
		s.splitAt = -1
		s.focus = 0
		s.loadPane(0)
		s.applyFocus()
		s.saveSession()
		return
	}
	i := s.focusedIndex()
	if i < 0 {
		i = 0
	}
	s.splitDoc(i)
}

func (s *shell) toggleShell() {
	if s.console == nil {
		return
	}
	if s.console.Open() {
		s.console.SetOpen(false)
		return
	}
	dir := s.chosen
	if s.docPath != "" {
		dir = filepath.Dir(s.docPath)
	}
	if err := s.console.Start(dir, s.wake); err != nil {
		if s.line3 != nil {
			s.line3.Set(err.Error())
		}
		s.writeFooter()
		return
	}
	s.console.SetFocused(true)
	s.armFocus = true
}

func (s *shell) splitDoc(i int) {
	if i < 0 || i >= len(s.docs) {
		return
	}
	s.remember()
	if s.active == i {
		other := -1
		for j := range s.docs {
			if j != i {
				other = j
				break
			}
		}
		if other >= 0 {
			s.active = other
		}
	}
	s.splitAt = i
	s.focus = 1
	if s.pair != nil {
		s.pair.SetSplit(true)
	}
	s.loadPane(0)
	s.loadPane(1)
	s.applyFocus()
	s.saveSession()
}

func (s *shell) saveSession() {
	if s.holdSave || s.sessionPath == "" {
		return
	}
	s.remember()
	st := session.State{Recent: s.recent, Active: s.active, Split: s.splitAt, Focus: s.focus, HideRawNumbers: s.hideRawNumbers, FormattedNumbers: s.formattedNumbers}
	for _, d := range s.docs {
		if d.path == "" {
			continue
		}
		st.Docs = append(st.Docs, session.Doc{Path: d.path, Raw: d.editing})
	}
	for path, on := range s.expanded {
		if on && strings.TrimSpace(path) != "" {
			st.Expanded = append(st.Expanded, path)
		}
	}
	if s.scroll != nil {
		st.Scroll = keptScroll(s.scroll.Get())
	}
	if s.scrollB != nil {
		st.ScrollB = keptScroll(s.scrollB.Get())
	}
	if s.split != nil {
		st.Sidebar = s.split.Width()
		st.HideSide = !s.split.Shown()
	}
	if s.railSplit != nil {
		st.Rail = s.railSplit.Shown()
	}
	if err := session.Save(s.sessionPath, st); err != nil {
		log.Printf("session: %v", err)
	}
}

func (s *shell) restoreDocs(st session.State) {
	s.recent = st.Recent
	s.hideRawNumbers = st.HideRawNumbers
	s.formattedNumbers = st.FormattedNumbers
	s.applyLineNumbers()
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("session: restore stopped: %v", rec)
		}
	}()
	active, split := "", ""
	if st.Active >= 0 && st.Active < len(st.Docs) {
		active = st.Docs[st.Active].Path
	}
	if st.Split >= 0 && st.Split < len(st.Docs) {
		split = st.Docs[st.Split].Path
	}
	for _, d := range st.Docs {
		s.openSaved(d.Path, d.Raw)
	}
	if len(s.docs) == 0 {
		return
	}
	if i := s.indexOf(active); i >= 0 {
		s.active = i
	} else {
		s.active = 0
	}
	s.splitAt = -1
	if i := s.indexOf(split); i >= 0 && i != s.active {
		s.splitAt = i
	}
	s.focus = 0
	if st.Focus == 1 && s.splitAt >= 0 {
		s.focus = 1
	}
	s.loadPane(0)
	if s.splitAt >= 0 {
		s.loadPane(1)
	}
	s.applyFocus()
	s.focusDoc()
	if s.scroll != nil {
		s.scroll.Set(keptScroll(st.Scroll))
	}
	if s.scrollB != nil {
		s.scrollB.Set(keptScroll(st.ScrollB))
	}
}

func (s *shell) openSaved(path string, raw bool) {
	path = filepath.Clean(strings.TrimSpace(path))
	before := len(s.docs)
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("session: skip %s: %v", path, rec)
			s.undoOpened(path, before)
		}
	}()
	if path == "" || path == "." {
		return
	}
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		log.Printf("session: skip %s", path)
		return
	}
	root, name, rel := s.placeOf(path)
	s.open(ui.Item{Path: path, Rel: rel, ProjectName: name, ProjectRoot: root})
	if s.openSavedHook != nil {
		s.openSavedHook(path)
	}
	if raw && s.docPath == path {
		s.showRaw(true)
		s.remember()
	}
}

// undoOpened drops a note that panicked while opening, so restore can continue.
func (s *shell) undoOpened(path string, before int) {
	if before < 0 {
		before = 0
	}
	if before > len(s.docs) {
		before = len(s.docs)
	}
	kept := append([]openDoc{}, s.docs[:before]...)
	for _, d := range s.docs[before:] {
		if d.path != path {
			kept = append(kept, d)
		}
	}
	s.docs = kept
	if len(s.docs) == 0 {
		s.active = -1
	} else if s.active >= len(s.docs) {
		s.active = len(s.docs) - 1
	}
	if s.splitAt >= len(s.docs) {
		s.splitAt = -1
	}
	if s.docPath == path {
		s.docPath = ""
		s.docRel = ""
		s.docProject = ""
		s.buf = nil
		s.editing = false
		s.pending = ""
		s.ignored = ""
	}
}

func (s *shell) placeOf(path string) (root, name, rel string) {
	rel = filepath.Base(path)
	best := ""
	for _, p := range s.projects {
		if p.Root == "" {
			continue
		}
		if path == p.Root || strings.HasPrefix(path, p.Root+string(filepath.Separator)) {
			if len(p.Root) > len(best) {
				best = p.Root
				name = p.Name
			}
		}
	}
	if best == "" {
		return "", "", rel
	}
	if r, err := filepath.Rel(best, path); err == nil {
		rel = r
	}
	return best, name, rel
}

func keptScroll(y float32) float32 {
	if y != y || y < 0 || y > 1e7 {
		return 0
	}
	return y
}

func (s *shell) syncTabs() {
	if s.tabs == nil {
		return
	}
	focus := s.focusedIndex()
	items := make([]ui.Tab, len(s.docs))
	for i, d := range s.docs {
		dirty := d.buf != nil && d.buf.Dirty()
		if i == focus && s.buf != nil {
			dirty = s.buf.Dirty()
		}
		items[i] = ui.Tab{Label: fileTitle(d.path), Dirty: dirty}
	}
	left := s.active
	if left < 0 || left >= len(items) {
		left = focus
	}
	s.tabs.Set(items, left, -1)
	if s.tabsB != nil {
		s.tabsB.Set(items, s.splitAt, -1)
	}
}

func (s *shell) fillRail() {
	if s.rail == nil {
		return
	}
	page := s.currentPage()
	if page == nil {
		return
	}
	s.rail.SetOutline(page.Outline())
	s.rail.SetLinks(page.Links())
	s.rail.SetPictures(page.Pictures())
	var info []string
	if s.docPath != "" || s.viewing != "" {
		if s.viewing != "" {
			info = append(info, s.docProject)
		} else {
			info = append(info, fileTitle(s.docPath))
		}
		if s.docRel != "" {
			info = append(info, doc.ShortPath(s.docRel))
		}
		if s.words != nil && s.words.Get() != "" {
			info = append(info, s.words.Get())
		}
		if s.chars != nil && s.chars.Get() != "" {
			info = append(info, s.chars.Get())
		}
	}
	s.rail.SetInfo(info)
}

func (s *shell) jump(line int) {
	if s.editing {
		ed := s.currentEditor()
		if ed == nil || ed.Buffer() == nil {
			return
		}
		ed.ShowLine(line)
		if s.buf != nil {
			s.stats(s.buf.Text())
			s.writeFooter()
		}
		if s.wake != nil {
			s.wake()
		}
		return
	}
	page := s.currentPage()
	if page == nil {
		return
	}
	y := page.LineY(line)
	if s.focus == 1 && s.scrollB != nil {
		s.scrollB.Set(y)
		return
	}
	if s.scroll != nil {
		s.scroll.Set(y)
	}
}

func (s *shell) listOf(root string) doc.List {
	if s.lib == nil {
		return doc.List{}
	}
	v := s.lib.View(root)
	return doc.List{ShowDot: v.ShowDot, ShowIgnored: v.ShowIgnored, AllDirs: v.AllDirs}
}

func (s *shell) chosenProject() string {
	if s.chosen != "" {
		return s.chosen
	}
	for _, p := range s.projects {
		if !p.Absent && p.Root != "" {
			return p.Root
		}
	}
	if s.lib != nil {
		for _, p := range s.lib.Paths {
			if p != "" {
				return p
			}
		}
	}
	return ""
}

func (s *shell) rereadProject(root string) {
	if s.remote != nil {
		s.remote.list()
		return
	}
	if root == "" {
		return
	}
	sep := string(filepath.Separator)
	var dirs []string
	for dir := range s.cache {
		if dir == root || strings.HasPrefix(dir, root+sep) {
			dirs = append(dirs, dir)
		}
	}
	for _, dir := range dirs {
		entries, err := doc.ReadDir(dir, root, s.listOf(root))
		if err != nil {
			delete(s.cache, dir)
			continue
		}
		s.cache[dir] = entries
	}
}

func (s *shell) syncActions() {
	if s.actions == nil {
		return
	}
	root := s.chosenProject()
	if root == "" {
		s.actions.Set("", false, false, false, false)
		return
	}
	v := library.View{}
	if s.lib != nil {
		v = s.lib.View(root)
	}
	s.actions.Set(filepath.Base(root), v.ShowDot, v.ShowIgnored, v.Names, !v.AllDirs)
}

func (s *shell) onProjectView(showDot, showIgnored bool) {
	if s.lib == nil {
		return
	}
	for _, root := range s.projectRoots() {
		v := s.lib.View(root)
		v.ShowDot = showDot
		v.ShowIgnored = showIgnored
		s.lib.SetView(root, v)
		s.rereadProject(root)
	}
	s.save()
	s.refresh()
}

func (s *shell) toggleRail() {
	if s.railSplit == nil {
		return
	}
	s.railSplit.Toggle()
	s.fillRail()
}

func (s *shell) documentColumn() widget.Widget {
	s.tabs = ui.NewTabs(func(i int) { s.showPane(0, i) }, s.closeDoc, s.splitDoc)
	s.tabs.OnMenu(s.showTabMenu)
	s.tabs.SetTheme(s.look)
	s.tabsB = ui.NewTabs(func(i int) { s.showPane(1, i) }, s.closeDoc, s.splitDoc)
	s.tabsB.OnMenu(s.showTabMenu)
	s.tabsB.SetTheme(s.look)
	view := scrollview.New(ui.ScrollInset(s.page),
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollStep(12),
		scrollview.ScrollYSignal(s.scroll),
		scrollview.PainterOpt(ui.ScrollPainter{}),
	)
	s.frame = ui.NewFrame(view, s.editor)
	viewB := scrollview.New(ui.ScrollInset(s.pageB),
		scrollview.DirectionOpt(scrollview.Vertical),
		scrollview.ScrollStep(12),
		scrollview.ScrollYSignal(s.scrollB),
		scrollview.PainterOpt(ui.ScrollPainter{}),
	)
	s.frameB = ui.NewFrame(viewB, s.editorB)
	pane := func(tabs *ui.Tabs, frame widget.Widget) widget.Widget {
		body := primitives.Box(primitives.Expanded(frame)).
			PaddingLeft(28).PaddingTop(28).PaddingBottom(28).
			Background(widget.Hex(s.look.Canvas))
		return primitives.Box(tabs, primitives.Expanded(body)).Background(widget.Hex(s.look.Canvas))
	}
	s.pair = ui.NewPair(pane(s.tabs, s.frame), pane(s.tabsB, s.frameB))
	s.pair.SetTheme(s.look)
	s.cmdLine = ui.NewCommandLine()
	s.cmdLine.SetTheme(s.look)
	children := []widget.Widget{primitives.Expanded(s.pair)}
	if s.console != nil {
		s.console.SetTheme(s.look)
		children = append(children, s.console)
	}
	column := primitives.Box(append(children, s.cmdLine)...).Background(widget.Hex(s.look.Canvas))
	s.rail = ui.NewRail(s.jump, s.follow)
	s.rail.SetTheme(s.look)
	s.railSplit = ui.NewSplit(s.rail, column, 300)
	s.railSplit.SetTheme(s.look)
	s.railSplit.SetRight(true)
	s.railSplit.Toggle()
	s.syncTabs()
	return s.railSplit
}
