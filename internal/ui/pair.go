package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// Pair shows one document, or two side by side after a tab is dropped.
type Pair struct {
	widget.WidgetBase
	a, b     widget.Widget
	on       bool
	width    float32
	theme    theme.Theme
	dragging bool
	startX   float32
	startW   float32
}

// NewPair returns a single pane. SetSplit opens the second.
func NewPair(a, b widget.Widget) *Pair {
	p := &Pair{a: a, b: b, theme: theme.Pairti()}
	p.SetVisible(true)
	p.SetEnabled(true)
	type parentSetter interface{ SetParent(widget.Widget) }
	if s, ok := a.(parentSetter); ok {
		s.SetParent(p)
	}
	if s, ok := b.(parentSetter); ok {
		s.SetParent(p)
	}
	return p
}

// SetTheme replaces the semantic theme.
func (p *Pair) SetTheme(t theme.Theme) { p.theme = t; p.SetNeedsRedraw(true) }

// SetSplit shows or hides the second document.
func (p *Pair) SetSplit(on bool) {
	if p.on == on {
		return
	}
	p.on = on
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

// Split reports whether both documents are showing.
func (p *Pair) Split() bool { return p.on }

// Layout gives the second pane half the width when the pair is split.
func (p *Pair) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.BiggestFinite(800, 600)
	if !p.on {
		place(ctx, p.a, geometry.NewRect(0, 0, size.Width, size.Height))
		place(ctx, p.b, geometry.Rect{})
		return size
	}
	p.width = pairWidth(p.width, size.Width)
	gap := splitDivider
	left := geometry.NewRect(0, 0, p.width, size.Height)
	rightX := p.width + gap
	place(ctx, p.a, left)
	place(ctx, p.b, geometry.NewRect(rightX, 0, size.Width-rightX, size.Height))
	return size
}

func pairWidth(current, total float32) float32 {
	if current <= 0 {
		current = (total - splitDivider) / 2
	}
	return SidebarWidth(current, total)
}

// Draw paints the visible panes and the divider.
func (p *Pair) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := p.Bounds()
	if b.IsEmpty() {
		return
	}
	canvas.PushTransform(b.Min)
	if p.a != nil {
		widget.StampScreenOrigin(p.a, canvas)
		widget.DrawChild(p.a, ctx, canvas)
	}
	if p.on && p.b != nil {
		widget.StampScreenOrigin(p.b, canvas)
		widget.DrawChild(p.b, ctx, canvas)
		x := p.width + splitDivider/2
		canvas.DrawLine(geometry.Pt(x, 0), geometry.Pt(x, b.Height()), rgb(p.theme.Rule), 1)
	}
	canvas.PopTransform()
}

// Event drags the divider. Other input goes to the pane under the pointer.
func (p *Pair) Event(ctx widget.Context, e event.Event) bool {
	if _, ok := e.(*event.KeyEvent); ok {
		if p.a != nil && p.a.Event(ctx, e) {
			return true
		}
		if p.on && p.b != nil && p.b.Event(ctx, e) {
			return true
		}
		return false
	}
	if we, ok := e.(*event.WheelEvent); ok {
		cp := *we
		cp.Position = we.Position.Sub(p.Bounds().Min)
		return p.dispatch(ctx, &cp, cp.Position)
	}
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	local := me.Position.Sub(p.Bounds().Min)
	if p.on && (p.dragging || p.divider().Contains(local)) {
		offerResize(ctx)
	}
	if p.on {
		switch me.MouseType {
		case event.MousePress:
			if me.Button == event.ButtonLeft && p.divider().Contains(local) {
				p.dragging = true
				p.startX = local.X
				p.startW = p.width
				return true
			}
		case event.MouseDrag, event.MouseMove:
			if p.dragging {
				if !me.Buttons.IsLeftPressed() {
					p.dragging = false
					return false
				}
				p.width = pairWidth(p.startW+(local.X-p.startX), p.Bounds().Width())
				p.MarkNeedsLayout()
				p.SetNeedsRedraw(true)
				return true
			}
		case event.MouseRelease:
			if p.dragging {
				p.dragging = false
				return true
			}
		}
	}
	cp := *me
	cp.Position = local
	return p.dispatch(ctx, &cp, local)
}

func (p *Pair) dispatch(ctx widget.Context, e event.Event, local geometry.Point) bool {
	if p.a != nil && contains(p.a, local) && p.a.Event(ctx, e) {
		return true
	}
	if p.on && p.b != nil && contains(p.b, local) && p.b.Event(ctx, e) {
		return true
	}
	return false
}

func (p *Pair) divider() geometry.Rect {
	if !p.on {
		return geometry.Rect{}
	}
	return geometry.NewRect(p.width, 0, splitDivider, p.Bounds().Height())
}

// Children returns both panes so focus can find an editor.
func (p *Pair) Children() []widget.Widget {
	return []widget.Widget{p.a, p.b}
}
