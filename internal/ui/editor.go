package ui

import (
	"runtime"
	"strconv"
	"unicode/utf8"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/theme"
)

const (
	editLine = float32(20)

	editGutter = float32(40)
)

// Editor draws a buffer and applies key events while it is focused.
type Editor struct {
	widget.WidgetBase
	lineNumbers bool
	buf         *edit.Buffer
	scroll      float32
	theme       theme.Theme
	onEdit      func()
	onSave      func()

	onCmd      func(string)
	onWin      func(rune)
	onActivate func()

	anchor           int
	dragging         bool
	file             string
	rows             []sourceRow
	cols             int
	rowFile, rowText string
	lineColors       [][]doc.Span
	cellRatio        float32
	customFace       string
	checkFace        bool
	pendingPassage   Passage
	verticalCol      int // preferred visual column for insert-mode arrow keys
}

// NewEditor returns an editor. onEdit runs after the buffer changes.
// onSave runs for Cmd-S or Ctrl-S.
func NewEditor(onEdit, onSave func()) *Editor {
	ratio, _ := loadSourceFont()
	e := &Editor{cellRatio: ratio, verticalCol: -1, checkFace: true, lineNumbers: true, buf: edit.New(""), theme: theme.Pairti(), onEdit: onEdit, onSave: onSave}
	e.SetVisible(true)
	e.SetEnabled(true)
	return e
}

// IsFocusable reports that the editor takes keyboard focus.
func (e *Editor) IsFocusable() bool { return e.IsVisible() && e.IsEnabled() }

// SetLineNumbers controls source coordinates in the raw editor.
func (e *Editor) SetLineNumbers(on bool) {
	if e.lineNumbers == on {
		return
	}
	e.lineNumbers = on
	e.MarkNeedsLayout()
	e.SetNeedsRedraw(true)
}

func (e *Editor) gutterWidth() float32 {
	if e.lineNumbers {
		return editGutter
	}
	return 4
}

// SetTheme replaces the semantic theme.
func (e *Editor) SetTheme(t theme.Theme) {
	if t.RawFont != e.theme.RawFont || t.Code != e.theme.Code {
		e.customFace = ""
		e.cellRatio, _ = loadSourceFont()
		e.checkFace = true
	}
	e.theme = t
	e.MarkNeedsLayout()
	e.SetNeedsRedraw(true)
}

func (e *Editor) lineH() float32 {
	code := e.theme.Code
	if code < 1 {
		code = 14
	}
	return editLine * code / 14
}

func (e *Editor) charW() float32 {
	code := e.theme.Code
	if code < 1 {
		code = 14
	}
	return e.cellRatio * code
}

// SetActivate runs when the editor is clicked, before the caret moves.
func (e *Editor) SetActivate(fn func()) { e.onActivate = fn }

// SetCommand receives a finished : command.
func (e *Editor) SetCommand(fn func(string)) { e.onCmd = fn }

// SetWindow receives a Ctrl-W target: h, l, or w.
func (e *Editor) SetWindow(fn func(rune)) { e.onWin = fn }

// ModeName is View, Edit, or Command.
func (e *Editor) ModeName() string { return e.buf.Modal.Name() }

// Command is the text after the colon.
func (e *Editor) Command() string { return e.buf.Modal.Cmd }

// CommandOn reports whether the : line is open.
func (e *Editor) CommandOn() bool { return e.buf.Modal.Mode == edit.ModeCommand }

// SetBuffer replaces the text and places the cursor at the start.
func (e *Editor) SetBuffer(text string) {
	e.UseBuffer(edit.New(text))
}

// SetFile is the path being edited. Markdown and known source files are coloured.
func (e *Editor) SetFile(path string) {
	if e.file == path {
		return
	}
	e.file = path
	e.rows = nil
	e.SetNeedsRedraw(true)
}

// UseBuffer shows an existing buffer, including its unsaved edits.
func (e *Editor) UseBuffer(b *edit.Buffer) {
	if b == nil {
		b = edit.New("")
	}
	e.buf = b
	e.pendingPassage = Passage{}
	e.verticalCol = -1
	e.scroll = 0
	e.MarkNeedsLayout()
	e.SetNeedsRedraw(true)
}

// Buffer is the text being edited.
func (e *Editor) Buffer() *edit.Buffer { return e.buf }

