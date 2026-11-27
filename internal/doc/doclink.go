package doc

import (
	"path/filepath"
	"strings"
)

// IsDocLink reports a doc:/// URI. The path after the scheme is a file
// under the project's doc directory.
func IsDocLink(raw string) bool {
	_, ok := docRel(raw)
	return ok
}

// DocFile is the file a doc:/// URI names. root is the project folder.
// A path that climbs out of doc/ is refused.
func DocFile(root, raw string) (string, bool) {
	rel, ok := docRel(raw)
	if !ok || strings.TrimSpace(root) == "" {
		return "", false
	}
	rel = filepath.Clean(filepath.FromSlash(rel))
	if rel == "." || rel == "" || filepath.IsAbs(rel) || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.Join(root, "doc", rel), true
}

func docRel(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "<>")
	if len(raw) < 4 || !strings.EqualFold(raw[:4], "doc:") {
		return "", false
	}
	rest := raw[4:]
	rest = strings.TrimLeft(rest, "/")
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	rest = strings.TrimRight(rest, ".,;:")
	rest = strings.TrimSpace(rest)
	if rest == "" || strings.ContainsAny(rest, " \t\r\n") {
		return "", false
	}
	return rest, true
}

// Linkify turns bare doc:/// URIs in text into links. Other words stay plain.
func Linkify(text string) []Span {
	if !strings.Contains(strings.ToLower(text), "doc:") {
		if text == "" {
			return nil
		}
		return []Span{{Text: text}}
	}
	return splitDocURIs(text)
}

// splitDocURIs turns bare doc:/// URIs in text into links. The surrounding
// words stay plain. A code span is not passed here.
func splitDocURIs(text string) []Span {
	lower := strings.ToLower(text)
	var out []Span
	i := 0
	for i < len(text) {
		j := strings.Index(lower[i:], "doc:")
		if j < 0 {
			out = append(out, Span{Text: text[i:]})
			break
		}
		j += i
		if j > i {
			out = append(out, Span{Text: text[i:j]})
		}
		k := j + 4
		for k < len(text) && docRune(text[k]) {
			k++
		}
		uri := strings.TrimRight(text[j:k], ".,;:")
		if !strings.Contains(uri, "/") {
			out = append(out, Span{Text: text[j:k]})
			i = k
			continue
		}
		out = append(out, Span{Text: uri, Link: uri})
		i = j + len(uri)
		if i < k {
			out = append(out, Span{Text: text[i:k]})
			i = k
		}
	}
	if len(out) == 0 && text != "" {
		return []Span{{Text: text}}
	}
	return out
}

func docRune(r byte) bool {
	switch r {
	case ' ', '\t', '\n', '\r', ',', ')', ']', '(', '[', '"', '\'', '<', '>', '`':
		return false
	default:
		return true
	}
}
