package doc

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// DirEntry is one child of a directory: a folder, a Markdown file, or source.
// Listing does not recurse. A large directory is capped.
type DirEntry struct {
	Name  string
	Abs   string
	Rel   string
	IsDir bool
	Title string
	More  bool
}

const (
	maxShown   = 400
	sniffLimit = 40
)

// skipDir names are not listed and are not walked. They are large generated
// trees. Entering one, especially node_modules, is the expensive mistake.
var skipDir = map[string]struct{}{
	".git": {}, "out": {}, "node_modules": {}, "vendor": {},
	"dist": {}, "target": {}, ".cache": {}, ".next": {}, "__pycache__": {},
	"bower_components": {}, ".turbo": {}, ".parcel-cache": {}, ".yarn": {},
	".venv": {}, "venv": {}, "__pypackages__": {}, ".pnpm-store": {},
	".npm": {}, ".tox": {}, "elm-stuff": {}, ".gradle": {}, "coverage": {},
}

// SkipName reports a directory the tree and the index both leave closed.
func SkipName(name string) bool {
	_, ok := skipDir[name]
	return ok
}

// List is how one project wants its tree filtered.
// The zero value hides dot-leading names, gitignored names, and directories
// that contain no Markdown or Typst document.
type List struct {
	ShowDot     bool
	ShowIgnored bool
	AllDirs     bool // also show directories that have no documents
}

// ReadDir lists the folders and Markdown files directly inside dir.
// Nested files stay inside their folder. Titles are read only when the
// folder holds few Markdown files.
func ReadDir(dir, projectRoot string, list List) ([]DirEntry, error) {
	return readDir(dir, projectRoot, list, maxShown, sniffLimit)
}

func readDir(dir, projectRoot string, list List, show, sniff int) ([]DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var dirs, files []DirEntry
	for _, e := range entries {
		name := e.Name()
		abs := filepath.Join(dir, name)
		rel, relErr := filepath.Rel(projectRoot, abs)
		if relErr != nil {
			rel = name
		}
		rel = filepath.ToSlash(rel)
		if _, skip := skipDir[name]; skip {
			continue
		}
		if !list.ShowDot && strings.HasPrefix(name, ".") {
			continue
		}
		if e.IsDir() {
			dirs = append(dirs, DirEntry{Name: name, Abs: abs, Rel: rel, IsDir: true, Title: name})
			continue
		}
		if !IsDocument(name) && !IsSource(name) {
			continue
		}
		files = append(files, DirEntry{Name: name, Abs: abs, Rel: rel, Title: name})
	}
	if !list.AllDirs {
		kept := dirs[:0]
		for _, d := range dirs {
			if hasDocument(d.Abs, list.ShowDot) {
				kept = append(kept, d)
			}
		}
		dirs = kept
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.Slice(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	out := append(dirs, files...)
	if !list.ShowIgnored {
		out = dropIgnored(projectRoot, out)
	}
	if md := markdownCount(out); md > 0 && md <= sniff {
		for i, e := range out {
			if !e.IsDir && IsMarkdown(e.Name) {
				out[i].Title = SniffTitle(e.Abs, e.Name)
			}
		}
	}
	if len(out) > show {
		extra := len(out) - show
		out = append(out[:show], DirEntry{
			Name:  fmt.Sprintf("%d more not shown", extra),
			Title: fmt.Sprintf("%d more not shown", extra),
			More:  true,
		})
	}
	return out, nil
}

func markdownCount(entries []DirEntry) int {
	n := 0
	for _, f := range entries {
		if !f.IsDir && IsMarkdown(f.Name) {
			n++
		}
	}
	return n
}
