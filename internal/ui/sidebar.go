package ui

import (
	"runtime"
	"time"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/icon"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const (
	sidebarRow = float32(26)
	doubleGap  = 400 * time.Millisecond
)

// Item is one sidebar row. Dir rows toggle. File rows open.
type Item struct {
	Depth       int
	Label       string
	Path        string
	Rel         string
	ProjectRoot string
	ProjectName string
	Dir         bool
	Expanded    bool
	More        bool
	Revision    string // Git commit for a historical file
	Note        string // report id when this row is a file inside a report
}

// Sidebar draws the project tree and hit-tests in the same coordinates it paints.
// It does not use the toolkit tree, whose click position and row bounds disagree.
type Sidebar struct {
	widget.WidgetBase
	items     []Item
	selected  string
	scroll    float32
	onPick    func(Item)
	onTab     func(Item)
	onMenu    func(Item, geometry.Point)
	onOrder   func(Item, int)
	onChevron func(Item)
	lastPath  string
	lastAt    time.Time
	theme     theme.Theme
}

// NewSidebar returns an empty sidebar. onPick runs for a directory toggle or a file.
func NewSidebar(onPick func(Item)) *Sidebar {
	s := &Sidebar{onPick: onPick, theme: theme.Pairti()}
	s.SetVisible(true)
	s.SetEnabled(true)
	return s
}

// SetTheme replaces the semantic theme.
func (s *Sidebar) SetTheme(t theme.Theme) { s.theme = t; s.SetNeedsRedraw(true) }

// OnMenu runs for a right click on a file or folder. The point is in window coordinates.
func (s *Sidebar) OnMenu(fn func(Item, geometry.Point)) { s.onMenu = fn }

// OnOrder runs when a report row's up (-1) or down (1) control is clicked.
func (s *Sidebar) OnOrder(fn func(Item, int)) { s.onOrder = fn }

// OnChevron runs when the disclosure triangle is clicked. The rest of a
// folder row still uses onPick.
func (s *Sidebar) OnChevron(fn func(Item)) { s.onChevron = fn }

// OnTab runs for a double click on a file. A single click still uses onPick.
func (s *Sidebar) OnTab(fn func(Item)) { s.onTab = fn }

// SetItems replaces the visible rows.
func (s *Sidebar) SetItems(items []Item) {
	s.items = items
	s.clampScroll()
	s.SetNeedsRedraw(true)
}

// Select marks the open file.
func (s *Sidebar) Select(path string) {
	s.selected = path
	s.SetNeedsRedraw(true)
}

// Selected is the marked file, or empty.
func (s *Sidebar) Selected() string { return s.selected }

// Layout fills the space the parent gives it. Rows scroll inside that space.
func (s *Sidebar) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.BiggestFinite(200, 200)
}

// Draw paints the rows. Coordinates match Layout bounds, which match Event.
func (s *Sidebar) Draw(_ widget.Context, canvas widget.Canvas) {
	if !s.IsVisible() {
		return
	}
	b := s.Bounds()
	if b.IsEmpty() {
		return
	}
	canvas.PushClip(b)
	s.clampScroll()
	ink := rgb(s.theme.Ink)
	muted := rgb(s.theme.Muted)
	y := b.Min.Y - s.scroll
	for i, it := range s.items {
		if y+sidebarRow >= b.Min.Y && y <= b.Max.Y {
			if it.Path != "" && it.Path == s.selected {
				canvas.DrawRect(geometry.NewRect(b.Min.X, y, b.Width(), sidebarRow), rgba(s.theme.Selection))
			}
			x := b.Min.X + 4 + float32(it.Depth)*16
			color := ink
			if it.More || it.Dir {
				color = muted
			}
			box := geometry.NewRect(x, y+(sidebarRow-14)/2, 14, 14)
			if it.Dir {
				chev := icon.ChevronRight
				folder := icon.FolderClosed
				if it.Expanded {
					chev = icon.ChevronDown
					folder = icon.FolderOpen
				}
				icon.Draw(canvas, chev, box, muted)
				icon.Draw(canvas, folder, box.TranslateXY(16, 0), muted)
				x += 34
			} else if !it.More && it.Path != "" {
				icon.DrawMulti(canvas, icon.FileMD, box.TranslateXY(16, 0), icon.Palette{
					icon.KeyPrimary:  muted,
					icon.KeyMarkdown: rgb(s.theme.Accent),
				})
				x += 34
			}
			right := b.Max.X - 8
			if it.Note != "" && !it.Dir {
				right -= orderW * 2
				s.drawOrder(canvas, b, y, i)
			}
			label := fitLabel(it.Label, right-x, s.theme.Meta)
			drawFace(canvas, label, geometry.NewRect(x, y, right-x, sidebarRow), s.theme.Meta, color, false, s.theme.UIFont)
		}
		y += sidebarRow
	}
	canvas.PopClip()
}

