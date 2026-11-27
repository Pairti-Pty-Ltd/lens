// Package agent is the in-editor MCP surface for the focused document.
//
// A tool edit changes the open buffer and leaves it unsaved. It does not write
// the file. An agent that writes the file itself is picked up by the existing
// file watcher: a clean buffer adopts the bytes, and a dirty buffer asks.
package agent

import (
	"fmt"
	"strings"

	"pairti-lense/internal/edit"
)

// Heading is one outline row.
type Heading struct {
	Number string `json:"number"`
	Text   string `json:"text"`
	Level  int    `json:"level"`
	Line   int    `json:"line"`
}

// Tab is one open document.
type Tab struct {
	Path    string `json:"path"`
	Title   string `json:"title"`
	Dirty   bool   `json:"dirty"`
	Focused bool   `json:"focused"`
}

// Snapshot is the focused document, for an agent that is co-authoring it.
type Snapshot struct {
	Path          string    `json:"path"`
	Rel           string    `json:"rel"`
	Project       string    `json:"project"`
	Title         string    `json:"title"`
	Dirty         bool      `json:"dirty"`
	Mode          string    `json:"mode"`
	Input         string    `json:"input"`
	MIME          string    `json:"mime"`
	Words         int       `json:"words"`
	Chars         int       `json:"chars"`
	Line          int       `json:"line"`
	Col           int       `json:"col"`
	Cursor        int       `json:"cursor"`
	HasSelection  bool      `json:"hasSelection"`
	SelectionFrom int       `json:"selectionFrom"`
	SelectionTo   int       `json:"selectionTo"`
	Selection     string    `json:"selection"`
	Selected      string    `json:"selected"`
	SelectedKind  string    `json:"selectedKind"`
	CurrentLine   string    `json:"currentLine"`
	Around        string    `json:"around"`
	Outline       []Heading `json:"outline"`
	Tabs          []Tab     `json:"tabs"`
	Text          string    `json:"text"`
}

// ReadIn chooses how much of the caret neighbourhood to return.
// The full text is always included. Radius is lines on each side of the caret.
// Zero means 20.
type ReadIn struct {
	Radius int `json:"radius,omitempty" jsonschema:"lines of context on each side of the caret"`
}

// ApplyIn replaces the focused document, or a rune span inside it.
// Set ReplaceAll, or set Span with From and To. From and To are rune offsets
// and To is exclusive. Text is the replacement. Expect, when set, must match
// the current document or span or the edit is refused.
type ApplyIn struct {
	Path       string `json:"path,omitempty" jsonschema:"absolute path; must be the focused document when set"`
	ReplaceAll bool   `json:"replaceAll,omitempty" jsonschema:"replace the whole document with text"`
	Span       bool   `json:"span,omitempty" jsonschema:"replace the rune span from,to with text"`
	From       int    `json:"from,omitempty" jsonschema:"rune offset where the span starts"`
	To         int    `json:"to,omitempty" jsonschema:"rune offset where the span ends, exclusive"`
	Text       string `json:"text,omitempty" jsonschema:"replacement text"`
	Expect     string `json:"expect,omitempty" jsonschema:"current document or span text; refuse the edit when it differs"`
}

// ApplyOut is the caret after an edit.
type ApplyOut struct {
	Path   string `json:"path"`
	Dirty  bool   `json:"dirty"`
	Line   int    `json:"line"`
	Col    int    `json:"col"`
	Cursor int    `json:"cursor"`
	Chars  int    `json:"chars"`
}

// Window is the numbered lines around a 1-based line.
func Window(text string, line, radius int) string {
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}
	if radius < 0 {
		radius = 0
	}
	if radius > 80 {
		radius = 80
	}
	if line < 1 {
		line = 1
	}
	if line > len(lines) {
		line = len(lines)
	}
	from := line - radius
	if from < 1 {
		from = 1
	}
	to := line + radius
	if to > len(lines) {
		to = len(lines)
	}
	var b strings.Builder
	for i := from; i <= to; i++ {
		if i > from {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%d|%s", i, lines[i-1])
	}
	return b.String()
}

// Apply edits b as one undo step. changed is false when the text is already
// the requested result. The file on disk is not touched.
func Apply(b *edit.Buffer, in ApplyIn) (bool, error) {
	if b == nil {
		return false, fmt.Errorf("no document open")
	}
	if in.ReplaceAll == in.Span {
		return false, fmt.Errorf("set replaceAll or span")
	}
	text := b.Text()
	runes := []rune(text)
	if in.ReplaceAll {
		if in.Expect != "" && in.Expect != text {
			return false, fmt.Errorf("document changed")
		}
		if in.Text == text {
			return false, nil
		}
		replace(b, 0, len(runes), in.Text)
		return true, nil
	}
	from, to := in.From, in.To
	if from < 0 || to < from || to > len(runes) {
		return false, fmt.Errorf("span %d:%d outside 0:%d", from, to, len(runes))
	}
	cur := string(runes[from:to])
	if in.Expect != "" && in.Expect != cur {
		return false, fmt.Errorf("document changed")
	}
	if cur == in.Text {
		return false, nil
	}
	replace(b, from, to, in.Text)
	return true, nil
}

func replace(b *edit.Buffer, from, to int, text string) {
	b.EndGroup()
	b.BeginGroup()
	if to > from {
		b.Select(from, to)
		b.DeleteSelection()
	} else {
		place(b, from)
	}
	if text != "" {
		b.InsertText(text)
	}
	b.EndGroup()
}

func place(b *edit.Buffer, off int) {
	line, col := 0, 0
	for i, r := range []rune(b.Text()) {
		if i >= off {
			break
		}
		if r == '\n' {
			line++
			col = 0
			continue
		}
		col++
	}
	b.Place(line, col)
}
