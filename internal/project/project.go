// Package project is one folder exported as a single document.
// The file order and template live in .pairti-project.json.
package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"pairti-lense/internal/doc"
)

const fileName = ".pairti-project.json"

// Settings is the export order and the brand used when the project is exported.
type Settings struct {
	Files    []string `json:"files"`
	Template string   `json:"template,omitempty"`
}

// Load reads the project file. A missing file is empty settings.
// Corrupt JSON is an error. The caller keeps going with the files on disk.
func Load(root string) (Settings, error) {
	b, err := os.ReadFile(filepath.Join(root, fileName))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Settings{}, nil
		}
		return Settings{}, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return Settings{}, nil
	}
	var s Settings
	if err := json.Unmarshal(b, &s); err != nil {
		return Settings{}, err
	}
	return s, nil
}

// Save writes the project file.
func Save(root string, s Settings) error {
	clean := Settings{Template: strings.TrimSpace(s.Template)}
	for _, rel := range s.Files {
		rel = cleanRel(rel)
		if rel != "" {
			clean.Files = append(clean.Files, rel)
		}
	}
	b, err := json.MarshalIndent(clean, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(filepath.Join(root, fileName), b, 0o644)
}

// Order is the saved list, skipping anything missing, then markdown files
// on disk that are not in the list yet.
func Order(root string, saved []string) ([]string, error) {
	found, err := scan(root)
	if err != nil {
		return nil, err
	}
	have := map[string]bool{}
	var out []string
	for _, rel := range saved {
		rel = cleanRel(rel)
		if rel == "" || have[rel] {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() || !doc.IsMarkdown(abs) {
			continue
		}
		have[rel] = true
		out = append(out, rel)
	}
	for _, rel := range found {
		if !have[rel] {
			out = append(out, rel)
		}
	}
	return out, nil
}

// Join reads the files in order into one Markdown document.
// The first file keeps its front matter. Later front matter is removed.
// A missing file is skipped.
func Join(root string, rels []string) ([]byte, error) {
	var b strings.Builder
	n := 0
	for _, rel := range rels {
		rel = cleanRel(rel)
		if rel == "" {
			continue
		}
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		if n > 0 {
			src = stripFront(src)
			b.WriteString("\n\n")
		}
		b.Write(retargetImages(src, filepath.Dir(filepath.Join(root, filepath.FromSlash(rel))), root))
		n++
	}
	if n == 0 {
		return nil, fmt.Errorf("no documents in the project")
	}
	return []byte(b.String()), nil
}

// retargetImages rewrites a relative image to a path from the report root,
// so a later file's picture is not resolved against the first file.
func retargetImages(src []byte, fromDir, root string) []byte {
	text := string(src)
	seen := map[string]bool{}
	var links []string
	for _, block := range doc.Blocks(src) {
		for _, sp := range block.Spans {
			if !sp.Image || sp.Link == "" || seen[sp.Link] {
				continue
			}
			lower := strings.ToLower(sp.Link)
			if strings.Contains(lower, "://") || strings.HasPrefix(lower, "data:") {
				continue
			}
			seen[sp.Link] = true
			links = append(links, sp.Link)
		}
	}
	sort.Slice(links, func(i, j int) bool { return len(links[i]) > len(links[j]) })
	for _, link := range links {
		abs := filepath.Clean(filepath.Join(fromDir, filepath.FromSlash(link)))
		if filepath.IsAbs(link) {
			abs = filepath.Clean(link)
		}
		rel, err := filepath.Rel(root, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		rel = filepath.ToSlash(rel)
		if rel == link {
			continue
		}
		text = replaceImageLink(text, link, rel)
	}
	return []byte(text)
}

func replaceImageLink(text, old, new string) string {
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(text[i:], old)
		if j < 0 {
			b.WriteString(text[i:])
			break
		}
		j += i
		prefix := text[:j]
		if strings.HasSuffix(prefix, "](") || strings.HasSuffix(prefix, "src=\"") || strings.HasSuffix(prefix, "src='") || strings.HasSuffix(prefix, "]: ") {
			b.WriteString(text[i:j])
			b.WriteString(new)
			i = j + len(old)
			continue
		}
		b.WriteString(text[i : j+len(old)])
		i = j + len(old)
	}
	return b.String()
}

func scan(root string) ([]string, error) {
	var out []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (skipDir(name) || strings.HasPrefix(name, ".")) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(name, ".") || !doc.IsMarkdown(path) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return nil
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	sort.Strings(out)
	return out, err
}

func skipDir(name string) bool {
	switch name {
	case ".git", "out", "node_modules", "vendor", "dist", "target", ".cache", ".next", "__pycache__":
		return true
	default:
		return false
	}
}

func cleanRel(rel string) string {
	rel = strings.TrimSpace(filepath.ToSlash(rel))
	rel = strings.TrimPrefix(rel, "./")
	if rel == "" || rel == "." || strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
		return ""
	}
	return rel
}

func stripFront(src []byte) []byte {
	s := string(src)
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return src
	}
	rest := s
	if strings.HasPrefix(rest, "---\r\n") {
		rest = rest[5:]
	} else {
		rest = rest[4:]
	}
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return src
	}
	body := rest[idx+4:]
	body = strings.TrimPrefix(body, "\r\n")
	body = strings.TrimPrefix(body, "\n")
	return []byte(body)
}
