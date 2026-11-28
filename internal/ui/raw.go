package ui

import (
	"strings"
	"unicode/utf8"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/theme"
)

// covers reports whether the open source line sits in the inclusive range.
func (p *Page) covers(start, end int) bool {
	if p.openLine <= 0 || start <= 0 {
		return false
	}
	if end < start {
		end = start
	}
	if p.buf != nil && (p.buf.Modal.Mode == edit.ModeVisual || p.buf.Modal.Mode == edit.ModeVisualLine) {
		for line := start; line <= end; line++ {
			if _, _, selected := p.buf.Modal.VisualColumns(line - 1); selected {
				return true
			}
		}
	}
	return p.openLine >= start && p.openLine <= end
}

func (p *Page) layoutRaw(b doc.Block, width float32) []rendered {
	return p.layoutRawSpan(b.Line, b.Line+b.Lines-1, b.Kind, width)
}

func (p *Page) layoutRawSpan(start, end int, kind string, width float32) []rendered {
	if start <= 0 {
		return nil
	}
	if end < start {
		end = start
	}
	var out []rendered
	for n := start; n <= end; n++ {
		out = append(out, p.rawRows(n, kind, n == start, width)...)
	}
	return out
}

func (p *Page) rawRows(n int, kind string, first bool, width float32) []rendered {
	text := p.sourceLine(n)
	th := p.theme
	ink := rgb(th.Ink)
	accent := rgb(th.Accent)
	size := th.Code
	family := th.CodeFont
	bold := false
	color := ink
	if strings.HasPrefix(kind, "h") {
		size = headingSize(th, kind)
		family = th.UIFont
		bold = true
		color = ink
	}
	chunks := p.wrapRaw(text, width, size, bold, family)
	var out []rendered
	for i, ch := range chunks {
		ln := rendered{src: n, col: ch.col}
		if i == 0 {
			ln.num = n
		}
		if strings.HasPrefix(kind, "h") && ch.col == 0 {
			if mark, rest, ok := splitHashes(ch.text); ok {
				ln.pieces = []piece{
					{text: mark, size: size, bold: true, color: accent, family: family},
					{text: rest, size: size, bold: true, color: ink, family: family},
				}
			} else {
				ln.pieces = []piece{{text: ch.text, size: size, bold: true, color: ink, family: family}}
			}
			if i == 0 && first {
				ln.gutter = kind
				ln.before = 18
				if kind == "h1" {
					ln.before = 4
				}
			}
		} else {
			ln.pieces = []piece{{text: ch.text, size: size, bold: bold, color: color, family: family}}
			if kind == "code" || kind == "html" {
				ln.fill = rgb(th.CodeFill)
			}
			if i == 0 && first {
				ln.before = 8
			}
		}
		out = append(out, ln)
	}
	return out
}

type rawChunk struct {
	text string
	col  int
}

// wrapRaw breaks a source line on the column. A break falls after a space
// when one fits, so the next row does not lead with that space. Every rune
// of the source line is kept.
func (p *Page) wrapRaw(text string, width, size float32, bold bool, family string) []rawChunk {
	if width < 8 {
		width = 8
	}
	runes := []rune(text)
	if len(runes) == 0 {
		return []rawChunk{{col: 0}}
	}
	var out []rawChunk
	i := 0
	for i < len(runes) {
		lastFit := i + 1
		lastSpace := -1
		for j := i + 1; j <= len(runes); j++ {
			w := p.advText(string(runes[i:j]), size, bold, family)
			if w > width && j > i+1 {
				break
			}
			lastFit = j
			if runes[j-1] == ' ' {
				lastSpace = j
			}
			if w > width {
				break
			}
		}
		end := lastFit
		if end < len(runes) && lastSpace > i {
			end = lastSpace
		}
		if end <= i {
			end = i + 1
		}
		out = append(out, rawChunk{text: string(runes[i:end]), col: i})
		i = end
	}
	return out
}

func headingSize(th theme.Theme, kind string) float32 {
	switch kind {
	case "h1":
		return th.H1
	case "h2":
		return th.H2
	default:
		return th.H3
	}
}

// splitHashes splits an ATX heading into the marks and the rest of the line.
// The rest keeps the space after the marks.
func splitHashes(text string) (mark, rest string, ok bool) {
	i := 0
	for i < len(text) && i < 6 && text[i] == '#' {
		i++
	}
	if i == 0 || i >= len(text) || text[i] != ' ' {
		return "", text, false
	}
	return text[:i], text[i:], true
}

