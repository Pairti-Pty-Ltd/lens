package edit

import "strings"

// Modes cover navigation, insertion, selection, and the command line.
const (
	ModeView = iota
	ModeEdit
	ModeCommand
	ModeVisual
	ModeVisualLine
)

// Key is one keypress. Code is set for keys that are not text.
// Rows is the visible line count, used by page motions.
type Key struct {
	Rune  rune
	Code  int
	Rows  int
	Super bool
	Ctrl  bool
	Shift bool
}

// Codes for keys that are not inserted as text.
const (
	CodeNone = iota
	CodeEsc
	CodeEnter
	CodeBackspace
	CodeDelete
	CodeLeft
	CodeRight
	CodeUp
	CodeDown
	CodeHome
	CodeEnd
)

// Result is what a key did to the buffer.
type Result struct {
	Changed bool
	Moved   bool
	Save    bool
	Cmd     string
	// Scroll is how many lines the view moves. Positive is toward the end.
	Screen int // signed wrapped-row motion
	Scroll int
	// Pull keeps the caret inside the view after a scroll that did not move it.
	Pull bool
	// Place is zz, zt, zb, H, M, or L. The editor owns the viewport.
	Place rune
	// Win selects a pane after Ctrl-W followed by Ctrl-H, Ctrl-L, or Ctrl-W.
	Win rune
}

// Modal is the view, edit, and command state for one buffer.
//
// View mode follows vim's operator-then-motion shape:
//
//	motions    h j k l w b e W B E 0 ^ $ gg G f F t T
//	delete     d{motion} dd D x X
//	change     c{motion} cc C s S
//	yank/put   y{motion} yy p P
//	other      r J ~ . u zz zt zb H M L
//	page       Ctrl-F Ctrl-B Ctrl-D Ctrl-U Ctrl-E Ctrl-Y
//	range      Ctrl-W then a motion, or v / V
//	insert     i a I A o O
type Modal struct {
	Mode                    int
	Cmd                     string
	count                   int
	pending                 byte
	op                      byte
	opN                     int
	seek                    byte
	sequence                []Key
	lastChange              []Key
	changed                 bool
	visualChange            bool
	object                  rune
	anchor                  int
	prompt                  rune
	query                   string
	backward                bool
	wholeWord               bool
	lastFind, lastTarget    rune
	searchMode              int
	visualFirst, visualLast int
	firstCol, lastCol       int
}

// undoChord reports a modifier chord for undo or redo.
// Undo is Cmd-Z or Ctrl-Z. Redo is Cmd-Shift-Z, Ctrl-Shift-Z, Ctrl-R, or Ctrl-Y.
func undoChord(k Key) (undo, redo, ok bool) {
	switch k.Rune {
	case 'z':
		if k.Shift {
			return false, true, true
		}
		return true, false, true
	case 'Z':
		return false, true, true
	case 'y', 'Y':
		if k.Ctrl {
			return false, true, true
		}
	case 'r', 'R':
		if k.Ctrl && !k.Super {
			return false, true, true
		}
	}
	return false, false, false
}

// Name returns the user-facing mode.
func (m Modal) Name() string {
	switch m.Mode {
	case ModeEdit:
		return "Edit"
	case ModeVisual:
		return "Visual"
	case ModeVisualLine:
		return "Visual Line"
	case ModeCommand:
		return "Command"
	default:
		return "View"
	}
}

// Reset returns to view mode.
func (m *Modal) Reset() {
	*m = Modal{lastChange: m.lastChange, query: m.query, backward: m.backward, wholeWord: m.wholeWord, lastFind: m.lastFind, lastTarget: m.lastTarget}
}

