package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const (
	menuW   = float32(220)
	menuRow = float32(28)
	menuPad = float32(8)

	MenuCopyPath     = "copy-path"
	MenuCopyRel      = "copy-rel"
	MenuReveal       = "reveal"
	MenuOpen         = "open"
	MenuTrash        = "trash"
	MenuDelete       = "delete"
	MenuReportAdd    = "report-add"
	MenuReportUp     = "report-up"
	MenuReportDown   = "report-down"
	MenuReportDrop   = "report-drop"
	MenuReportRename = "report-rename"
	MenuReportDelete = "report-delete"
)

// TreeMenu is the file-tree context menu. Delete asks for a second click.
type TreeMenu struct {
	widget.WidgetBase
	theme      theme.Theme
	on         bool
	armed      bool
	item       Item
	at         geometry.Point
	addReport  bool
	reportFile bool
	reportDir  bool
	commit     func(Item, string)
}

// NewTreeMenu returns a hidden menu. commit runs for a chosen action.
func NewTreeMenu(commit func(Item, string)) *TreeMenu {
	m := &TreeMenu{theme: theme.Pairti(), commit: commit}
	m.SetVisible(true)
	m.SetEnabled(true)
	return m
}

// SetTheme replaces the semantic theme.
func (m *TreeMenu) SetTheme(t theme.Theme) {
	m.theme = t
	m.SetNeedsRedraw(true)
}

// SetReport chooses the report rows. add is "Add to report" on a file in the tree.
// file is a note inside a report: up, down, and remove.
// dir is a report folder: rename and delete the report.
func (m *TreeMenu) SetReport(add, file, dir bool) {
	m.addReport = add
	m.reportFile = file
	m.reportDir = dir
}

// Show opens the menu for one row. at is the pointer in window coordinates.
func (m *TreeMenu) Show(it Item, at geometry.Point) {
	m.on = true
	m.armed = false
	m.item = it
	m.at = at
	m.SetNeedsRedraw(true)
}

// Hide closes the menu.
func (m *TreeMenu) Hide() {
	if !m.on {
		return
	}
	m.on = false
	m.armed = false
	m.SetNeedsRedraw(true)
}

// Shown reports whether the menu is open.
func (m *TreeMenu) Shown() bool { return m.on }

// Layout fills the window. The card is placed inside Draw.
func (m *TreeMenu) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.BiggestFinite(800, 600)
}

// Draw paints the menu. The delete confirmation stays hidden until Delete is pressed.
func (m *TreeMenu) Draw(_ widget.Context, canvas widget.Canvas) {
	if !m.on || canvas == nil {
		return
	}
	card, hits := m.place()
	canvas.DrawRoundRect(card, rgb(exportCard), 8)
	canvas.StrokeRoundRect(card, rgb(m.theme.Rule), 8, 1)
	title := m.fitted(canvas, m.item.Label, menuW-menuPad*2)
	drawFace(canvas, title, geometry.NewRect(card.Min.X+menuPad, card.Min.Y+menuPad, menuW-menuPad*2, 20), 13, rgb(m.theme.Muted), false, m.theme.UIFont)
	for _, h := range hits {
		label := menuLabel(h.id, m.armed)
		color := rgb(m.theme.Ink)
		if h.id == MenuDelete && m.armed {
			color = rgb(0xE06C75)
		}
		drawFace(canvas, label, geometry.NewRect(h.box.Min.X+10, h.box.Min.Y+4, h.box.Width()-16, 20), 13, color, false, m.theme.UIFont)
	}
}

// Event handles a row, Escape, and a click outside.
func (m *TreeMenu) Event(ctx widget.Context, ev event.Event) bool {
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
		m.press(h.id)
		invalidate(ctx)
		return true
	}
	return true
}

// Children returns nil.
func (m *TreeMenu) Children() []widget.Widget { return nil }

// Spot is the centre of a row, for tests.
func (m *TreeMenu) Spot(id string) (geometry.Point, bool) {
	_, hits := m.place()
	for _, h := range hits {
		if h.id == id {
			return geometry.Pt(h.box.Min.X+h.box.Width()/2, h.box.Min.Y+h.box.Height()/2), true
		}
	}
	return geometry.Point{}, false
}

func (m *TreeMenu) press(id string) {
	if id == MenuDelete && !m.armed {
		m.armed = true
		m.SetNeedsRedraw(true)
		return
	}
	item := m.item
	m.Hide()
	if m.commit != nil {
		m.commit(item, id)
	}
}

func (m *TreeMenu) place() (geometry.Rect, []exportHit) {
	var ids []string
	if m.reportDir {
		ids = []string{MenuReportRename, MenuReportDelete}
	} else {
		if m.addReport && !m.item.Dir && m.item.Path != "" {
			ids = append(ids, MenuReportAdd)
		}
		if m.reportFile && !m.item.Dir && m.item.Path != "" {
			ids = append(ids, MenuReportUp, MenuReportDown, MenuReportDrop)
		}
		ids = append(ids, MenuCopyPath, MenuCopyRel, MenuReveal, MenuOpen, MenuTrash, MenuDelete)
	}
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

func (m *TreeMenu) fitted(canvas widget.Canvas, text string, width float32) string {
	if text == "" || textWidth(canvas, text, 13, rgb(m.theme.Muted), false, m.theme.UIFont) <= width {
		return text
	}
	r := []rune(text)
	for n := len(r) - 1; n > 8; n-- {
		head := n / 2
		tail := n - head
		s := string(r[:head]) + "…" + string(r[len(r)-tail:])
		if textWidth(canvas, s, 13, rgb(m.theme.Muted), false, m.theme.UIFont) <= width {
			return s
		}
	}
	return "…"
}

func menuLabel(id string, armed bool) string {
	switch id {
	case MenuCopyPath:
		return "Copy Path"
	case MenuCopyRel:
		return "Copy Relative Path"
	case MenuReveal:
		return "Reveal in Finder"
	case MenuOpen:
		return "Open in App"
	case MenuTrash:
		return "Move to Bin"
	case MenuDelete:
		if armed {
			return "Delete forever"
		}
		return "Delete"
	case MenuReportAdd:
		return "Add to report"
	case MenuReportUp:
		return "Move up"
	case MenuReportDown:
		return "Move down"
	case MenuReportDrop:
		return "Remove from report"
	case MenuReportRename:
		return "Rename"
	case MenuReportDelete:
		return "Delete report"
	default:
		return id
	}
}