// Event handles clicks and the wheel. Positions are in the parent's coordinates.
func (s *Sidebar) Event(ctx widget.Context, e event.Event) bool {
	if we, ok := e.(*event.WheelEvent); ok {
		if !s.Bounds().Contains(we.Position) {
			return false
		}
		s.scrollBy(WheelScroll(we.DeltaY(), runtime.GOOS == "darwin"))
		s.SetNeedsRedraw(true)
		ctx.InvalidateRect(s.Bounds())
		return true
	}
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	switch me.MouseType {
	case event.MouseDoubleClick:
		return s.activate(ctx, me, true)
	case event.MousePress:
		if me.Button == event.ButtonRight {
			return s.rightClick(ctx, me)
		}
		return s.activate(ctx, me, me.ClickCount >= 2)
	default:
		return false
	}
}

// activate opens a file. tab asks for a new tab. Two presses on the same
// file inside doubleGap are a double click, because the window delivers
// each press with ClickCount 1.
func (s *Sidebar) activate(ctx widget.Context, me *event.MouseEvent, tab bool) bool {
	if !tab && me.Button != event.ButtonLeft {
		return false
	}
	if !s.Bounds().Contains(me.Position) {
		return false
	}
	i := RowIndex(me.Position.Y, s.Bounds().Min.Y, s.scroll, sidebarRow, len(s.items))
	if i < 0 {
		return false
	}
	it := s.items[i]
	if it.More {
		return true
	}
	if it.Dir && s.chevronAt(me.Position, i) {
		if s.onChevron != nil {
			s.onChevron(it)
		} else if s.onPick != nil {
			s.onPick(it)
		}
		s.SetNeedsRedraw(true)
		if ctx != nil {
			ctx.Invalidate()
		}
		return true
	}
	if dir, ok := s.orderAt(me.Position, i); ok {
		if s.onOrder != nil {
			s.onOrder(it, dir)
		}
		s.SetNeedsRedraw(true)
		if ctx != nil {
			ctx.Invalidate()
		}
		return true
	}
	if !it.Dir && it.Path != "" {
		s.selected = it.Path
	}
	if !it.Dir && it.Path != "" {
		now := time.Now()
		if !tab && it.Path == s.lastPath && now.Sub(s.lastAt) <= doubleGap {
			tab = true
		}
		if tab {
			s.lastPath = ""
			s.lastAt = time.Time{}
		} else {
			s.lastPath = it.Path
			s.lastAt = now
		}
	}
	if tab && !it.Dir && s.onTab != nil {
		s.onTab(it)
	} else if s.onPick != nil {
		s.onPick(it)
	}
	// The document lives outside this widget. A sidebar-only dirty
	// rect leaves the new note unpainted until the next event.
	s.SetNeedsRedraw(true)
	if ctx != nil {
		ctx.Invalidate()
	}
	return true
}