// Handle applies one key. A false return means the key was not used.
func (m *Modal) Handle(b *Buffer, k Key) (Result, bool) {
	if b == nil || (k.Code == CodeNone && k.Rune == 0) {
		// Native platforms send a physical press followed by a text event.
		// Only the latter carries a printable Vim command.
		return Result{}, false
	}
	if m.Mode == ModeView && m.op == 0 && m.pending == 0 && m.seek == 0 && k.Rune == '.' && !k.Ctrl && !k.Super {
		count := max(1, m.count)
		m.clearPending()
		m.sequence = nil
		var result Result
		for i := 0; i < count && len(m.lastChange) > 0; i++ {
			for _, key := range m.lastChange {
				r, _ := m.handle(b, key)
				result.Changed = result.Changed || r.Changed
			}
		}
		return result, true
	}
	insertWordErase := m.Mode == ModeEdit && k.Ctrl && !k.Super && k.Rune == 'w'
	wasVisual := m.Mode == ModeVisual || m.Mode == ModeVisualLine
	wasCommand := m.Mode == ModeCommand
	undo := m.Mode == ModeView && m.seek == 0 && m.op == 0 && k.Rune == 'u'
	res, used := m.handle(b, k)
	if !used {
		return res, false
	}
	if wasVisual {
		m.sequence = nil
		m.visualChange = true
	}
	if !wasVisual && !k.Super && (!k.Ctrl || insertWordErase) && !wasCommand && m.Mode != ModeCommand {
		m.sequence = append(m.sequence, k)
		if res.Changed && !undo {
			m.changed = true
		}
	}
	if m.Mode == ModeView && m.op == 0 && m.seek == 0 && m.pending == 0 && m.count == 0 {
		if m.changed && !m.visualChange {
			m.lastChange = append(m.lastChange[:0], m.sequence...)
		}
		m.sequence = nil
		m.changed = false
		m.visualChange = false
	}
	return res, true
}

func (m *Modal) clearPending() {
	m.op, m.pending, m.seek, m.object, m.count, m.opN = 0, 0, 0, 0, 0, 0
}

func (m *Modal) handle(b *Buffer, k Key) (Result, bool) {
	if m.Mode == ModeVisual && m.pending == 'W' {
		m.pending = 0
		if k.Ctrl && !k.Super && (k.Rune == 'h' || k.Rune == 'l' || k.Rune == 'w') {
			m.Mode = ModeView
			b.Select(0, 0)
			return Result{Win: k.Rune}, true
		}
	}
	if k.Ctrl && !k.Super && (m.Mode == ModeVisual || m.Mode == ModeVisualLine) && strings.ContainsRune("fbduye", k.Rune) {
		res, used := m.viewCtrl(b, k)
		m.selectVisual(b)
		return res, used
	}
	if k.Ctrl && !k.Super && m.Mode == ModeView {
		if res, ok := m.viewCtrl(b, k); ok {
			return res, true
		}
	}
	if k.Ctrl && !k.Super && m.Mode == ModeEdit && k.Rune == 'w' {
		end := b.pos
		start := end
		for start > 0 && (b.r[start-1] == ' ' || b.r[start-1] == '\t') {
			start--
		}
		if start == end && start > 0 && b.r[start-1] == '\n' {
			start--
		} else if start > 0 && b.r[start-1] != '\n' {
			class := wordClass(b.r[start-1])
			for start > 0 && wordClass(b.r[start-1]) == class && b.r[start-1] != '\n' {
				start--
			}
		}
		changed := b.DeleteSpan(start, end)
		b.pos = start
		b.col = b.column()
		return Result{Changed: changed}, true
	}
	if k.Super || k.Ctrl {
		if k.Rune == 's' || k.Rune == 'S' {
			return Result{Save: true}, true
		}
		if undo, redo, ok := undoChord(k); ok {
			if m.Mode == ModeVisual || m.Mode == ModeVisualLine {
				m.Mode = ModeView
				b.Select(0, 0)
			}
			b.EndGroup()
			n := m.count
			if n < 1 {
				n = 1
			}
			m.count = 0
			changed := false
			for i := 0; i < n; i++ {
				var did bool
				if redo {
					did = b.Redo()
				} else if undo {
					did = b.Undo()
				}
				if !did {
					break
				}
				changed = true
			}
			return Result{Changed: changed, Moved: true}, true
		}
		return Result{}, false
	}
	switch m.Mode {
	case ModeVisual, ModeVisualLine:
		return m.visual(b, k), true
	case ModeCommand:
		return m.command(b, k), true
	case ModeEdit:
		return m.insert(b, k), true
	default:
		return m.normal(b, k), true
	}
}

