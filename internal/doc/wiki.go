package doc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode"
)

// A wiki link is [[target]], [[target|label]], or [[target#heading]].
// The target is a note title or a file name, the way Obsidian resolves it.

type wikiPage struct {
	path  string
	title string
	stem  string
}

var wikiCache sync.Map

// ForgetWiki drops the name index for a project so the next link sees new notes.
func ForgetWiki(root string) {
	if root == "" {
		return
	}
	wikiCache.Delete(filepath.Clean(root))
}

// WikiFile finds the note named by an Obsidian link target.
// root is the project folder. from is the note that contains the link.
func WikiFile(root, from, name string) (string, bool) {
	name = strings.TrimSpace(name)
	root = strings.TrimSpace(root)
	if name == "" || root == "" {
		return "", false
	}
	root = filepath.Clean(root)
	if strings.ContainsAny(name, `/\`) {
		if p, ok := wikiPath(root, from, name); ok {
			return p, true
		}
	}
	return wikiMatch(root, from, name, true)
}

func wikiMatch(root, from, name string, retry bool) (string, bool) {
	wantFold := foldName(name)
	wantSlug := slug(name)
	fromDir := filepath.Dir(from)
	bestPath := ""
	bestScore := 0
	bestRel := 0
	bestSame := false
	for _, p := range wikiPages(root) {
		score := 0
		if p.title != "" && foldName(p.title) == wantFold {
			score = 4
		} else if foldName(p.stem) == wantFold {
			score = 3
		} else if wantSlug != "" && (slug(p.title) == wantSlug || slug(p.stem) == wantSlug) {
			score = 2
		}
		if score == 0 {
			continue
		}
		same := filepath.Dir(p.path) == fromDir
		rel := len(p.path)
		if score > bestScore || (score == bestScore && same && !bestSame) || (score == bestScore && same == bestSame && (bestPath == "" || rel < bestRel)) {
			bestScore = score
			bestPath = p.path
			bestRel = rel
			bestSame = same
		}
	}
	if bestPath == "" {
		return "", false
	}
	if _, err := os.Stat(bestPath); err != nil && retry {
		ForgetWiki(root)
		return wikiMatch(root, from, name, false)
	}
	return bestPath, true
}

// CreateWiki writes a new note for a link that names nothing yet.
// The file is dir/slug.md and starts with the title. An existing file is left as it is.
func CreateWiki(dir, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || strings.ContainsAny(name, `/\`) || strings.ContainsRune(name, 0) {
		return "", fmt.Errorf("missing link")
	}
	base := slug(name)
	if base == "" {
		base = "note"
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, base+".md")
	for n := 2; ; n++ {
		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			return "", err
		}
		path = filepath.Join(dir, fmt.Sprintf("%s-%d.md", base, n))
	}
	title := strings.ReplaceAll(name, "\"", "'")
	body := "---\ntitle: " + title + "\n---\n\n# " + name + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func wikiPages(root string) []wikiPage {
	if v, ok := wikiCache.Load(root); ok {
		return v.([]wikiPage)
	}
	var pages []wikiPage
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (SkipName(name) || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !IsMarkdown(name) {
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > MaxBytes || info.Size() == 0 {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		buf := make([]byte, 8192)
		n, _ := f.Read(buf)
		f.Close()
		src := buf[:n]
		title := Meta(src)["title"]
		if title == "" {
			title = Title(src)
		}
		stem := strings.TrimSuffix(name, filepath.Ext(name))
		pages = append(pages, wikiPage{path: path, title: title, stem: stem})
		return nil
	})
	wikiCache.Store(root, pages)
	return pages
}

func wikiPath(root, from, name string) (string, bool) {
	name = filepath.Clean(filepath.FromSlash(name))
	if name == "." || name == "" || filepath.IsAbs(name) || name == ".." || strings.HasPrefix(name, ".."+string(filepath.Separator)) {
		return "", false
	}
	fromDir := root
	if from != "" {
		fromDir = filepath.Dir(from)
	}
	cands := []string{
		filepath.Join(root, name),
		filepath.Join(fromDir, name),
	}
	if !strings.HasSuffix(strings.ToLower(name), ".md") && !strings.HasSuffix(strings.ToLower(name), ".markdown") {
		cands = append(cands, filepath.Join(root, name+".md"), filepath.Join(fromDir, name+".md"))
	}
	for _, c := range cands {
		c = filepath.Clean(c)
		if !strings.HasPrefix(c, root+string(filepath.Separator)) && c != root {
			continue
		}
		info, err := os.Stat(c)
		if err == nil && !info.IsDir() {
			return c, true
		}
	}
	return "", false
}

func foldName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func slug(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			b.WriteRune(r)
			dash = false
			continue
		}
		if !dash && b.Len() > 0 {
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

// splitWiki turns [[target]] and [[target|label]] into links.
// The link value is wiki:target or wiki:target#heading. A code span is not passed here.
func splitWiki(text string) []Span {
	var out []Span
	i := 0
	for i < len(text) {
		j := strings.Index(text[i:], "[[")
		if j < 0 {
			out = append(out, Span{Text: text[i:]})
			break
		}
		j += i
		if j > i && text[j-1] == '\\' {
			out = append(out, Span{Text: text[i:j-1] + "[["})
			i = j + 2
			continue
		}
		start := j
		if j > i && text[j-1] == '!' {
			start = j - 1
		}
		endRel := strings.Index(text[j+2:], "]]")
		if endRel < 0 {
			out = append(out, Span{Text: text[i:]})
			break
		}
		end := j + 2 + endRel
		if start > i {
			out = append(out, Span{Text: text[i:start]})
		}
		target, heading, label := wikiParts(text[j+2 : end])
		if target == "" {
			out = append(out, Span{Text: text[start : end+2]})
			i = end + 2
			continue
		}
		link := "wiki:" + target
		if heading != "" {
			link += "#" + heading
		}
		out = append(out, Span{Text: label, Link: link})
		i = end + 2
	}
	if len(out) == 0 && text != "" {
		return []Span{{Text: text}}
	}
	return out
}

func wikiParts(body string) (target, heading, label string) {
	body = strings.TrimSpace(body)
	if i := strings.Index(body, "|"); i >= 0 {
		label = strings.TrimSpace(body[i+1:])
		body = strings.TrimSpace(body[:i])
	}
	if h := strings.Index(body, "#"); h >= 0 {
		heading = strings.TrimSpace(strings.TrimPrefix(body[h+1:], "^"))
		body = strings.TrimSpace(body[:h])
	}
	target = body
	if label == "" {
		label = target
	}
	return target, heading, label
}
