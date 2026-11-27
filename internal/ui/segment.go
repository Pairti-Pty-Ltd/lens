package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/icon"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// Segment is the Formatted / Raw control. Two icons share one rounded track.
// The selected half is accent. on receives "Formatted" or "Raw".
type Segment struct {
	widget.WidgetBase
	theme  theme.Theme
	active string
	on     func(string)
}

// NewViewSegment returns the document view switch, with Formatted selected.
func NewViewSegment(on func(string)) *Segment {
	s := &Segment{theme: theme.Pairti(), active: "Formatted", on: on}
	s.SetVisible(true)
	s.SetEnabled(true)
	return s
}

// SetTheme replaces the semantic theme.
func (s *Segment) SetTheme(t theme.Theme) {
	s.theme = t
	s.SetNeedsRedraw(true)
}

// SetShown hides the control when no document is open.
func (s *Segment) SetShown(on bool) {
	if s.IsVisible() == on {
		return
	}
	s.SetVisible(on)
	s.MarkNeedsLayout()
	s.SetNeedsRedraw(true)
}

// SetActive selects Formatted or Raw.
func (s *Segment) SetActive(label string) {
	if label != "" && label != "Formatted" && label != "Raw" {
		return
	}
	if s.active == label {
		return
	}
	s.active = label
	s.SetNeedsRedraw(true)
}

// Layout is two toolbar slots side by side, or nothing while hidden.
func (s *Segment) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	if !s.IsVisible() {
		return geometry.Sz(0, 0)
	}
	return c.Constrain(geometry.Sz(toolSlot*2, toolSlot))
}

// Draw paints the track, the active half, and the two icons.
func (s *Segment) Draw(_ widget.Context, canvas widget.Canvas) {
	if !s.IsVisible() {
		return
	}
	b := s.Bounds()
	if b.Width() < 2 || b.Height() < 2 {
		return
	}
	canvas.DrawRoundRect(b, rgb(s.theme.CodeFill), 6)
	half := b.Width() / 2
	active := geometry.NewRect(b.Min.X, b.Min.Y, half, b.Height())
	if s.active == "Raw" {
		active.Min.X += half
	}
	if s.active != "" {
		canvas.DrawRoundRect(active, rgba(s.theme.Selection), 6)
	}
	canvas.StrokeRoundRect(b, rgb(s.theme.Rule), 6, 1)
	canvas.DrawLine(
		geometry.Pt(b.Min.X+half, b.Min.Y+4),
		geometry.Pt(b.Min.X+half, b.Max.Y-4),
		rgb(s.theme.Rule), 1,
	)
	s.icon(canvas, b.Min.X, iconDoc, s.active == "Formatted")
	s.icon(canvas, b.Min.X+half, iconCode, s.active == "Raw")
}

func (s *Segment) icon(canvas widget.Canvas, x float32, data icon.IconData, on bool) {
	color := rgb(s.theme.Muted)
	if on {
		color = rgb(s.theme.Accent)
	}
	box := geometry.NewRect(x+(toolSlot-16)/2, s.Bounds().Min.Y+6, 16, 16)
	icon.Draw(canvas, data, box, color)
}

// Event hits one half. A move names Formatted or Raw.
func (s *Segment) Event(ctx widget.Context, e event.Event) bool {
	if !s.IsVisible() {
		return false
	}
	if s.hover(ctx, e) {
		return false
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	if !s.Bounds().Contains(me.Position) {
		return false
	}
	label := "Formatted"
	if me.Position.X >= s.Bounds().Min.X+s.Bounds().Width()/2 {
		label = "Raw"
	}
	s.SetActive(label)
	if s.on != nil {
		s.on(label)
	}
	if ctx != nil {
		ctx.Invalidate()
	}
	return true
}

func (s *Segment) hover(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if me.MouseType == event.MouseLeave {
		offerTip(ctx, s.theme, "", geometry.Rect{})
		return true
	}
	local, over := hoverPoint(s, me)
	if !over {
		return false
	}
	label := "Formatted"
	x := float32(0)
	if local.X >= s.Bounds().Width()/2 {
		label = "Raw"
		x = s.Bounds().Width() / 2
	}
	origin := screenOrigin(s)
	offerTip(ctx, s.theme, label, geometry.NewRect(origin.X+x, origin.Y, toolSlot, toolSlot))
	return true
}

// Children returns nil.
func (s *Segment) Children() []widget.Widget { return nil }
