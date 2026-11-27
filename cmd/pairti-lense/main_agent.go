package main

import (
	"fmt"
	"log"
	"path/filepath"

	"pairti-lense/internal/agent"
	"pairti-lense/internal/doc"
)

func (s *shell) listenMCP(path string) error {
	ep, err := agent.LoadEndpoint(path)
	if err != nil {
		log.Printf("mcp: %v", err)
		quarantine(path, err)
		ep = agent.Endpoint{}
	}
	if ep.Token == "" {
		ep.Token, err = agent.NewToken()
		if err != nil {
			return err
		}
	}
	addr := agent.PreferredAddr(ep.URL)
	srv, listenURL, err := agent.Serve(s.agent, ep.Token, addr)
	if err != nil {
		return err
	}
	s.mcpSrv = srv
	s.mcpURL = listenURL
	s.mcpToken = ep.Token
	ep.URL = listenURL
	if err := agent.SaveEndpoint(path, ep); err != nil {
		return err
	}
	log.Printf("mcp %s", listenURL)
	return nil
}

// noteMCP shows the listen address when no document is open.
func (s *shell) noteMCP() {
	if s.docPath != "" || s.line3 == nil {
		return
	}
	msg := s.mcpURL
	if s.mcpErr != "" {
		msg = s.mcpErr
	}
	s.line3.Set(msg)
	s.writeFooter()
}

func (s *shell) drainAgent() bool {
	if s.agent == nil {
		return false
	}
	changed := false
	for {
		req, ok := s.agent.Take()
		if !ok {
			return changed
		}
		switch req.Kind {
		case agent.KindRead:
			req.Reply(agent.Response{Snap: s.agentSnapshot(req.Read)})
		case agent.KindEdit:
			out, err := s.applyAgent(req.Edit)
			req.Reply(agent.Response{Applied: out, Err: err})
			if err == nil {
				changed = true
			}
		default:
			req.Reply(agent.Response{Err: fmt.Errorf("unknown request")})
		}
	}
}

func (s *shell) agentSnapshot(in agent.ReadIn) agent.Snapshot {
	snap := agent.Snapshot{
		Path:    s.docPath,
		Rel:     s.docRel,
		Project: s.docProject,
		Mode:    "Formatted",
		Input:   "View",
		MIME:    "text/plain",
		Outline: []agent.Heading{},
		Tabs:    []agent.Tab{},
	}
	if s.editing {
		snap.Mode = "Raw"
	} else if page := s.currentPage(); s.docPath != "" && page != nil && page.Buffer() == nil {
		snap.Mode = "Preview"
	}
	if name := s.modeName(); name != "" {
		snap.Input = name
	} else if s.editing {
		snap.Input = "Edit"
	}
	if doc.IsMarkdown(s.docPath) {
		snap.MIME = "text/markdown"
	}
	focus := s.focusedIndex()
	for i, d := range s.docs {
		title := filepath.Base(d.path)
		if d.buf != nil && doc.IsMarkdown(d.path) {
			if t := doc.Title([]byte(d.buf.Text())); t != "" {
				title = t
			}
		}
		snap.Tabs = append(snap.Tabs, agent.Tab{
			Path:    d.path,
			Title:   title,
			Dirty:   d.buf != nil && d.buf.Dirty(),
			Focused: i == focus,
		})
	}
	if s.buf == nil || s.docPath == "" {
		return snap
	}
	text := s.buf.Text()
	snap.Text = text
	snap.Dirty = s.buf.Dirty()
	snap.Chars = len([]rune(text))
	snap.Words = doc.CountWords(doc.Render(s.docPath, []byte(text)))
	snap.Cursor = s.buf.Cursor()
	snap.Line = s.buf.CaretLine() + 1
	snap.Col = s.buf.CaretCol()
	lines := s.buf.Lines()
	if snap.Line >= 1 && snap.Line <= len(lines) {
		snap.CurrentLine = lines[snap.Line-1]
	}
	radius := in.Radius
	if radius <= 0 {
		radius = 20
	}
	snap.Around = agent.Window(text, snap.Line, radius)
	if from, to, ok := s.buf.Selection(); ok {
		runes := []rune(text)
		if to > len(runes) {
			to = len(runes)
		}
		if from < 0 {
			from = 0
		}
		snap.HasSelection = true
		snap.SelectionFrom = from
		snap.SelectionTo = to
		snap.Selection = string(runes[from:to])
		snap.Selected = snap.Selection
		snap.SelectedKind = "selection"
	} else {
		snap.Selected = snap.CurrentLine
		snap.SelectedKind = "line"
	}
	title := filepath.Base(s.docPath)
	if doc.IsMarkdown(s.docPath) {
		if t := doc.Title([]byte(text)); t != "" {
			title = t
		}
	}
	snap.Title = title
	if page := s.currentPage(); page != nil {
		for _, h := range page.Outline() {
			snap.Outline = append(snap.Outline, agent.Heading{
				Number: h.Number,
				Text:   h.Text,
				Level:  h.Level,
				Line:   h.Line,
			})
		}
	}
	return snap
}

func (s *shell) applyAgent(in agent.ApplyIn) (agent.ApplyOut, error) {
	if s.buf == nil || s.docPath == "" {
		return agent.ApplyOut{}, fmt.Errorf("no document open")
	}
	if s.pending != "" {
		return agent.ApplyOut{}, fmt.Errorf("resolve the change on disk first")
	}
	if in.Path != "" && filepath.Clean(in.Path) != filepath.Clean(s.docPath) {
		return agent.ApplyOut{}, fmt.Errorf("focused document is %s", s.docPath)
	}
	changed, err := agent.Apply(s.buf, in)
	if err != nil {
		return agent.ApplyOut{}, err
	}
	if changed {
		s.afterAgentEdit()
	}
	return s.applyOut(), nil
}

func (s *shell) applyOut() agent.ApplyOut {
	out := agent.ApplyOut{Path: s.docPath}
	if s.buf == nil {
		return out
	}
	out.Dirty = s.buf.Dirty()
	out.Line = s.buf.CaretLine() + 1
	out.Col = s.buf.CaretCol()
	out.Cursor = s.buf.Cursor()
	out.Chars = len([]rune(s.buf.Text()))
	return out
}

func (s *shell) afterAgentEdit() {
	text := s.buf.Text()
	s.bindBuf(s.buf)
	blocks := doc.Render(s.docPath, []byte(text))
	title := filepath.Base(s.docPath)
	if doc.IsMarkdown(s.docPath) {
		if t := doc.Title([]byte(text)); t != "" {
			title = t
		}
	}
	s.retitle(s.docPath, title)
	if page := s.currentPage(); page != nil {
		reopen := page.OpenLine()
		page.SetPath(s.docPath)
		page.UseBuffer(s.buf)
		page.SetSource(filepath.Dir(s.docPath), blocks)
		if reopen > 0 {
			page.Reopen(reopen)
		}
	}
	if ed := s.currentEditor(); ed != nil {
		if ed.Buffer() != s.buf {
			ed.UseBuffer(s.buf)
		} else {
			ed.MarkNeedsLayout()
			ed.SetNeedsRedraw(true)
		}
	}
	if s.line1 != nil {
		s.line1.Set(fileTitle(s.docPath))
	}
	s.stats(text)
	if s.line3 != nil {
		s.line3.Set(s.modeLine())
	}
	s.syncAction()
	s.syncCommand()
	s.refresh()
	s.fillRail()
}