func (s *Sidebar) rightClick(ctx widget.Context, me *event.MouseEvent) bool {
	if !s.Bounds().Contains(me.Position) {
		return false
	}
	i := RowIndex(me.Position.Y, s.Bounds().Min.Y, s.scroll, sidebarRow, len(s.items))
	if i < 0 {
		return true
	}
	it := s.items[i]
	if it.More || (it.Path == "" && !(it.Dir && it.Rel != "" && it.Rel != "new")) {
		return true
	}
	if it.Path != "" {
		s.selected = it.Path
	}
	if s.onMenu != nil {
		origin := screenOrigin(s)
		local := me.Position.Sub(s.Bounds().Min)
		s.onMenu(it, geometry.Pt(origin.X+local.X, origin.Y+local.Y))
	}
	s.SetNeedsRedraw(true)
	if ctx != nil {
		ctx.Invalidate()
	}
	return true
}

// Children returns nil.
func (s *Sidebar) Children() []widget.Widget { return nil }

const orderW = float32(18)

func (s *Sidebar) drawOrder(canvas widget.Canvas, b geometry.Rect, y float32, i int) {
	up := s.orderBox(b, y, 0)
	down := s.orderBox(b, y, 1)
	upCol := rgb(s.theme.Muted)
	downCol := rgb(s.theme.Muted)
	if !canOrder(s.items, i, -1) {
		upCol = rgb(s.theme.Rule)
	}
	if !canOrder(s.items, i, 1) {
		downCol = rgb(s.theme.Rule)
	}
	drawFace(canvas, "↑", up, 13, upCol, false, s.theme.UIFont)
	drawFace(canvas, "↓", down, 13, downCol, false, s.theme.UIFont)
}

func (s *Sidebar) chevronAt(p geometry.Point, i int) bool {
	if i < 0 || i >= len(s.items) || !s.items[i].Dir {
		return false
	}
	b := s.Bounds()
	y := b.Min.Y - s.scroll + float32(i)*sidebarRow
	x := b.Min.X + 4 + float32(s.items[i].Depth)*16
	return geometry.NewRect(x, y+(sidebarRow-14)/2, 14, 14).Contains(p)
}

func (s *Sidebar) orderBox(b geometry.Rect, y float32, which int) geometry.Rect {
	x := b.Max.X - 8 - orderW*2 + float32(which)*orderW
	return geometry.NewRect(x, y, orderW, sidebarRow)
}

func (s *Sidebar) orderAt(p geometry.Point, i int) (int, bool) {
	if i < 0 || i >= len(s.items) || s.items[i].Note == "" || s.items[i].Dir {
		return 0, false
	}
	b := s.Bounds()
	y := b.Min.Y - s.scroll + float32(i)*sidebarRow
	if s.orderBox(b, y, 0).Contains(p) {
		return -1, canOrder(s.items, i, -1)
	}
	if s.orderBox(b, y, 1).Contains(p) {
		return 1, canOrder(s.items, i, 1)
	}
	return 0, false
}

func canOrder(items []Item, i, dir int) bool {
	j := i + dir
	if j < 0 || j >= len(items) {
		return false
	}
	return items[i].Note != "" && items[j].Note == items[i].Note && !items[j].Dir
}

func fitLabel(s string, px, size float32) string {
	if size <= 0 {
		size = 13
	}
	max := int(px / (size * 0.5))
	if max < 4 {
		return s
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max-1]) + "…"
}

func (s *Sidebar) scrollBy(dy float32) {
	s.scroll += dy
	s.clampScroll()
}

func (s *Sidebar) clampScroll() {
	maxScroll := float32(len(s.items))*sidebarRow - s.Bounds().Height()
	if maxScroll < 0 {
		maxScroll = 0
	}
	if s.scroll < 0 {
		s.scroll = 0
	}
	if s.scroll > maxScroll {
		s.scroll = maxScroll
	}
}
