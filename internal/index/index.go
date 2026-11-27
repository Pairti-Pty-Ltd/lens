// Package index is the lexical note index. Markdown on disk stays the source.
// A query ranks title, headings, and body and returns one snippet.
package index

import (
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"unicode"

	"pairti-lense/internal/doc"
)

const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// Hit is one ranked note. Score stays off the palette row.
type Hit struct {
	Path    string
	Root    string
	Rel     string
	Title   string
	Snippet string
}

type posting struct {
	doc int
	tf  int
}

type record struct {
	path, root, rel string
	title           string
	head            string
	body            string
	mtime           int64
	size            int64
	len             int
	tf              map[string]int
}

// Index answers keyword queries over the open projects.
type Index struct {
	mu   sync.RWMutex
	docs []*record
	post map[string][]posting
	df   map[string]int
	avg  float64
}

// New returns an empty index.
func New() *Index {
	return &Index{post: map[string][]posting{}, df: map[string]int{}}
}

// Sync reads Markdown under root. Unchanged files stay. Cache directories
// such as node_modules are not opened.
func (x *Index) Sync(root string) {
	root = filepath.Clean(root)
	seen := map[string]os.FileInfo{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (doc.SkipName(name) || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !doc.IsMarkdown(name) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > doc.MaxBytes {
			return nil
		}
		seen[path] = info
		return nil
	})

	x.mu.Lock()
	fresh := x.docs[:0]
	for _, rec := range x.docs {
		if rec.root != root {
			fresh = append(fresh, rec)
			continue
		}
		info, ok := seen[rec.path]
		if !ok {
			continue
		}
		if info.ModTime().UnixNano() == rec.mtime && info.Size() == rec.size {
			fresh = append(fresh, rec)
			delete(seen, rec.path)
		}
	}
	x.docs = fresh
	x.mu.Unlock()

	for path, info := range seen {
		b, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		x.mu.Lock()
		x.putLocked(path, root, b, info.ModTime().UnixNano(), info.Size())
		x.mu.Unlock()
	}
	x.reindex()
}

// Put replaces one note, usually after a save.
func (x *Index) Put(path, root string, src []byte) {
	info, err := os.Stat(path)
	mtime, size := int64(0), int64(len(src))
	if err == nil {
		mtime = info.ModTime().UnixNano()
		size = info.Size()
	}
	x.mu.Lock()
	x.putLocked(path, root, src, mtime, size)
	x.mu.Unlock()
	x.reindex()
}

// Drop removes one path and any note under it.
func (x *Index) Drop(path string) {
	path = filepath.Clean(path)
	x.mu.Lock()
	fresh := x.docs[:0]
	prefix := path + string(filepath.Separator)
	for _, rec := range x.docs {
		if rec.path == path || strings.HasPrefix(rec.path, prefix) {
			continue
		}
		fresh = append(fresh, rec)
	}
	x.docs = fresh
	x.mu.Unlock()
	x.reindex()
}

// Keep drops notes whose project is no longer open.
func (x *Index) Keep(roots []string) {
	set := map[string]bool{}
	for _, r := range roots {
		set[filepath.Clean(r)] = true
	}
	x.mu.Lock()
	fresh := x.docs[:0]
	for _, rec := range x.docs {
		if set[rec.root] {
			fresh = append(fresh, rec)
		}
	}
	x.docs = fresh
	x.mu.Unlock()
	x.reindex()
}

func (x *Index) putLocked(path, root string, src []byte, mtime, size int64) {
	path = filepath.Clean(path)
	root = filepath.Clean(root)
	rel, err := filepath.Rel(root, path)
	if err != nil {
		rel = filepath.Base(path)
	}
	title, heads, body := splitNote(src, filepath.Base(path))
	rec := &record{
		path: path, root: root, rel: filepath.ToSlash(rel),
		title: title, head: heads, body: body,
		mtime: mtime, size: size,
	}
	rec.tf, rec.len = weighted(title, heads, body)
	replaced := false
	for i, old := range x.docs {
		if old.path == path {
			x.docs[i] = rec
			replaced = true
			break
		}
	}
	if !replaced {
		x.docs = append(x.docs, rec)
	}
}

func (x *Index) reindex() {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.post = map[string][]posting{}
	x.df = map[string]int{}
	var total int
	for id, rec := range x.docs {
		total += rec.len
		for term, tf := range rec.tf {
			x.post[term] = append(x.post[term], posting{doc: id, tf: tf})
			x.df[term]++
		}
	}
	if len(x.docs) == 0 {
		x.avg = 0
		return
	}
	x.avg = float64(total) / float64(len(x.docs))
}

