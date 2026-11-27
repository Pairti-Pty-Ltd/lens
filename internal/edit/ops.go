package edit

// DeleteSpan removes runes [from, to) as one undo step.
func (b *Buffer) DeleteSpan(from, to int) bool {
	from, to = order(from, to, len(b.r))
	if from == to {
		return false
	}
	b.snapshot()
	b.r = append(b.r[:from], b.r[to:]...)
	b.pos = from
	b.clampView()
	return true
}

// CopySpan stores [from, to) for paste. linewise pastes as whole lines.
func (b *Buffer) CopySpan(from, to int, linewise bool) {
	from, to = order(from, to, len(b.r))
	if linewise && to > from && b.r[to-1] == '\n' {
		to--
	}
	b.reg = append([]rune(nil), b.r[from:to]...)
	b.regLine = linewise
}

// Put inserts the last copied text after or before the caret.
func (b *Buffer) Put(after bool) bool {
	if len(b.reg) == 0 && !b.regLine {
		return false
	}
	text := append([]rune(nil), b.reg...)
	if b.regLine {
		line := b.lineBounds()[b.CaretLine()]
		at := line[0]
		first := at
		if after && line[1] == len(b.r) {
			at = line[1]
			first = at + 1
			text = append([]rune{'\n'}, text...)
		} else {
			if after {
				at = line[1] + 1
			}
			first = at
			text = append(text, '\n')
		}
		b.splice(at, text)
		b.pos = first
		b.firstText()
		b.clampView()
		return true
	}
	at := b.pos
	if after && at < len(b.r) && b.r[at] != '\n' {
		at++
	}
	b.splice(at, text)
	if after && b.pos > 0 {
		b.pos--
	}
	b.clampView()
	return true
}

// ReplaceOne overwrites the rune under the caret.
func (b *Buffer) ReplaceOne(r rune) bool {
	if b.pos >= len(b.r) || b.r[b.pos] == '\n' || r == 0 || r == '\n' {
		return false
	}
	b.snapshot()
	b.r[b.pos] = r
	return true
}

// JoinLine pulls the next line up, with a space between.
func (b *Buffer) JoinLine() bool {
	bounds := b.lineBounds()
	line := b.CaretLine()
	if line < 0 || line+1 >= len(bounds) {
		return false
	}
	nl := bounds[line][1]
	if nl >= len(b.r) || b.r[nl] != '\n' {
		return false
	}
	b.snapshot()
	next := nl + 1
	for next < len(b.r) && (b.r[next] == ' ' || b.r[next] == '\t') {
		next++
	}
	gap := []rune{' '}
	if bounds[line][1] == bounds[line][0] {
		gap = nil
	}
	b.r = append(b.r[:nl], append(gap, b.r[next:]...)...)
	b.pos = nl
	if len(gap) == 0 && b.pos > bounds[line][0] {
		b.pos--
	}
	b.clampView()
	return true
}

// ToggleCase flips the rune under the caret and steps right.
func (b *Buffer) ToggleCase(n int) bool {
	if n < 1 {
		n = 1
	}
	if b.pos >= len(b.r) {
		return false
	}
	b.snapshot()
	changed := false
	for i := 0; i < n && b.pos < len(b.r); i++ {
		r := b.r[b.pos]
		if r == '\n' {
			break
		}
		switch {
		case r >= 'a' && r <= 'z':
			b.r[b.pos] = r - 'a' + 'A'
			changed = true
		case r >= 'A' && r <= 'Z':
			b.r[b.pos] = r - 'A' + 'a'
			changed = true
		}
		if b.pos+1 < len(b.r) && b.r[b.pos+1] != '\n' {
			b.pos++
		}
	}
	b.col = b.column()
	if !changed {
		b.Undo()
		return false
	}
	return true
}

func (b *Buffer) splice(at int, add []rune) {
	if at < 0 {
		at = 0
	}
	if at > len(b.r) {
		at = len(b.r)
	}
	b.snapshot()
	b.r = append(b.r[:at], append(append([]rune(nil), add...), b.r[at:]...)...)
	b.pos = at + len(add)
	b.col = b.column()
	b.mark = -1
}

