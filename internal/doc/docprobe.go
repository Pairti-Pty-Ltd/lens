package doc

import (
	"os"
	"path/filepath"
	"sync"
	"time"
)

// A directory probe answers whether a folder holds a Markdown or Typst
// file. The answer is cached. A repeat stats the directories already
// seen and does not read them again. skipDir names are never opened.
type probe struct {
	dir     string
	mtime   time.Time
	showDot bool
	has     bool
	found   string
	subs    []subStamp
}

type subStamp struct {
	path  string
	mtime time.Time
}

var docProbes sync.Map

func hasDocument(dir string, showDot bool) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	if e, ok := docProbes.Load(dir); ok {
		p := e.(probe)
		if p.live(info.ModTime(), showDot) {
			return p.has
		}
	}
	has, found, subs := walkDocs(dir, showDot)
	docProbes.Store(dir, probe{
		dir: dir, mtime: info.ModTime(), showDot: showDot,
		has: has, found: found, subs: subs,
	})
	return has
}

func (p probe) live(mtime time.Time, showDot bool) bool {
	if p.showDot != showDot || !p.mtime.Equal(mtime) {
		return false
	}
	if p.has {
		if p.found == "" {
			return false
		}
		_, err := os.Stat(p.found)
		return err == nil
	}
	for _, sub := range p.subs {
		e, ok := docProbes.Load(sub.path)
		if !ok {
			return false
		}
		child := e.(probe)
		info, err := os.Stat(sub.path)
		if err != nil || !info.IsDir() || !child.live(info.ModTime(), showDot) || child.has {
			return false
		}
	}
	return true
}

func walkDocs(dir string, showDot bool) (has bool, found string, subs []subStamp) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false, "", nil
	}
	var kids []os.DirEntry
	for _, e := range entries {
		name := e.Name()
		if _, skip := skipDir[name]; skip {
			continue
		}
		if !showDot && stringsHasDot(name) {
			continue
		}
		if e.IsDir() {
			kids = append(kids, e)
			continue
		}
		if IsDocument(name) {
			return true, filepath.Join(dir, name), nil
		}
	}
	for _, e := range kids {
		abs := filepath.Join(dir, e.Name())
		info, err := e.Info()
		if err != nil || !info.IsDir() {
			continue
		}
		childHas, childFound, childSubs := walkDocs(abs, showDot)
		docProbes.Store(abs, probe{
			dir: abs, mtime: info.ModTime(), showDot: showDot,
			has: childHas, found: childFound, subs: childSubs,
		})
		if childHas {
			return true, childFound, nil
		}
		subs = append(subs, subStamp{path: abs, mtime: info.ModTime()})
	}
	return false, "", subs
}

func stringsHasDot(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
