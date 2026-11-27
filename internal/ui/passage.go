package ui

import "pairti-lense/internal/edit"

// Passage identifies a source line and its vertical position in the viewport.
// The caret anchors an editing session. Scrolling past it anchors the top line.
type Passage struct {
	Line   int
	Offset float32
}

func (p *Page) Passage() Passage {
	h, y, ok := p.windowGeom()
	if !ok {
		h = p.Bounds().Height()
	}
	if p.buf != nil {
		line := p.buf.CaretLine() + 1
		if top, height, found := p.lineBox(line); found && top >= y && top+height <= y+h {
			return Passage{line, top - y}
		}
	}
	return Passage{Line: p.lineAt(y)}
}

// ResumeFormatted renders the active block and keeps keyboard editing available.
// Resolve the scroll offset after the destination width has been laid out.
func (p *Page) ResumeFormatted(anchor Passage) {
	if p.buf != nil && p.buf.Modal.Mode != edit.ModeView {
		p.Reopen(p.buf.CaretLine() + 1)
	}
	p.anchor = anchor
	p.MarkNeedsLayout()
}

func (e *Editor) Passage() Passage {
	rows := e.visualRows()
	caret := rowAt(rows, e.buf.CaretLine(), e.buf.CaretCol())
	y := float32(caret) * e.lineH()
	if y >= e.scroll && y+e.lineH() <= e.scroll+e.Bounds().Height() {
		return Passage{e.buf.CaretLine() + 1, y - e.scroll}
	}
	top := min(len(rows)-1, int(e.scroll/e.lineH()))
	return Passage{Line: rows[top].line + 1}
}

func (e *Editor) restoreRows(anchor Passage, rows []sourceRow) {
	col := 0
	if anchor.Line == e.buf.CaretLine()+1 {
		col = e.buf.CaretCol()
	}
	e.scroll = max(0, float32(rowAt(rows, anchor.Line-1, col))*e.lineH()-anchor.Offset)
}

func (e *Editor) RestorePassage(anchor Passage) {
	e.pendingPassage = anchor
	e.restoreRows(anchor, e.visualRows())
	e.clampScroll()
	e.MarkNeedsLayout()
	e.SetNeedsRedraw(true)
}
