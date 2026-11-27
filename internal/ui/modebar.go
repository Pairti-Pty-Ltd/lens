package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/icon"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const (
	// SideFiles is the document tree.
	SideFiles = iota
	// SideReports is the report list.
	SideReports
	// SideHistory is the read-only git history.
	SideHistory
)

// ModeBar switches the sidebar between files, reports, and history.
type ModeBar struct {
	widget.WidgetBase
	theme     theme.Theme
	labels    []string
	mode      int
	on        func(mode int)
	available func(int) bool
}

// NewModeBar returns a Files | Reports | History control.
func NewModeBar(on func(int)) *ModeBar {
	m := &ModeBar{theme: theme.Pairti(), labels: []string{"Files", "Reports", "History"}, on: on}
	m.SetVisible(true)
	m.SetEnabled(true)
	return m
}

// NewHistoryScopeBar switches the Git log between a file and its repository.
func NewHistoryScopeBar(on func(int)) *ModeBar {
	m := NewModeBar(on)
	m.labels = []string{"This file", "Repository"}
	return m
}

// SetAvailable dims modes the host cannot open and prevents their selection.
func (m *ModeBar) SetAvailable(f func(int) bool) { m.available = f; m.SetNeedsRedraw(true) }

// SetTheme replaces the colours.
func (m *ModeBar) SetTheme(t theme.Theme) {
	m.theme = t
	m.SetNeedsRedraw(true)
}

// SetMode selects Files, Reports, or History.
func (m *ModeBar) SetMode(mode int) {
	m.mode = mode
	m.SetNeedsRedraw(true)
}

// SetReports selects Reports when true, and Files otherwise.
func (m *ModeBar) SetReports(on bool) {
	if on {
		m.SetMode(SideReports)
		return
	}
	m.SetMode(SideFiles)
}

// Layout is one row.
func (m *ModeBar) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	w := c.MaxWidth
	if w <= 0 || w >= geometry.Infinity {
		w = 200
	}
	return c.Constrain(geometry.Sz(w, 26))
}

// Draw paints the two labels.
func (m *ModeBar) Draw(_ widget.Context, canvas widget.Canvas) {
	if !m.IsVisible() {
		return
	}
	b := m.Bounds()
	if b.Width() < 2 {
		return
	}
	canvas.DrawRoundRect(b, rgb(m.theme.CodeFill), 6)
	width := b.Width() / float32(len(m.labels))
	for i, label := range m.labels {
		box := geometry.NewRect(b.Min.X+float32(i)*width, b.Min.Y, width, b.Height())
		if i == m.mode {
			canvas.DrawRoundRect(box, rgb(exportWell), 6)
		}
		m.label(canvas, box, label, i)
	}
	canvas.StrokeRoundRect(b, rgb(m.theme.Rule), 6, 1)
}

func (m *ModeBar) label(canvas widget.Canvas, box geometry.Rect, text string, mode int) {
	col := rgb(m.theme.Muted)
	if m.available != nil && !m.available(mode) {
		col.A = 0.4
	} else if mode == m.mode {
		col = rgb(m.theme.Accent)
	}
	drawCenter(canvas, text, box, 13, col, false, m.theme.UIFont)
}

// Event selects the half under the pointer.
func (m *ModeBar) Event(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	b := m.Bounds()
	if !b.Contains(me.Position) {
		return false
	}
	mode := min(len(m.labels)-1, int((me.Position.X-b.Min.X)/b.Width()*float32(len(m.labels))))
	if m.available != nil && !m.available(mode) {
		return true
	}
	if mode != m.mode {
		m.mode = mode
		m.SetNeedsRedraw(true)
		if m.on != nil {
			m.on(mode)
		}
	}
	if ctx != nil {
		ctx.Invalidate()
	}
	return true
}

// Children returns nil.
func (m *ModeBar) Children() []widget.Widget { return nil }

// ReportChoice is one library folder the reports list can show.
type ReportChoice struct {
	Name string
	Root string
}

// ReportBar is the reports toolbar. The name opens the project list.
// The plus creates a report in the project that is showing.
type ReportBar struct {
	widget.WidgetBase
	theme    theme.Theme
	add      func()
	pick     func(root string)
	choices  []ReportChoice
	selected string
	open     bool
}

const reportRow = float32(28)

// NewReportBar returns the project control and the plus.
func NewReportBar(add func(), pick func(string)) *ReportBar {
	r := &ReportBar{theme: theme.Pairti(), add: add, pick: pick}
	r.SetVisible(true)
	r.SetEnabled(true)
	return r
}