func (m *Modal) insert(b *Buffer, k Key) Result {
	switch k.Code {
	case CodeEsc:
		b.EndGroup()
		m.Mode = ModeView
		if b.CaretCol() > 0 {
			b.Left()
		}
		b.clampView()
		return Result{Moved: true}
	case CodeEnter:
		b.Insert('\n')
		return Result{Changed: true}
	case CodeBackspace:
		b.Backspace()
		return Result{Changed: true}
	case CodeDelete:
		b.Delete()
		return Result{Changed: true}
	case CodeHome:
		b.LineStart()
		return Result{Moved: true}
	case CodeEnd:
		b.LineEnd(true)
		return Result{Moved: true}
	case CodeLeft:
		b.Left()
		return Result{Moved: true}
	case CodeRight:
		b.Right()
		return Result{Moved: true}
	case CodeUp:
		b.Up()
		return Result{Moved: true}
	case CodeDown:
		b.Down()
		return Result{Moved: true}
	}
	if k.Rune == 0 || k.Rune < 32 {
		return Result{}
	}
	b.Insert(k.Rune)
	return Result{Changed: true}
}

func (m *Modal) command(b *Buffer, k Key) Result {
	switch k.Code {
	case CodeEsc:
		m.Mode = m.searchMode
		m.Cmd = ""
		return Result{Moved: true}
	case CodeEnter:
		cmd := m.Cmd
		m.Mode = ModeView
		m.Cmd = ""
		return m.submit(b, cmd)
	case CodeBackspace:
		r := []rune(m.Cmd)
		if len(r) > 0 {
			m.Cmd = string(r[:len(r)-1])
		}
		return Result{Moved: true}
	}
	if k.Rune == 0 || k.Rune < 32 {
		return Result{}
	}
	m.Cmd += string(k.Rune)
	return Result{Moved: true}
}

// viewCtrl is the common ctrl chords in view mode.
// Ctrl-F and Ctrl-B move a page. Ctrl-D and Ctrl-U move half a page.
// Ctrl-E and Ctrl-Y scroll one line. Ctrl-W starts a character range.
func (m *Modal) viewCtrl(b *Buffer, k Key) (Result, bool) {
	rows := k.Rows
	if rows < 1 {
		rows = 20
	}
	page := rows - 2
	if page < 1 {
		page = 1
	}
	half := rows / 2
	if half < 1 {
		half = 1
	}
	switch k.Rune {
	case 'f':
		return m.shift(b, page), true
	case 'b':
		return m.shift(b, -page), true
	case 'd':
		return m.shift(b, half), true
	case 'u':
		return m.shift(b, -half), true
	case 'e':
		return Result{Scroll: 1, Pull: true}, true
	case 'y':
		return Result{Scroll: -1, Pull: true}, true
	case 'w':
		m.clearPending()
		m.Mode = ModeVisual
		m.anchor = b.pos
		m.selectVisual(b)
		m.pending = 'W'
		return Result{Moved: true}, true
	default:
		return Result{}, false
	}
}

func (m *Modal) shift(b *Buffer, lines int) Result {
	if lines > 0 {
		for i := 0; i < lines; i++ {
			b.Down()
		}
	} else {
		for i := 0; i < -lines; i++ {
			b.Up()
		}
	}
	m.count = 0
	return Result{Moved: true, Scroll: lines}
}

