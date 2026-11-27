package ui

import (
	"runtime"
	"strings"

	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/icon"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

const (
	railIcons = float32(36)
	railTitle = float32(28)
	railModeN = 4
)

// Rail is the right sidebar. The first mode is the table of contents.
// The others are the document, its links, and its images.
type Rail struct {
	widget.WidgetBase
	theme  theme.Theme
	mode   int
	heads  []Heading
	info   []string
	links  []Ref
	pics   []Ref
	rows   []railRow
	scroll float32
	onJump func(int)
	onOpen func(string)
}

type railRow struct {
	y, h   float32
	line   int
	url    string
	number string
	text   []string
	level  int
}

// NewRail returns a contents sidebar. onJump scrolls to a source line.
// onOpen follows a link.
func NewRail(onJump func(int), onOpen func(string)) *Rail {
	r := &Rail{theme: theme.Pairti(), onJump: onJump, onOpen: onOpen}
	r.SetVisible(true)
	r.SetEnabled(true)
	return r
}

// SetTheme replaces the semantic theme.
func (r *Rail) SetTheme(t theme.Theme) {
	r.theme = t
	r.MarkNeedsLayout()
	r.SetNeedsRedraw(true)
}

// SetOutline replaces the table of contents.
func (r *Rail) SetOutline(h []Heading) {
	r.heads = h
	r.MarkNeedsLayout()
	r.SetNeedsRedraw(true)
}

// Outline is the current table of contents.
func (r *Rail) Outline() []Heading { return r.heads }

// Home scrolls the rail back to the first row.
func (r *Rail) Home() {
	r.scroll = 0
	r.MarkNeedsLayout()
	r.SetNeedsRedraw(true)
}

// ShowContents selects heading navigation, including when another rail mode is open.
func (r *Rail) ShowContents() {
	r.mode = 0
	r.Home()
}

// SetInfo replaces the document lines. At most four are shown.
func (r *Rail) SetInfo(lines []string) {
	if len(lines) > 4 {
		lines = lines[:4]
	}
	r.info = lines
	r.MarkNeedsLayout()
	r.SetNeedsRedraw(true)
}

// SetLinks replaces the link list.
func (r *Rail) SetLinks(links []Ref) {
	r.links = links
	r.MarkNeedsLayout()
	r.SetNeedsRedraw(true)
}

// SetPictures replaces the image list.
func (r *Rail) SetPictures(pics []Ref) {
	r.pics = pics
	r.MarkNeedsLayout()
	r.SetNeedsRedraw(true)
}

// Mode is the selected sidebar, 0 through 3.
func (r *Rail) Mode() int { return r.mode }

// Layout fills the pane and wraps the contents rows.
func (r *Rail) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	size := c.BiggestFinite(280, 400)
	r.rows = r.build(size.Width)
	return size
}

func (r *Rail) build(width float32) []railRow {
	top := railIcons + railTitle
	size := r.theme.Meta
	if size <= 0 {
		size = 13
	}
	switch r.mode {
	case 1:
		var rows []railRow
		y := top
		for _, line := range r.info {
			rows = append(rows, railRow{y: y, h: 22, text: []string{fitLabel(line, width-24, size)}})
			y += 22
		}
		return rows
	case 2:
		return r.refRows(r.links, width, top, size, false)
	case 3:
		return r.refRows(r.pics, width, top, size, true)
	default:
		var rows []railRow
		y := top
		numW := float32(52)
		for _, h := range r.heads {
			indent := float32(h.Level-1) * 12
			if indent < 0 {
				indent = 0
			}
			textW := width - 16 - indent - numW
			if textW < 40 {
				textW = 40
			}
			lines := wrapWords(h.Text, int(textW/(size*0.55)))
			if len(lines) == 0 {
				lines = []string{h.Text}
			}
			rh := float32(len(lines))*size*r.theme.Line + 6
			if rh < 24 {
				rh = 24
			}
			rows = append(rows, railRow{
				y: y, h: rh, line: h.Line, number: h.Number, text: lines, level: h.Level,
			})
			y += rh
		}
		return rows
	}
}

func (r *Rail) refRows(refs []Ref, width, top, size float32, jump bool) []railRow {
	var rows []railRow
	y := top
	for _, ref := range refs {
		row := railRow{y: y, h: 24, text: []string{fitLabel(ref.Label, width-24, size)}}
		if jump {
			row.line = ref.Line
		} else {
			row.url = ref.URL
			row.line = ref.Line
		}
		rows = append(rows, row)
		y += 24
	}
	return rows
}