func order(from, to, n int) (int, int) {
	if from > to {
		from, to = to, from
	}
	if from < 0 {
		from = 0
	}
	if to > n {
		to = n
	}
	if to < from {
		to = from
	}
	return from, to
}

// MotionSpan is the rune range a view-mode motion covers.
// linewise ranges include the trailing newline of the last line.
func (b *Buffer) MotionSpan(kind rune, n int, arg rune) (from, to int, linewise, ok bool) {
	if n < 1 {
		n = 1
	}
	pos := b.pos
	switch kind {
	case '{', '}':
		b.paragraph(kind == '}', n)
		return min(pos, b.pos), max(pos, b.pos), false, b.pos != pos
	case '%':
		if !b.matchingBracket() {
			return pos, pos, false, false
		}
		return min(pos, b.pos), max(pos, b.pos) + 1, false, true
	case 'h':
		for i := 0; i < n; i++ {
			if b.pos == 0 || b.r[b.pos-1] == '\n' {
				break
			}
			b.pos--
		}
		return b.pos, pos, false, b.pos != pos
	case 'l':
		for i := 0; i < n && b.pos < len(b.r) && b.r[b.pos] != '\n'; i++ {
			b.pos++
		}
		return pos, b.pos, false, b.pos != pos
	case 'j', 'k':
		line := b.CaretLine()
		for i := 0; i < n; i++ {
			if kind == 'j' {
				b.Down()
			} else {
				b.Up()
			}
		}
		return b.lineSpan(line, b.CaretLine())
	case 'w', 'W':
		for i := 0; i < n; i++ {
			b.wordStep(1, kind == 'W')
		}
		return pos, b.pos, false, b.pos != pos
	case 'b', 'B':
		for i := 0; i < n; i++ {
			b.wordStep(-1, kind == 'B')
		}
		return b.pos, pos, false, b.pos != pos
	case 'e', 'E':
		for i := 0; i < n; i++ {
			b.wordEnd(kind == 'E')
		}
		end := b.pos + 1
		if end > len(b.r) {
			end = len(b.r)
		}
		return pos, end, false, end != pos
	case '0':
		b.LineStart()
		return b.pos, pos, false, b.pos != pos
	case '^':
		b.firstText()
		return b.pos, pos, false, b.pos != pos
	case '$':
		if n > 1 {
			b.GoLine(b.CaretLine() + n)
		}
		b.LineEnd(true)
		return pos, b.pos, false, b.pos != pos
	case 'G':
		line := b.CaretLine()
		b.GoLine(n)
		return b.lineSpan(line, b.CaretLine())
	case 'g':
		line := b.CaretLine()
		b.GoLine(1)
		return b.lineSpan(line, b.CaretLine())
	case 'f', 'F', 't', 'T':
		return b.findSpan(kind, arg, n, pos)
	default:
		return 0, 0, false, false
	}
}

func (b *Buffer) lineSpan(a, c int) (from, to int, linewise, ok bool) {
	bounds := b.lineBounds()
	if len(bounds) == 0 {
		return 0, 0, true, false
	}
	if a > c {
		a, c = c, a
	}
	if a < 0 {
		a = 0
	}
	if c >= len(bounds) {
		c = len(bounds) - 1
	}
	from = bounds[a][0]
	to = bounds[c][1]
	if c+1 < len(bounds) {
		to++
	}
	if to > len(b.r) {
		to = len(b.r)
	}
	return from, to, true, from != to
}