func (p *Page) drawCaret(canvas widget.Canvas, ln rendered, x, y, h float32) {
	if p.buf == nil {
		return
	}
	line := ln.src
	if line == 0 {
		line = ln.num
	}
	length := utf8.RuneCountInString(p.sourceLine(line))
	col := min(p.buf.CaretCol(), length)
	local, ok := caretInRow(ln, col, length)
	if !ok {
		return
	}
	w := p.prefixWidth(ln, local)
	cw := float32(1.5)
	if p.buf.Modal.Mode != edit.ModeEdit {
		cw = p.prefixWidth(ln, local+1) - w
		if cw <= 0 {
			cw = p.theme.Body * 0.5
		}
	}
	drawModalCaret(canvas, geometry.NewRect(x+w, y+2, cw, h-4), rgb(p.theme.Accent), p.buf.Modal.Mode != edit.ModeEdit)
}

// caretInRow reports the caret offset within this visual row. A caret at the
// end of the source line sits on the last row.
func caretInRow(ln rendered, caret, lineLen int) (int, bool) {
	start := ln.col
	n := 0
	for _, pc := range ln.pieces {
		n += len([]rune(pc.text))
	}
	end := start + n
	if caret >= start && caret < end {
		return caret - start, true
	}
	if caret == lineLen && end == lineLen {
		return caret - start, true
	}
	return 0, false
}

func (p *Page) prefixWidth(ln rendered, count int) float32 {
	var width float32
	for _, pc := range ln.pieces {
		if count <= 0 {
			break
		}
		length := utf8.RuneCountInString(pc.text)
		if count < length {
			pc.text = string([]rune(pc.text)[:count])
		}
		width += p.pieceWidth(pc)
		count -= length
	}
	return width
}

// LineTop is the y of a source line inside the page bounds.
func (p *Page) LineTop(line int) float32 {
	y := p.Bounds().Min.Y
	for _, ln := range p.lines {
		y += ln.before
		if ln.src == line || ln.num == line {
			return y
		}
		y += p.rowHeight(ln)
	}
	return y
}

func (p *Page) hitLine(y float32) int {
	line, _, _ := p.hitSource(y)
	return line
}

func (p *Page) hitSource(y float32) (line, col int, text string) {
	yy := p.Bounds().Min.Y
	for _, ln := range p.lines {
		yy += ln.before
		h := p.rowHeight(ln)
		if y >= yy && y < yy+h {
			line = ln.src
			if line == 0 {
				line = ln.num
			}
			var b strings.Builder
			for _, pc := range ln.pieces {
				b.WriteString(pc.text)
			}
			return line, ln.col, b.String()
		}
		yy += h
	}
	return 0, 0, ""
}

func (p *Page) fitCol(text string, dx, size float32, bold bool, family string) int {
	if dx < 0 {
		dx = 0
	}
	rs := []rune(text)
	best := 0
	var previous float32
	for i := 1; i <= len(rs); i++ {
		w := p.advText(string(rs[:i]), size, bold, family)
		if dx < (previous+w)/2 {
			break
		}
		best = i
		previous = w
	}
	return best
}

func (p *Page) openAt(ctx widget.Context, at geometry.Point) bool {
	if p.buf == nil {
		return false
	}
	line, col := p.pointSource(at)
	if line <= 0 {
		return false
	}
	p.buf.Select(0, 0)
	p.buf.Place(line-1, col)
	if p.buf.Modal.Mode != edit.ModeEdit {
		p.buf.Modal.Reset()
	}
	p.openLine = line
	p.SetFocused(true)
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
	if ctx != nil {
		ctx.RequestFocus(p)
	}
	if p.onEdit != nil {
		p.onEdit()
	}
	return true
}

func (p *Page) kindAt(line int) string {
	for _, b := range p.blocks {
		if b.Line <= 0 {
			continue
		}
		end := b.Line + b.Lines - 1
		if end < b.Line {
			end = b.Line
		}
		if line >= b.Line && line <= end {
			return b.Kind
		}
	}
	return "p"
}

func (p *Page) sourceLine(n int) string {
	if p.buf == nil {
		return ""
	}
	return p.buf.LineText(n)
}