// Search ranks notes for a keyword query. The last word matches a prefix
// so a half-typed word still hits. At most limit hits come back.
func (x *Index) Search(query string, limit int) []Hit {
	terms := tokens(query)
	if len(terms) == 0 || limit <= 0 {
		return nil
	}
	x.mu.RLock()
	defer x.mu.RUnlock()
	if len(x.docs) == 0 {
		return nil
	}
	n := float64(len(x.docs))
	type acc struct {
		score float64
		got   int
	}
	score := map[int]*acc{}
	for i, term := range terms {
		words := []string{term}
		if i == len(terms)-1 && len(term) >= 2 {
			words = x.expand(term)
		}
		seenDoc := map[int]bool{}
		for _, word := range words {
			df := float64(x.df[word])
			if df == 0 {
				continue
			}
			idf := math.Log(1 + (n-df+0.5)/(df+0.5))
			for _, p := range x.post[word] {
				if seenDoc[p.doc] {
					continue
				}
				seenDoc[p.doc] = true
				dl := float64(x.docs[p.doc].len)
				if dl < 1 {
					dl = 1
				}
				avg := x.avg
				if avg < 1 {
					avg = dl
				}
				tf := float64(p.tf)
				denom := tf + bm25K1*(1-bm25B+bm25B*dl/avg)
				add := idf * (tf * (bm25K1 + 1)) / denom
				a := score[p.doc]
				if a == nil {
					a = &acc{}
					score[p.doc] = a
				}
				a.score += add
				a.got++
			}
		}
	}
	if len(score) == 0 {
		return nil
	}
	all := 0
	for _, a := range score {
		if a.got == len(terms) {
			all++
		}
	}
	type pair struct {
		id    int
		score float64
	}
	var rows []pair
	for id, a := range score {
		if all > 0 && a.got != len(terms) {
			continue
		}
		rows = append(rows, pair{id, a.score + float64(a.got)})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].score == rows[j].score {
			return x.docs[rows[i].id].title < x.docs[rows[j].id].title
		}
		return rows[i].score > rows[j].score
	})
	if len(rows) > limit {
		rows = rows[:limit]
	}
	out := make([]Hit, len(rows))
	for i, row := range rows {
		rec := x.docs[row.id]
		out[i] = Hit{
			Path: rec.path, Root: rec.root, Rel: rec.rel, Title: rec.title,
			Snippet: snippet(rec.body, rec.head, terms),
		}
	}
	return out
}

func (x *Index) expand(prefix string) []string {
	exact := false
	var out []string
	for term := range x.post {
		if term == prefix {
			exact = true
		}
		if strings.HasPrefix(term, prefix) {
			out = append(out, term)
		}
	}
	sort.Strings(out)
	if len(out) > 32 {
		out = out[:32]
	}
	if exact && !hasTerm(out, prefix) {
		out = append(out, prefix)
	}
	if len(out) == 0 {
		return []string{prefix}
	}
	return out
}

func hasTerm(list []string, term string) bool {
	for _, s := range list {
		if s == term {
			return true
		}
	}
	return false
}

func splitNote(src []byte, name string) (title, heads, body string) {
	text := string(src)
	if meta := doc.Meta(src); meta["title"] != "" {
		title = strings.TrimSpace(meta["title"])
	}
	var b strings.Builder
	var h strings.Builder
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		trim := strings.TrimSpace(line)
		if n, rest, ok := heading(trim); ok {
			if title == "" && n == 1 {
				title = rest
			}
			if h.Len() > 0 {
				h.WriteByte('\n')
			}
			h.WriteString(rest)
			continue
		}
		if trim == "---" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(line)
	}
	if title == "" {
		title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	return title, h.String(), strings.TrimSpace(b.String())
}

func heading(line string) (int, string, bool) {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 || i > 6 || i == len(line) || line[i] != ' ' {
		return 0, "", false
	}
	return i, strings.TrimSpace(line[i:]), true
}

func weighted(title, heads, body string) (map[string]int, int) {
	tf := map[string]int{}
	add := func(text string, w int) {
		for _, tok := range tokens(text) {
			tf[tok] += w
		}
	}
	add(title, 4)
	add(heads, 2)
	add(body, 1)
	n := 0
	for _, c := range tf {
		n += c
	}
	return tf, n
}

func tokens(s string) []string {
	var out []string
	var b strings.Builder
	flush := func() {
		if b.Len() == 0 {
			return
		}
		tok := b.String()
		b.Reset()
		if len(tok) < 2 || stop(tok) {
			return
		}
		out = append(out, tok)
	}
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return out
}

func stop(s string) bool {
	switch s {
	case "the", "a", "an", "of", "and", "to", "in", "for", "on", "at":
		return true
	default:
		return false
	}
}

func snippet(body, heads string, terms []string) string {
	text := strings.Join(strings.Fields(body), " ")
	if text == "" {
		text = strings.Join(strings.Fields(heads), " ")
	}
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	at := -1
	for _, term := range terms {
		if i := strings.Index(lower, term); i >= 0 && (at < 0 || i < at) {
			at = i
		}
	}
	runes := []rune(text)
	focus := 0
	if at > 0 && at <= len(text) {
		focus = len([]rune(text[:at]))
	}
	start := focus - 40
	if start < 0 {
		start = 0
	}
	end := start + 90
	if end > len(runes) {
		end = len(runes)
	}
	s := string(runes[start:end])
	if start > 0 {
		s = "…" + s
	}
	if end < len(runes) {
		s += "…"
	}
	return s
}
