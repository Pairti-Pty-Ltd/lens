// Package ui draws a Markdown file as text on a GoGPU scroll surface.
package ui

import (
	"context"
	"errors"
	"image"
	"strconv"
	"strings"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/diagram"
	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/theme"
)

const (
	readNum     = float32(40) // source line, right aligned
	readTag     = float32(28) // h1, h2, …
	readGutter  = readNum + readTag
	diagramMaxH = float32(2400)
	// proseLead is the line box for reading text. Theme.Line (1.6) left too
	// much air between wrapped lines of one paragraph.
	proseLead = float32(1.3)
	// interAscent is Inter Regular's (ascent - (ascent+descent)/2) / em.
	// DrawStyledText centers the em box in the rect, so a smaller face in
	// the same rect sits above the row's baseline by this fraction of the
	// size difference.
	interAscent = float32(0.9688 - 1.210/2)
)

// Page is the rendered Markdown. The first heading is the document title,
// so the caller does not paint that title again above the page.
type Page struct {
	anchor Passage
	widget.WidgetBase
	wrapWidth   float32
	lineNumbers bool
	blocks      []doc.Block
	lines       []rendered
	theme       theme.Theme
	dir         string
	path        string
	pics        map[string]*cachedPic
	remoteLeft  int
	diagrams    map[string]diagramHit
	hits        []linkHit
	// measure is the width of a run in the face that draws it. Layout before
	// the first draw uses a stand-in. A short stand-in lets the line run
	// under the scrollbar.
	measure  func(text string, size float32, bold bool, family string) float32
	follow   func(string)
	buf      *edit.Buffer
	openLine int

	onEdit        func()
	onSave        func()
	onCmd         func(string)
	onWin         func(rune)
	dragging      bool
	dragAnchor    int
	frontExpanded bool
	onScroll      func(float32)
}

type linkHit struct {
	url  string
	rect geometry.Rect
}

type diagramHit struct {
	img image.Image
	err error
}

type piece struct {
	sourceLine, sourceCol      int // position of a formatted text run before its markup is revealed
	text                       string
	size                       float32
	bold, italic, strike, code bool
	bullet, box, checked       bool
	pad                        float32 // blank advance, for a hanging indent
	color                      widget.Color
	family                     string
	img                        image.Image
	imgW, imgH                 float32
	link                       string
}

type rendered struct {
	pieces    []piece
	cells     []string
	cellLinks []string
	cellLines [][][]piece // column, visual line, styled run
	cols      []float32   // shared table column widths
	wrap      [][]string  // cell text wrapped to cols
	figure    bool
	tableH    float32
	tableEnd  bool
	rule      bool
	gap       bool
	quote     bool
	num       int
	src       int // source line this row belongs to, even when the number is not drawn
	col       int // rune offset of this row in that source line
	gutter    string
	lang      string
	before    float32
	fill      widget.Color
	yaml      int // 1 badge, 2 body line, 3 closing fence
	img       image.Image
	imgW      float32
	imgH      float32
}

// NewPage returns an empty page.
func NewPage() *Page {
	p := &Page{theme: theme.Pairti()}
	p.SetVisible(true)
	p.SetEnabled(true)
	return p
}

