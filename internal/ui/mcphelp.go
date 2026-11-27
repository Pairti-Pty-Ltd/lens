package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// MCPHelp shows the stable local address and bearer token.
type MCPHelp struct {
	widget.WidgetBase
	theme      theme.Theme
	on         bool
	url, token string
	problem    string
}

// NewMCPHelp returns a hidden card.
func NewMCPHelp() *MCPHelp {
	m := &MCPHelp{theme: theme.Pairti()}
	m.SetVisible(true)
	m.SetEnabled(true)
	return m
}

// SetTheme replaces the colours.
func (m *MCPHelp) SetTheme(t theme.Theme) {
	m.theme = t
	m.SetNeedsRedraw(true)
}

// Show opens the card. problem replaces the token line when the listener failed.
func (m *MCPHelp) Show(url, token, problem string) {
	m.on = true
	m.url = url
	m.token = token
	m.problem = problem
	m.SetNeedsRedraw(true)
}

// Hide closes the card.
func (m *MCPHelp) Hide() {
	if !m.on {
		return
	}
	m.on = false
	m.SetNeedsRedraw(true)
}

// Shown reports whether the card is open.
func (m *MCPHelp) Shown() bool { return m.on }

// Layout fills the window. The card is placed in Draw.
func (m *MCPHelp) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.BiggestFinite(800, 600)
}

// Draw paints the address once and the token once.
func (m *MCPHelp) Draw(_ widget.Context, canvas widget.Canvas) {
	if !m.on || canvas == nil {
		return
	}
	card := m.card()
	canvas.DrawRoundRect(card, rgb(exportCard), 12)
	canvas.StrokeRoundRect(card, rgb(m.theme.Rule), 12, 1)
	x := card.Min.X + 20
	w := card.Width() - 40
	y := card.Min.Y + 18
	drawFace(canvas, "MCP", geometry.NewRect(x, y, w, 22), 16, rgb(m.theme.Ink), false, m.theme.UIFont)
	y += 28
	drawFace(canvas, m.url, geometry.NewRect(x, y, w, 20), 13, rgb(m.theme.Accent), false, m.theme.UIFont)
	y += 24
	line := "Bearer " + m.token
	if m.problem != "" {
		line = m.problem
	}
	drawFace(canvas, line, geometry.NewRect(x, y, w, 20), 13, rgb(m.theme.Ink), false, m.theme.UIFont)
	y += 24
	drawFace(canvas, "This machine only. The address stays.", geometry.NewRect(x, y, w, 20), 13, rgb(m.theme.Muted), false, m.theme.UIFont)
}

// Event closes on Escape or a click outside the card.
func (m *MCPHelp) Event(ctx widget.Context, e event.Event) bool {
	if !m.on {
		return false
	}
	if ke, ok := e.(*event.KeyEvent); ok {
		if ke.IsPress() && ke.Key == event.KeyEscape {
			m.Hide()
			invalidate(ctx)
		}
		return true
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return true
	}
	if !m.card().Contains(me.Position) {
		m.Hide()
		invalidate(ctx)
	}
	return true
}

// Children returns nil.
func (m *MCPHelp) Children() []widget.Widget { return nil }

func (m *MCPHelp) card() geometry.Rect {
	win := m.Bounds()
	const width = float32(520)
	const height = float32(132)
	x := win.Min.X + (win.Width()-width)/2
	y := win.Min.Y + 72
	if x < win.Min.X+16 {
		x = win.Min.X + 16
	}
	return geometry.NewRect(x, y, width, height)
}
