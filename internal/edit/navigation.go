package edit

import (
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"
)

func reverseFind(kind rune) rune {
	switch kind {
	case 'f':
		return 'F'
	case 'F':
		return 'f'
	case 't':
		return 'T'
	default:
		return 't'
	}
}

func (m *Modal) Prompt() rune {
	if m.prompt == '/' || m.prompt == '?' {
		return m.prompt
	}
	return ':'
}

func (m *Modal) submit(b *Buffer, cmd string) Result {
	if m.prompt == '/' || m.prompt == '?' {
		m.Mode = m.searchMode
		if cmd != "" {
			m.query = cmd
			m.wholeWord = false
		}
		m.backward = m.prompt == '?'
		m.search(b, m.backward, 1)
		m.RefreshSelection(b)
		return Result{Moved: true}
	}
	cmd = strings.TrimSpace(cmd)
	if len(cmd) > 0 && strings.ContainsRune("dy", rune(cmd[len(cmd)-1])) {
		spec := strings.TrimSpace(cmd[:len(cmd)-1])
		if first, last, ok := m.lineRange(b, spec); ok {
			b.Select(0, 0)
			b.Place(first-1, 0)
			return m.lineOp(b, cmd[len(cmd)-1], last-first+1)
		}
	}
	if cmd == "$" {
		b.LastLine()
		return Result{Moved: true}
	}
	if line, err := strconv.Atoi(strings.TrimSpace(cmd)); err == nil && line >= 0 {
		b.GoLine(line)
		return Result{Moved: true}
	}
	return Result{Cmd: cmd, Moved: true}
}

// Search visits literal matches in source order, wrapping around the buffer.
func (m *Modal) search(b *Buffer, backward bool, count int) {
	if m.query == "" {
		return
	}
	text := b.Text()
	var hits []int
	pos := 0
	for len(text) > 0 {
		i := strings.Index(text, m.query)
		if i < 0 {
			break
		}
		pos += utf8.RuneCountInString(text[:i])
		end := pos + utf8.RuneCountInString(m.query)
		if !m.wholeWord || ((pos == 0 || wordClass(b.r[pos-1]) != 1) && (end == len(b.r) || wordClass(b.r[end]) != 1)) {
			hits = append(hits, pos)
		}
		pos = end
		text = text[i+len(m.query):]
	}
	if len(hits) == 0 {
		return
	}
	i := sort.SearchInts(hits, b.pos+1) + count - 1
	if backward {
		i = sort.SearchInts(hits, b.pos) - count
	}
	i = (i%len(hits) + len(hits)) % len(hits)
	b.pos = hits[i]
	b.col = b.column()
}

func (m *Modal) RefreshSelection(b *Buffer) {
	if m.Mode == ModeVisual || m.Mode == ModeVisualLine {
		m.selectVisual(b)
	}
}

// StartCommand opens a command or search prompt in the current view.
func (m *Modal) StartCommand(prompt rune) {
	m.clearPending()
	m.searchMode = m.Mode
	m.prompt, m.Mode, m.Cmd = prompt, ModeCommand, ""
}

// Drag selects the half-open interval between two insertion positions.
func (m *Modal) Drag(b *Buffer, anchor, end int) {
	m.Reset()
	b.EndGroup()
	if anchor == end {
		b.Select(0, 0)
		return
	}
	m.Mode = ModeVisual
	m.anchor = anchor
	b.pos = end
	if end > anchor {
		b.pos--
	} else {
		m.anchor--
	}
	b.col = b.column()
	m.selectVisual(b)
}

func (b *Buffer) paragraph(forward bool, n int) {
	bounds := b.lineBounds()
	line := b.CaretLine()
	dir := 1
	if !forward {
		dir = -1
	}
	for ; n > 0; n-- {
		prevBlank := bounds[line][0] == bounds[line][1]
		for line+dir >= 0 && line+dir < len(bounds) {
			line += dir
			blank := bounds[line][0] == bounds[line][1]
			if blank && !prevBlank {
				break
			}
			prevBlank = blank
		}
	}
	b.Place(line, 0)
	if forward && line == len(bounds)-1 {
		b.LineEnd(true)
	}
}

func (b *Buffer) matchingBracket() bool {
	const pairs = "()[]{}"
	end := b.lineBounds()[b.CaretLine()][1]
	start := b.pos
	for start < end && !strings.ContainsRune(pairs, b.r[start]) {
		start++
	}
	if start == end {
		return false
	}
	a := b.r[start]
	index := strings.IndexRune(pairs, a)
	z := rune(pairs[index^1])
	dir := 1
	if index%2 == 1 {
		dir = -1
	}
	depth := 1
	for i := start + dir; i >= 0 && i < len(b.r); i += dir {
		switch b.r[i] {
		case a:
			depth++
		case z:
			depth--
			if depth == 0 {
				b.pos = i
				b.col = b.column()
				return true
			}
		}
	}
	return false
}

func (m *Modal) lineRange(b *Buffer, spec string) (int, int, bool) {
	end := len(b.lineBounds())
	if spec == "%" {
		return 1, end, true
	}
	if spec == "'<,'>" && (m.searchMode == ModeVisual || m.searchMode == ModeVisualLine) {
		return m.visualFirst + 1, m.visualLast + 1, true
	}
	address := func(s string) (int, bool) {
		switch strings.TrimSpace(s) {
		case ".":
			return b.CaretLine() + 1, true
		case "$":
			return end, true
		}
		n, err := strconv.Atoi(strings.TrimSpace(s))
		return n, err == nil && n >= 1 && n <= end
	}
	a, z, multiple := strings.Cut(spec, ",")
	first, ok := address(a)
	if !ok || !multiple {
		return first, first, ok
	}
	last, ok := address(z)
	return first, last, ok && last >= first
}
