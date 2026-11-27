// Package edit is the Markdown buffer for one open file.
package edit

import "strings"

type snap struct {
	r   []rune
	pos int
}

// Buffer is the source of one document. Cursor is a rune index.
type Buffer struct {
	Modal     Modal // shared by Raw and Formatted while this document is open
	r         []rune
	pos       int
	col       int
	keep      string
	text      string
	textValid bool
	lineText  []string
	undo      []snap
	redo      []snap
	grouping  bool
	grouped   bool
	mark      int
	markEnd   int
	reg       []rune
	regLine   bool
}

// New returns a buffer whose saved text is s.
func New(s string) *Buffer {
	b := &Buffer{r: []rune(s), keep: s, text: s, textValid: true, mark: -1}
	b.col = b.column()
	return b
}

// BeginGroup coalesces following edits into one undo step.
// An insert session, from i until Escape, is one group.
func (b *Buffer) BeginGroup() {
	if b.grouping {
		return
	}
	b.grouping = true
	b.grouped = false
}

// EndGroup closes the current undo step.
func (b *Buffer) EndGroup() {
	b.grouping = false
	b.grouped = false
}

func (b *Buffer) snapshot() {
	b.textValid = false
	b.lineText = nil
	if b.grouping && b.grouped {
		return
	}
	b.undo = append(b.undo, b.capture())
	if len(b.undo) > 100 {
		b.undo = b.undo[1:]
	}
	b.redo = nil
	if b.grouping {
		b.grouped = true
	}
}

func (b *Buffer) capture() snap {
	return snap{r: append([]rune(nil), b.r...), pos: b.pos}
}

func (b *Buffer) restore(s snap) {
	b.textValid = false
	b.lineText = nil
	b.r = s.r
	if s.pos > len(b.r) {
		s.pos = len(b.r)
	}
	if s.pos < 0 {
		s.pos = 0
	}
	b.pos = s.pos
	b.col = b.column()
	b.mark = -1
}

func (b *Buffer) CanUndo() bool { return len(b.undo) > 0 }
func (b *Buffer) CanRedo() bool { return len(b.redo) > 0 }

// Undo reverses the last change and leaves it available to Redo.
func (b *Buffer) Undo() bool {
	if len(b.undo) == 0 {
		return false
	}
	b.EndGroup()
	b.redo = append(b.redo, b.capture())
	s := b.undo[len(b.undo)-1]
	b.undo = b.undo[:len(b.undo)-1]
	b.restore(s)
	return true
}

// Redo reapplies the last undone change.
func (b *Buffer) Redo() bool {
	if len(b.redo) == 0 {
		return false
	}
	b.EndGroup()
	b.undo = append(b.undo, b.capture())
	s := b.redo[len(b.redo)-1]
	b.redo = b.redo[:len(b.redo)-1]
	b.restore(s)
	return true
}

// Select marks the rune range [from, to). An empty range clears it.
func (b *Buffer) Select(from, to int) {
	if from > to {
		from, to = to, from
	}
	if from < 0 {
		from = 0
	}
	if to > len(b.r) {
		to = len(b.r)
	}
	if to <= from {
		b.mark = -1
		b.markEnd = 0
		return
	}
	b.mark = from
	b.markEnd = to
}

// Selection is the marked rune range.
func (b *Buffer) Selection() (int, int, bool) {
	if b.mark < 0 || b.markEnd <= b.mark {
		return 0, 0, false
	}
	return b.mark, b.markEnd, true
}

// DeleteSelection removes the marked range.
func (b *Buffer) DeleteSelection() bool {
	from, to, ok := b.Selection()
	if !ok {
		return false
	}
	b.snapshot()
	b.r = append(b.r[:from], b.r[to:]...)
	b.pos = from
	b.col = b.column()
	b.mark = -1
	return true
}

// InsertText inserts s as one edit.
func (b *Buffer) InsertText(s string) {
	rs := []rune(s)
	if len(rs) == 0 {
		return
	}
	b.snapshot()
	b.r = append(b.r[:b.pos], append(rs, b.r[b.pos:]...)...)
	b.pos += len(rs)
	b.col = b.column()
	b.mark = -1
}

// Text is the buffer contents.
func (b *Buffer) Text() string {
	if !b.textValid {
		b.text = string(b.r)
		b.textValid = true
	}
	return b.text
}