func (m *Modal) normal(b *Buffer, k Key) Result {
	if k.Code == CodeEsc {
		m.clearPending()
		b.Select(0, 0)
		return Result{}
	}
	if m.pending == 'z' {
		m.pending = 0
		switch k.Rune {
		case 'z', 't', 'b':
			return Result{Place: k.Rune}
		default:
			return Result{}
		}
	}
	n := m.count
	had := m.count > 0
	if n < 1 {
		n = 1
	}
	if m.seek != 0 {
		kind := rune(m.seek)
		m.seek = 0
		if k.Rune == 0 || k.Rune < 32 {
			m.op = 0
			return Result{}
		}
		if kind != 'r' {
			m.lastFind, m.lastTarget = kind, k.Rune
		}
		return m.useMotion(b, kind, n, k.Rune)
	}
	if k.Rune >= '1' && k.Rune <= '9' || (k.Rune == '0' && m.count > 0) {
		m.count = min(len(b.r)+1, m.count*10+int(k.Rune-'0'))
		return Result{Moved: true}
	}
	if m.pending == 'g' {
		m.pending = 0
		if (k.Rune == 'j' || k.Rune == 'k') && m.op == 0 {
			m.count = 0
			if k.Rune == 'k' {
				n = -n
			}
			return Result{Screen: n, Moved: true}
		}
		if k.Rune == 'g' {
			if m.op != 0 {
				return m.useMotion(b, 'g', n, 0)
			}
			if m.count > 0 {
				b.GoLine(m.count)
			} else {
				b.GoLine(1)
			}
			m.count = 0
			return Result{Moved: true}
		}
	}
	if k.Rune == ';' || k.Rune == ',' {
		kind := m.lastFind
		if kind == 0 {
			m.clearPending()
			return Result{}
		}
		if k.Rune == ',' {
			kind = reverseFind(kind)
		}
		if kind == 't' || kind == 'T' {
			adj := b.pos + 1
			if kind == 'T' {
				adj = b.pos - 1
			}
			if adj >= 0 && adj < len(b.r) && b.r[adj] == m.lastTarget {
				n++
			}
		}
		return m.useMotion(b, kind, n, m.lastTarget)
	}
	if m.op != 0 && (k.Rune == rune(m.op)) {
		op := m.op
		count := min(len(b.r)+1, n*m.opN)
		m.clearPending()
		return m.lineOp(b, op, count)
	}
	if m.op != 0 {
		if m.object != 0 {
			if k.Rune == 'w' || k.Rune == 'W' {
				from, to := b.wordObject(m.object == 'a', k.Rune == 'W', min(len(b.r)+1, n*m.opN))
				return m.applyRange(b, from, to)
			}
			m.clearPending()
			return Result{}
		}
		switch k.Rune {
		case 'i', 'a':
			m.object = k.Rune
			return Result{}
		case 'f', 'F', 't', 'T':
			m.seek = byte(k.Rune)
			return Result{}
		}
		if mot, arg, ok := motionOf(k); ok {
			return m.useMotion(b, mot, n, arg)
		}
		if k.Rune == 'g' {
			m.pending = 'g'
			return Result{Moved: true}
		}
		m.clearPending()
		return Result{}
	}
	m.count = 0
	if motion, _, ok := motionOf(k); ok && motion != 'G' {
		return m.useMotion(b, motion, n, 0)
	}
	switch k.Rune {
	case 'G':
		if had {
			b.GoLine(n)
		} else {
			b.LastLine()
		}
	case 'g':
		m.pending = 'g'
		m.count = n
		if n == 1 && !had {
			m.count = 0
		}
		return Result{Moved: true}
	case 'd', 'c', 'y':
		m.op = byte(k.Rune)
		m.opN = n
		return Result{Moved: true}
	case 'D':
		m.op = 'd'
		m.opN = n
		return m.useMotion(b, '$', 1, 0)
	case 'C':
		m.op = 'c'
		m.opN = n
		return m.useMotion(b, '$', 1, 0)
	case 'x', 'X':
		from, to, selected := b.Selection()
		if !selected {
			bounds := b.lineBounds()[b.CaretLine()]
			if k.Rune == 'x' {
				from, to = b.pos, min(bounds[1], b.pos+n)
			} else {
				from, to = max(bounds[0], b.pos-n), b.pos
			}
		}
		if from == to {
			return Result{}
		}
		b.CopySpan(from, to, false)
		b.Select(0, 0)
		return Result{Changed: b.DeleteSpan(from, to)}
	case 's':
		b.BeginGroup()
		start := b.pos
		b.DeleteAt(n)
		b.pos = min(start, len(b.r))
		b.col = b.column()
		m.Mode = ModeEdit
		return Result{Changed: true}
	case 'S':
		m.op = 'c'
		return m.lineOp(b, 'c', n)
	case 'p', 'P':
		b.BeginGroup()
		changed := false
		for i := 0; i < n; i++ {
			if !b.Put(k.Rune == 'p') {
				break
			}
			changed = true
		}
		b.EndGroup()
		return Result{Changed: changed}
	case 'r':
		m.count = n
		m.seek = 'r'
		return Result{Moved: true}
	case 'J':
		if b.JoinLine() {
			return Result{Changed: true}
		}
		return Result{}
	case 'z':
		m.pending = 'z'
		return Result{Moved: true}
	case 'H', 'M', 'L':
		return Result{Place: k.Rune}
	case '~':
		if b.ToggleCase(n) {
			return Result{Changed: true}
		}
		return Result{}
	case 'f', 'F', 't', 'T':
		m.seek = byte(k.Rune)
		m.count = n
		return Result{Moved: true}
	case 'v', 'V':
		m.Mode = ModeVisual
		if k.Rune == 'V' {
			m.Mode = ModeVisualLine
		}
		m.anchor = b.pos
		m.selectVisual(b)
		return Result{Moved: true}
	case 'I':
		b.BeginGroup()
		b.firstText()
		m.Mode = ModeEdit
	case 'u':
		changed := false
		for i := 0; i < n; i++ {
			if !b.Undo() {
				break
			}
			changed = true
		}
		return Result{Changed: changed, Moved: true}
	case 'i':
		b.BeginGroup()
		m.Mode = ModeEdit
	case 'a':
		b.BeginGroup()
		if b.pos < len(b.r) && b.r[b.pos] != '\n' {
			b.Right()
		}
		m.Mode = ModeEdit
	case 'A':
		b.BeginGroup()
		b.LineEnd(true)
		m.Mode = ModeEdit
	case 'o':
		b.BeginGroup()
		b.OpenLine(true)
		m.Mode = ModeEdit
		return Result{Changed: true}
	case 'O':
		b.BeginGroup()
		b.OpenLine(false)
		m.Mode = ModeEdit
		return Result{Changed: true}
	case '*', '#':
		if b.pos < len(b.r) && wordClass(b.r[b.pos]) == 1 {
			from, to := b.wordObject(false, false, 1)
			m.query = string(b.r[from:to])
			m.wholeWord = true
			m.backward = k.Rune == '#'
			b.pos = from
			b.col = b.column()
			m.search(b, m.backward, n)
		}
	case 'n', 'N':
		m.search(b, m.backward != (k.Rune == 'N'), n)
	case ':', '/', '?':
		m.StartCommand(k.Rune)
	default:
		return Result{}
	}
	return Result{Moved: true}
}

