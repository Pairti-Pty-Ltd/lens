// Package library is the set of project folders this install has open.
package library

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// View is one project's tree filter. The zero value hides dot-leading
// names, gitignored names, and directories with no Markdown or Typst file.
type View struct {
	ShowDot     bool `json:"showDot"`
	ShowIgnored bool `json:"showIgnored"`
	Names       bool `json:"names,omitempty"`   // file names in the tree, instead of titles
	AllDirs     bool `json:"allDirs,omitempty"` // also show directories that have no documents
}

// Library is a list of absolute project paths. Removing a path does not
// delete the folder. A path whose folder is gone stays in the list.
// Views holds a project only when a filter is turned on.
type Library struct {
	Paths   []string            `json:"paths"`
	Views   map[string]View     `json:"views,omitempty"`
	OpenRaw bool                `json:"openRaw,omitempty"`
	Tags    map[string][]string `json:"tags,omitempty"`
}

// Load reads a library. A missing file is an empty library.
func Load(path string) (*Library, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Library{}, nil
		}
		return nil, err
	}
	var lib Library
	if err := json.Unmarshal(b, &lib); err != nil {
		return nil, err
	}
	if lib.Paths == nil {
		lib.Paths = []string{}
	}
	return &lib, nil
}

// Save writes the library. The directory is created if needed.
func (l *Library) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

// Add appends an absolute, cleaned directory path. A duplicate is ignored.
// The path is kept even when the directory does not exist yet.
func (l *Library) Add(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	for _, p := range l.Paths {
		if p == abs {
			return nil
		}
	}
	l.Paths = append(l.Paths, abs)
	return nil
}

// AddTag records tag on path. A repeat is ignored.
func (l *Library) AddTag(path, tag string) {
	path = cleanPath(path)
	tag = strings.TrimSpace(tag)
	if path == "" || tag == "" {
		return
	}
	if l.Tags == nil {
		l.Tags = map[string][]string{}
	}
	for _, have := range l.Tags[path] {
		if strings.EqualFold(have, tag) {
			return
		}
	}
	l.Tags[path] = append(l.Tags[path], tag)
}

// HasTag reports whether path carries tag.
func (l *Library) HasTag(path, tag string) bool {
	path = cleanPath(path)
	for _, have := range l.Tags[path] {
		if strings.EqualFold(have, tag) {
			return true
		}
	}
	return false
}

// Remove drops a path. It does not delete the folder. An unknown path is ignored.
func (l *Library) Remove(dir string) {
	abs := cleanPath(dir)
	out := l.Paths[:0]
	for _, p := range l.Paths {
		if p != abs {
			out = append(out, p)
		}
	}
	l.Paths = out
	delete(l.Views, abs)
}

// View returns the saved filter for a project. An unknown project is hidden.
func (l *Library) View(dir string) View {
	if l == nil || l.Views == nil {
		return View{}
	}
	return l.Views[cleanPath(dir)]
}

// SetView stores the filter. Both hidden drops the entry.
func (l *Library) SetView(dir string, v View) {
	if l.Views == nil {
		l.Views = map[string]View{}
	}
	abs := cleanPath(dir)
	if !v.ShowDot && !v.ShowIgnored && !v.Names && !v.AllDirs {
		delete(l.Views, abs)
		if len(l.Views) == 0 {
			l.Views = nil
		}
		return
	}
	l.Views[abs] = v
}

func cleanPath(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return filepath.Clean(dir)
	}
	return filepath.Clean(abs)
}

// Absent reports whether path is not a directory.
func Absent(path string) bool {
	info, err := os.Stat(path)
	return err != nil || !info.IsDir()
}