// Layout is the full source height. A parent scroll view then sizes the thumb
// to the document. A tight max height still clamps, and the editor scrolls itself.
func (e *Editor) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	w := c.MaxWidth
	if w <= 0 || w >= geometry.Infinity {
		w = 400
	}
	rows := e.sourceRows(w)
	h := float32(len(rows)) * e.lineH()
	size := c.Constrain(geometry.Sz(w, h))
	if e.pendingPassage.Line > 0 {
		e.restoreRows(e.pendingPassage, rows)
		e.pendingPassage = Passage{}
	}
	e.scroll = min(e.scroll, max(0, h-size.Height))
	return size
}

// Draw paints the source and a caret.
func (e *Editor) Draw(_ widget.Context, canvas widget.Canvas) {
	if !e.IsVisible() {
		return
	}
	b := e.Bounds()
	if b.IsEmpty() {
		return
	}
	e.resolveSourceFace(canvas)
	canvas.PushClip(b)
	ink := rgb(e.theme.Ink)
	muted := rgb(e.theme.Muted)
	rows := e.visualRows()
	lineH, charW := e.lineH(), e.charW()
	x := b.Min.X + e.gutterWidth()
	from, to, selected := e.buf.Selection()
	caretCol := e.buf.CaretCol()
	caret := rowAt(rows, e.buf.CaretLine(), caretCol)
	first := max(0, int(e.scroll/lineH))
	for i := first; i < len(rows); i++ {
		row := rows[i]
		y := b.Min.Y + float32(i)*lineH - e.scroll
		if y >= b.Max.Y {
			break
		}
		if e.lineNumbers && row.col == 0 {
			num := strconv.Itoa(row.line + 1)
			nw := textWidth(canvas, num, e.theme.Meta, muted, false, e.theme.UIFont)
			drawFace(canvas, num, geometry.NewRect(b.Min.X+editGutter-nw-8, y, nw+2, lineH), e.theme.Meta, muted, false, e.theme.UIFont)
		}
		if selected {
			e.drawSel(canvas, x, y, row.start, row.length, from, to)
		}
		e.drawSource(canvas, row.text, row.spans, x, y, lineH, charW, ink)
		if i == caret && e.IsFocused() {
			cx := x + float32(caretCol-row.col)*charW
			cw := float32(1)
			if e.buf.Modal.Mode != edit.ModeEdit {
				cw = charW
			}
			drawModalCaret(canvas, geometry.NewRect(cx, y+2, cw, lineH-4), rgb(e.theme.Accent), e.buf.Modal.Mode != edit.ModeEdit)
		}
	}
	canvas.PopClip()
}