func motionOf(k Key) (rune, rune, bool) {
	switch k.Code {
	case CodeHome:
		return '0', 0, true
	case CodeEnd:
		return '$', 0, true
	case CodeLeft:
		return 'h', 0, true
	case CodeRight:
		return 'l', 0, true
	case CodeUp:
		return 'k', 0, true
	case CodeDown:
		return 'j', 0, true
	}
	switch k.Rune {
	case 'h', 'l', 'j', 'k', 'w', 'b', 'e', 'W', 'B', 'E', '0', '^', '$', '{', '}', '%':
		return k.Rune, 0, true
	case 'G':
		return 'G', 0, true
	default:
		return 0, 0, false
	}
}

func (m *Modal) useMotion(b *Buffer, kind rune, n int, arg rune) Result {
	if kind == 'r' {
		m.clearPending()
		end := b.lineBounds()[b.CaretLine()][1]
		if n > end-b.pos {
			return Result{}
		}
		b.BeginGroup()
		ok := false
		for i := 0; i < n; i++ {
			if i > 0 {
				b.Right()
			}
			ok = b.ReplaceOne(arg)
		}
		b.EndGroup()
		return Result{Changed: ok}
	}
	op := m.op
	count := min(len(b.r)+1, n*max(1, m.opN))
	if op != 0 && kind == 'G' && m.count == 0 && m.opN <= 1 {
		count = len(b.lineBounds())
	}
	origin := b.Cursor()
	var from, to int
	var linewise, ok bool
	if op == 'c' && (kind == 'w' || kind == 'W') && origin < len(b.r) && !isSpace(b.r[origin]) {
		from, to = origin, b.changeWordEnd(count, kind == 'W')
		ok = true
	} else {
		from, to, linewise, ok = b.MotionSpan(kind, count, arg)
	}
	if op == 'c' && kind == '$' && !ok {
		return m.applyRange(b, origin, origin)
	}
	if !ok {
		m.clearPending()
		b.pos = origin
		b.col = b.column()
		return Result{}
	}
	if op == 0 {
		m.clearPending()
		if kind == 'j' || kind == 'k' {
			want := b.col
			b.clampView()
			b.col = want
		} else {
			b.clampView()
		}
		return Result{Moved: true}
	}
	destination := b.CaretLine()
	b.pos = origin
	b.col = b.column()
	if linewise {
		line := b.CaretLine()
		b.Place(min(line, destination), 0)
		res := m.lineOp(b, op, max(line, destination)-min(line, destination)+1)
		if op == 'y' {
			b.pos = origin
			b.col = b.column()
		}
		return res
	}
	return m.applyRange(b, from, to)
}

