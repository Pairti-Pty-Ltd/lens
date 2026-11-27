package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const sidePad = float32(8)
const sideGap = float32(6)

// SidePane is the sidebar column. The search field is the full inner width,
// so the same inset sits on its left and its right.
type SidePane struct {
	widget.WidgetBase
	search  widget.Widget
	actions widget.Widget
	bar     widget.Widget
	history *ModeBar
	reports *ReportBar
	mode    int
	tree    widget.Widget
	theme   theme.Theme
}

// NewSidePane returns the sidebar column.
func NewSidePane(search, actions, tree widget.Widget) *SidePane {
	p := &SidePane{search: search, actions: actions, tree: tree, theme: theme.Pairti()}
	p.SetVisible(true)
	p.SetEnabled(true)
	type parentSetter interface{ SetParent(widget.Widget) }
	for _, child := range []widget.Widget{search, actions, tree} {
		if child == nil {
			continue
		}
		if s, ok := child.(parentSetter); ok {
			s.SetParent(p)
		}
	}
	return p
}

// SetBar puts the Files | Reports switch at the top of the column.
func (p *SidePane) SetBar(bar widget.Widget) {
	p.bar = bar
	if bar == nil {
		return
	}
	if s, ok := bar.(interface{ SetParent(widget.Widget) }); ok {
		s.SetParent(p)
	}
}

func (p *SidePane) SetHistoryBar(bar *ModeBar) {
	p.history = bar
	bar.SetParent(p)
}

// SetReportBar is the plus control shown while Reports is selected.
func (p *SidePane) SetReportBar(bar *ReportBar) {
	p.reports = bar
	if bar == nil {
		return
	}
	bar.SetParent(p)
}

// SetMode shows the controls for Files, Reports, or History.
func (p *SidePane) SetMode(mode int) {
	if p.mode == mode {
		return
	}
	p.mode = mode
	if mode != SideReports && p.reports != nil {
		p.reports.CloseMenu()
	}
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

// SetReports shows the report controls when on, and the file controls otherwise.
func (p *SidePane) SetReports(on bool) {
	if on {
		p.SetMode(SideReports)
		return
	}
	p.SetMode(SideFiles)
}

// SetTheme replaces the ground colour.
func (p *SidePane) SetTheme(t theme.Theme) {
	p.theme = t
	p.SetNeedsRedraw(true)
}

// Layout gives the search field equal insets and the tree the remaining height.
func (p *SidePane) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.BiggestFinite(200, 400)
	inner := size.Width - sidePad*2
	if inner < 0 {
		inner = 0
	}
	y := sidePad
	if p.bar != nil {
		const h = float32(26)
		widget.LayoutChild(p.bar, ctx, geometry.Constraints{
			MinWidth: inner, MaxWidth: inner, MinHeight: h, MaxHeight: h,
		})
		setBounds(p.bar, geometry.NewRect(sidePad, y, inner, h))
		y += h + sideGap
	}
	if p.mode == SideHistory && p.history != nil {
		widget.LayoutChild(p.history, ctx, geometry.Constraints{MinWidth: inner, MaxWidth: inner, MinHeight: 26, MaxHeight: 26})
		setBounds(p.history, geometry.NewRect(sidePad, y, inner, 26))
		y += 26 + sideGap
	}
	if p.mode == SideReports {
		if p.reports != nil {
			got := widget.LayoutChild(p.reports, ctx, geometry.Constraints{
				MinWidth: inner, MaxWidth: inner, MinHeight: reportRow, MaxHeight: geometry.Infinity,
			})
			h := got.Height
			if h < reportRow {
				h = reportRow
			}
			setBounds(p.reports, geometry.NewRect(sidePad, y, inner, h))
			y += h + sideGap
		}
	} else if p.mode == SideFiles && p.search != nil {
		const h = float32(26)
		widget.LayoutChild(p.search, ctx, geometry.Constraints{
			MinWidth: inner, MaxWidth: inner, MinHeight: h, MaxHeight: h,
		})
		setBounds(p.search, geometry.NewRect(sidePad, y, inner, h))
		y += h + sideGap
	}
	if p.mode == SideFiles && p.actions != nil {
		got := widget.LayoutChild(p.actions, ctx, geometry.Constraints{
			MinWidth: 0, MaxWidth: inner, MinHeight: 0, MaxHeight: toolSlot,
		})
		h := got.Height
		w := got.Width
		if w > inner {
			w = inner
		}
		setBounds(p.actions, geometry.NewRect(sidePad, y, w, h))
		if h > 0 {
			y += h + sideGap
		}
	}
	treeH := size.Height - y - sidePad
	if treeH < 0 {
		treeH = 0
	}
	if p.tree != nil {
		widget.LayoutChild(p.tree, ctx, geometry.Constraints{
			MinWidth: inner, MaxWidth: inner, MinHeight: treeH, MaxHeight: treeH,
		})
		setBounds(p.tree, geometry.NewRect(sidePad, y, inner, treeH))
	}
	return size
}

