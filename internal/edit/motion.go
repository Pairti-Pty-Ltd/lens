package edit

import "unicode"

// LineStart moves to the first rune of the current line.
func (b *Buffer) LineStart() {
	b.Place(b.CaretLine(), 0)
}

// LineEnd moves to the last rune of the current line.
// past is the position after that rune, for insert mode.
func (b *Buffer) LineEnd(past bool) {
	line := b.CaretLine()
	bounds := b.lineBounds()
	if line < 0 || line >= len(bounds) {
		return
	}
	start, end := bounds[line][0], bounds[line][1]
	if past || end == start {
		b.pos = end
	} else {
		b.pos = end - 1
	}
	b.col = b.column()
}

// GoLine moves to the first rune of a 1-based line.
func (b *Buffer) GoLine(line int) {
	n := len(b.lineBounds())
	if n == 0 {
		b.pos = 0
		b.col = 0
		return
	}
	if line < 1 {
		line = 1
	}
	if line > n {
		line = n
	}
	b.Place(line-1, 0)
	b.clampView()
}

// LastLine moves to the first rune of the last line.
func (b *Buffer) LastLine() {
	b.GoLine(len(b.lineBounds()))
}

// MoveWord moves n words forward. A word is letters, digits, and underscores.
func (b *Buffer) MoveWord(n int) {
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		b.word(1)
	}
	b.clampView()
}

// MoveWordBack moves n words backward.
func (b *Buffer) MoveWordBack(n int) {
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		b.word(-1)
	}
	b.clampView()
}

// MoveWordEnd moves to the last rune of the nth word.
func (b *Buffer) MoveWordEnd(n int) {
	if n < 1 {
		n = 1
	}
	for i := 0; i < n; i++ {
		if b.pos < len(b.r) {
			b.pos++
		}
		for b.pos < len(b.r) && isSpace(b.r[b.pos]) {
			b.pos++
		}
		if b.pos >= len(b.r) {
			break
		}
		kind := wordClass(b.r[b.pos])
		for b.pos+1 < len(b.r) && wordClass(b.r[b.pos+1]) == kind && !isSpace(b.r[b.pos+1]) {
			b.pos++
		}
	}
	b.col = b.column()
	b.clampView()
}

// DeleteAt deletes n runes under the caret. A newline is left in place.
func (b *Buffer) DeleteAt(n int) {
	if b.DeleteSelection() {
		return
	}
	if n < 1 {
		n = 1
	}
	b.snapshot()
	for i := 0; i < n; i++ {
		if b.pos >= len(b.r) || b.r[b.pos] == '\n' {
			break
		}
		b.r = append(b.r[:b.pos], b.r[b.pos+1:]...)
	}
	b.clampView()
}

// DeleteLines deletes n whole lines from the caret line.
func (b *Buffer) DeleteLines(n int) {
	if n < 1 {
		n = 1
	}
	b.snapshot()
	bounds := b.lineBounds()
	line := b.CaretLine()
	if line < 0 || line >= len(bounds) {
		return
	}
	last := line + n - 1
	if last >= len(bounds) {
		last = len(bounds) - 1
	}
	from := bounds[line][0]
	to := bounds[last][1]
	if last+1 < len(bounds) {
		to++
	} else if from > 0 {
		from--
	}
	if to > len(b.r) {
		to = len(b.r)
	}
	b.r = append(b.r[:from], b.r[to:]...)
	if from > len(b.r) {
		from = len(b.r)
	}
	b.pos = from
	b.col = b.column()
	b.clampView()
}

// OpenLine inserts a line below or above the caret and sits on it.
func (b *Buffer) OpenLine(below bool) {
	if below {
		b.LineEnd(true)
		b.Insert('\n')
		return
	}
	b.LineStart()
	b.Insert('\n')
	b.Left()
}

func (b *Buffer) word(dir int) {
	if dir < 0 {
		if b.pos > 0 {
			b.pos--
		}
		for b.pos > 0 && isSpace(b.r[b.pos]) {
			b.pos--
		}
		if b.pos >= len(b.r) {
			b.col = b.column()
			return
		}
		kind := wordClass(b.r[b.pos])
		for b.pos > 0 && wordClass(b.r[b.pos-1]) == kind && !isSpace(b.r[b.pos-1]) {
			b.pos--
		}
		b.col = b.column()
		return
	}
	if b.pos >= len(b.r) {
		return
	}
	kind := wordClass(b.r[b.pos])
	if !isSpace(b.r[b.pos]) {
		for b.pos < len(b.r) && wordClass(b.r[b.pos]) == kind && !isSpace(b.r[b.pos]) {
			b.pos++
		}
	}
	for b.pos < len(b.r) && isSpace(b.r[b.pos]) {
		b.pos++
	}
	b.col = b.column()
}

func (b *Buffer) clampView() {
	if b.pos < len(b.r) && b.r[b.pos] != '\n' {
		b.col = b.column()
		return
	}
	if b.pos > 0 && (b.pos == len(b.r) || b.r[b.pos] == '\n') {
		_, col := b.lineCol()
		if col > 0 {
			b.pos--
		}
	}
	b.col = b.column()
}

func isSpace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}

func wordClass(r rune) int {
	if isSpace(r) {
		return 0
	}
	if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
		return 1
	}
	return 2
}