func (m *Modal) applyRange(b *Buffer, from, to int) Result {
	op := m.op
	m.clearPending()

	switch op {
	case 'y':
		b.CopySpan(from, to, false)
		return Result{Moved: true}
	case 'c':
		b.CopySpan(from, to, false)
		b.BeginGroup()
		b.DeleteSpan(from, to)
		b.pos = from
		b.col = b.column()
		m.Mode = ModeEdit
		return Result{Changed: true}
	default:
		b.CopySpan(from, to, false)
		if b.DeleteSpan(from, to) {
			return Result{Changed: true}
		}
		return Result{}
	}
}

func (m *Modal) lineOp(b *Buffer, op byte, n int) Result {
	if n < 1 {
		n = 1
	}
	m.clearPending()
	switch op {
	case 'y':
		line := b.CaretLine()
		from, to, _, ok := b.lineSpan(line, line+n-1)
		if ok {
			b.CopySpan(from, to, true)
		}
		return Result{Moved: true}
	case 'c':
		line := b.CaretLine()
		bounds := b.lineBounds()
		if line < 0 || line >= len(bounds) {
			return Result{}
		}
		last := line + n - 1
		if last >= len(bounds) {
			last = len(bounds) - 1
		}
		from := bounds[line][0]
		to := bounds[last][1]
		b.CopySpan(from, to, true)
		b.BeginGroup()
		b.DeleteSpan(from, to)
		b.pos = from
		b.col = b.column()
		m.Mode = ModeEdit
		return Result{Changed: true}
	default:
		line := b.CaretLine()
		from, to, _, _ := b.lineSpan(line, line+n-1)
		b.CopySpan(from, to, true)
		b.DeleteLines(n)
		return Result{Changed: true}
	}
}

