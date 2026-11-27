package ui

import (
	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/edit"
)

// windowGeom is the viewport height and the current scroll offset.
// The page's own bounds are the document, so they are not the window.
func (p *Page) windowGeom() (height, scroll float32, ok bool) {
	var w widget.Widget = p
	for {
		next := parentOf(w)
		if next == nil {
			break
		}
		if sv, is := next.(*scrollview.Widget); is {
			h := sv.ViewportSize().Height
			_, y := sv.ScrollOffset()
			if h > 1 {
				return h, y, true
			}
		}
		w = next
	}
	return 0, 0, false
}

func parentOf(w widget.Widget) widget.Widget {
	parent, ok := w.(interface{ Parent() widget.Widget })
	if !ok {
		return nil
	}
	next := parent.Parent()
	if next == nil || next == w {
		return nil
	}
	return next
}

func (p *Page) bodyLine() float32 {
	h := p.theme.Body * p.theme.Line
	if h < 1 {
		h = 20
	}
	return h
}

func (p *Page) contentH() float32 {
	var h float32
	for _, ln := range p.lines {
		h += ln.before + p.rowHeight(ln)
	}
	return h
}

// followNav turns a navigation result into a scroll offset and a caret.
// Page motions move by a viewport. The caret stays on the same screen row.
func (p *Page) followNav(cur, viewH float32, prevLine, prevCol int, res edit.Result) {
	if p.buf == nil || viewH < 1 {
		return
	}
	y, line := p.navTarget(cur, viewH, prevLine, res)
	if line >= 0 {
		p.buf.Place(line, prevCol)
	}
	if p.onScroll != nil {
		p.onScroll(y)
	}
}

func (p *Page) navTarget(cur, viewH float32, prevLine int, res edit.Result) (float32, int) {
	max := p.contentH() - viewH
	if max < 0 {
		max = 0
	}
	clamp := func(v float32) float32 {
		if v < 0 {
			return 0
		}
		if v > max {
			return max
		}
		return v
	}
	body := p.bodyLine()
	caret := p.buf.CaretLine() + 1
	switch res.Place {
	case 'z':
		top, _, _ := p.lineBox(caret)
		return clamp(top - viewH/2), -1
	case 't':
		top, _, _ := p.lineBox(caret)
		return clamp(top), -1
	case 'b':
		top, h, _ := p.lineBox(caret)
		return clamp(top + h - viewH), -1
	case 'H':
		return cur, p.lineAt(cur) - 1
	case 'M':
		return cur, p.lineAt(cur+viewH/2) - 1
	case 'L':
		return cur, p.lineAt(cur+viewH-body) - 1
	}
	if res.Scroll != 0 && res.Pull {
		step := p.rowPx(cur, res.Scroll < 0)
		y := clamp(cur + float32(signOf(res.Scroll))*step)
		top := p.lineAt(y)
		bot := p.lineAt(y + viewH - 1)
		c := caret
		if c < top {
			return y, top - 1
		}
		if c > bot {
			return y, bot - 1
		}
		return y, -1
	}
	if res.Scroll != 0 {
		y := clamp(cur + float32(res.Scroll)*body)
		top, _, _ := p.lineBox(prevLine + 1)
		screen := top - cur
		if screen < 0 {
			screen = 0
		}
		if screen > viewH-body {
			screen = viewH - body
		}
		return y, p.lineAt(y+screen) - 1
	}
	return p.revealScroll(cur, viewH), -1
}

func (p *Page) revealScroll(cur, viewH float32) float32 {
	top, h, ok := p.lineBox(p.buf.CaretLine() + 1)
	if !ok {
		return cur
	}
	if top < cur {
		return top
	}
	if top+h > cur+viewH {
		y := top + h - viewH
		if y < 0 {
			return 0
		}
		return y
	}
	return cur
}

func (p *Page) lineBox(line int) (top, height float32, ok bool) {
	var y float32
	for _, ln := range p.lines {
		h := ln.before + p.rowHeight(ln)
		if ln.num == line || ln.src == line {
			if h < 1 {
				h = p.bodyLine()
			}
			return y, h, true
		}
		y += h
	}
	return 0, p.bodyLine(), false
}

func (p *Page) lineAt(y float32) int {
	var at float32
	last := 1
	for _, ln := range p.lines {
		h := ln.before + p.rowHeight(ln)
		if ln.num > 0 {
			last = ln.num
		} else if ln.src > 0 {
			last = ln.src
		}
		if at+h > y {
			return last
		}
		at += h
	}
	return last
}

func (p *Page) rowPx(at float32, up bool) float32 {
	var y float32
	var prev float32
	body := p.bodyLine()
	for _, ln := range p.lines {
		h := ln.before + p.rowHeight(ln)
		if h < 1 {
			h = body
		}
		if up {
			if y+h > at+0.5 {
				if prev > 0 {
					return prev
				}
				return h
			}
			prev = h
		} else if y+h > at+0.5 {
			return h
		}
		y += h
	}
	if up && prev > 0 {
		return prev
	}
	return body
}

func signOf(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}
