package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const (
	tabH      = float32(28)
	tabPad    = float32(8)
	tabGap    = float32(6)
	tabCloseW = float32(20)
)

// Tab is one open document in the tab bar.
type Tab struct {
	Label string
	Dirty bool
}

// Tabs draws the open documents. A drag that ends on the right of the bar
// asks for that document to sit beside the other one.
type Tabs struct {
	widget.WidgetBase
	items    []Tab
	active   int
	side     int
	theme    theme.Theme
	onSelect func(int)
	onClose  func(int)
	onSplit  func(int)
	onMenu   func(int, geometry.Point)
	widths   []float32
	press    int
	pressX   float32
	dragging bool
}

// NewTabs returns an empty bar. side is the document already on the right, or -1.
func NewTabs(onSelect, onClose, onSplit func(int)) *Tabs {
	t := &Tabs{
		theme: theme.Pairti(), active: -1, side: -1, press: -1,
		onSelect: onSelect, onClose: onClose, onSplit: onSplit,
	}
	t.SetVisible(true)
	t.SetEnabled(true)
	return t
}

// OnMenu runs for a right click on a tab. The point is in window coordinates.
func (t *Tabs) OnMenu(fn func(int, geometry.Point)) { t.onMenu = fn }

// SetTheme replaces the semantic theme.
func (t *Tabs) SetTheme(th theme.Theme) {
	t.theme = th
	t.SetNeedsRedraw(true)
}

// Set replaces the tabs. active is the focused document. side is the other pane.
func (t *Tabs) Set(items []Tab, active, side int) {
	t.items = items
	t.active = active
	t.side = side
	t.MarkNeedsLayout()
	t.SetNeedsRedraw(true)
}

// Layout sizes one row. Tabs share the width.
func (t *Tabs) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	w := c.MaxWidth
	if w <= 0 || w >= geometry.Infinity {
		w = 480
	}
	t.widths = tabWidths(len(t.items), w)
	return geometry.Sz(w, tabH)
}

func tabWidths(n int, total float32) []float32 {
	if n == 0 {
		return nil
	}
	w := total / float32(n)
	if w > 168 {
		w = 168
	}
	if w < 48 {
		w = 48
	}
	out := make([]float32, n)
	for i := range out {
		out[i] = w
	}
	return out
}

// Draw paints the filename, a dirty dot, and a close mark.
func (t *Tabs) Draw(_ widget.Context, canvas widget.Canvas) {
	if !t.IsVisible() {
		return
	}
	b := t.Bounds()
	ink := rgb(t.theme.Ink)
	muted := rgb(t.theme.Muted)
	accent := rgb(t.theme.Accent)
	x := b.Min.X
	for i, tab := range t.items {
		if i >= len(t.widths) {
			break
		}
		w := t.widths[i]
		if t.active == i || t.side == i {
			canvas.DrawRect(geometry.NewRect(x, b.Min.Y, w, tabH), rgb(t.theme.Surface))
		}
		color := muted
		if t.active == i {
			color = ink
			canvas.DrawRect(geometry.NewRect(x, b.Min.Y+tabH-2, w, 2), accent)
		} else if t.side == i {
			canvas.DrawRect(geometry.NewRect(x, b.Min.Y+tabH-2, w, 2), rgb(t.theme.Rule))
		}
		label := tab.Label
		if tab.Dirty {
			label = "● " + label
		}
		textW := tabLabelWidth(w)
		label = fitMeasured(canvas, label, textW, t.theme.Meta, t.theme.UIFont, color)
		textBox := geometry.NewRect(x+tabPad, b.Min.Y, textW, tabH)
		canvas.PushClip(textBox)
		drawFace(canvas, label, geometry.NewRect(textBox.Min.X, b.Min.Y+4, textW, 20), t.theme.Meta, color, false, t.theme.UIFont)
		canvas.PopClip()
		drawFace(canvas, "×", geometry.NewRect(x+w-tabCloseW, b.Min.Y+4, tabCloseW, 20), t.theme.Meta, muted, false, t.theme.UIFont)
		x += w
	}

}