// Event handles clicks and typing.
func (e *Editor) Event(ctx widget.Context, ev event.Event) bool {
	switch v := ev.(type) {
	case *event.WheelEvent:
		if !e.Bounds().Contains(v.Position) {
			return false
		}
		// The parent scroll view owns the thumb when the whole source fits
		// in the bounds it gave us.
		if float32(len(e.visualRows()))*e.lineH() <= e.Bounds().Height()+1 {
			return false
		}
		e.scrollBy(WheelScroll(v.DeltaY(), runtime.GOOS == "darwin"))
		e.SetNeedsRedraw(true)
		return true
	case *event.MouseEvent:
		if v.Button != event.ButtonLeft && v.MouseType != event.MouseDrag {
			return false
		}
		if v.MouseType == event.MousePress {
			if !e.Bounds().Contains(v.Position) {
				return false
			}
			if ctx != nil {
				ctx.RequestFocus(e)
			}
			e.verticalCol = -1
			e.place(v.Position)
			e.anchor = e.buf.Cursor()
			e.dragging = true
			e.buf.Select(0, 0)
			if e.buf.Modal.Mode != edit.ModeEdit {
				e.buf.Modal.Reset()
			}
			e.SetNeedsRedraw(true)
			if e.onActivate != nil {
				e.onActivate()
			}
			return true
		}
		if !e.dragging {
			return false
		}
		if v.MouseType == event.MouseDrag {
			e.verticalCol = -1
			e.place(v.Position)
			e.buf.Modal.Drag(e.buf, e.anchor, e.buf.Cursor())
			e.SetNeedsRedraw(true)
			return true
		}
		if v.MouseType == event.MouseRelease {
			e.dragging = false
			return true
		}
		return false
	case *event.KeyEvent:
		if !ownsKeyFocus(ctx, e) || v.KeyType == event.KeyRelease {
			return false
		}
		key := keyOf(v)
		if key.Code == edit.CodeNone && key.Rune == 0 {
			return false
		}
		key.Rows = e.screenLines()
		prevLine := 0
		oldScroll := e.scroll
		if e.buf != nil {
			prevLine = e.caretRow()
		}
		var res edit.Result
		used := true
		if e.buf.Modal.Mode == edit.ModeEdit && !key.Ctrl && !key.Super && (key.Code == edit.CodeUp || key.Code == edit.CodeDown) {
			rows := e.visualRows()
			if e.verticalCol < 0 {
				e.verticalCol = e.buf.CaretCol() - rows[prevLine].col
			}
			next := prevLine + 1
			if key.Code == edit.CodeUp {
				next = prevLine - 1
			}
			if next >= 0 && next < len(rows) {
				col := e.verticalCol
				if next+1 < len(rows) && rows[next+1].line == rows[next].line {
					col = min(col, rows[next].length-1)
				}
				e.placeRow(next, col)
			}
		} else {
			e.verticalCol = -1
			res, used = e.buf.Modal.Handle(e.buf, key)
		}
		if !used {
			return false
		}
		if res.Screen != 0 {
			rows := e.visualRows()
			next := max(0, min(len(rows)-1, prevLine+res.Screen))
			e.placeRow(next, min(e.buf.CaretCol()-rows[prevLine].col, max(0, rows[next].length-1)))
			e.buf.Modal.RefreshSelection(e.buf)
		}
		if res.Save && e.onSave != nil {
			e.onSave()
		}
		if res.Cmd != "" && e.onCmd != nil {
			e.onCmd(res.Cmd)
		}
		if res.Win != 0 && e.onWin != nil {
			e.onWin(res.Win)
			return true
		}
		e.applyView(res)
		// A page scrolls from the current view. The caret then sits on the
		// same screen row. reveal would jump back to wherever the caret was
		// before the wheel moved the document.
		// A one-line scroll keeps the caret where it is. reveal would
		// drag the view back to that caret and cancel Ctrl-E and Ctrl-Y.
		if res.Scroll != 0 && !res.Pull {
			row := prevLine - int(oldScroll/e.lineH())
			if row < 0 {
				row = 0
			}
			if max := e.screenLines() - 1; row > max {
				row = max
			}
			e.caretAtScreen(row)
		} else if !res.Pull {
			e.reveal()
		}
		if res.Scroll != 0 || res.Place != 0 {
			e.buf.Modal.RefreshSelection(e.buf)
		}
		e.SetNeedsRedraw(true)
		if e.onEdit != nil {
			e.onEdit()
		}
		return true
	default:
		return false
	}
}

// Children returns nil.
func (e *Editor) Children() []widget.Widget { return nil }

func (e *Editor) drawSource(canvas widget.Canvas, line string, spans []doc.Span, x, y, lineH, charW float32, ink widget.Color) {
	if !spansCover(spans, line) {
		drawFace(canvas, rawDisplay(line), geometry.NewRect(x, y, float32(utf8.RuneCountInString(line))*charW+1, lineH), e.theme.Code, ink, false, e.sourceFamily(false, false))
		return
	}
	for _, sp := range spans {
		w := float32(utf8.RuneCountInString(sp.Text)) * charW
		rect := geometry.NewRect(x, y, w+1, lineH)
		col := toneColor(e.theme, sp.Tone)
		if sp.Italic {
			if styled, ok := canvas.(widget.StyledTextDrawer); ok {
				styled.DrawStyledText(rawDisplay(sp.Text), rect, widget.TextStyle{
					FontFamily: e.sourceFamily(sp.Bold, sp.Italic), FontSize: e.theme.Code,
					Bold: sp.Bold, Italic: true, Color: col,
				})
				x += w
				continue
			}
		}
		drawFace(canvas, rawDisplay(sp.Text), rect, e.theme.Code, col, sp.Bold, e.sourceFamily(sp.Bold, sp.Italic))
		x += w
	}
}

func spansCover(spans []doc.Span, line string) bool {
	if len(spans) == 0 {
		return false
	}
	n := 0
	for _, sp := range spans {
		n += len(sp.Text)
	}
	return n == len(line)
}

func (e *Editor) drawSel(canvas widget.Canvas, x, y float32, start, n, from, to int) {
	lo := from - start
	hi := to - start
	if hi <= 0 || lo >= n {
		return
	}
	if lo < 0 {
		lo = 0
	}
	if hi > n {
		hi = n
	}
	canvas.DrawRect(geometry.NewRect(x+float32(lo)*e.charW(), y+2, float32(hi-lo)*e.charW(), e.lineH()-4), rgba(e.theme.Selection))
}