func (p *Page) literalParagraph(b doc.Block) bool {
	if p.buf == nil || b.Kind != "p" || b.Line <= 0 || b.Lines > 1 || b.Text != p.buf.LineText(b.Line) {
		return false
	}
	for _, sp := range b.Spans {
		if sp != (doc.Span{Text: sp.Text}) {
			return false
		}
	}
	return strings.Join(strings.Fields(b.Text), " ") == b.Text
}

func (p *Page) pointSource(at geometry.Point) (int, int) {
	y := p.Bounds().Min.Y
	for _, ln := range p.lines {
		y += ln.before
		h := p.rowHeight(ln)
		if at.Y >= y && at.Y < y+h {
			line := ln.src
			if line == 0 {
				line = ln.num
			}
			col := ln.col
			dx := at.X - p.Bounds().Min.X - p.gutterWidth()
			if ln.quoteLine != 0 {
				dx -= quotePad
			}
			pieces := ln.pieces
			if ln.gridCell && len(ln.cellLines[0]) > 0 {
				idx := int((at.Y - y - cellPadY) / (p.theme.Body * proseLead))
				pieces = ln.cellLines[0][min(max(idx, 0), len(ln.cellLines[0])-1)]
				dx -= cellPad
			}
			for _, pc := range pieces {
				if pc.sourceLine > 0 {
					line, col = pc.sourceLine, pc.sourceCol
				}
				w := p.pieceWidth(pc)
				if dx <= w {
					return line, col + p.fitCol(pc.text, dx, pc.size, pc.bold, pc.family)
				}
				dx -= w
				col += utf8.RuneCountInString(pc.text)
			}
			return line, col
		}
		y += h
	}
	return 0, 0
}

func (p *Page) editKey(ctx widget.Context, e event.Event) bool {
	v, ok := e.(*event.KeyEvent)
	if !ok || p.buf == nil || !ownsKeyFocus(ctx, p) || v.KeyType == event.KeyRelease {
		return false
	}
	key := keyOf(v)
	if key.Code == edit.CodeNone && key.Rune == 0 {
		return false
	}
	if key.Code == edit.CodeEnter && p.buf.Modal.Mode == edit.ModeView && p.openLine == 1 && p.kindAt(1) == "front" {
		p.toggleDetails()
		return true
	}
	prevLine, prevCol := p.buf.CaretLine(), p.buf.CaretCol()
	viewH, cur, measured := p.windowGeom()
	line := p.bodyLine()
	// Bounds are the whole document once a scroll view has measured it.
	// A page count taken from that height jumps to the end.
	if measured && viewH > line {
		key.Rows = int(viewH / line)
	} else if p.Parent() == nil && p.Bounds().Height() > line {
		key.Rows = int(p.Bounds().Height() / line)
		viewH = p.Bounds().Height()
	}
	if key.Rows < 1 {
		key.Rows = 20
	}
	res, used := p.buf.Modal.Handle(p.buf, key)
	if !used {
		return false
	}
	if p.openLine == 0 {
		p.Reopen(p.buf.CaretLine() + 1)
	}
	if res.Win != 0 && p.onWin != nil {
		p.onWin(res.Win)
		return true
	}
	if res.Screen != 0 {
		p.moveScreen(res.Screen)
	}
	if res.Save && p.onSave != nil {
		p.onSave()
	}
	if res.Cmd != "" && p.onCmd != nil {
		p.onCmd(res.Cmd)
	}
	if p.openLine == 0 {
		if ctx != nil {
			ctx.Invalidate()
		}
		if p.onEdit != nil {
			p.onEdit()
		}
		return true
	}
	if viewH > 1 {
		p.followNav(cur, viewH, prevLine, prevCol, res)
		if res.Scroll != 0 || res.Place != 0 {
			p.buf.Modal.RefreshSelection(p.buf)
		}
	}
	if !p.frontExpanded && len(p.blocks) > 0 && p.blocks[0].Kind == "front" && p.buf.CaretLine() < p.blocks[0].Lines && (p.buf.CaretLine() > 0 || p.buf.Modal.Mode == edit.ModeEdit || res.Changed) {
		p.frontExpanded = true
	}
	if res.Changed {
		p.openLine = p.buf.CaretLine() + 1
		p.SetBlocks(doc.Render(p.path, []byte(p.buf.Text())))
	} else {
		p.openLine = p.buf.CaretLine() + 1
		p.MarkNeedsLayout()
		p.SetNeedsRedraw(true)
	}
	if ctx != nil {
		ctx.Invalidate()
	}
	if p.onEdit != nil {
		p.onEdit()
	}
	return true
}