// Draw paints the ground and the three rows.
func (p *SidePane) Draw(ctx widget.Context, canvas widget.Canvas) {
	b := p.Bounds()
	if b.IsEmpty() {
		return
	}
	canvas.DrawRect(b, rgb(p.theme.Surface))
	canvas.PushTransform(b.Min)
	for _, child := range p.Children() {
		widget.StampScreenOrigin(child, canvas)
		child.Draw(ctx, canvas)
	}
	canvas.PopTransform()
}

// Event sends input to the row under the pointer.
func (p *SidePane) Event(ctx widget.Context, e event.Event) bool {
	if me, ok := e.(*event.MouseEvent); ok {
		local := *me
		local.Position = me.Position.Sub(p.Bounds().Min)
		if me.MouseType == event.MousePress && p.reports != nil && p.reports.MenuOpen() && !p.reports.Bounds().Contains(local.Position) {
			p.reports.CloseMenu()
		}
		for i := len(p.Children()) - 1; i >= 0; i-- {
			child := p.Children()[i]
			if child == nil {
				continue
			}
			if bw, ok := child.(interface{ Bounds() geometry.Rect }); ok && !bw.Bounds().Contains(local.Position) {
				continue
			}
			if child.Event(ctx, &local) {
				return true
			}
		}
		return false
	}
	if we, ok := e.(*event.WheelEvent); ok {
		local := *we
		local.Position = we.Position.Sub(p.Bounds().Min)
		for i := len(p.Children()) - 1; i >= 0; i-- {
			child := p.Children()[i]
			if child == nil {
				continue
			}
			if bw, ok := child.(interface{ Bounds() geometry.Rect }); ok && !bw.Bounds().Contains(local.Position) {
				continue
			}
			if child.Event(ctx, &local) {
				return true
			}
		}
		return false
	}
	for i := len(p.Children()) - 1; i >= 0; i-- {
		if p.Children()[i] != nil && p.Children()[i].Event(ctx, e) {
			return true
		}
	}
	return false
}

// Children is the search field, the action row, and the tree.
func (p *SidePane) Children() []widget.Widget {
	var out []widget.Widget
	for _, child := range p.shown() {
		if child != nil {
			out = append(out, child)
		}
	}
	return out
}

func (p *SidePane) shown() []widget.Widget {
	var out []widget.Widget
	add := func(w widget.Widget) {
		if w != nil {
			out = append(out, w)
		}
	}
	add(p.bar)
	switch p.mode {
	case SideReports:
		add(p.reports)
	case SideHistory:
		if p.history != nil {
			add(p.history)
		}
	default:
		add(p.search)
		add(p.actions)
	}
	add(p.tree)
	return out
}

func setBounds(w widget.Widget, r geometry.Rect) {
	if s, ok := w.(interface{ SetBounds(geometry.Rect) }); ok {
		s.SetBounds(r)
	}
}