// VisualColumns gives the inclusive columns selected on a zero-based source line.
func (m *Modal) VisualColumns(line int) (int, int, bool) {
	if (m.Mode != ModeVisual && m.Mode != ModeVisualLine) || line < m.visualFirst || line > m.visualLast {
		return 0, 0, false
	}
	start, end := 0, int(^uint(0)>>1)
	if m.Mode == ModeVisual {
		if line == m.visualFirst {
			start = m.firstCol
		}
		if line == m.visualLast {
			end = m.lastCol
		}
	}
	return start, end, true
}

func (m *Modal) selectVisual(b *Buffer) {
	from, to := min(m.anchor, b.pos), max(m.anchor, b.pos)
	bounds := b.lineBounds()
	for i, line := range bounds {
		if from >= line[0] && from <= line[1] {
			m.visualFirst, m.firstCol = i, from-line[0]
		}
		if to >= line[0] && to <= line[1] {
			m.visualLast, m.lastCol = i, to-line[0]
			break
		}
	}
	if m.Mode == ModeVisualLine {
		from, to = bounds[m.visualFirst][0], bounds[m.visualLast][1]
	}
	b.Select(from, min(to+1, len(b.r)))
}

func (m *Modal) visual(b *Buffer, k Key) Result {
	linewise := m.Mode == ModeVisualLine
	if k.Code == CodeEsc || (k.Rune == 'v' && !linewise) || (k.Rune == 'V' && linewise) {
		b.Select(0, 0)
		m.Mode = ModeView
		m.clearPending()
		return Result{Moved: true}
	}
	if m.object != 0 {
		around := m.object == 'a'
		m.object = 0
		if k.Rune == 'w' || k.Rune == 'W' {
			from, to := b.wordObject(around, k.Rune == 'W', max(1, m.count))
			m.count = 0
			if to > from {
				m.Mode = ModeVisual
				m.anchor = from
				b.pos = to - 1
				b.col = b.column()
				m.selectVisual(b)
			}
		}
		return Result{Moved: true}
	}
	if m.seek == 0 {
		switch k.Rune {
		case ':':
			m.searchMode = m.Mode
			m.prompt = ':'
			m.Cmd = "'<,'>"
			m.Mode = ModeCommand
			return Result{Moved: true}
		case 'i', 'a':
			m.object = k.Rune
			return Result{}
		case 'v', 'V':
			m.Mode = ModeVisual
			if k.Rune == 'V' {
				m.Mode = ModeVisualLine
			}
		case 'o':
			m.anchor, b.pos = b.pos, m.anchor
			b.col = b.column()
		case 'd', 'x', 'c', 'y':
			from, to, _ := b.Selection()
			b.Select(0, 0)
			m.Mode = ModeView
			if linewise {
				b.Place(m.visualFirst, 0)
				op := byte(k.Rune)
				if op == 'x' {
					op = 'd'
				}
				return m.lineOp(b, op, m.visualLast-m.visualFirst+1)
			}
			b.pos = from
			b.col = b.column()
			m.op = byte(k.Rune)
			return m.applyRange(b, from, to)
		default:
			_, _, motion := motionOf(k)
			if !motion && !(k.Rune >= '0' && k.Rune <= '9') && k.Rune != 'g' && k.Rune != 'f' && k.Rune != 'F' && k.Rune != 't' && k.Rune != 'T' && k.Rune != ';' && k.Rune != ',' && k.Rune != 'n' && k.Rune != 'N' && k.Rune != '/' && !strings.ContainsRune("?*#HML", k.Rune) {
				return Result{}
			}
			res := m.normal(b, k)
			if res.Screen != 0 || res.Place != 0 || m.Mode == ModeCommand {
				return res
			}
		}
	} else {
		m.normal(b, k)
	}
	m.selectVisual(b)
	return Result{Moved: true}
}