func keyOf(v *event.KeyEvent) edit.Key {
	k := edit.Key{Rune: v.Rune, Super: v.IsSuper(), Ctrl: v.IsCtrl(), Shift: v.IsShift()}
	if k.Ctrl || k.Super {
		switch {
		case k.Rune >= 1 && k.Rune <= 26:
			k.Rune = 'a' + k.Rune - 1
		case k.Rune >= 'A' && k.Rune <= 'Z':
			k.Rune += 'a' - 'A'
		case k.Rune == 0 && v.Key >= event.KeyA && v.Key <= event.KeyZ:
			k.Rune = rune('a' + (v.Key - event.KeyA))
		}
	}
	switch v.Key {
	case event.KeyHome:
		k.Code = edit.CodeHome
	case event.KeyEnd:
		k.Code = edit.CodeEnd
	case event.KeyEscape:
		k.Code = edit.CodeEsc
	case event.KeyEnter:
		k.Code = edit.CodeEnter
	case event.KeyBackspace:
		k.Code = edit.CodeBackspace
	case event.KeyDelete:
		k.Code = edit.CodeDelete
	case event.KeyLeft:
		k.Code = edit.CodeLeft
	case event.KeyRight:
		k.Code = edit.CodeRight
	case event.KeyUp:
		k.Code = edit.CodeUp
	case event.KeyDown:
		k.Code = edit.CodeDown
	}
	return k
}

func (p *Page) moveScreen(delta int) {
	width := max(1, p.Bounds().Width()-p.gutterWidth())
	line, col := p.buf.CaretLine()+1, p.buf.CaretCol()
	// Lay out the destination source lines as they will appear while active.
	rows := func(n int) []rendered {
		width := width
		for _, b := range p.blocks {
			if b.QuoteLine != 0 && n >= b.Line && n < b.Line+max(1, b.Lines) {
				width = max(1, width-2*quotePad)
			}
			if b.Line == n && p.literalParagraph(b) {
				return p.layoutOne(b, width, p.theme)
			}
		}
		return p.rawRows(n, p.kindAt(n), false, width)
	}
	current := rows(line)
	i := len(current) - 1
	for i > 0 && col < current[i].col {
		i--
	}
	offset := col - current[i].col
	last := len(p.buf.Lines())
	for delta != 0 {
		if delta > 0 {
			if i+1 < len(current) {
				i++
			} else if line < last {
				line++
				current = rows(line)
				i = 0
			} else {
				break
			}
			delta--
		} else {
			if i > 0 {
				i--
			} else if line > 1 {
				line--
				current = rows(line)
				i = len(current) - 1
			} else {
				break
			}
			delta++
		}
	}
	length := 0
	for _, pc := range current[i].pieces {
		length += utf8.RuneCountInString(pc.text)
	}
	p.buf.Place(line-1, current[i].col+min(offset, max(0, length-1)))
	p.buf.Modal.RefreshSelection(p.buf)
}

// Map display runs to source in reading order, including repeated words and soft breaks.
func (p *Page) mapFlowSource(rows []rendered, b doc.Block) {
	line, col := b.Line, 0
	rest := p.sourceLine(line)
	last := line + max(1, b.Lines) - 1
	for i := range rows {
		mapped := false
		for j := range rows[i].pieces {
			pc := &rows[i].pieces[j]
			if pc.text == "" {
				continue
			}
			s, l, c := rest, line, col
			for {
				at := strings.Index(s, pc.text)
				if at >= 0 {
					c += utf8.RuneCountInString(s[:at])
					pc.sourceLine, pc.sourceCol = l, c
					if !mapped {
						rows[i].src, rows[i].col = l, c
						mapped = true
					}
					line, col, rest = l, c+utf8.RuneCountInString(pc.text), s[at+len(pc.text):]
					break
				}
				if l == last || strings.TrimSpace(pc.text) == "" {
					break
				}
				l++
				c = 0
				s = p.sourceLine(l)
			}
		}
	}
}