// SetLineNumbers shows source coordinates and heading tags together.
func (p *Page) SetLineNumbers(on bool) {
	if p.lineNumbers == on {
		return
	}
	p.lineNumbers = on
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

func (p *Page) gutterWidth() float32 {
	if p.lineNumbers {
		return readGutter
	}
	return 16 // leaves room for the quote bar
}

// SetTheme replaces the semantic theme.
func (p *Page) SetTheme(t theme.Theme) {
	p.theme = t
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

// SetBlocks replaces the rendered document and asks for layout.
// Relative images resolve only after SetSource.
func (p *Page) SetBlocks(blocks []doc.Block) {
	p.blocks = blocks
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

// SetFollow is called with the destination when a link is clicked.
func (p *Page) SetWindow(fn func(rune)) { p.onWin = fn }

func (p *Page) SetFollow(fn func(string)) { p.follow = fn }

// SetInline wires preview editing. onEdit runs when the open line changes.
// onSave runs for Cmd-S or Ctrl-S. onCmd runs a finished : command.
func (p *Page) SetInline(onEdit, onSave func(), onCmd func(string)) {
	p.onEdit = onEdit
	p.onSave = onSave
	p.onCmd = onCmd
}

// OnScroll receives the vertical offset after a navigation key.
// The scroll view owns the thumb. The page only reports the offset.
func (p *Page) OnScroll(fn func(float32)) { p.onScroll = fn }

// ModeName is View, Edit, or Command while a line is open.
func (p *Page) ModeName() string {
	if p.buf == nil {
		return ""
	}
	return p.buf.Modal.Name()
}

// Command is the text after the colon.
func (p *Page) Command() string {
	if p.buf == nil {
		return ""
	}
	return p.buf.Modal.Cmd
}

// CommandOn reports whether the : line is open.
func (p *Page) CommandOn() bool { return p.openLine > 0 && p.buf.Modal.Mode == edit.ModeCommand }

// Close leaves the open line and returns to the reading view.
func (p *Page) Close() {
	p.openLine = 0
	p.anchor = Passage{}
	if p.buf != nil {
		p.buf.EndGroup()
		p.buf.Modal.Reset()
	}
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

// UseBuffer is the document the preview edits. A different buffer closes the open line.
func (p *Page) UseBuffer(b *edit.Buffer) {
	if p.buf != b {
		p.dragging = false
		p.anchor = Passage{}
		p.frontExpanded = false
		p.openLine = 0
	}
	p.buf = b
}

// Buffer is the text being edited in the preview, or nil.
func (p *Page) Buffer() *edit.Buffer { return p.buf }

// Open reports whether a source line is showing its Markdown.
func (p *Page) Open() bool { return p.openLine > 0 }

// OpenLine is the 1-based source line showing Markdown, or 0.
func (p *Page) OpenLine() int { return p.openLine }

// Reopen shows line as Markdown again after a save.
func (p *Page) Reopen(line int) {
	if p.buf == nil || line <= 0 {
		p.openLine = 0
		return
	}
	n := len(p.buf.Lines())
	if line > n {
		line = n
	}
	p.openLine = line
	if line > 1 && p.kindAt(line) == "front" {
		p.frontExpanded = true
	}
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

// IsFocusable reports that an open document can take the caret.
func (p *Page) IsFocusable() bool {
	return p.buf != nil && p.IsVisible() && p.IsEnabled()
}

// SetPath is the open file. An edit highlights source again from this path.
func (p *Page) SetPath(path string) {
	if p.path != path {
		p.frontExpanded = false
	}
	p.path = path
}

// SetSource is the open document: dir is the file's directory, used to
// resolve inline images. It drops the decode cache so a replaced file is read.
func (p *Page) SetSource(dir string, blocks []doc.Block) {
	if p.buf == nil {
		p.frontExpanded = false
	}
	p.dir = dir
	p.pics = nil
	p.diagrams = nil
	p.remoteLeft = maxRemote
	p.openLine = 0
	p.anchor = Passage{}
	p.SetBlocks(blocks)
}

// Layout wraps the blocks to the available width.
func (p *Page) Layout(_ widget.Context, constraints geometry.Constraints) geometry.Size {
	width := constraints.MaxWidth
	if width <= 0 || width >= geometry.Infinity {
		width = 880
	}
	content := width - p.gutterWidth()
	content = max(1, content)
	p.wrapWidth = content
	p.lines = p.layoutBlocks(content)
	if p.anchor.Line > 0 {
		if p.onScroll != nil {
			offset := max(0, p.LineY(p.anchor.Line)-p.anchor.Offset)
			if height, _, ok := p.windowGeom(); ok {
				offset = min(offset, max(0, p.contentH()-height))
			}
			p.onScroll(offset)
		}
		// The first draw supplies the font measurements used for final wrapping.
		if p.measure != nil {
			p.anchor = Passage{}
		}
	}
	var h float32
	for _, ln := range p.lines {
		h += ln.before + p.rowHeight(ln)
	}
	if h < 1 {
		h = 1
	}
	return constraints.Constrain(geometry.Sz(content+p.gutterWidth(), h))
}

func (p *Page) rowHeight(ln rendered) float32 {
	if ln.gap {
		return p.theme.Meta * 1.25
	}
	if ln.rule {
		return 16
	}
	if ln.img != nil {
		h := ln.imgH + 8
		if h < 1 {
			h = 1
		}
		return h
	}
	if ln.yaml == 1 {
		h := p.theme.Meta * 1.6
		if h < 22 {
			h = 22
		}
		return h
	}
	if ln.yaml > 1 {
		return p.theme.Body * 1.35
	}
	size := p.theme.Body
	for _, pc := range ln.pieces {
		if pc.size > size {
			size = pc.size
		}
	}
	if ln.cols != nil {
		n := 1
		for _, lines := range ln.wrap {
			if len(lines) > n {
				n = len(lines)
			}
		}
		if ln.figure {
			return 8 + 28 + 4 + float32(n)*p.theme.Meta*p.theme.Line + 8
		}
		for _, lines := range ln.cellLines {
			n = max(n, len(lines))
		}
		return float32(n)*p.theme.Body*proseLead + 2*cellPadY
	}
	mul := proseLead
	if ln.fill.A > 0 {
		mul = 1.35
	}
	h := size * mul
	for _, pc := range ln.pieces {
		if pc.imgH > h {
			h = pc.imgH
		}
	}
	return h
}

// Draw paints the wrapped lines.
func (p *Page) Draw(_ widget.Context, canvas widget.Canvas) {
	if !p.IsVisible() {
		return
	}
	b := p.Bounds()
	p.hits = p.hits[:0]
	canvas.PushClip(b)
	defer canvas.PopClip()
	y := b.Min.Y
	x0 := b.Min.X + p.gutterWidth()
	width := b.Width() - p.gutterWidth()
	if width < 1 {
		width = 1
	}
	rule := rgb(p.theme.Rule)
	needsWrap := p.measure == nil || p.wrapWidth != width
	if p.measure == nil {
		p.measure = func(text string, size float32, bold bool, family string) float32 {
			return textWidth(canvas, text, size, rgb(p.theme.Ink), bold, family)
		}
	}
	if needsWrap {
		p.wrapWidth = width
		p.lines = p.layoutBlocks(width)
		p.MarkNeedsLayout()
	}
	for i := range p.lines {
		ln := p.lines[i]
		y += ln.before
		h := p.rowHeight(ln)
		if p.lineNumbers && ln.num > 0 {
			p.drawNum(canvas, ln.num, b.Min.X, y, h, p.lineSize(ln))
		}
		if ln.gap {
			y += h
			continue
		}
		if ln.rule {
			canvas.DrawLine(geometry.Pt(x0, y+8), geometry.Pt(x0+width, y+8), rule, 1)
			y += h
			continue
		}
		if ln.cols != nil {
			if ln.figure {
				p.drawFigure(canvas, x0, y, ln)
			} else {
				p.drawTableRow(canvas, x0, y, h, ln)
			}
			y += h
			continue
		}
		if ln.yaml >= 2 && (i == 0 || p.lines[i-1].yaml < 2) {
			cardH := h
			for j := i + 1; j < len(p.lines) && p.lines[j].yaml >= 2; j++ {
				cardH += p.lines[j].before + p.rowHeight(p.lines[j])
			}
			canvas.DrawRoundRect(geometry.NewRect(x0, y, width, cardH), rgb(p.theme.Surface), 6)
		}
		if ln.yaml == 1 {
			p.drawFrontHead(canvas, x0, y, h)
			y += h
			continue
		}
		if ln.fill.A > 0 {
			canvas.DrawRect(geometry.NewRect(x0, y, width, h), ln.fill)
		}
		if p.lineNumbers && ln.gutter != "" {
			gy := y + baselineShift(p.lineSize(ln), p.theme.Meta)
			drawFace(canvas, ln.gutter, geometry.NewRect(b.Min.X+readNum, gy, readTag, h), p.theme.Meta, rgb(p.theme.Muted), false, p.theme.UIFont)
		}
		if ln.quote {
			canvas.DrawLine(geometry.Pt(x0-14, y+2), geometry.Pt(x0-14, y+h-2), rgb(p.theme.Muted), 2)
		}
		x := x0
		if ln.yaml >= 2 {
			x += 12
		}
		if p.buf != nil {
			if lo, hi, selected := p.buf.Modal.VisualColumns(ln.src - 1); selected {
				length := 0
				for _, pc := range ln.pieces {
					length += len([]rune(pc.text))
				}
				left := max(0, lo-ln.col)
				right := min(length-1, hi-ln.col) + 1
				if right > left || length == 0 {
					start := p.prefixWidth(ln, left)
					end := p.prefixWidth(ln, right)
					canvas.DrawRect(geometry.NewRect(x+start, y, max(3, end-start), h), rgba(p.theme.Selection))
				}
			}
		}
		var end float32
		for _, pc := range ln.pieces {
			adv := p.advance(canvas, pc)
			if pc.img != nil {
				dy := (h - pc.imgH) / 2
				if dy < 0 {
					dy = 0
				}
				canvas.DrawImage(pc.img, geometry.Pt(x, y+dy))
			} else if pc.bullet {
				canvas.DrawRect(geometry.NewRect(x+2, y+pc.size*0.45, 4, 4), pc.color)
			} else if pc.box {
				box := geometry.NewRect(x+1, y+pc.size*0.28, 11, 11)
				canvas.StrokeRect(box, pc.color, 1)
				if pc.checked {
					canvas.DrawLine(geometry.Pt(x+3, y+pc.size*0.55), geometry.Pt(x+6, y+pc.size*0.75), rgb(p.theme.Accent), 1.5)
					canvas.DrawLine(geometry.Pt(x+6, y+pc.size*0.75), geometry.Pt(x+11, y+pc.size*0.32), rgb(p.theme.Accent), 1.5)
				}
			} else if pc.text != "" {
				if pc.code {
					canvas.DrawRect(geometry.NewRect(x, y+2, adv, pc.size+2), rgb(p.theme.CodeFill))
				}
				rect := geometry.NewRect(x, y, adv+1, h)
				p.drawPiece(canvas, pc, rect)
				if pc.link != "" {
					p.hits = append(p.hits, linkHit{url: pc.link, rect: geometry.NewRect(x, y, adv, h)})
				}
				if pc.strike {
					mid := y + h/2
					canvas.DrawLine(geometry.Pt(x, mid), geometry.Pt(x+adv, mid), pc.color, 1)
				}
			}
			x += adv
			end = x
		}
		if ln.img != nil {
			canvas.DrawImage(ln.img, geometry.Pt(x, y+4))
		}
		if p.buf != nil && p.openLine > 0 && (ln.src == p.openLine || ln.num == p.openLine) {
			p.drawCaret(canvas, ln, x0, y, h)
		}
		if ln.lang != "" {
			lw := textWidth(canvas, ln.lang, p.theme.Meta, rgb(p.theme.Muted), false, p.theme.UIFont)
			if end+12+lw < x0+width {
				ly := y + baselineShift(p.lineSize(ln), p.theme.Meta)
				drawFace(canvas, ln.lang, geometry.NewRect(x0+width-lw, ly, lw+1, h), p.theme.Meta, rgb(p.theme.Muted), false, p.theme.UIFont)
			}
		}
		y += h
	}
}

func (p *Page) drawFrontHead(canvas widget.Canvas, x, y, h float32) {
	color := rgb(p.theme.Muted)
	xMid, yMid := x+5, y+h/2
	if p.frontExpanded {
		canvas.DrawLine(geometry.Pt(xMid-4, yMid-2), geometry.Pt(xMid, yMid+2), color, 1.5)
		canvas.DrawLine(geometry.Pt(xMid, yMid+2), geometry.Pt(xMid+4, yMid-2), color, 1.5)
	} else {
		canvas.DrawLine(geometry.Pt(xMid-2, yMid-4), geometry.Pt(xMid+2, yMid), color, 1.5)
		canvas.DrawLine(geometry.Pt(xMid+2, yMid), geometry.Pt(xMid-2, yMid+4), color, 1.5)
	}
	label := "Document details"
	w := textWidth(canvas, label, p.theme.Meta, color, false, p.theme.UIFont)
	drawFace(canvas, label, geometry.NewRect(x+16, y, w+2, h), p.theme.Meta, color, false, p.theme.UIFont)
}

func (p *Page) drawNum(canvas widget.Canvas, n int, x, y, rowH, lineSize float32) {
	s := strconv.Itoa(n)
	w := textWidth(canvas, s, p.theme.Meta, rgb(p.theme.Muted), false, p.theme.UIFont)
	y += baselineShift(lineSize, p.theme.Meta)
	drawFace(canvas, s, geometry.NewRect(x+readNum-w-8, y, w+2, rowH), p.theme.Meta, rgb(p.theme.Muted), false, p.theme.UIFont)
}

// baselineShift moves a smaller face down so its baseline matches lineSize
// when both are drawn in a rect of the same height.
func baselineShift(lineSize, faceSize float32) float32 {
	if lineSize <= faceSize {
		return 0
	}
	return interAscent * (lineSize - faceSize)
}

func (p *Page) lineSize(ln rendered) float32 {
	size := p.theme.Body
	for _, pc := range ln.pieces {
		if pc.size > size {
			size = pc.size
		}
	}
	return size
}

func (p *Page) advance(canvas widget.Canvas, pc piece) float32 {
	if pc.pad > 0 {
		return pc.pad
	}
	if pc.img != nil {
		return pc.imgW
	}
	if pc.bullet || pc.box {
		return 16
	}
	if pc.text == "" {
		return 0
	}
	return textWidth(canvas, pc.text, pc.size, pc.color, pc.bold, pc.family)
}

func (p *Page) drawPiece(canvas widget.Canvas, pc piece, rect geometry.Rect) {
	if pc.italic {
		if styled, ok := canvas.(widget.StyledTextDrawer); ok {
			styled.DrawStyledText(pc.text, rect, widget.TextStyle{
				FontFamily: pc.family, FontSize: pc.size, Bold: pc.bold, Italic: true, Color: pc.color,
			})
			return
		}
	}
	drawFace(canvas, pc.text, rect, pc.size, pc.color, pc.bold, pc.family)
}

func (p *Page) drawTableRow(canvas widget.Canvas, x0, y, h float32, ln rendered) {
	rule := rgb(p.theme.Rule)
	var sum float32
	for _, c := range ln.cols {
		sum += c
	}
	if ln.tableH > 0 {
		card := geometry.NewRect(x0, y, sum, ln.tableH)
		canvas.DrawRoundRect(card, rgb(p.theme.Surface), 8)
		canvas.StrokeRoundRect(card, rule, 8, 1)
		canvas.PushClip(geometry.NewRect(x0, y, sum, h))
		canvas.DrawRoundRect(card, mixHex(p.theme.Surface, p.theme.Ink, 0.08), 8)
		canvas.PopClip()
	}
	if !ln.tableEnd {
		canvas.DrawLine(geometry.Pt(x0, y+h), geometry.Pt(x0+sum, y+h), rule, 1)
	}
	x := x0
	lineH := p.theme.Body * proseLead
	for i, col := range ln.cols {
		canvas.PushClip(geometry.NewRect(x, y, col, h))
		for li, runs := range ln.cellLines[i] {
			left := x + cellPad
			top := y + cellPadY + float32(li)*lineH
			for _, pc := range runs {
				w := p.pieceWidth(pc)
				if pc.bullet {
					canvas.DrawCircle(geometry.Pt(left+4, top+lineH/2), 3, rgb(p.theme.Number))
				} else {
					rect := geometry.NewRect(left, top, w+1, lineH)
					if pc.code {
						canvas.DrawRect(rect, rgb(p.theme.CodeFill))
					}
					p.drawPiece(canvas, pc, rect)
					if pc.strike {
						canvas.DrawLine(geometry.Pt(left, top+lineH/2), geometry.Pt(left+w, top+lineH/2), pc.color, 1)
					}
					if pc.link != "" {
						p.hits = append(p.hits, linkHit{url: pc.link, rect: rect})
					}
				}
				left += w
			}
		}
		canvas.PopClip()
		// Keep wiki cells clickable in the whitespace surrounding their text.
		if i < len(ln.cellLinks) && ln.cellLinks[i] != "" {
			p.hits = append(p.hits, linkHit{url: ln.cellLinks[i], rect: geometry.NewRect(x, y, col, h)})
		}
		x += col
	}
}

func (p *Page) drawFigure(canvas widget.Canvas, x0, y float32, ln rendered) {
	numC := rgb(p.theme.Number)
	muted := rgb(p.theme.Muted)
	x := x0
	for i, col := range ln.cols {
		num := ""
		if i < len(ln.cells) {
			num = ln.cells[i]
		}
		nw := textWidth(canvas, num, 28, numC, true, p.theme.UIFont)
		nx := x + (col-nw)/2
		if nx < x {
			nx = x
		}
		drawFace(canvas, num, geometry.NewRect(nx, y+8, nw+2, 28), 28, numC, true, p.theme.UIFont)
		var lines []string
		if i < len(ln.wrap) {
			lines = ln.wrap[i]
		}
		lineH := p.theme.Meta * p.theme.Line
		for li, text := range lines {
			if text == "" {
				continue
			}
			tw := textWidth(canvas, text, p.theme.Meta, muted, false, p.theme.UIFont)
			tx := x + (col-tw)/2
			if tx < x {
				tx = x
			}
			drawFace(canvas, text, geometry.NewRect(tx, y+8+28+4+float32(li)*lineH, tw+2, lineH), p.theme.Meta, muted, false, p.theme.UIFont)
		}
		x += col
	}
}

func splitBullet(s string) (bool, string) {
	for _, m := range []string{"• ", "● ", "* "} {
		if strings.HasPrefix(s, m) {
			return true, strings.TrimPrefix(s, m)
		}
	}
	return false, s
}

// Event edits the source line under the pointer, or opens a link.
// The scroll view owns scrolling.
func (p *Page) Event(ctx widget.Context, e event.Event) bool {
	if p.editKey(ctx, e) {
		return true
	}
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if me.MouseType == event.MouseRelease && p.dragging {
		p.dragging = false
		return true
	}
	if me.MouseType == event.MouseDrag && p.dragging && p.buf != nil {
		line, col := p.pointSource(me.Position)
		if line <= 0 {
			return false
		}
		p.buf.Place(line-1, col)
		p.buf.Modal.Drag(p.buf, p.dragAnchor, p.buf.Cursor())
		p.openLine = p.buf.CaretLine() + 1
		p.MarkNeedsLayout()
		p.SetNeedsRedraw(true)
		if p.onEdit != nil {
			p.onEdit()
		}
		return true
	}
	if me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	if len(p.lines) > 0 && p.lines[0].yaml == 1 {
		top := p.Bounds().Min.Y + p.lines[0].before
		if me.Position.Y >= top && me.Position.Y < top+p.rowHeight(p.lines[0]) {
			p.toggleDetails()
			return true
		}
	}
	if p.openLine == 0 {
		for _, h := range p.hits {
			if h.rect.Contains(me.Position) {
				if p.follow != nil {
					p.follow(h.url)
				}
				return true
			}
		}
	}
	if p.openAt(ctx, me.Position) {
		p.dragging = true
		p.dragAnchor = p.buf.Cursor()
		return true
	}
	return false
}

func (p *Page) toggleDetails() {
	p.frontExpanded = !p.frontExpanded
	if !p.frontExpanded && p.buf != nil && p.openLine > 0 && p.kindAt(p.openLine) == "front" {
		p.buf.EndGroup()
		p.buf.Place(0, 0)
		p.openLine = 1
		p.buf.Modal.Reset()
		if p.onEdit != nil {
			p.onEdit()
		}
	}
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

// Children returns nil. The page draws its own lines.
func (p *Page) Children() []widget.Widget { return nil }

func (p *Page) layoutBlocks(width float32) []rendered {
	th := p.theme
	var lines []rendered
	last := 0
	for i := 0; i < len(p.blocks); i++ {
		b := p.blocks[i]
		if b.Kind == "th" || b.Kind == "tr" {
			j := i + 1
			for j < len(p.blocks) && p.blocks[j].Kind == "tr" {
				j++
			}
			lines = append(lines, p.placeBlock(b, last, width)...)
			last = advanceLine(advanceLine(last, b), p.blocks[j-1])
			if p.covers(b.Line, last) {
				lines = append(lines, p.layoutRawSpan(b.Line, last, "p", width)...)
			} else {
				lines = append(lines, p.layoutTable(p.blocks[i:j], width)...)
			}
			i = j - 1
			continue
		}
		lines = append(lines, p.placeBlock(b, last, width)...)
		last = advanceLine(last, b)
		rows := p.layoutOne(b, width, th)
		if b.Kind == "code" && i > 0 && p.blocks[i-1].Kind == "code" {
			for n := range rows {
				rows[n].before = 0
			}
		}
		lines = append(lines, rows...)
	}
	// Markdown omits trailing blank lines, including the new line created by o.
	if p.buf != nil && p.openLine > last {
		lines = append(lines, p.placeBlock(doc.Block{Line: p.openLine + 1}, last, width)...)
	}
	return lines
}

func advanceLine(last int, b doc.Block) int {
	if b.Line <= 0 {
		return last
	}
	end := b.Line + b.Lines - 1
	if end < b.Line {
		end = b.Line
	}
	if end > last {
		return end
	}
	return last
}

func (p *Page) placeBlock(b doc.Block, last int, width float32) []rendered {
	if b.Line <= last+1 {
		return nil
	}
	var lines []rendered
	for n := last + 1; n < b.Line; n++ {
		if p.covers(n, n) {
			lines = append(lines, p.rawRows(n, "p", false, width)...)
			continue
		}
		lines = append(lines, rendered{num: n, gap: true, src: n})
	}
	return lines
}

const cellPad = float32(8)
const cellPadY = float32(3)

func (p *Page) layoutTable(rows []doc.Block, width float32) []rendered {
	n := 0
	for _, r := range rows {
		if len(r.Cells) > n {
			n = len(r.Cells)
		}
	}
	if n == 0 || width < 1 {
		return nil
	}
	if fig, ok := figureRow(rows); ok {
		return p.layoutFigure(fig, rows[1], width)
	}
	nat := make([]float32, n)
	mins := make([]float32, n)
	share := width / float32(n)
	all := make([][][]piece, len(rows))
	for i, row := range rows {
		all[i] = make([][]piece, n)
		for c, text := range row.Cells {
			spans := []doc.Span{{Text: text}}
			if c < len(row.CellSpans) {
				spans = row.CellSpans[c]
			}
			runs := p.tableRuns(spans, row.Kind == "th")
			all[i][c] = runs
			nat[c] = max(nat[c], p.piecesWidth(runs)+2*cellPad)
			for _, run := range runs {
				for _, word := range strings.Fields(run.text) {
					mins[c] = max(mins[c], min(share, p.advText(word, run.size, run.bold, run.family)+2*cellPad))
				}
			}
		}
	}
	for c := range mins {
		mins[c] = max(mins[c], min(share, p.theme.Body*2+2*cellPad))
		nat[c] = max(nat[c], mins[c])
		if nat[c] <= share {
			mins[c] = nat[c]
		}
	}
	cols := fitColumns(nat, mins, width)
	var out []rendered
	for i, r := range rows {
		lines := make([][][]piece, n)
		for c, runs := range all[i] {
			lines[c] = p.wrapTableRuns(runs, max(float32(1), cols[c]-2*cellPad))
		}
		ln := rendered{cols: cols, cellLines: lines, cellLinks: r.CellLinks, src: r.Line}
		if i > 0 && rows[0].Line > 0 {
			ln.src = rows[0].Line + i + 1
		}
		if i == 0 {
			ln.before = 10
			ln.num = r.Line
		}
		if i == len(rows)-1 {
			ln.tableEnd = true
		}
		out = append(out, ln)
	}
	var total float32
	for i := range out {
		total += p.rowHeight(out[i])
	}
	if len(out) > 0 {
		out[0].tableH = total
	}
	return out
}

func figureRow(rows []doc.Block) (doc.Block, bool) {
	if len(rows) != 2 || rows[0].Kind != "th" || len(rows[0].Cells) < 2 {
		return doc.Block{}, false
	}
	for _, c := range rows[0].Cells {
		if !numericCell(c) {
			return doc.Block{}, false
		}
	}
	return rows[0], true
}

func numericCell(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 3 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (p *Page) layoutFigure(head, body doc.Block, width float32) []rendered {
	n := len(head.Cells)
	cols := make([]float32, n)
	each := width / float32(n)
	for i := range cols {
		cols[i] = each
	}
	char := p.theme.Meta * 0.5
	wrap := make([][]string, n)
	for c := 0; c < n; c++ {
		text := ""
		if c < len(body.Cells) {
			text = body.Cells[c]
		}
		wrap[c] = wrapCell(text, each-8, char)
	}
	return []rendered{{
		cols: cols, wrap: wrap, cells: head.Cells, figure: true,
		before: 12, num: head.Line, src: head.Line,
	}}
}

func fitColumns(nat, mins []float32, width float32) []float32 {
	cols := append([]float32(nil), nat...)
	sum := float32(0)
	flex := float32(0)
	for i, c := range cols {
		sum += c
		if c > mins[i] {
			flex += c - mins[i]
		}
	}
	if sum <= width {
		cols[len(cols)-1] += width - sum
		return cols
	}
	need := sum - width
	if flex > 0 && need < flex {
		for i := range cols {
			if cols[i] <= mins[i] {
				continue
			}
			cols[i] -= need * (cols[i] - mins[i]) / flex
		}
		return cols
	}
	cols = append([]float32(nil), mins...)
	sum = 0
	for _, c := range cols {
		sum += c
	}
	if sum > width && sum > 0 {
		scale := width / sum
		for i := range cols {
			cols[i] *= scale
		}
	} else if sum < width {
		cols[len(cols)-1] += width - sum
	}
	return cols
}

func wrapCell(text string, inner, char float32) []string {
	max := int(inner / char)
	if max < 1 {
		max = 1
	}
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return []string{""}
	}
	var lines []string
	var cur []string
	used := 0
	flush := func() {
		if len(cur) == 0 {
			return
		}
		lines = append(lines, strings.Join(cur, " "))
		cur = nil
		used = 0
	}
	for _, word := range fields {
		for len([]rune(word)) > max {
			if used > 0 {
				flush()
			}
			r := []rune(word)
			lines = append(lines, string(r[:max]))
			word = string(r[max:])
		}
		if word == "" {
			continue
		}
		n := len([]rune(word))
		if used > 0 && used+1+n > max {
			flush()
		}
		if used > 0 {
			used += 1 + n
		} else {
			used = n
		}
		cur = append(cur, word)
	}
	flush()
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func (p *Page) layoutOne(b doc.Block, width float32, th theme.Theme) []rendered {
	if b.Kind == "front" {
		return p.layoutFront(b, th, width)
	}
	literal := p.literalParagraph(b)
	if !literal && p.covers(b.Line, b.Line+b.Lines-1) {
		return p.layoutRaw(b, width)
	}
	ink := rgb(th.Ink)
	muted := rgb(th.Muted)
	accent := rgb(th.Accent)
	codeFill := rgb(th.CodeFill)
	size := th.Body
	before := float32(8)
	color := ink
	forceBold := false
	fill := widget.Color{}
	face := th.UIFont
	switch b.Kind {
	case "h1":
		size, forceBold, before = th.H1, true, 4
	case "h2":
		size, forceBold, before = th.H2, true, 12
	case "h3", "h4", "h5", "h6":
		size, forceBold, before = th.H3, true, 8
	case "p":
		color = soften(th)
		before = 4
	case "alert":
		before = 2
	case "code", "html":
		size, before, color, fill, face = th.Code, 8, muted, codeFill, th.CodeFont
	case "hr":
		return []rendered{{rule: true, before: 8, num: b.Line}}
	case "li", "quote", "fn":
		before = 2
		if b.Kind == "fn" {
			color = soften(th)
		}
	}
	spans := b.Spans
	if len(spans) == 0 && b.Text != "" {
		spans = []doc.Span{{Text: b.Text, Bold: forceBold}}
	}
	for i := range spans {
		if forceBold {
			spans[i].Bold = true
		}
	}
	if b.Kind != "code" && b.Kind != "html" {
		rows := p.layoutFlow(b, spans, width, size, before, color, forceBold, fill, face)
		if p.buf != nil && b.Line > 0 {
			p.mapFlowSource(rows, b)
		}
		return rows
	}
	diagramNote := ""
	if b.Kind == "code" {
		img, note, ok := p.diagramOf(b)
		if ok && img != nil {
			return p.diagramFigure(b, img, width, before)
		}
		diagramNote = note
	}
	wrapped := p.codeRows(b)
	if len(wrapped) == 0 {
		wrapped = [][]doc.Span{{{Text: ""}}}
	}
	indent := strings.Repeat("  ", b.Depth)
	var out []rendered
	for i, row := range wrapped {
		var pieces []piece
		if indent != "" {
			pieces = append(pieces, piece{text: indent, size: size, color: muted, family: face})
		}
		if b.Kind == "li" && i == 0 {
			pieces = append(pieces, markerPiece(b.Marker, size, muted, face))
		} else if b.Kind == "li" {
			pieces = append(pieces, piece{text: "   ", size: size, color: muted, family: face})
		}
		for _, sp := range row {
			c := color
			fam := face
			bold := sp.Bold || forceBold
			if sp.Code {
				c = muted
				fam = th.CodeFont
			}
			if sp.Tone != "" {
				c = toneColor(th, sp.Tone)
				bold = bold || sp.Tone == "keyword"
			}
			if sp.Link != "" {
				c = accent
			}
			pieces = append(pieces, piece{
				text: sp.Text, size: size, bold: bold,
				italic: sp.Italic, strike: sp.Strike, code: sp.Code,
				color: c, family: fam,
			})
		}
		visual := wrapCode(pieces, width, size)
		for v, rowPieces := range visual {
			ln := rendered{pieces: rowPieces, fill: fill, quote: b.Kind == "quote"}
			if b.Kind == "code" || b.Kind == "html" {
				if b.Line > 0 {
					ln.src = b.Line + i
					if v == 0 {
						ln.num = ln.src
					}
				}
				if i == 0 && v == 0 {
					ln.lang = b.Marker
				}
			} else if i == 0 && v == 0 && b.Line > 0 {
				ln.num = b.Line
			}
			if i == 0 && v == 0 && strings.HasPrefix(b.Kind, "h") {
				ln.gutter = b.Kind
			}
			if i == 0 && v == 0 {
				ln.before = before
			}
			out = append(out, ln)
		}
	}
	if diagramNote != "" {
		noteLn := rendered{
			pieces: []piece{{text: diagramNote, size: th.Meta, color: muted, family: th.UIFont}},
			before: before,
		}
		if len(out) > 0 {
			out[0].before = 2
		}
		out = append([]rendered{noteLn}, out...)
	}
	return out
}

func (p *Page) layoutFront(b doc.Block, th theme.Theme, width float32) []rendered {
	out := []rendered{{yaml: 1, before: 2}}
	if !p.frontExpanded {
		out[0].num, out[0].src = b.Line, b.Line
		return out
	}
	literal := p.literalParagraph(b)
	if !literal && p.covers(b.Line, b.Line+b.Lines-1) {
		return append(out, p.layoutRaw(b, width)...)
	}
	out[0].num, out[0].src = b.Line, b.Line
	lines := strings.Split(b.Text, "\n")
	if b.Text == "" {
		lines = nil
	}
	for i, line := range lines {
		var pieces []piece
		key, val, ok := strings.Cut(line, ":")
		if ok {
			pieces = append(pieces, piece{text: key + ":", size: th.Body, color: rgb(th.Accent), family: th.UIFont})
			for _, sp := range doc.Linkify(val) {
				pc := piece{text: sp.Text, size: th.Body, color: rgb(th.Muted), family: th.UIFont, link: sp.Link}
				if sp.Link != "" {
					pc.color = rgb(th.Accent)
				}
				pieces = append(pieces, pc)
			}
		} else if strings.TrimSpace(line) != "" {
			pieces = append(pieces, piece{text: line, size: th.Body, color: rgb(th.Muted), family: th.UIFont})
		}
		n := b.Line + 1 + i
		out = append(out, rendered{yaml: 2, pieces: pieces, num: n, src: n})
	}
	closeLine := b.Line + b.Lines - 1
	out = append(out, rendered{
		yaml: 3, num: closeLine, src: closeLine,
		pieces: []piece{{text: "---", size: th.Body, color: rgb(th.Muted), family: th.UIFont}},
	})
	return out
}

func (p *Page) diagramOf(b doc.Block) (image.Image, string, bool) {
	if !diagram.Known(b.Marker) || strings.TrimSpace(b.Text) == "" {
		return nil, "", false
	}
	key := strings.ToLower(strings.TrimSpace(b.Marker)) + "\n" + b.Text
	if p.diagrams == nil {
		p.diagrams = map[string]diagramHit{}
	}
	hit, ok := p.diagrams[key]
	if !ok {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		hit.img, hit.err = diagram.Render(ctx, b.Marker, b.Text)
		cancel()
		p.diagrams[key] = hit
	}
	if hit.err != nil || hit.img == nil {
		return nil, diagramNote(hit.err), true
	}
	return hit.img, "", true
}

func (p *Page) codeRows(b doc.Block) [][]doc.Span {
	if b.Kind == "code" && b.Marker != "" && !diagram.Known(b.Marker) {
		if rows := doc.TokenLines("", b.Marker, b.Text); len(rows) > 0 {
			return rows
		}
	}
	if len(b.Spans) > 0 && !strings.Contains(b.Text, "\n") {
		return [][]doc.Span{b.Spans}
	}
	parts := strings.Split(b.Text, "\n")
	if len(parts) == 0 {
		return [][]doc.Span{{{Text: ""}}}
	}
	rows := make([][]doc.Span, len(parts))
	for i, line := range parts {
		rows[i] = []doc.Span{{Text: line}}
	}
	return rows
}

// wrapCode breaks a source line on the column. It does not reflow words.
func wrapCode(pieces []piece, width, size float32) [][]piece {
	if size < 1 {
		size = 1
	}
	adv := size * 0.6
	if width < adv {
		width = adv
	}
	var rows [][]piece
	var cur []piece
	var used float32
	flush := func() {
		if len(cur) == 0 {
			return
		}
		rows = append(rows, cur)
		cur = nil
		used = 0
	}
	for _, pc := range pieces {
		runes := []rune(pc.text)
		if len(runes) == 0 {
			continue
		}
		for len(runes) > 0 {
			room := int((width - used) / adv)
			if room < 1 {
				if len(cur) > 0 {
					flush()
					continue
				}
				room = 1
			}
			if room > len(runes) {
				room = len(runes)
			}
			next := pc
			next.text = string(runes[:room])
			cur = append(cur, next)
			used += float32(room) * adv
			runes = runes[room:]
		}
	}
	if len(cur) == 0 {
		return [][]piece{nil}
	}
	flush()
	return rows
}

func toneColor(th theme.Theme, tone string) widget.Color {
	switch tone {
	case "comment":
		return rgb(th.Muted)
	case "keyword":
		return rgb(th.Accent)
	case "string":
		return rgb(th.String)
	case "num":
		return rgb(th.Number)
	case "punct":
		return rgb(th.Muted)
	case "heading":
		return rgb(th.Accent)
	case "strong", "emph":
		return rgb(th.Ink)
	default:
		return rgb(th.Ink)
	}
}

func diagramNote(err error) string {
	var missing *diagram.NotInstalledError
	if errors.As(err, &missing) {
		return missing.Error()
	}
	return "diagram failed"
}

func (p *Page) diagramFigure(b doc.Block, img image.Image, width, before float32) []rendered {
	maxW := width
	if maxW < 48 {
		maxW = 48
	}
	// Fit the width only. A height cap crushes a wide diagram and the labels go soft.
	fw, fh := fitPicture(img.Bounds().Dx(), img.Bounds().Dy(), maxW, diagramMaxH)
	fitted := scalePicture(img, fw, fh)
	return []rendered{{
		img:    fitted,
		imgW:   float32(fitted.Bounds().Dx()),
		imgH:   float32(fitted.Bounds().Dy()),
		before: before,
		num:    b.Line,
	}}
}

func (p *Page) layoutFlow(b doc.Block, spans []doc.Span, width, size, before float32, color widget.Color, forceBold bool, fill widget.Color, face string) []rendered {
	muted := rgb(p.theme.Muted)
	accent := rgb(p.theme.Accent)
	head := p.linePrefix(b, size, muted, face)
	hang := p.piecesWidth(head)
	textW := width - hang
	if textW < 48 {
		textW = 48
	}
	var out []rendered
	var cur []piece
	var curW float32
	flush := func() {
		if len(cur) == 0 {
			return
		}
		ln := rendered{pieces: cur}
		if hang > 0 {
			if len(out) == 0 {
				ln.pieces = append(append([]piece{}, head...), cur...)
			} else {
				ln.pieces = append([]piece{{pad: hang}}, cur...)
			}
		}
		out = append(out, ln)
		cur = nil
		curW = 0
	}
	add := func(pc piece) {
		adv := p.pieceWidth(pc)
		if adv > textW && pc.text != "" {
			flush()
			for _, chunk := range p.wrapRaw(pc.text, textW, pc.size, pc.bold, pc.family) {
				if len(cur) > 0 {
					flush()
				}
				part := pc
				part.text = chunk.text
				cur = append(cur, part)
				curW = p.pieceWidth(part)
			}
			return
		}
		if len(cur) > 0 && curW+adv > textW {
			flush()
		}
		cur = append(cur, pc)
		curW += adv
	}
	space := func() {
		if len(cur) == 0 {
			return
		}
		last := cur[len(cur)-1]
		if last.text == " " {
			return
		}
		pc := piece{text: " ", size: size, color: color, family: face}
		if curW+p.pieceWidth(pc) > textW {
			flush()
			return
		}
		add(pc)
	}
	word := func(sp doc.Span, text string) {
		if text == "" {
			return
		}
		if sp.Note {
			add(piece{
				text: text, size: size * 0.65, color: accent, family: face, link: sp.Link,
			})
			return
		}
		if !tightPunct(text) && !stuckTo(cur, sp.Link) {
			space()
		}
		pc := piece{
			text: text, size: size, bold: sp.Bold || forceBold,
			italic: sp.Italic, strike: sp.Strike, code: sp.Code,
			color: color, family: face,
		}
		if sp.Code {
			pc.color = muted
			pc.family = p.theme.CodeFont
		}
		if sp.Link != "" && !tightPunct(text) {
			pc.color = accent
			pc.link = sp.Link
		}
		add(pc)
	}
	for _, sp := range spans {
		if sp.Image {
			img := p.loadPicture(sp.Link)
			if img == nil {
				word(doc.Span{}, imageLabel(sp.Text, sp.Link))
				if len(cur) > 0 {
					cur[len(cur)-1].color = muted
					cur[len(cur)-1].family = face
				}
				continue
			}
			// An image stays in the line, like a Zettlr preview. It keeps its
			// pixel size unless it is wider than the column, and it wraps to
			// the next line of the same paragraph when it does not fit.
			nw, nh := img.Bounds().Dx(), img.Bounds().Dy()
			fw, fh := fitPicture(nw, nh, textW, diagramMaxH)
			fitted := p.fitted(sp.Link, img, fw, fh)
			adv := float32(fitted.Bounds().Dx())
			gap := float32(0)
			if len(cur) > 0 && cur[len(cur)-1].text != " " {
				gap = p.advText(" ", size, false, face)
			}
			if len(cur) > 0 && curW+gap+adv > textW {
				flush()
				gap = 0
			}
			if gap > 0 {
				add(piece{text: " ", size: size, color: color, family: face})
			}
			add(piece{img: fitted, imgW: adv, imgH: float32(fitted.Bounds().Dy())})
			continue
		}
		parts := strings.Split(sp.Text, "\n")
		for i, part := range parts {
			if i > 0 {
				flush()
			}
			for _, w := range strings.Fields(part) {
				word(sp, w)
			}
		}
	}
	flush()
	if len(out) == 0 {
		return nil
	}
	for i := range out {
		out[i].quote = b.Kind == "quote"
		out[i].fill = fill
	}
	if b.Line > 0 {
		out[0].num = b.Line
	}
	if strings.HasPrefix(b.Kind, "h") {
		out[0].gutter = b.Kind
	}
	out[0].before = before
	for i := range out {
		if out[i].src == 0 {
			out[i].src = b.Line
		}
	}
	return out
}

// linePrefix is the marker drawn on the first line of a block. Wrapped
// lines keep a blank of the same width, so the text stacks under the text
// and not under the number.
func (p *Page) linePrefix(b doc.Block, size float32, muted widget.Color, face string) []piece {
	var pre []piece
	if b.Depth > 0 {
		pre = append(pre, piece{text: strings.Repeat("  ", b.Depth), size: size, color: muted, family: face})
	}
	switch b.Kind {
	case "li", "fn":
		pre = append(pre, markerPiece(b.Marker, size, muted, face))
	case "alert":
		if b.Marker != "" {
			pre = append(pre, piece{text: alertMark(b.Marker) + " ", size: size, color: alertColor(p.theme, b.Marker), family: face})
		} else {
			pre = append(pre, piece{text: "    ", size: size, color: muted, family: face})
		}
	}
	return pre
}

func (p *Page) piecesWidth(pcs []piece) float32 {
	var w float32
	for _, pc := range pcs {
		w += p.pieceWidth(pc)
	}
	return w
}

func (p *Page) pieceWidth(pc piece) float32 {
	if pc.pad > 0 {
		return pc.pad
	}
	if pc.img != nil {
		return pc.imgW
	}
	if pc.bullet || pc.box {
		return 16
	}
	if pc.text == "" {
		return 0
	}
	return p.advText(pc.text, pc.size, pc.bold, pc.family)
}

func (p *Page) advText(text string, size float32, bold bool, family string) float32 {
	if p.measure != nil {
		return p.measure(text, size, bold, family)
	}
	// Courier runs wider than Inter. A half-em stand-in puts the end of a
	// code chip under the scrollbar.
	em := float32(0.5)
	if family != "" && family == p.theme.CodeFont {
		em = 0.62
	}
	return float32(len([]rune(text))) * size * em
}

func stuckTo(cur []piece, link string) bool {
	if len(cur) == 0 || !doc.IsDocLink(link) {
		return false
	}
	last := cur[len(cur)-1].text
	if last == "" {
		return false
	}
	r := []rune(last)
	switch r[len(r)-1] {
	case '(', '[', '{':
		return true
	default:
		return false
	}
}

func tightPunct(s string) bool {
	if s == "" || len([]rune(s)) > 3 {
		return false
	}
	for _, r := range s {
		switch r {
		case ',', '.', ';', ':', '!', '?', ')', ']':
		default:
			return false
		}
	}
	return true
}

func markerPiece(marker string, size float32, color widget.Color, face string) piece {
	switch marker {
	case "", "·":
		return piece{bullet: true, size: size, color: color}
	case "[x]":
		return piece{box: true, checked: true, size: size, color: color}
	case "[ ]":
		return piece{box: true, size: size, color: color}
	default:
		return piece{text: marker + "  ", size: size, color: color, family: face}
	}
}

func wrapSpans(spans []doc.Span, maxChars int) [][]doc.Span {
	var lines [][]doc.Span
	var cur []doc.Span
	used := 0
	flush := func() {
		if len(cur) == 0 {
			return
		}
		lines = append(lines, cur)
		cur = nil
		used = 0
	}
	for _, sp := range spans {
		if sp.Image {
			continue
		}
		parts := strings.Split(sp.Text, "\n")
		for i, part := range parts {
			if i > 0 {
				flush()
			}
			for _, word := range strings.Fields(part) {
				n := len([]rune(word))
				if used > 0 && used+1+n > maxChars {
					flush()
				}
				if used > 0 {
					cur = append(cur, doc.Span{Text: " "})
					used++
				}
				cur = append(cur, doc.Span{
					Text: word, Bold: sp.Bold, Italic: sp.Italic,
					Code: sp.Code, Strike: sp.Strike, Link: sp.Link, Image: sp.Image,
				})
				used += n
			}
		}
	}
	flush()
	if len(lines) == 0 {
		return [][]doc.Span{{{Text: ""}}}
	}
	return lines
}
