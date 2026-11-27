package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/icon"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// ProjectActions is the side-panel toolbar under the search field.
// The eye shows dot-leading names. The slash shows gitignored names.
// Accent means that set is shown.
type ProjectActions struct {
	widget.WidgetBase
	theme                theme.Theme
	name                 string
	showDot, showIgnored bool
	docs                 bool
	names                bool
	on                   func(showDot, showIgnored bool)
	onDocs               func(bool)
	onNames              func(names bool)
}

const nameGap = float32(12)

// NewProjectActions returns an empty pane. on receives the new pair of flags.
func NewProjectActions(on func(showDot, showIgnored bool)) *ProjectActions {
	p := &ProjectActions{theme: theme.Pairti(), on: on}
	p.SetVisible(true)
	p.SetEnabled(true)
	return p
}

// SetTheme replaces the semantic theme.
func (p *ProjectActions) SetTheme(t theme.Theme) {
	p.theme = t
	p.SetNeedsRedraw(true)
}

// OnNames runs when the tree should show file names instead of titles.
func (p *ProjectActions) OnNames(fn func(bool)) { p.onNames = fn }

// OnDocs runs when the tree should show only directories that hold a document.
func (p *ProjectActions) OnDocs(fn func(bool)) { p.onDocs = fn }

// Set replaces the project, the filters, and the name/title choice.
// An empty name collapses the pane. names false shows titles.
// docs true hides directories that contain no Markdown or Typst file.
func (p *ProjectActions) Set(name string, showDot, showIgnored, names, docs bool) {
	p.name = name
	p.showDot = showDot
	p.showIgnored = showIgnored
	p.names = names
	p.docs = docs
	p.MarkNeedsLayout()
	p.SetNeedsRedraw(true)
}

// Layout is one icon row, or nothing when no project is chosen.
func (p *ProjectActions) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	w := toolSlot*3 + nameGap + toolSlot*2
	if c.MaxWidth > 0 && c.MaxWidth < w {
		w = c.MaxWidth
	}
	h := float32(0)
	if p.name != "" {
		h = toolSlot
	}
	return c.Constrain(geometry.Sz(w, h))
}

// Draw paints the two icons the way the main toolbar paints its tools.
func (p *ProjectActions) Draw(_ widget.Context, canvas widget.Canvas) {
	if !p.IsVisible() || p.name == "" {
		return
	}
	origin := p.Bounds().Min
	p.drawSlot(canvas, origin.X, iconEye, p.showDot)
	p.drawSlot(canvas, origin.X+toolSlot, iconIgnore, p.showIgnored)
	p.drawSlot(canvas, origin.X+toolSlot*2, iconDocs, p.docs)
	p.drawNames(canvas, origin.X+toolSlot*3+nameGap)
}

func (p *ProjectActions) drawNames(canvas widget.Canvas, x float32) {
	b := geometry.NewRect(x, p.Bounds().Min.Y, toolSlot*2, toolSlot)
	canvas.DrawRoundRect(b, rgb(p.theme.CodeFill), 6)
	half := b.Width() / 2
	active := geometry.NewRect(b.Min.X, b.Min.Y, half, b.Height())
	if p.names {
		active.Min.X += half
	}
	canvas.DrawRoundRect(active, rgba(p.theme.Selection), 6)
	canvas.StrokeRoundRect(b, rgb(p.theme.Rule), 6, 1)
	canvas.DrawLine(
		geometry.Pt(b.Min.X+half, b.Min.Y+4),
		geometry.Pt(b.Min.X+half, b.Max.Y-4),
		rgb(p.theme.Rule), 1,
	)
	p.drawMark(canvas, b.Min.X, iconTitle, !p.names)
	p.drawMark(canvas, b.Min.X+half, iconFile, p.names)
}

func (p *ProjectActions) drawMark(canvas widget.Canvas, x float32, data icon.IconData, on bool) {
	color := rgb(p.theme.Muted)
	if on {
		color = rgb(p.theme.Accent)
	}
	box := geometry.NewRect(x+(toolSlot-16)/2, p.Bounds().Min.Y+6, 16, 16)
	icon.Draw(canvas, data, box, color)
}

func (p *ProjectActions) drawSlot(canvas widget.Canvas, x float32, data icon.IconData, on bool) {
	color := rgb(p.theme.Muted)
	if on {
		color = rgb(p.theme.Accent)
	}
	box := geometry.NewRect(x+6, p.Bounds().Min.Y+6, 16, 16)
	icon.Draw(canvas, data, box, color)
}

// Event toggles the row under the pointer. A move names the filter.
func (p *ProjectActions) Event(ctx widget.Context, e event.Event) bool {
	if p.hover(ctx, e) {
		return false
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	if p.name == "" || !p.Bounds().Contains(me.Position) {
		return false
	}
	x := me.Position.X - p.Bounds().Min.X
	seg := toolSlot*3 + nameGap
	switch {
	case x >= 0 && x < toolSlot:
		p.showDot = !p.showDot
	case x >= toolSlot && x < toolSlot*2:
		p.showIgnored = !p.showIgnored
	case x >= toolSlot*2 && x < toolSlot*3:
		p.docs = !p.docs
		p.SetNeedsRedraw(true)
		if p.onDocs != nil {
			p.onDocs(p.docs)
		}
		if ctx != nil {
			ctx.Invalidate()
		}
		return true
	case x >= seg && x < seg+toolSlot*2:
		names := x >= seg+toolSlot
		if names == p.names {
			return true
		}
		p.names = names
		p.SetNeedsRedraw(true)
		if p.onNames != nil {
			p.onNames(names)
		}
		if ctx != nil {
			ctx.Invalidate()
		}
		return true
	default:
		return true
	}
	if p.on != nil {
		p.on(p.showDot, p.showIgnored)
	}
	p.SetNeedsRedraw(true)
	if ctx != nil {
		ctx.Invalidate()
	}
	return true
}

func (p *ProjectActions) hover(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if me.MouseType == event.MouseLeave {
		offerTip(ctx, p.theme, "", geometry.Rect{})
		return true
	}
	local, over := hoverPoint(p, me)
	if !over || p.name == "" {
		return false
	}
	label := ""
	x := float32(0)
	seg := toolSlot*3 + nameGap
	switch {
	case local.X >= 0 && local.X < toolSlot:
		label = "Show hidden files"
		if p.showDot {
			label = "Hide hidden files"
		}
	case local.X >= toolSlot && local.X < toolSlot*2:
		label = "Show gitignored files"
		if p.showIgnored {
			label = "Hide gitignored files"
		}
		x = toolSlot
	case local.X >= toolSlot*2 && local.X < toolSlot*3:
		label = "Documents"
		if !p.docs {
			label = "All folders"
		}
		x = toolSlot * 2
	case local.X >= seg && local.X < seg+toolSlot:
		label = "Titles"
		x = seg
	case local.X >= seg+toolSlot && local.X < seg+toolSlot*2:
		label = "Filenames"
		x = seg + toolSlot
	}
	origin := screenOrigin(p)
	offerTip(ctx, p.theme, label, geometry.NewRect(origin.X+x, origin.Y, toolSlot, toolSlot))
	return true
}

// Children returns nil.
func (p *ProjectActions) Children() []widget.Widget { return nil }
