package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const splitDivider = float32(6)

// Split is a left sidebar and a document pane. The sidebar width is in pixels.
// A window resize does not move it. Only a drag on the divider does.
// The widget does not register a window-wide drag gesture.
type Split struct {
	widget.WidgetBase
	first    widget.Widget
	second   widget.Widget
	width    float32
	shown    bool
	right    bool
	theme    theme.Theme
	dragging bool
	startX   float32
	startW   float32
}

// NewSplit returns a split with a sidebar of the given pixel width.
func NewSplit(first, second widget.Widget, sidebar float32) *Split {
	s := &Split{first: first, second: second, width: sidebar, shown: true, theme: theme.Pairti()}
	s.SetVisible(true)
	s.SetEnabled(true)
	type parentSetter interface{ SetParent(widget.Widget) }
	if p, ok := first.(parentSetter); ok {
		p.SetParent(s)
	}
	if p, ok := second.(parentSetter); ok {
		p.SetParent(s)
	}
	return s
}

// SetTheme replaces the semantic theme.
func (s *Split) SetTheme(t theme.Theme) { s.theme = t; s.SetNeedsRedraw(true) }

// Toggle shows or hides the sidebar.
func (s *Split) Toggle() {
	s.shown = !s.shown
	s.MarkNeedsLayout()
	s.SetNeedsRedraw(true)
}

// Shown reports whether the sidebar is visible.
func (s *Split) Shown() bool { return s.shown }

// SetShown shows or hides the sidebar.
func (s *Split) SetShown(on bool) {
	if s.shown == on {
		return
	}
	s.Toggle()
}

// Width is the sidebar width in pixels.
func (s *Split) Width() float32 { return s.width }

// SetWidth sets the sidebar width in pixels.
func (s *Split) SetWidth(w float32) {
	if w < 1 {
		return
	}
	s.width = w
	s.MarkNeedsLayout()
	s.SetNeedsRedraw(true)
}

// SetRight puts the first pane on the right. The file tree stays on the left.
func (s *Split) SetRight(on bool) {
	s.right = on
	s.MarkNeedsLayout()
	s.SetNeedsRedraw(true)
}

// Layout places the panes. The sidebar keeps its pixel width.
func (s *Split) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.BiggestFinite(800, 600)
	if !s.shown {
		place(ctx, s.first, geometry.Rect{})
		place(ctx, s.second, geometry.NewRect(0, 0, size.Width, size.Height))
		return size
	}
	s.width = SidebarWidth(s.width, size.Width)
	gap := splitDivider
	if s.right {
		docW := size.Width - s.width - gap
		place(ctx, s.second, geometry.NewRect(0, 0, docW, size.Height))
		place(ctx, s.first, geometry.NewRect(docW+gap, 0, s.width, size.Height))
		return size
	}
	place(ctx, s.first, geometry.NewRect(0, 0, s.width, size.Height))
	place(ctx, s.second, geometry.NewRect(s.width+gap, 0, size.Width-s.width-gap, size.Height))
	return size
}

func place(ctx widget.Context, child widget.Widget, rect geometry.Rect) {
	if child == nil {
		return
	}
	widget.LayoutChild(child, ctx, geometry.Constraints{
		MinWidth: rect.Width(), MaxWidth: rect.Width(),
		MinHeight: rect.Height(), MaxHeight: rect.Height(),
	})
	if setter, ok := child.(interface{ SetBounds(geometry.Rect) }); ok {
		setter.SetBounds(rect)
	}
}

// Draw paints both panes and a one-pixel divider. Children stay in local coordinates.
func (s *Split) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := s.Bounds()
	if b.IsEmpty() {
		return
	}
	canvas.PushTransform(b.Min)
	if s.shown && s.first != nil {
		widget.StampScreenOrigin(s.first, canvas)
		widget.DrawChild(s.first, ctx, canvas)
	}
	if s.second != nil {
		widget.StampScreenOrigin(s.second, canvas)
		widget.DrawChild(s.second, ctx, canvas)
	}
	if s.shown {
		x := s.width + splitDivider/2
		if s.right {
			x = b.Width() - s.width - splitDivider/2
		}
		canvas.DrawLine(geometry.Pt(x, 0), geometry.Pt(x, b.Height()), rgb(s.theme.Rule), 1)
	}
	canvas.PopTransform()
}

// Event drags the divider only when the pointer went down on it.
func (s *Split) Event(ctx widget.Context, e event.Event) bool {
	if _, ok := e.(*event.KeyEvent); ok {
		if s.second != nil && s.second.Event(ctx, e) {
			return true
		}
		if s.first != nil && s.first.Event(ctx, e) {
			return true
		}
		return false
	}
	if we, ok := e.(*event.WheelEvent); ok {
		cp := *we
		cp.Position = we.Position.Sub(s.Bounds().Min)
		return s.dispatch(ctx, &cp, cp.Position)
	}
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	local := me.Position.Sub(s.Bounds().Min)
	if s.dragging || s.divider().Contains(local) {
		offerResize(ctx)
	}
	switch me.MouseType {
	case event.MousePress:
		if me.Button == event.ButtonLeft && s.divider().Contains(local) {
			s.dragging = true
			s.startX = local.X
			s.startW = s.width
			return true
		}
	case event.MouseDrag, event.MouseMove:
		if s.dragging {
			if !me.Buttons.IsLeftPressed() {
				s.dragging = false
				return false
			}
			delta := local.X - s.startX
			if s.right {
				delta = -delta
			}
			s.width = SidebarWidth(s.startW+delta, s.Bounds().Width())
			s.MarkNeedsLayout()
			s.SetNeedsRedraw(true)
			return true
		}
	case event.MouseRelease:
		if s.dragging {
			s.dragging = false
			return true
		}
	}
	cp := *me
	cp.Position = local
	return s.dispatch(ctx, &cp, local)
}

func (s *Split) dispatch(ctx widget.Context, e event.Event, local geometry.Point) bool {
	if s.first != nil && contains(s.first, local) && s.first.Event(ctx, e) {
		return true
	}
	if s.second != nil && contains(s.second, local) && s.second.Event(ctx, e) {
		return true
	}
	return false
}

func offerResize(ctx widget.Context) {
	if ctx == nil {
		return
	}
	ctx.SetCursor(widget.CursorResizeEW)
}

func (s *Split) divider() geometry.Rect {
	if !s.shown {
		return geometry.Rect{}
	}
	x := s.width
	if s.right {
		x = s.Bounds().Width() - s.width - splitDivider
	}
	return geometry.NewRect(x, 0, splitDivider, s.Bounds().Height())
}

func (s *Split) Children() []widget.Widget {
	var out []widget.Widget
	if s.first != nil {
		out = append(out, s.first)
	}
	if s.second != nil {
		out = append(out, s.second)
	}
	return out
}

func contains(w widget.Widget, p geometry.Point) bool {
	type bounds interface{ Bounds() geometry.Rect }
	b, ok := w.(bounds)
	return ok && b.Bounds().Contains(p)
}