func (e *Editor) place(p geometry.Point) {
	b := e.Bounds()
	line := int((p.Y - b.Min.Y + e.scroll) / e.lineH())
	col := int((p.X - (b.Min.X + e.gutterWidth())) / e.charW())
	e.placeRow(line, col)
}

// ShowLine moves the caret to a 1-based source line and scrolls that line to the top.
func (e *Editor) ShowLine(line int) {
	if e.buf == nil {
		return
	}
	e.buf.GoLine(line)
	e.scroll = float32(e.caretRow()) * e.lineH()
	maxScroll := float32(len(e.visualRows()))*e.lineH() - e.Bounds().Height()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.scroll > maxScroll {
		e.scroll = maxScroll
	}
	if e.scroll < 0 {
		e.scroll = 0
	}
	e.SetNeedsRedraw(true)
}

func (e *Editor) screenLines() int {
	lh := e.lineH()
	if lh < 1 {
		lh = 1
	}
	n := int(e.Bounds().Height() / lh)
	if n < 1 {
		n = 1
	}
	return n
}

func (e *Editor) applyView(res edit.Result) {
	switch res.Place {
	case 'z':
		e.scrollToCaret(0)
	case 't':
		e.scrollToCaret(-1)
	case 'b':
		e.scrollToCaret(1)
	case 'H':
		e.caretAtScreen(0)
	case 'M':
		e.caretAtScreen(e.screenLines() / 2)
	case 'L':
		row := e.screenLines() - 1
		if row < 0 {
			row = 0
		}
		e.caretAtScreen(row)
	}
	if res.Scroll != 0 {
		e.scrollBy(float32(res.Scroll) * e.lineH())
		if res.Pull {
			e.pullCaret()
		}
	}
}

// scrollToCaret puts the caret line at the top, middle, or bottom.
// where -1 is the top, 0 the middle, 1 the bottom.
func (e *Editor) scrollToCaret(where int) {
	y := float32(e.caretRow()) * e.lineH()
	view := e.Bounds().Height()
	switch where {
	case -1:
		e.scroll = y
	case 1:
		e.scroll = y - view + e.lineH()
	default:
		e.scroll = y - view/2
	}
	e.clampScroll()
}

func (e *Editor) caretAtScreen(row int) {
	if e.buf == nil {
		return
	}
	e.placeRow(int(e.scroll/e.lineH())+row, 0)
}

func (e *Editor) pullCaret() {
	if e.buf == nil {
		return
	}
	top := int(e.scroll / e.lineH())
	bot := top + e.screenLines() - 1
	if bot < top {
		bot = top
	}
	if e.caretRow() < top {
		e.placeRow(top, 0)
	}
	if e.caretRow() > bot {
		e.placeRow(bot, 0)
	}
}

// RevealCaret scrolls after a menu command moves the shared buffer cursor.
func (e *Editor) RevealCaret() {
	e.MarkNeedsLayout()
	e.reveal()
	e.SetNeedsRedraw(true)
}

func (e *Editor) reveal() {
	y := float32(e.caretRow()) * e.lineH()
	if y < e.scroll {
		e.scroll = y
	}
	if y+e.lineH() > e.scroll+e.Bounds().Height() {
		e.scroll = y + e.lineH() - e.Bounds().Height()
	}
	if e.scroll < 0 {
		e.scroll = 0
	}
}

func (e *Editor) scrollBy(dy float32) {
	e.scroll += dy
	e.clampScroll()
}

func (e *Editor) clampScroll() {
	n := 1
	if e.buf != nil {
		n = len(e.visualRows())
	}
	maxScroll := float32(n)*e.lineH() - e.Bounds().Height()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if e.scroll < 0 {
		e.scroll = 0
	}
	if e.scroll > maxScroll {
		e.scroll = maxScroll
	}
}

func drawModalCaret(canvas widget.Canvas, rect geometry.Rect, accent widget.Color, block bool) {
	if block {
		fill := accent
		fill.A = 0.32
		canvas.DrawRect(rect, fill)
		rect.Min.Y = rect.Max.Y - 2
	}
	canvas.DrawRect(rect, accent)
}

// Split panes can retain a visual focus flag while the sidebar owns keyboard focus.
func ownsKeyFocus(ctx widget.Context, w interface {
	widget.Widget
	widget.Focusable
}) bool {
	return w.IsFocused() && (ctx == nil || ctx.FocusedWidget() == nil || ctx.IsFocused(w))
}
