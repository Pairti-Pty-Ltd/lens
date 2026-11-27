package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const (
	TabClose       = "close"
	TabCloseOthers = "close-others"
	TabCloseRight  = "close-right"
	TabCloseLeft   = "close-left"
	TabCloseAll    = "close-all"
)

// TabMenu is the tab-bar context menu. It names the tab once, then the
// close commands that have something to close.
type TabMenu struct {
	widget.WidgetBase
	theme theme.Theme
	on    bool
	label string
	index int
	count int
	at    geometry.Point
	act   func(int, string)
}

// NewTabMenu returns a hidden menu. act receives the tab index and the command.
func NewTabMenu(act func(int, string)) *TabMenu {
	m := &TabMenu{theme: theme.Pairti(), act: act}
	m.SetVisible(true)
	m.SetEnabled(true)
	return m
}

// SetTheme replaces the semantic theme.
func (m *TabMenu) SetTheme(t theme.Theme) {
	m.theme = t
	m.SetNeedsRedraw(true)
}

// Show opens the menu for one tab. at is the pointer in window coordinates.
// count is how many tabs are open. index is the one under the pointer.
func (m *TabMenu) Show(label string, index, count int, at geometry.Point) {
	m.on = true
	m.label = label
	m.index = index
	m.count = count
	m.at = at
	m.SetNeedsRedraw(true)
}

// Hide closes the menu.
func (m *TabMenu) Hide() {
	if !m.on {
		return
	}
	m.on = false
	m.SetNeedsRedraw(true)
}

// Shown reports whether the menu is open.
func (m *TabMenu) Shown() bool { return m.on }

// Layout fills the window. The card is placed inside Draw.
func (m *TabMenu) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.BiggestFinite(800, 600)
}

// Draw paints the tab name once and the close commands.
func (m *TabMenu) Draw(_ widget.Context, canvas widget.Canvas) {
	if !m.on || canvas == nil {
		return
	}
	card, hits := m.place()
	canvas.DrawRoundRect(card, rgb(exportCard), 8)
	canvas.StrokeRoundRect(card, rgb(m.theme.Rule), 8, 1)
	title := fitMiddle(canvas, m.label, menuW-menuPad*2, 13, rgb(m.theme.Muted), m.theme.UIFont)
	drawFace(canvas, title, geometry.NewRect(card.Min.X+menuPad, card.Min.Y+menuPad, menuW-menuPad*2, 20), 13, rgb(m.theme.Muted), false, m.theme.UIFont)
	for _, h := range hits {
		drawFace(canvas, tabMenuLabel(h.id), geometry.NewRect(h.box.Min.X+10, h.box.Min.Y+4, h.box.Width()-16, 20), 13, rgb(m.theme.Ink), false, m.theme.UIFont)
	}
}

// Event handles a command, Escape, and a click outside.
func (m *TabMenu) Event(ctx widget.Context, ev event.Event) bool {
	if !m.on {
		return false
	}
	if ke, ok := ev.(*event.KeyEvent); ok {
		if ke.IsPress() && ke.Key == event.KeyEscape {
			m.Hide()
			invalidate(ctx)
		}
		return true
	}
	me, ok := ev.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return true
	}
	card, hits := m.place()
	if !card.Contains(me.Position) {
		m.Hide()
		invalidate(ctx)
		return true
	}
	for _, h := range hits {
		if !h.box.Contains(me.Position) {
			continue
		}
		index := m.index
		id := h.id
		m.Hide()
		if m.act != nil {
			m.act(index, id)
		}
		invalidate(ctx)
		return true
	}
	return true
}

// Children returns nil.
func (m *TabMenu) Children() []widget.Widget { return nil }

// Spot is the centre of a command, for tests.
func (m *TabMenu) Spot(id string) (geometry.Point, bool) {
	_, hits := m.place()
	for _, h := range hits {
		if h.id == id {
			return geometry.Pt(h.box.Min.X+h.box.Width()/2, h.box.Min.Y+h.box.Height()/2), true
		}
	}
	return geometry.Point{}, false
}

func (m *TabMenu) place() (geometry.Rect, []exportHit) {
	ids := tabMenuIDs(m.index, m.count)
	h := menuPad + 22 + float32(len(ids))*menuRow + menuPad
	win := m.Bounds()
	x, y := m.at.X, m.at.Y
	if x+menuW > win.Max.X-8 {
		x = win.Max.X - 8 - menuW
	}
	if y+h > win.Max.Y-8 {
		y = win.Max.Y - 8 - h
	}
	if x < win.Min.X+8 {
		x = win.Min.X + 8
	}
	if y < win.Min.Y+8 {
		y = win.Min.Y + 8
	}
	card := geometry.NewRect(x, y, menuW, h)
	var hits []exportHit
	ry := y + menuPad + 22
	for _, id := range ids {
		hits = append(hits, exportHit{id, geometry.NewRect(x+4, ry, menuW-8, menuRow)})
		ry += menuRow
	}
	return card, hits
}

func tabMenuIDs(index, count int) []string {
	ids := []string{TabClose}
	if count > 1 {
		ids = append(ids, TabCloseOthers)
	}
	if index >= 0 && index < count-1 {
		ids = append(ids, TabCloseRight)
	}
	if index > 0 {
		ids = append(ids, TabCloseLeft)
	}
	if count > 1 {
		ids = append(ids, TabCloseAll)
	}
	return ids
}

func tabMenuLabel(id string) string {
	switch id {
	case TabClose:
		return "Close"
	case TabCloseOthers:
		return "Close Others"
	case TabCloseRight:
		return "Close to the Right"
	case TabCloseLeft:
		return "Close to the Left"
	case TabCloseAll:
		return "Close All"
	default:
		return id
	}
}

func fitMiddle(canvas widget.Canvas, text string, width, size float32, col widget.Color, family string) string {
	if text == "" || textWidth(canvas, text, size, col, false, family) <= width {
		return text
	}
	r := []rune(text)
	for n := len(r) - 1; n > 8; n-- {
		head := n / 2
		tail := n - head
		s := string(r[:head]) + "…" + string(r[len(r)-tail:])
		if textWidth(canvas, s, size, col, false, family) <= width {
			return s
		}
	}
	return "…"
}