// Event selects, closes, or splits. A drag onto the right half splits.
// A move over the close mark names it.
func (t *Tabs) Event(ctx widget.Context, e event.Event) bool {
	if t.hover(ctx, e) {
		return false
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.Button != event.ButtonLeft && me.Button != event.ButtonRight && me.MouseType != event.MouseDrag {
		return false
	}
	local := me.Position.X - t.Bounds().Min.X
	switch me.MouseType {
	case event.MousePress:
		if me.Button == event.ButtonRight {
			return t.rightClick(ctx, me, local)
		}
		if !t.Bounds().Contains(me.Position) {
			return false
		}
		i := t.hit(local)
		if i < 0 {
			return false
		}
		t.press = i
		t.pressX = me.Position.X
		t.dragging = false
		if t.closing(i, local) {
			if t.onClose != nil {
				t.onClose(i)
			}
			t.press = -1
			t.redraw(ctx)
			return true
		}
		if t.onSelect != nil {
			t.onSelect(i)
		}
		t.redraw(ctx)
		return true
	case event.MouseDrag:
		if t.press < 0 {
			return false
		}
		if me.Position.X-t.pressX > 8 || t.pressX-me.Position.X > 8 {
			t.dragging = true
		}
		return true
	case event.MouseRelease:
		if t.press < 0 {
			return false
		}
		i := t.press
		dragged := t.dragging
		t.press = -1
		t.dragging = false
		if dragged && t.onSplit != nil && (local > t.Bounds().Width()*0.55 || me.Position.X-t.pressX > 72) {
			t.onSplit(i)
			t.redraw(ctx)
		}
		return true
	default:
		return false
	}
}

func (t *Tabs) rightClick(ctx widget.Context, me *event.MouseEvent, local float32) bool {
	if !t.Bounds().Contains(me.Position) {
		return false
	}
	i := t.hit(local)
	if i < 0 {
		return true
	}
	if t.onMenu != nil {
		origin := screenOrigin(t)
		at := me.Position.Sub(t.Bounds().Min)
		t.onMenu(i, geometry.Pt(origin.X+at.X, origin.Y+at.Y))
	}
	t.redraw(ctx)
	return true
}

func (t *Tabs) redraw(ctx widget.Context) {
	t.SetNeedsRedraw(true)
	if ctx != nil {
		ctx.Invalidate()
	}
}

func (t *Tabs) hover(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if me.MouseType == event.MouseLeave {
		offerTip(ctx, t.theme, "", geometry.Rect{})
		return true
	}
	local, over := hoverPoint(t, me)
	if !over {
		return false
	}
	label := ""
	var box geometry.Rect
	i := t.hit(local.X)
	if t.closing(i, local.X) {
		label = "Close"
		var x float32
		for n := 0; n < i; n++ {
			x += t.widths[n]
		}
		origin := screenOrigin(t)
		box = geometry.NewRect(origin.X+x+t.widths[i]-tabCloseW, origin.Y, tabCloseW, tabH)
	}
	offerTip(ctx, t.theme, label, box)
	return true
}

func (t *Tabs) hit(localX float32) int {
	var x float32
	for i, w := range t.widths {
		if localX >= x && localX < x+w {
			return i
		}
		x += w
	}
	return -1
}

func tabLabelWidth(w float32) float32 {
	room := w - tabPad - tabGap - tabCloseW
	if room < 0 {
		return 0
	}
	return room
}

func fitMeasured(canvas widget.Canvas, s string, px, size float32, family string, col widget.Color) string {
	if s == "" || px <= 0 {
		return ""
	}
	if textWidth(canvas, s, size, col, false, family) <= px {
		return s
	}
	r := []rune(s)
	lo, hi := 0, len(r)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		trial := string(r[:mid]) + "…"
		if textWidth(canvas, trial, size, col, false, family) <= px {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	if lo < 1 {
		return "…"
	}
	return string(r[:lo]) + "…"
}

func (t *Tabs) closing(i int, localX float32) bool {
	if i < 0 || i >= len(t.widths) {
		return false
	}
	var x float32
	for n := 0; n < i; n++ {
		x += t.widths[n]
	}
	return localX >= x+t.widths[i]-tabCloseW && localX < x+t.widths[i]
}

// Children returns nil.
func (t *Tabs) Children() []widget.Widget { return nil }
