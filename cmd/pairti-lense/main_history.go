package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gogpu/ui/core/scrollview"

	"pairti-lense/internal/gitlog"
	"pairti-lense/internal/ui"
)

const historyBatch = 80

func (s *shell) loadHistory() {
	root, rel := s.historyScope()
	if s.histScopeBar != nil {
		mode := 0
		if rel == "" {
			mode = 1
		}
		s.histScopeBar.SetMode(mode)
	}
	if root == s.histRoot && rel == s.histRel && s.commits != nil {
		return
	}
	s.histRoot, s.histRel = root, rel
	s.histOpen = map[string]bool{}
	s.histFiles = map[string][]string{}
	s.commits = nil
	s.histErr = ""
	if root == "" {
		s.histErr = "Open a project"
		return
	}
	s.readHistory(historyBatch)
	s.branchFor = ""
	s.noteBranch()
}

func (s *shell) readHistory(limit int) {
	commits, err := gitlog.Log(s.histRoot, s.histRel, limit+1)
	if err != nil {
		s.histErr = "Not a git repository"
		s.commits = []gitlog.Commit{}
		return
	}
	if commits == nil {
		commits = []gitlog.Commit{}
	}
	s.histMore = len(commits) > limit
	s.commits = commits[:min(len(commits), limit)]
}

func (s *shell) historyScope() (root, rel string) {
	root = s.projectRoot()
	if root == "" {
		return "", ""
	}
	top, err := gitlog.Top(root)
	if err != nil {
		return root, ""
	}
	if !s.histRepo && s.docPath != "" {
		path := s.docPath
		// Git returns a physical worktree root, including on macOS where /var
		// aliases /private/var. Preserve the name of a tracked symlink itself.
		if dir, err := filepath.EvalSymlinks(filepath.Dir(path)); err == nil {
			path = filepath.Join(dir, filepath.Base(path))
		}
		if r, err := filepath.Rel(top, path); err == nil && r != "." && r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator)) {
			rel = filepath.ToSlash(r)
		}
	}
	return top, rel
}

func (s *shell) historyItems() []ui.Item {
	root, rel := s.historyScope()
	if root != s.histRoot || rel != s.histRel {
		s.loadHistory()
	}
	if s.histErr != "" {
		return []ui.Item{{Label: s.histErr}}
	}
	if len(s.commits) == 0 {
		return []ui.Item{{Label: "No commits"}}
	}
	var out []ui.Item
	for _, c := range s.commits {
		open := s.histOpen[c.Hash]
		out = append(out, ui.Item{
			Label:       c.Date + " · " + gitlogCut(c.Subject, 48),
			Path:        "commit:" + c.Hash,
			Rel:         c.Hash,
			ProjectRoot: s.histRoot,
			Dir:         true,
			Expanded:    open,
		})
		if !open {
			continue
		}
		files, ok := s.histFiles[c.Hash]
		if !ok {
			out = append(out, ui.Item{Depth: 1, Label: "Could not load changed files"})
			continue
		}
		label := fmt.Sprintf("%d files changed", len(files))
		if len(files) == 1 {
			label = "1 file changed"
		}
		out = append(out, ui.Item{Depth: 1, Label: label})
		for _, rel := range files {
			out = append(out, ui.Item{
				Depth: 1, Label: filepath.Base(rel), Rel: rel, Revision: c.Hash,
				Path: "commitfile:" + c.Hash + ":" + rel, ProjectRoot: s.histRoot,
			})
		}
	}
	if s.histMore {
		out = append(out, ui.Item{Label: "Load older commits", Path: "history:older"})
	}
	return out
}

func (s *shell) onHistory(it ui.Item) {
	if it.Path == "history:older" && s.histMore {
		s.readHistory(len(s.commits) + historyBatch)
		s.refresh()
		return
	}
	if it.Rel == "" {
		return
	}
	if strings.HasPrefix(it.Path, "commitfile:") {
		if s.side != nil {
			s.side.Select(it.Path)
		}
		s.showDiff(it.Revision, it.Rel)
		return
	}
	if strings.HasPrefix(it.Path, "commit:") {
		rel := s.histRel
		if s.side != nil {
			s.side.Select(it.Path)
		}
		s.showDiff(it.Rel, rel)
	}
}

func (s *shell) toggleHist(hash string) {
	if s.histOpen == nil {
		s.histOpen = map[string]bool{}
	}
	s.histOpen[hash] = !s.histOpen[hash]
	if s.histOpen[hash] {
		if _, ok := s.histFiles[hash]; !ok {
			files, err := gitlog.Files(s.histRoot, hash)
			if err == nil {
				s.histFiles[hash] = files
			}
		}
	}
	s.refresh()
}

// A comparison lives above the document, with its own scroll position. Only the
// UI thread installs results. Superseded requests retain their own result channel.
type historyComparison struct {
	pane          int
	frame         *ui.Frame
	view          *ui.DiffView
	scroll        *scrollview.Widget
	cancel        context.CancelFunc
	ready         <-chan comparisonResult
	subject, date string
}

type comparisonResult struct {
	model gitlog.Comparison
	err   error
}

func (s *shell) closeDiff() {
	if d := s.diff; d != nil {
		d.cancel()
		d.frame.SetOverlay(nil)
		s.diff = nil
	}
}

func (s *shell) showDiff(hash, rel string) {
	frame := s.currentFrame()
	if frame == nil {
		return
	}
	s.closeDiff()
	view := ui.NewDiffView()
	view.SetTheme(s.look)
	view.SetMessage("Loading changes…")
	scroll := scrollview.New(ui.ScrollInset(view),
		scrollview.DirectionOpt(scrollview.Vertical), scrollview.ScrollStep(12),
		scrollview.PainterOpt(ui.ScrollPainter{}))
	ctx, cancel := context.WithCancel(context.Background())
	ready := make(chan comparisonResult, 1)
	d := &historyComparison{pane: s.focus, frame: frame, view: view, scroll: scroll, cancel: cancel, ready: ready}
	for _, c := range s.commits {
		if c.Hash == hash {
			d.subject, d.date = c.Subject, c.Date
			break
		}
	}
	s.diff = d
	frame.SetOverlay(scroll)
	if p := s.currentPage(); p != nil {
		p.SetFocused(false)
	}
	if e := s.currentEditor(); e != nil {
		e.SetFocused(false)
	}
	s.focusDoc()
	s.syncAction()
	s.syncCommand()
	root, wake := s.histRoot, s.wake
	go func() {
		model, err := gitlog.CompareContext(ctx, root, hash, rel)
		if ctx.Err() != nil {
			return
		}
		ready <- comparisonResult{model, err}
		if wake != nil {
			wake()
		}
	}()
}

func (s *shell) receiveDiff() bool {
	d := s.diff
	if d == nil {
		return false
	}
	select {
	case res := <-d.ready:
		d.ready = nil
		if res.err != nil {
			d.view.SetMessage("Could not load changes: " + res.err.Error())
		} else {
			d.view.SetComparison(d.subject, d.date, res.model)
		}
		return true
	default:
		return false
	}
}

func gitlogCut(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= n {
		return string(r)
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}
