package main

import (
	"path/filepath"
	"slices"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/project"
	"pairti-lense/internal/ui"
)

// reportMenuCommands snapshots the structure; validation and checkmarks use
// live shell state. Folder paths and report IDs identify each action even
// when the titles shown by two projects are identical.
func (s *shell) reportMenuCommands() []menuCommand {
	s.useReports()
	var commands []menuCommand
	disabled := func() bool { return false }
	add := func(group, title string, run func(), enabled, checked func() bool) {
		commands = append(commands, menuCommand{group: group, title: title, run: run, enabled: enabled, checked: checked})
	}
	counts := map[string]int{}
	for _, p := range s.projects {
		counts[p.Name]++
	}
	for _, p := range s.projects {
		label := p.Name
		if label == "" || counts[label] > 1 {
			label = p.Root
		}
		group := menuGroup("Reports", "Open Report", label)
		if p.Absent {
			add(group, "Project Unavailable", nil, disabled, nil)
			continue
		}
		book := s.book
		if p.Root != s.reportRoot {
			var err error
			book, err = project.LoadBook(p.Root)
			if err != nil {
				add(group, "Reports Unavailable", nil, disabled, nil)
				continue
			}
		}
		names := map[string]int{}
		for _, rep := range book.Reports {
			names[rep.Name]++
		}
		for _, rep := range book.Reports {
			title := rep.Name
			if names[rep.Name] > 1 {
				title += " (" + rep.ID + ")"
			}
			add(group, title, func() {
				s.pickReportProject(p.Root)
				s.goToSidebar(ui.SideReports)
				s.openReport(rep.ID)
			}, nil, func() bool { return s.reportRoot == p.Root && s.book.Open == rep.ID })
		}
		if len(book.Reports) == 0 {
			add(group, "No Reports", nil, disabled, nil)
		}
	}
	if len(s.projects) == 0 {
		add(menuGroup("Reports", "Open Report"), "No Open Projects", nil, disabled, nil)
	}

	hasProject := func() bool { return s.projectKnown(s.reportRoot) }
	current := func() bool { _, ok := s.book.Current(); return hasProject() && ok }
	add("Reports", "New Report", func() {
		s.goToSidebar(ui.SideReports)
		s.newReport()
		s.openReport(s.book.Open)
	}, hasProject, nil)
	add("Reports", "Rename Report…", func() {
		s.onTreeMenu(ui.Item{Rel: s.book.Open}, ui.MenuReportRename)
	}, current, nil)
	add("Reports", "Delete Report", func() {
		s.onTreeMenu(ui.Item{Rel: s.book.Open}, ui.MenuReportDelete)
		if s.viewing != "" {
			s.restoreDoc()
		}
	}, current, nil)
	add("Reports", "Add Current Document", func() {
		rel, _ := filepath.Rel(s.reportRoot, s.docPath)
		s.addToReport(ui.Item{Path: s.docPath, Rel: filepath.ToSlash(rel), ProjectRoot: s.reportRoot})
	}, func() bool {
		if !current() || !doc.IsMarkdown(s.docPath) {
			return false
		}
		rel, err := filepath.Rel(s.reportRoot, s.docPath)
		rep, _ := s.book.Current()
		return err == nil && filepath.IsLocal(rel) && !slices.Contains(rep.Files, filepath.ToSlash(rel))
	}, nil)

	rep, _ := s.book.Current()
	root := s.reportRoot
	for _, action := range []struct {
		title, id string
		dir       int
	}{
		{"Move Document Up", ui.MenuReportUp, -1},
		{"Move Document Down", ui.MenuReportDown, 1},
		{"Remove Document", ui.MenuReportDrop, 0},
	} {
		group := menuGroup("Reports", action.title)
		for i, rel := range rep.Files {
			enabled := func() bool {
				return current() && s.reportRoot == root && s.book.Open == rep.ID && i+action.dir >= 0 && i+action.dir < len(rep.Files)
			}
			add(group, rel, func() {
				s.onTreeMenu(ui.Item{Note: rep.ID, Rel: rel}, action.id)
				if s.viewing == rep.ID {
					s.openReport(rep.ID)
				}
			}, enabled, nil)
		}
		if len(rep.Files) == 0 {
			add(group, "No Documents", nil, disabled, nil)
		}
	}
	add("Reports", "Export Report…", func() {
		if !s.reportMode {
			s.goToSidebar(ui.SideReports)
		}
		s.openExport()
	}, func() bool {
		r, ok := s.book.Current()
		return ok && hasProject() && len(r.Files) > 0
	}, nil)
	return commands
}

func (s *shell) renameReport(name string) {
	if s.book.Rename(s.renameID, name) {
		s.saveReports()
		if s.viewing == s.renameID {
			s.openReport(s.renameID)
		} else {
			s.refresh()
		}
	}
}