// Dirty reports unsaved edits.
func (b *Buffer) Dirty() bool { return b.Text() != b.keep }

// Cursor is the rune index.
func (b *Buffer) Cursor() int { return b.pos }

// Saved is the text last loaded or written.
func (b *Buffer) Saved() string { return b.keep }

// MarkSaved records the current text as what is on disk.
func (b *Buffer) MarkSaved() { b.MarkSavedText(b.Text()) }

// MarkSavedText records the snapshot acknowledged by an asynchronous save.
// Edits made while the save was in flight remain dirty.
func (b *Buffer) MarkSavedText(text string) { b.keep = text }

// SetSaved sets the on-disk text without changing the buffer.
// Dirty is then whether Text differs from that disk text.
func (b *Buffer) SetSaved(s string) { b.keep = s }

// Insert puts r before the cursor.
func (b *Buffer) Insert(r rune) {
	b.snapshot()
	b.mark = -1
	b.r = append(b.r, 0)
	copy(b.r[b.pos+1:], b.r[b.pos:])
	b.r[b.pos] = r
	b.pos++
	b.col = b.column()
}

// Backspace deletes the rune before the cursor.
func (b *Buffer) Backspace() {
	if b.DeleteSelection() {
		return
	}
	if b.pos == 0 {
		return
	}
	b.snapshot()
	b.pos--
	b.r = append(b.r[:b.pos], b.r[b.pos+1:]...)
	b.col = b.column()
}

// Delete removes the rune at the cursor.
func (b *Buffer) Delete() {
	if b.DeleteSelection() {
		return
	}
	if b.pos >= len(b.r) {
		return
	}
	b.snapshot()
	b.r = append(b.r[:b.pos], b.r[b.pos+1:]...)
}

// Left moves the cursor one rune toward the start.
func (b *Buffer) Left() {
	if b.pos > 0 {
		b.pos--
	}
	b.col = b.column()
}

// Right moves the cursor one rune toward the end.
func (b *Buffer) Right() {
	if b.pos < len(b.r) {
		b.pos++
	}
	b.col = b.column()
}

// Up moves to the previous line, keeping the column.
func (b *Buffer) Up() { b.vertical(-1) }

// Down moves to the next line, keeping the column.
func (b *Buffer) Down() { b.vertical(1) }

// Place sets the cursor to a line and column. Both are clamped.
func (b *Buffer) Place(line, col int) {
	lines := b.lineBounds()
	if len(lines) == 0 {
		b.pos = 0
		b.col = 0
		return
	}
	if line < 0 {
		line = 0
	}
	if line >= len(lines) {
		line = len(lines) - 1
	}
	start, end := lines[line][0], lines[line][1]
	if col < 0 {
		col = 0
	}
	if col > end-start {
		col = end - start
	}
	b.pos = start + col
	b.col = col
}

// Lines are the buffer split on newlines, without the newline runes.
func (b *Buffer) Lines() []string {
	if len(b.r) == 0 {
		return []string{""}
	}
	return strings.Split(b.Text(), "\n")
}

// LineText reads a one-based source line without splitting the document for every row.
func (b *Buffer) LineText(line int) string {
	if b.lineText == nil {
		b.lineText = strings.Split(b.Text(), "\n")
	}
	if line < 1 || line > len(b.lineText) {
		return ""
	}
	return b.lineText[line-1]
}

// CaretLine is the zero-based line of the cursor.
func (b *Buffer) CaretLine() int {
	line, _ := b.lineCol()
	return line
}

// CaretCol is the zero-based column of the cursor.
func (b *Buffer) CaretCol() int { return b.column() }

func (b *Buffer) vertical(dir int) {
	want := b.col
	line, _ := b.lineCol()
	b.Place(line+dir, want)
	b.col = want
}

func (b *Buffer) column() int {
	_, col := b.lineCol()
	return col
}

func (b *Buffer) lineCol() (int, int) {
	line := 0
	start := 0
	for i := 0; i < b.pos && i < len(b.r); i++ {
		if b.r[i] == '\n' {
			line++
			start = i + 1
		}
	}
	return line, b.pos - start
}

func (b *Buffer) lineBounds() [][2]int {
	var out [][2]int
	start := 0
	for i, r := range b.r {
		if r == '\n' {
			out = append(out, [2]int{start, i})
			start = i + 1
		}
	}
	out = append(out, [2]int{start, len(b.r)})
	return out
}
