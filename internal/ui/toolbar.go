package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/icon"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const toolSlot = float32(28)

// Tool is one toolbar button. Mark draws short text when there is no icon.
// Gap inserts a break before the button, the way Zettlr groups its tools.
type Tool struct {
	Name string
	Icon icon.IconData
	Mark string
	Gap  bool
}

// ToolBar is the icon row. on receives the tool name.
type ToolBar struct {
	widget.WidgetBase
	active    string
	tools     []Tool
	theme     theme.Theme
	on        func(string)
	available func(string) bool
	spans     []toolSpan
}

type toolSpan struct {
	name string
	x    float32
}

// NewToolBar returns the icon row.
func NewToolBar(tools []Tool, on func(string)) *ToolBar {
	b := &ToolBar{tools: tools, on: on, theme: theme.Pairti()}
	b.SetVisible(true)
	b.SetEnabled(true)
	return b
}

// SetAvailable lets document actions reflect the currently displayed surface.
func (b *ToolBar) SetAvailable(fn func(string) bool) { b.available = fn }

// SetTheme replaces the semantic theme.
func (b *ToolBar) SetTheme(t theme.Theme) {
	b.theme = t
	b.MarkNeedsLayout()
	b.SetNeedsRedraw(true)
}

// SetActive highlights the selected tool.
func (b *ToolBar) SetActive(name string) {
	if b.active == name {
		return
	}
	b.active = name
	b.SetNeedsRedraw(true)
}

// Layout sizes the row to the slots and the group gaps.
func (b *ToolBar) Layout(widget.Context, geometry.Constraints) geometry.Size {
	b.spans = b.measure()
	w := float32(4)
	if n := len(b.spans); n > 0 {
		w = b.spans[n-1].x + toolSlot
	}
	return geometry.Sz(w, toolSlot)
}

// Draw paints a 16px icon, or the short mark, in each slot.
func (b *ToolBar) Draw(_ widget.Context, canvas widget.Canvas) {
	if !b.IsVisible() {
		return
	}
	origin := b.Bounds().Min
	for i, sp := range b.spans {
		muted := rgb(b.theme.Muted)
		if sp.name == b.active {
			muted = rgb(b.theme.Accent)
		}
		if b.available != nil && !b.available(sp.name) {
			muted.A *= 0.35
		}
		t := b.tools[i]
		box := geometry.NewRect(origin.X+sp.x+6, origin.Y+6, 16, 16)
		if t.Mark != "" {
			drawFace(canvas, t.Mark, geometry.NewRect(origin.X+sp.x, origin.Y+4, toolSlot, 20), 11, muted, false, b.theme.UIFont)
			continue
		}
		icon.Draw(canvas, t.Icon, box, muted)
	}
}

// Event hits one slot. A move names the icon under the pointer.
func (b *ToolBar) Event(ctx widget.Context, e event.Event) bool {
	if b.hover(ctx, e) {
		return false
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	if !b.Bounds().Contains(me.Position) {
		return false
	}
	x := me.Position.X - b.Bounds().Min.X
	for _, sp := range b.spans {
		if x >= sp.x && x < sp.x+toolSlot {
			if b.on != nil && (b.available == nil || b.available(sp.name)) {
				b.on(sp.name)
			}
			ctx.Invalidate()
			return true
		}
	}
	return false
}

func (b *ToolBar) hover(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if me.MouseType == event.MouseLeave {
		offerTip(ctx, b.theme, "", geometry.Rect{})
		return true
	}
	local, over := hoverPoint(b, me)
	if !over {
		return false
	}
	name := ""
	var box geometry.Rect
	origin := screenOrigin(b)
	for _, sp := range b.spans {
		if local.X >= sp.x && local.X < sp.x+toolSlot {
			name = sp.name
			box = geometry.NewRect(origin.X+sp.x, origin.Y, toolSlot, toolSlot)
			break
		}
	}
	offerTip(ctx, b.theme, name, box)
	return true
}

// Children returns nil.
func (b *ToolBar) Children() []widget.Widget { return nil }

// Slot is the tool's rectangle in window coordinates, after layout.
func (b *ToolBar) Slot(name string) geometry.Rect {
	origin := screenOrigin(b)
	for _, sp := range b.spans {
		if sp.name == name {
			return geometry.NewRect(origin.X+sp.x, origin.Y, toolSlot, toolSlot)
		}
	}
	return geometry.Rect{}
}

func (b *ToolBar) measure() []toolSpan {
	var spans []toolSpan
	x := float32(0)
	for _, t := range b.tools {
		if t.Gap && len(spans) > 0 {
			x += 10
		}
		spans = append(spans, toolSpan{name: t.Name, x: x})
		x += toolSlot
	}
	return spans
}

// Narrow forces a stock text field down to one short line.
// The toolkit field reports a height of 48.
type Narrow struct {
	widget.WidgetBase
	child widget.Widget
}

// NewNarrow wraps a field at 26px.
func NewNarrow(child widget.Widget) *Narrow {
	n := &Narrow{child: child}
	n.SetVisible(true)
	n.SetEnabled(true)
	if s, ok := child.(interface{ SetParent(widget.Widget) }); ok {
		s.SetParent(n)
	}
	return n
}

// Layout gives the child a 26px line.
func (n *Narrow) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	const h = float32(26)
	width := c.MaxWidth
	if width <= 0 || width >= geometry.Infinity {
		width = 160
	}
	if n.child != nil {
		widget.LayoutChild(n.child, ctx, geometry.Constraints{
			MinWidth: width, MaxWidth: width,
			MinHeight: h, MaxHeight: h,
		})
		if s, ok := n.child.(interface{ SetBounds(geometry.Rect) }); ok {
			s.SetBounds(geometry.NewRect(0, 0, width, h))
		}
	}
	return c.Constrain(geometry.Sz(width, h))
}

// Draw paints the field inside this row. The child's bounds are local,
// so the row's position has to be applied or the field stays at the top.
func (n *Narrow) Draw(ctx widget.Context, canvas widget.Canvas) {
	if n.child == nil {
		return
	}
	canvas.PushTransform(n.Bounds().Min)
	widget.StampScreenOrigin(n.child, canvas)
	widget.DrawChild(n.child, ctx, canvas)
	canvas.PopTransform()
}

// Event forwards to the field. Mouse coordinates are in this widget's parent
// space; the field's bounds are local, so the point is shifted first.
func (n *Narrow) Event(ctx widget.Context, e event.Event) bool {
	if n.child == nil {
		return false
	}
	if me, ok := e.(*event.MouseEvent); ok {
		cp := *me
		cp.Position = me.Position.Sub(n.Bounds().Min)
		return n.child.Event(ctx, &cp)
	}
	if we, ok := e.(*event.WheelEvent); ok {
		cp := *we
		cp.Position = we.Position.Sub(n.Bounds().Min)
		return n.child.Event(ctx, &cp)
	}
	return n.child.Event(ctx, e)
}

// Children is the wrapped field.
func (n *Narrow) Children() []widget.Widget {
	if n.child == nil {
		return nil
	}
	return []widget.Widget{n.child}
}