// SetChoices is the library folders and the folder whose reports are showing.
func (r *ReportBar) SetChoices(choices []ReportChoice, selected string) {
	r.choices = choices
	r.selected = selected
	r.SetNeedsRedraw(true)
}

// MenuOpen reports whether the project list is open.
func (r *ReportBar) MenuOpen() bool { return r.open }

// CloseMenu closes the project list.
func (r *ReportBar) CloseMenu() {
	if !r.open {
		return
	}
	r.open = false
	r.MarkNeedsLayout()
	r.SetNeedsRedraw(true)
}

// SetTheme replaces the colours.
func (r *ReportBar) SetTheme(t theme.Theme) {
	r.theme = t
	r.SetNeedsRedraw(true)
}

// Layout is the name row, and the project list while it is open.
func (r *ReportBar) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	w := c.MaxWidth
	if w <= 0 || w >= geometry.Infinity {
		w = 200
	}
	h := reportRow
	if r.open {
		h += float32(len(r.choices)) * reportRow
	}
	return c.Constrain(geometry.Sz(w, h))
}

// Draw paints the project name, the plus, and the open list.
func (r *ReportBar) Draw(_ widget.Context, canvas widget.Canvas) {
	if !r.IsVisible() {
		return
	}
	b := r.Bounds()
	if b.Width() < toolSlot+8 {
		return
	}
	field := geometry.NewRect(b.Min.X, b.Min.Y, b.Width()-toolSlot, reportRow)
	canvas.DrawRoundRect(field, rgb(r.theme.CodeFill), 6)
	canvas.StrokeRoundRect(field, rgb(r.theme.Rule), 6, 1)
	chev := geometry.NewRect(field.Max.X-22, field.Min.Y+6, 16, 16)
	icon.Draw(canvas, icon.ChevronDown, chev, rgb(r.theme.Muted))
	name := fitRun(canvas, r.label(), 13, rgb(r.theme.Ink), r.theme.UIFont, chev.Min.X-field.Min.X-12)
	drawFace(canvas, name, geometry.NewRect(field.Min.X+10, field.Min.Y+4, chev.Min.X-field.Min.X-12, 20), 13, rgb(r.theme.Ink), false, r.theme.UIFont)
	box := geometry.NewRect(b.Max.X-toolSlot+6, b.Min.Y+6, 16, 16)
	if r.add != nil {
		icon.Draw(canvas, icon.Add, box, rgb(r.theme.Muted))
	}
	if !r.open {
		return
	}
	top := b.Min.Y + reportRow
	listH := float32(len(r.choices)) * reportRow
	list := geometry.NewRect(b.Min.X, top, b.Width()-toolSlot, listH)
	canvas.DrawRoundRect(list, rgb(r.theme.Surface), 6)
	canvas.StrokeRoundRect(list, rgb(r.theme.Rule), 6, 1)
	for i, c := range r.choices {
		row := geometry.NewRect(list.Min.X, top+float32(i)*reportRow, list.Width(), reportRow)
		col := rgb(r.theme.Muted)
		if c.Root == r.selected {
			col = rgb(r.theme.Accent)
		}
		label := fitRun(canvas, c.Name, 13, col, r.theme.UIFont, row.Width()-20)
		drawFace(canvas, label, geometry.NewRect(row.Min.X+10, row.Min.Y+4, row.Width()-20, 20), 13, col, false, r.theme.UIFont)
	}
}

func (r *ReportBar) label() string {
	for _, c := range r.choices {
		if c.Root == r.selected {
			return c.Name
		}
	}
	if len(r.choices) > 0 {
		return r.choices[0].Name
	}
	return "Project"
}

// Event opens the project list, chooses one, or creates a report.
func (r *ReportBar) Event(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	b := r.Bounds()
	if !b.Contains(me.Position) {
		return false
	}
	localX := me.Position.X - b.Min.X
	localY := me.Position.Y - b.Min.Y
	if localY < reportRow {
		if localX >= b.Width()-toolSlot {
			if r.add != nil {
				r.add()
			}
		} else {
			r.open = !r.open
			r.MarkNeedsLayout()
			r.SetNeedsRedraw(true)
		}
		if ctx != nil {
			ctx.Invalidate()
		}
		return true
	}
	if !r.open {
		return false
	}
	i := int((localY - reportRow) / reportRow)
	if i >= 0 && i < len(r.choices) {
		root := r.choices[i].Root
		r.selected = root
		r.open = false
		r.MarkNeedsLayout()
		r.SetNeedsRedraw(true)
		if r.pick != nil {
			r.pick(root)
		}
	}
	if ctx != nil {
		ctx.Invalidate()
	}
	return true
}

// Children returns nil.
func (r *ReportBar) Children() []widget.Widget { return nil }