func wrapWords(text string, max int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if max < 8 {
		max = 8
	}
	words := strings.Fields(text)
	var lines []string
	var cur string
	for _, w := range words {
		if cur == "" {
			cur = w
			continue
		}
		if len([]rune(cur))+1+len([]rune(w)) > max {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur += " " + w
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

// Draw paints the mode icons, the title, and the rows.
func (r *Rail) Draw(_ widget.Context, canvas widget.Canvas) {
	if !r.IsVisible() {
		return
	}
	b := r.Bounds()
	canvas.PushClip(b)
	defer canvas.PopClip()
	muted := rgb(r.theme.Muted)
	ink := rgb(r.theme.Ink)
	accent := rgb(r.theme.Accent)
	slot := float32(28)
	pad := float32(8)
	for i := 0; i < railModeN; i++ {
		x := b.Min.X + pad + float32(i)*slot
		box := geometry.NewRect(x, b.Min.Y+6, 22, 22)
		if i == r.mode {
			canvas.DrawRoundRect(box, rgba(r.theme.Selection), 4)
		}
		color := muted
		if i == r.mode {
			color = accent
		}
		icon.Draw(canvas, railIcon(i), geometry.NewRect(x+3, b.Min.Y+9, 16, 16), color)
	}
	drawFace(canvas, r.title(), geometry.NewRect(b.Min.X+12, b.Min.Y+railIcons, b.Width()-24, railTitle), r.theme.Body, ink, true, r.theme.UIFont)
	origin := b.Min.Y + railIcons + railTitle - r.scroll
	numW := float32(52)
	for _, row := range r.rows {
		y := origin + row.y - (railIcons + railTitle)
		if y+row.h < b.Min.Y+railIcons+railTitle || y > b.Max.Y {
			continue
		}
		indent := float32(row.level-1) * 12
		if row.level == 0 {
			indent = 0
		}
		x := b.Min.X + 12 + indent
		if row.number != "" {
			drawFace(canvas, row.number, geometry.NewRect(x, y, numW, r.theme.Meta*r.theme.Line), r.theme.Meta, accent, false, r.theme.UIFont)
			x += numW
		}
		lineH := r.theme.Meta * r.theme.Line
		for i, line := range row.text {
			drawFace(canvas, line, geometry.NewRect(x, y+float32(i)*lineH, b.Max.X-x-8, lineH), r.theme.Meta, ink, false, r.theme.UIFont)
		}
	}
}

func (r *Rail) title() string {
	switch r.mode {
	case 1:
		return "Document"
	case 2:
		return "Links"
	case 3:
		return "Images"
	default:
		return "Table of contents"
	}
}

func railIcon(mode int) icon.IconData {
	switch mode {
	case 1:
		return iconDoc
	case 2:
		return iconCopy
	case 3:
		return iconClip
	default:
		return iconList
	}
}

func railTip(mode int) string {
	switch mode {
	case 1:
		return "Document"
	case 2:
		return "Links"
	case 3:
		return "Images"
	default:
		return "Table of contents"
	}
}

func (r *Rail) hover(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok {
		return false
	}
	if me.MouseType == event.MouseLeave {
		offerTip(ctx, r.theme, "", geometry.Rect{})
		return true
	}
	local, over := hoverPoint(r, me)
	if !over {
		return false
	}
	if local.Y < 0 || local.Y >= railIcons {
		offerTip(ctx, r.theme, "", geometry.Rect{})
		return true
	}
	slot := float32(28)
	i := int((local.X - 8) / slot)
	if i < 0 || i >= railModeN {
		offerTip(ctx, r.theme, "", geometry.Rect{})
		return true
	}
	origin := screenOrigin(r)
	offerTip(ctx, r.theme, railTip(i), geometry.NewRect(origin.X+8+float32(i)*slot, origin.Y, slot, railIcons))
	return true
}

// Event switches mode, jumps to a heading, or follows a link.
// A move over the mode icons names that icon.
func (r *Rail) Event(ctx widget.Context, e event.Event) bool {
	if r.hover(ctx, e) {
		return false
	}
	if we, ok := e.(*event.WheelEvent); ok {
		if !r.Bounds().Contains(we.Position) {
			return false
		}
		r.scrollBy(WheelScroll(we.DeltaY(), runtime.GOOS == "darwin"))
		r.SetNeedsRedraw(true)
		if ctx != nil {
			ctx.InvalidateRect(r.Bounds())
		}
		return true
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	if !r.Bounds().Contains(me.Position) {
		return false
	}
	local := me.Position.Sub(r.Bounds().Min)
	if local.Y < railIcons {
		slot := float32(28)
		i := int((local.X - 8) / slot)
		if i >= 0 && i < railModeN {
			r.mode = i
			r.scroll = 0
			r.MarkNeedsLayout()
			r.SetNeedsRedraw(true)
			if ctx != nil {
				ctx.Invalidate()
			}
			return true
		}
		return true
	}
	y := local.Y + r.scroll
	for _, row := range r.rows {
		if y < row.y || y >= row.y+row.h {
			continue
		}
		if row.url != "" && r.onOpen != nil {
			r.onOpen(row.url)
		} else if row.line > 0 && r.onJump != nil {
			r.onJump(row.line)
		}
		return true
	}
	return true
}

func (r *Rail) scrollBy(dy float32) {
	r.scroll += dy
	var content float32
	if n := len(r.rows); n > 0 {
		last := r.rows[n-1]
		content = last.y + last.h - railIcons - railTitle
	}
	maxScroll := content - (r.Bounds().Height() - railIcons - railTitle)
	if maxScroll < 0 {
		maxScroll = 0
	}
	if r.scroll < 0 {
		r.scroll = 0
	}
	if r.scroll > maxScroll {
		r.scroll = maxScroll
	}
}

// Children returns nil.
func (r *Rail) Children() []widget.Widget { return nil }
