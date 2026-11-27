package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/gogpu/gogpu"
	"pairti-lense/internal/ui"
)

const maxRecentPaths = 20

type fileActions struct {
	pick         func(gogpu.FileDialogOptions) ([]string, error)
	newPath      func(gogpu.FileDialogOptions) (string, error)
	reveal       func(string) error
	confirmClose func(string) closeChoice
}

type closeChoice int

const (
	cancelClose closeChoice = iota
	saveClose
	discardClose
)

func (s *shell) fileMenuCommands() []menuCommand {
	document := func() bool {
		return s.docPath != "" && s.buf != nil && s.pending == "" && s.diff == nil && s.viewing == ""
	}
	commands := []menuCommand{
		{"File", "New", "n", menuCommandKey, s.newFile, nil, nil},
		{"File", "Open Document…", "o", menuCommandKey, func() { s.chooseOpenPath(false) }, nil, nil},
		{"File", "Open Project Folder…", "o", menuCommandKey | menuShift, func() { s.chooseOpenPath(true) }, nil, nil},
	}
	counts := make(map[string]int)
	for _, path := range s.recent {
		counts[filepath.Base(path)]++
	}
	group := menuGroup("File", "Open Recent")
	for _, path := range s.recent {
		label := filepath.Base(path)
		if counts[label] > 1 || label == "Clear Menu" {
			label = path
		}
		commands = append(commands, menuCommand{group: group, title: label, run: func() { s.openPath(path) }})
	}
	commands = append(commands,
		menuCommand{group: group, title: "Clear Menu", run: func() { s.setRecent(nil) }, enabled: func() bool { return len(s.recent) > 0 }},
		menuCommand{"File", "Close Tab", "w", menuCommandKey, func() { s.closeFileTab(s.focusedIndex()) }, func() bool { return s.focusedIndex() >= 0 }, nil},
		menuCommand{"File", "Close Window", "w", menuCommandKey | menuShift, func() {
			if s.quit != nil {
				s.quit()
			}
		}, nil, nil},
		menuCommand{"File", "Save", "s", menuCommandKey, s.saveDoc, document, nil},
		menuCommand{"File", "Export Document…", "", 0, func() { s.openExportScope(false) }, document, nil},
		menuCommand{"File", "Remove Project from Library", "", 0, s.removeChosen, func() bool { return s.chosen != "" && slices.Contains(s.libPaths(), s.chosen) }, nil},
		menuCommand{"File", "Reveal in Finder", "", 0, func() {
			if err := s.files.reveal(s.docPath); err != nil {
				s.fileError(err)
			}
		}, func() bool { return s.docPath != "" }, nil},
	)
	return commands
}

func (s *shell) rememberPath(path string) {
	if s.holdSave || path == "" {
		return
	}
	path = filepath.Clean(path)
	if len(s.recent) > 0 && s.recent[0] == path {
		return
	}
	next := []string{path}
	for _, old := range s.recent {
		if old != path {
			next = append(next, old)
		}
		if len(next) == maxRecentPaths {
			break
		}
	}
	s.setRecent(next)
}

func (s *shell) setRecent(paths []string) {
	s.recent = paths
	s.invalidateMenus()
	s.saveSession()
}

func (s *shell) fileError(err error) {
	if s.line3 != nil {
		s.line3.Set(err.Error())
		s.writeFooter()
	}
	s.redraw()
}

func (s *shell) chooseOpenPath(directory bool) {
	opts := gogpu.FileDialogOptions{Title: "Open Document", Multiple: true, Filters: []gogpu.FileTypeFilter{{Name: "Markdown", Extensions: []string{"*.md", "*.markdown"}}}}
	if directory {
		opts = gogpu.FileDialogOptions{Title: "Open Project Folder", Directory: true}
	}
	paths, err := s.files.pick(opts)
	if err != nil {
		s.fileError(err)
		return
	}
	for _, p := range paths {
		s.openPath(p)
	}
}

// File pickers, Open Recent and Finder all use this entry point.
func (s *shell) openPath(path string) {
	if path == "" {
		return
	}
	path, err := filepath.Abs(path)
	if err != nil {
		s.fileError(err)
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) && slices.Contains(s.recent, path) {
			s.setRecent(slices.DeleteFunc(s.recent, func(p string) bool { return p == path }))
		}
		s.fileError(err)
		return
	}
	if info.IsDir() {
		s.goToSidebar(ui.SideFiles)
		s.add(path)
		if s.side != nil {
			s.side.Select(path)
		}
		return
	}
	if !info.Mode().IsRegular() {
		s.fileError(fmt.Errorf("cannot open %s", path))
		return
	}
	if s.reportMode || s.histMode {
		s.setSide(ui.SideFiles)
	}
	root, name, rel := s.placeOf(path)
	s.open(ui.Item{Path: path, ProjectRoot: root, ProjectName: name, Rel: rel})
}

func (s *shell) newFile() {
	if s.chosen != "" {
		s.newDocIn(s.chosen)
		return
	}
	if s.projectRoot() != "" {
		s.newDocIn(s.projectRoot())
		return
	}
	path, err := s.files.newPath(gogpu.FileDialogOptions{Title: "New Document", DefaultFilename: "untitled.md"})
	if err != nil {
		s.fileError(err)
		return
	}
	if path == "" {
		return
	}
	if filepath.Ext(path) == "" {
		path += ".md"
	}
	if err := createNote(path); err != nil {
		s.fileError(err)
		return
	}
	s.openPath(path)
}

func createNote(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, err = f.WriteString("# \n\n")
	return errors.Join(err, f.Close())
}

func (s *shell) canCloseFile(i int) bool {
	if i < 0 || i >= len(s.docs) {
		return false
	}
	d := s.docs[i]
	if d.buf == nil || !d.buf.Dirty() {
		return true
	}
	s.show(i)
	s.line3.Set("Unsaved: " + filepath.Base(d.path))
	s.writeFooter()
	s.redraw()
	if s.files.confirmClose == nil {
		return false
	}
	switch s.files.confirmClose(d.path) {
	case saveClose:
		s.saveDoc()
		return !d.buf.Dirty()
	case discardClose:
		return true
	}
	return false
}

func (s *shell) closeFileTab(i int) {
	if s.canCloseFile(i) {
		s.dropDoc(i)
	}
}
