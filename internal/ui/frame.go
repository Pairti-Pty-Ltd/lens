package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// Frame shows one of two children and gives it the whole viewport.
type Frame struct {
	widget.WidgetBase
	a, b    widget.Widget
	useB    bool
	overlay widget.Widget
}

// NewFrame returns a frame showing a.
func NewFrame(a, b widget.Widget) *Frame {
	f := &Frame{a: a, b: b}
	f.SetVisible(true)
	f.SetEnabled(true)
	type parentSetter interface{ SetParent(widget.Widget) }
	if p, ok := a.(parentSetter); ok {
		p.SetParent(f)
	}
	if p, ok := b.(parentSetter); ok {
		p.SetParent(f)
	}
	return f
}

// SetOverlay temporarily covers the current document without changing its view state.
func (f *Frame) SetOverlay(w widget.Widget) {
	if f.overlay == w {
		return
	}
	if child, ok := f.overlay.(interface{ SetParent(widget.Widget) }); ok {
		child.SetParent(nil)
	}
	f.overlay = w
	if child, ok := w.(interface{ SetParent(widget.Widget) }); ok {
		child.SetParent(f)
	}
	f.MarkNeedsLayout()
	f.SetNeedsRedraw(true)
}

// ShowB selects which child is visible.
func (f *Frame) ShowB(on bool) {
	if f.useB == on {
		return
	}
	f.useB = on
	f.MarkNeedsLayout()
	f.SetNeedsRedraw(true)
}

// Layout sizes the visible child to the viewport.
func (f *Frame) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.BiggestFinite(200, 200)
	rect := geometry.NewRect(0, 0, size.Width, size.Height)
	child := f.visible()
	if child != nil {
		widget.LayoutChild(child, ctx, geometry.Constraints{
			MinWidth: size.Width, MaxWidth: size.Width,
			MinHeight: size.Height, MaxHeight: size.Height,
		})
		if setter, ok := child.(interface{ SetBounds(geometry.Rect) }); ok {
			setter.SetBounds(rect)
		}
	}
	return size
}

// Draw paints the visible child in local coordinates.
func (f *Frame) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := f.Bounds()
	if b.IsEmpty() {
		return
	}
	child := f.visible()
	if child == nil {
		return
	}
	canvas.PushTransform(b.Min)
	widget.StampScreenOrigin(child, canvas)
	widget.DrawChild(child, ctx, canvas)
	canvas.PopTransform()
}

// Event sends input to the visible child. Keys pass through unchanged.
func (f *Frame) Event(ctx widget.Context, e event.Event) bool {
	child := f.visible()
	if child == nil {
		return false
	}
	if _, ok := e.(*event.KeyEvent); ok {
		return child.Event(ctx, e)
	}
	if me, ok := e.(*event.MouseEvent); ok {
		cp := *me
		cp.Position = me.Position.Sub(f.Bounds().Min)
		if !contains(child, cp.Position) {
			return false
		}
		return child.Event(ctx, &cp)
	}
	if we, ok := e.(*event.WheelEvent); ok {
		cp := *we
		cp.Position = we.Position.Sub(f.Bounds().Min)
		if !contains(child, cp.Position) {
			return false
		}
		return child.Event(ctx, &cp)
	}
	return child.Event(ctx, e)
}

// Children returns both panes so focus can find the editor.
func (f *Frame) Children() []widget.Widget {
	if f.overlay != nil {
		return []widget.Widget{f.overlay}
	}
	return []widget.Widget{f.a, f.b}
}

func (f *Frame) visible() widget.Widget {
	if f.overlay != nil {
		return f.overlay
	}
	if f.useB {
		return f.b
	}
	return f.a
}