func (b *Buffer) findSpan(kind rune, arg rune, n, origin int) (from, to int, linewise, ok bool) {
	if arg == 0 || arg == '\n' {
		return 0, 0, false, false
	}
	line := b.CaretLine()
	bounds := b.lineBounds()
	if line < 0 || line >= len(bounds) {
		return 0, 0, false, false
	}
	start, end := bounds[line][0], bounds[line][1]
	pos := origin
	switch kind {
	case 'f', 't':
		for i := 0; i < n; i++ {
			j := pos + 1
			found := -1
			for ; j < end; j++ {
				if b.r[j] == arg {
					found = j
					break
				}
			}
			if found < 0 {
				return 0, 0, false, false
			}
			pos = found
		}
		to = pos
		if kind == 'f' {
			to = pos + 1
		}
		b.pos = to - 1
		b.col = b.column()
		return origin, to, false, to > origin
	default:
		for i := 0; i < n; i++ {
			j := pos - 1
			found := -1
			for ; j >= start; j-- {
				if b.r[j] == arg {
					found = j
					break
				}
			}
			if found < 0 {
				return 0, 0, false, false
			}
			pos = found
		}
		from = pos
		if kind == 'T' {
			from = pos + 1
		}
		b.pos = from
		b.col = b.column()
		return from, origin, false, from < origin
	}
}

func (b *Buffer) firstText() {
	b.LineStart()
	bounds := b.lineBounds()
	line := b.CaretLine()
	if line < 0 || line >= len(bounds) {
		return
	}
	end := bounds[line][1]
	for b.pos < end && (b.r[b.pos] == ' ' || b.r[b.pos] == '\t') {
		b.pos++
	}
	b.col = b.column()
}

func (b *Buffer) wordStep(dir int, word bool) {
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
		for b.pos > 0 && sameWord(b.r[b.pos-1], b.r[b.pos], word) && !isSpace(b.r[b.pos-1]) {
			b.pos--
		}
		b.col = b.column()
		return
	}
	if b.pos >= len(b.r) {
		return
	}
	if !isSpace(b.r[b.pos]) {
		r := b.r[b.pos]
		for b.pos < len(b.r) && sameWord(r, b.r[b.pos], word) && !isSpace(b.r[b.pos]) {
			b.pos++
		}
	}
	for b.pos < len(b.r) && isSpace(b.r[b.pos]) {
		b.pos++
	}
	b.col = b.column()
}

func (b *Buffer) wordEnd(word bool) {
	if b.pos < len(b.r) {
		b.pos++
	}
	for b.pos < len(b.r) && isSpace(b.r[b.pos]) {
		b.pos++
	}
	if b.pos >= len(b.r) {
		return
	}
	r := b.r[b.pos]
	for b.pos+1 < len(b.r) && sameWord(r, b.r[b.pos+1], word) && !isSpace(b.r[b.pos+1]) {
		b.pos++
	}
	b.col = b.column()
}

func sameWord(a, c rune, word bool) bool {
	if word {
		return !isSpace(a) && !isSpace(c)
	}
	return wordClass(a) == wordClass(c)
}

// changeWordEnd stops at the end of the current word, preserving the space
// that separates the replacement from the following word.
func (b *Buffer) changeWordEnd(n int, big bool) int {
	pos := b.pos
	for i := 0; i < n && pos < len(b.r); i++ {
		if i > 0 {
			for pos < len(b.r) && isSpace(b.r[pos]) {
				pos++
			}
		}
		if pos == len(b.r) {
			break
		}
		start := pos
		for pos < len(b.r) && sameWord(b.r[start], b.r[pos], big) {
			pos++
		}
	}
	return pos
}

func (b *Buffer) wordObject(around, big bool, n int) (int, int) {
	if b.pos >= len(b.r) || b.r[b.pos] == '\n' {
		return b.pos, b.pos
	}
	from := b.pos
	for from > 0 && b.r[from-1] != '\n' && sameWord(b.r[from-1], b.r[b.pos], big) {
		from--
	}
	to := b.changeWordEnd(n, big)
	if around {
		end := to
		for to < len(b.r) && (b.r[to] == ' ' || b.r[to] == '\t') {
			to++
		}
		if to == end {
			for from > 0 && (b.r[from-1] == ' ' || b.r[from-1] == '\t') {
				from--
			}
		}
	}
	return from, to
}
