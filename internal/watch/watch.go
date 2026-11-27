// Package watch reports when an open file is written by another process.
package watch

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"pairti-lense/internal/doc"
)

// dirSettle waits for a burst of creates to finish. A new folder is often
// empty for a moment, and the documents-only tree hides it until the note
// inside it exists. Refreshing on the first event misses that note.
const dirSettle = 200 * time.Millisecond

// Watch reads the current file after a burst of writes settles, and notices
// when a directory that is open in the tree gains or loses a child.
type Watch struct {
	mu      sync.Mutex
	fs      *fsnotify.Watcher
	dir     string
	name    string
	gen     int
	text    string
	ready   bool
	tree    map[string]bool
	watched map[string]bool
	dirty   []string
	pending map[string]bool
	pendGen int
	timer   *time.Timer
	wake    func()
	stop    chan struct{}
}

// New starts a watcher. wake is called after a settled read and may run
// off the UI goroutine. RequestRedraw is safe for that.
func New(wake func()) (*Watch, error) {
	fs, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	w := &Watch{
		fs: fs, wake: wake, stop: make(chan struct{}),
		tree: map[string]bool{}, watched: map[string]bool{}, pending: map[string]bool{},
	}
	go w.loop()
	return w, nil
}

// Set watches path. An empty path clears the watch.
func (w *Watch) Set(path string) {
	if w == nil {
		return
	}
	w.mu.Lock()
	w.gen++
	gen := w.gen
	w.ready = false
	old := w.dir
	w.dir = ""
	w.name = ""
	if path != "" {
		w.dir = filepath.Dir(path)
		w.name = filepath.Base(path)
	}
	dir := w.dir
	w.mu.Unlock()
	if old != "" && old != dir {
		w.unwatch(old)
	}
	if dir != "" {
		w.watch(dir)
	}
	_ = gen
}

// Tree watches the directories that are open in the sidebar.
// A directory that is no longer open is dropped, except the open file's folder.
func (w *Watch) Tree(paths []string) {
	if w == nil {
		return
	}
	next := map[string]bool{}
	for _, p := range paths {
		p = filepath.Clean(p)
		if p != "" && p != "." {
			next[p] = true
		}
	}
	w.mu.Lock()
	prev := w.tree
	fileDir := w.dir
	w.tree = next
	w.mu.Unlock()
	for dir := range prev {
		if !next[dir] && dir != fileDir {
			w.unwatch(dir)
		}
	}
	for dir := range next {
		w.watch(dir)
	}
	w.mu.Lock()
	var drop []string
	for dir := range w.watched {
		if dir != fileDir && !next[dir] && !underAny(dir, next) {
			drop = append(drop, dir)
		}
	}
	w.mu.Unlock()
	for _, dir := range drop {
		w.unwatch(dir)
	}
}

// TakeDirs returns directories whose listing changed, once.
func (w *Watch) TakeDirs() []string {
	if w == nil {
		return nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.dirty) == 0 {
		return nil
	}
	out := append([]string(nil), w.dirty...)
	w.dirty = nil
	return out
}

func (w *Watch) watch(dir string) {
	w.mu.Lock()
	if w.watched[dir] {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()
	if err := w.fs.Add(dir); err != nil {
		return
	}
	w.mu.Lock()
	w.watched[dir] = true
	w.mu.Unlock()
}

func (w *Watch) unwatch(dir string) {
	w.mu.Lock()
	if !w.watched[dir] {
		w.mu.Unlock()
		return
	}
	keep := w.tree[dir] || w.dir == dir
	w.mu.Unlock()
	if keep {
		return
	}
	_ = w.fs.Remove(dir)
	w.mu.Lock()
	delete(w.watched, dir)
	w.mu.Unlock()
}

// Take returns the latest settled file text once.
func (w *Watch) Take() (string, bool) {
	if w == nil {
		return "", false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.ready {
		return "", false
	}
	w.ready = false
	return w.text, true
}

// Close stops the watcher.
func (w *Watch) Close() {
	if w == nil {
		return
	}
	close(w.stop)
	_ = w.fs.Close()
}

func (w *Watch) loop() {
	for {
		select {
		case <-w.stop:
			return
		case ev, ok := <-w.fs.Events:
			if !ok {
				return
			}
			w.onEvent(ev)
		case _, ok := <-w.fs.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watch) onEvent(ev fsnotify.Event) {
	if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return
	}
	name := filepath.Clean(ev.Name)
	parent := filepath.Clean(filepath.Dir(name))
	w.mu.Lock()
	openFile := w.name != "" && name == filepath.Join(w.dir, w.name)
	fileHit := openFile && ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename) != 0
	var gen int
	var file string
	if fileHit {
		w.gen++
		gen = w.gen
		file = filepath.Join(w.dir, w.name)
	}
	tree := false
	if ev.Op&(fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0 && w.seeingLocked(parent) {
		w.markUpLocked(parent)
		tree = true
	}
	w.mu.Unlock()
	if fileHit {
		time.AfterFunc(100*time.Millisecond, func() { w.settle(gen, file) })
	}
	if ev.Op&(fsnotify.Create|fsnotify.Rename) != 0 {
		if st, err := os.Stat(name); err == nil && st.IsDir() {
			// Watch the new folder so the note written inside it is seen.
			// The parent watch does not report grandchildren.
			w.watch(name)
			w.mu.Lock()
			w.markUpLocked(parent)
			w.mu.Unlock()
			tree = true
		}
	}
	if ev.Op&fsnotify.Write != 0 && !openFile {
		if st, err := os.Stat(name); err == nil && st.IsDir() {
			w.mu.Lock()
			if w.seeingLocked(name) {
				w.markUpLocked(name)
				tree = true
			}
			w.mu.Unlock()
		}
	}
	if ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0 {
		w.unwatch(name)
	}
	if tree {
		w.scheduleWake()
	}
}

func (w *Watch) seeingLocked(dir string) bool {
	return w.tree[dir] || w.watched[dir]
}

// markUpLocked records dir and the expanded ancestor whose listing shows it.
// The caller holds w.mu. The refresh waits until the burst settles.
func (w *Watch) markUpLocked(dir string) {
	dir = filepath.Clean(dir)
	for dir != "" && dir != "." {
		if w.tree[dir] || w.watched[dir] {
			w.pending[dir] = true
		}
		if w.tree[dir] {
			return
		}
		next := filepath.Dir(dir)
		if next == dir {
			return
		}
		dir = next
	}
}

func (w *Watch) scheduleWake() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pendGen++
	gen := w.pendGen
	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(dirSettle, func() { w.flushPending(gen) })
}

func (w *Watch) flushPending(gen int) {
	w.mu.Lock()
	if gen != w.pendGen {
		w.mu.Unlock()
		return
	}
	for dir := range w.pending {
		if !has(w.dirty, dir) {
			w.dirty = append(w.dirty, dir)
		}
	}
	w.pending = map[string]bool{}
	w.mu.Unlock()
	if w.wake != nil {
		w.wake()
	}
}

func underAny(dir string, roots map[string]bool) bool {
	for root := range roots {
		if dir == root || strings.HasPrefix(dir, root+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func has(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}

func (w *Watch) settle(gen int, path string) {
	w.mu.Lock()
	if gen != w.gen {
		w.mu.Unlock()
		return
	}
	w.mu.Unlock()
	b, err := doc.Read(path)
	if err != nil {
		return
	}
	w.mu.Lock()
	if gen != w.gen {
		w.mu.Unlock()
		return
	}
	w.text = string(b)
	w.ready = true
	w.mu.Unlock()
	if w.wake != nil {
		w.wake()
	}
}
