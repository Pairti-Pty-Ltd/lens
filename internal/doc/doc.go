// Package doc lists Markdown files in a project and turns them into blocks.
package doc

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	// MaxBytes is the largest file the viewer will load.
	MaxBytes = 1 << 20
	// sniffBytes is how much of a file is read to find a title.
	sniffBytes = 64 << 10
	// segmentLimit is the longest path segment shown whole.
	segmentLimit = 40
)

// Project is one folder in the library.
type Project struct {
	Name   string
	Root   string
	Absent bool
	Docs   []Document
}

// Document is one Markdown file.
type Document struct {
	Abs   string
	Rel   string
	Title string
}

// Span is one styled run inside a block.
// Tone is a source-token role: comment, keyword, string, num, punct.
// Empty means the reading colour.
type Span struct {
	Text   string
	Bold   bool
	Italic bool
	Code   bool
	Strike bool
	Marked bool
	Link   string
	Image  bool
	Note   bool // a footnote reference, drawn smaller than the sentence
	Tone   string
}

// Block is one rendered piece of a Markdown file.
// Text is the plain reading of Spans. Marker is a list marker or a code language.
// Line is the 1-based source line. Lines is how many source lines the block covers.
// A following row with Line 0 belongs to the same source range (a table body row).
type Block struct {
	Kind           string
	Text           string
	Spans          []Span
	Cells          []string // table row, one entry per column
	CellSpans      [][]Span // inline Markdown in each table cell
	QuoteLine      int      // first content line of the containing blockquote, zero outside
	GridParagraphs []Block  // one-column Pandoc cell, with document-relative lines
	GridRows       []Block  // rows of a multi-column Pandoc table
	CellLinks      []string // wiki or other link for that column, or ""
	Depth          int
	Marker         string
	Line           int
	Lines          int
}

// TooLargeError is returned when a file is over MaxBytes.
type TooLargeError struct {
	Size int64
}

func (e *TooLargeError) Error() string {
	return fmt.Sprintf("%d bytes is over the 1 MiB limit", e.Size)
}

// Walk lists Markdown documents under root. A missing root is an absent project.
func Walk(root string) Project {
	name := filepath.Base(root)
	p := Project{Name: name, Root: root}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		p.Absent = true
		return p
	}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d == nil {
			return nil
		}
		if d.IsDir() {
			if path != root && (d.Name() == ".git" || d.Name() == "out") {
				return filepath.SkipDir
			}
			return nil
		}
		lower := strings.ToLower(d.Name())
		if !strings.HasSuffix(lower, ".md") && !strings.HasSuffix(lower, ".markdown") {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			rel = d.Name()
		}
		p.Docs = append(p.Docs, Document{
			Abs:   path,
			Rel:   filepath.ToSlash(rel),
			Title: SniffTitle(path, d.Name()),
		})
		return nil
	})
	return p
}

// SniffTitle is the first ATX heading, otherwise filename.
func SniffTitle(path, filename string) string {
	f, err := os.Open(path)
	if err != nil {
		return filename
	}
	defer f.Close()
	buf := make([]byte, sniffBytes)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return filename
	}
	if title := Title(buf[:n]); title != "" {
		return title
	}
	return filename
}

// Title returns the first ATX heading in src, or "" when there is none.
func Title(src []byte) string {
	for _, line := range strings.Split(string(src), "\n") {
		line = strings.TrimRight(line, "\r")
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" || trimmed[0] != '#' {
			continue
		}
		i := 0
		for i < len(trimmed) && trimmed[i] == '#' {
			i++
		}
		if i == 0 || i > 6 || i == len(trimmed) || trimmed[i] != ' ' {
			continue
		}
		text := strings.TrimSpace(trimmed[i:])
		text = strings.TrimRight(text, " #")
		text = strings.TrimSpace(text)
		if text != "" {
			return text
		}
	}
	return ""
}

// Read loads a Markdown file. Files over MaxBytes are refused.
func Read(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxBytes {
		return nil, &TooLargeError{Size: info.Size()}
	}
	return os.ReadFile(path)
}

// ShortPath cuts any segment longer than 40 runes in the middle.
func ShortPath(rel string) string {
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	for i, part := range parts {
		parts[i] = cutSegment(part)
	}
	return strings.Join(parts, "/")
}

func cutSegment(s string) string {
	r := []rune(s)
	if len(r) <= segmentLimit {
		return s
	}
	return string(r[:18]) + "…" + string(r[len(r)-18:])
}

// Wrap breaks text on spaces so each line is at most maxChars runes.
func Wrap(s string, maxChars int) []string {
	if maxChars < 8 {
		maxChars = 8
	}
	s = strings.ReplaceAll(s, "\r\n", "\n")
	var lines []string
	for _, para := range strings.Split(s, "\n") {
		if strings.TrimSpace(para) == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, wrapLine(para, maxChars)...)
	}
	return lines
}

func wrapLine(s string, maxChars int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var cur string
	for _, w := range words {
		if utf8.RuneCountInString(w) > maxChars {
			if cur != "" {
				lines = append(lines, cur)
				cur = ""
			}
			r := []rune(w)
			for len(r) > maxChars {
				lines = append(lines, string(r[:maxChars]))
				r = r[maxChars:]
			}
			cur = string(r)
			continue
		}
		if cur == "" {
			cur = w
			continue
		}
		if utf8.RuneCountInString(cur)+1+utf8.RuneCountInString(w) > maxChars {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur += " " + w
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}
