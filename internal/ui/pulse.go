package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// Pulse is the window root. It ticks on the UI thread, then draws one child.
type Pulse struct {
	widget.WidgetBase
	child widget.Widget
	tick  func(widget.Context)
}

// NewPulse returns a root that calls tick before each layout.
func NewPulse(child widget.Widget, tick func(widget.Context)) *Pulse {
	p := &Pulse{child: child, tick: tick}
	p.SetVisible(true)
	p.SetEnabled(true)
	if child != nil {
		if s, ok := child.(interface{ SetParent(widget.Widget) }); ok {
			s.SetParent(p)
		}
	}
	return p
}

// TickAnimation runs on the UI thread before layout.
func (p *Pulse) TickAnimation(ctx widget.Context) {
	if p.tick != nil {
		p.tick(ctx)
	}
}

// Layout gives the child the window.
func (p *Pulse) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	if p.child == nil {
		return c.Constrain(geometry.Sz(0, 0))
	}
	size := widget.LayoutChild(p.child, ctx, c)
	if s, ok := p.child.(interface{ SetBounds(geometry.Rect) }); ok {
		s.SetBounds(geometry.FromPointSize(geometry.Pt(0, 0), size))
	}
	return size
}

// Draw paints the child.
func (p *Pulse) Draw(ctx widget.Context, canvas widget.Canvas) {
	if p.child == nil {
		return
	}
	widget.DrawChild(p.child, ctx, canvas)
	drawTip(canvas, p.Bounds())
}

// Event forwards to the child.
func (p *Pulse) Event(ctx widget.Context, e event.Event) bool {
	if p.child == nil {
		return false
	}
	return p.child.Event(ctx, e)
}

// Children is the wrapped tree.
func (p *Pulse) Children() []widget.Widget {
	if p.child == nil {
		return nil
	}
	return []widget.Widget{p.child}
}
