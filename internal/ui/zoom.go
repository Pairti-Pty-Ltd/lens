package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const zoomMid = float32(88)

// Zoom is the reading-size control: smaller, the current stop, larger.
// It uses the same track as Formatted / Raw. on receives -1 or 1.
type Zoom struct {
	widget.WidgetBase
	theme theme.Theme
	label string
	on    func(int)
}

// NewZoom returns a control showing 100%.
func NewZoom(on func(int)) *Zoom {
	z := &Zoom{theme: theme.Pairti(), label: "100%", on: on}
	z.SetVisible(true)
	z.SetEnabled(true)
	return z
}

// SetTheme replaces the colours. The label stays at 13pt so the control does not grow.
func (z *Zoom) SetTheme(t theme.Theme) {
	z.theme = t
	z.SetNeedsRedraw(true)
}

// SetLabel shows the current stop, such as 100% or Fit width.
func (z *Zoom) SetLabel(s string) {
	if s == "" || s == z.label {
		return
	}
	z.label = s
	z.SetNeedsRedraw(true)
}

// Label is the current stop.
func (z *Zoom) Label() string { return z.label }

// Layout is minus, the label, and plus.
func (z *Zoom) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.Constrain(geometry.Sz(toolSlot*2+zoomMid, toolSlot))
}

// Draw paints the track, the current stop, and the two steps.
func (z *Zoom) Draw(_ widget.Context, canvas widget.Canvas) {
	if !z.IsVisible() {
		return
	}
	b := z.Bounds()
	if b.Width() < 2 || b.Height() < 2 {
		return
	}
	canvas.DrawRoundRect(b, rgb(z.theme.CodeFill), 6)
	mid := geometry.NewRect(b.Min.X+toolSlot, b.Min.Y, zoomMid, b.Height())
	canvas.DrawRoundRect(mid, rgba(z.theme.Selection), 6)
	canvas.StrokeRoundRect(b, rgb(z.theme.Rule), 6, 1)
	rule := rgb(z.theme.Rule)
	canvas.DrawLine(geometry.Pt(b.Min.X+toolSlot, b.Min.Y+4), geometry.Pt(b.Min.X+toolSlot, b.Max.Y-4), rule, 1)
	canvas.DrawLine(geometry.Pt(mid.Max.X, b.Min.Y+4), geometry.Pt(mid.Max.X, b.Max.Y-4), rule, 1)
	z.mark(canvas, b.Min.X, "−")
	z.mark(canvas, mid.Max.X, "+")
	ink := rgb(z.theme.Ink)
	tw := textWidth(canvas, z.label, 13, ink, false, z.theme.UIFont)
	x := mid.Min.X + (mid.Width()-tw)/2
	if x < mid.Min.X+2 {
		x = mid.Min.X + 2
	}
	drawFace(canvas, z.label, geometry.NewRect(x, b.Min.Y+4, tw+2, b.Height()-8), 13, ink, false, z.theme.UIFont)
}

func (z *Zoom) mark(canvas widget.Canvas, x float32, s string) {
	col := rgb(z.theme.Ink)
	tw := textWidth(canvas, s, 16, col, false, z.theme.UIFont)
	drawFace(canvas, s, geometry.NewRect(x+(toolSlot-tw)/2, z.Bounds().Min.Y+4, tw+2, toolSlot-8), 16, col, false, z.theme.UIFont)
}

// Event steps the size. The middle is the label.
func (z *Zoom) Event(ctx widget.Context, e event.Event) bool {
	if !z.IsVisible() {
		return false
	}
	if z.hover(ctx, e) {
		return false
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	if !z.Bounds().Contains(me.Position) {
		return false
	}
	local := me.Position.X - z.Bounds().Min.X
	dir := 0
	switch {
	case local < toolSlot:
		dir = -1
	case local >= toolSlot+zoomMid:
		dir = 1
	}
	if dir != 0 && z.on != nil {
		z.on(dir)
	}
	if ctx != nil {
		ctx.Invalidate()
	}
	return true
}

func (z *Zoom) hover(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if me.MouseType == event.MouseLeave {
		offerTip(ctx, z.theme, "", geometry.Rect{})
		return true
	}
	local, over := hoverPoint(z, me)
	if !over {
		return false
	}
	label := z.label
	x := toolSlot
	w := zoomMid
	switch {
	case local.X < toolSlot:
		label, x, w = "Zoom out", 0, toolSlot
	case local.X >= toolSlot+zoomMid:
		label, x, w = "Zoom in", toolSlot+zoomMid, toolSlot
	}
	origin := screenOrigin(z)
	offerTip(ctx, z.theme, label, geometry.NewRect(origin.X+x, origin.Y, w, toolSlot))
	return true
}

// Children returns nil.
func (z *Zoom) Children() []widget.Widget { return nil }
