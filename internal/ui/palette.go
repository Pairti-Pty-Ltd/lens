package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/icon"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/theme"
)

const (
	palW     = float32(640)
	palTitle = float32(32)
	palMeta  = float32(20)
	palGap   = float32(4)
	palPad   = float32(18)
	palQuery = float32(28)
	palShow  = 6
	palIcon  = float32(16)
)

// PaletteHit is one ranked note. Path is for opening. Rel is the short path drawn.
type PaletteHit struct {
	Title   string
	Rel     string
	Snippet string
	Path    string
	Root    string
}

// Palette is the note query. The sidebar field still filters the tree.
// Each row leads with the document mark. The open row is the title, one path, and one snippet.
type Palette struct {
	widget.WidgetBase
	theme   theme.Theme
	on      bool
	query   string
	hits    []PaletteHit
	at      int
	onQuery func(string)
	onOpen  func(PaletteHit)
}

// NewPalette returns a hidden palette.
func NewPalette(onQuery func(string), onOpen func(PaletteHit)) *Palette {
	p := &Palette{theme: theme.Pairti(), onQuery: onQuery, onOpen: onOpen}
	p.SetVisible(true)
	p.SetEnabled(true)
	return p
}

// SetTheme replaces the semantic theme.
func (p *Palette) SetTheme(t theme.Theme) {
	p.theme = t
	p.SetNeedsRedraw(true)
}

// Show opens an empty query.
func (p *Palette) Show() {
	p.on = true
	p.query = ""
	p.hits = nil
	p.at = 0
	p.SetNeedsRedraw(true)
}

// Hide closes the palette.
func (p *Palette) Hide() {
	if !p.on {
		return
	}
	p.on = false
	p.SetNeedsRedraw(true)
}

// Shown reports whether the palette is open.
func (p *Palette) Shown() bool { return p.on }

// SetHits replaces the rows for the current query.
func (p *Palette) SetHits(hits []PaletteHit) {
	p.hits = hits
	if p.at >= len(hits) {
		p.at = 0
	}
	p.SetNeedsRedraw(true)
}

// Query is the text in the field.
func (p *Palette) Query() string { return p.query }

// Layout fills the window.
func (p *Palette) Layout(_ widget.Context, c geometry.Constraints) geometry.Size {
	return c.BiggestFinite(800, 600)
}

// Draw paints the query once and the selected note's path and snippet.
func (p *Palette) Draw(_ widget.Context, canvas widget.Canvas) {
	if !p.on || canvas == nil {
		return
	}
	card := p.card()
	canvas.DrawRoundRect(card, rgb(exportCard), 12)
	canvas.StrokeRoundRect(card, rgb(p.theme.Rule), 12, 1)
	q := p.query
	col := rgb(p.theme.Ink)
	if q == "" {
		q = "Search notes"
		col = rgb(p.theme.Muted)
	}
	qx := card.Min.X + palPad
	qw := card.Width() - palPad*2
	drawFace(canvas, q, geometry.NewRect(qx, card.Min.Y+palPad, qw, palQuery), 17, col, false, p.theme.UIFont)
	if p.query != "" {
		caret := textWidth(canvas, p.query, 17, rgb(p.theme.Ink), false, p.theme.UIFont)
		canvas.DrawRect(geometry.NewRect(qx+caret+1, card.Min.Y+palPad+1, 1, 18), rgb(p.theme.Accent))
	}
	ruleY := card.Min.Y + palPad + palQuery
	canvas.DrawLine(geometry.Pt(qx, ruleY), geometry.Pt(qx+qw, ruleY), rgb(p.theme.Rule), 1)
	ink := rgb(p.theme.Ink)
	muted := rgb(p.theme.Muted)
	tx := qx + palIcon + 12
	tw := card.Max.X - palPad - tx
	for _, row := range p.rows(card) {
		h := p.hits[row.i]
		titleY := row.y
		iconY := row.y + (palTitle-palIcon)/2
		if row.i == p.at {
			canvas.DrawRoundRect(geometry.NewRect(card.Min.X+8, row.y, card.Width()-16, row.h), rgb(exportWell), 8)
			titleY = row.y + 6
			iconY = titleY + (palTitle-palIcon)/2
		}
		p.mark(canvas, h.Path, qx, iconY)
		title := fitTail(canvas, h.Title, tw, 15, ink, p.theme.UIFont)
		drawFace(canvas, title, geometry.NewRect(tx, titleY, tw, palTitle), 15, ink, false, p.theme.UIFont)
		if row.i != p.at {
			continue
		}
		rel := fitMiddle(canvas, h.Rel, tw, 13, muted, p.theme.UIFont)
		drawFace(canvas, rel, geometry.NewRect(tx, titleY+palTitle, tw, palMeta), 13, muted, false, p.theme.UIFont)
		snip := fitTail(canvas, h.Snippet, tw, 13, muted, p.theme.UIFont)
		if snip != "" {
			drawFace(canvas, snip, geometry.NewRect(tx, titleY+palTitle+palMeta, tw, palMeta), 13, muted, false, p.theme.UIFont)
		}
	}
}

// Event edits the query and opens the selected note.
func (p *Palette) Event(ctx widget.Context, ev event.Event) bool {
	if !p.on {
		return false
	}
	ke, ok := ev.(*event.KeyEvent)
	if ok && ke.IsPress() {
		switch ke.Key {
		case event.KeyEscape:
			p.Hide()
		case event.KeyEnter:
			p.open()
		case event.KeyDown:
			if p.at+1 < len(p.hits) {
				p.at++
			}
		case event.KeyUp:
			if p.at > 0 {
				p.at--
			}
		case event.KeyBackspace:
			r := []rune(p.query)
			if len(r) > 0 {
				p.query = string(r[:len(r)-1])
				p.at = 0
				if p.onQuery != nil {
					p.onQuery(p.query)
				}
			}
		default:
			if ke.Rune >= 32 && !ke.IsSuper() && !ke.IsCtrl() {
				p.query += string(ke.Rune)
				p.at = 0
				if p.onQuery != nil {
					p.onQuery(p.query)
				}
			}
		}
		p.SetNeedsRedraw(true)
		invalidate(ctx)
		return true
	}
	me, ok := ev.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return true
	}
	card := p.card()
	if !card.Contains(me.Position) {
		p.Hide()
		invalidate(ctx)
		return true
	}
	if i, ok := p.hitAt(me.Position.Y); ok {
		p.at = i
		p.open()
	}
	invalidate(ctx)
	return true
}

// Children returns nil.
func (p *Palette) Children() []widget.Widget { return nil }

func (p *Palette) open() {
	if p.at < 0 || p.at >= len(p.hits) {
		return
	}
	hit := p.hits[p.at]
	p.Hide()
	if p.onOpen != nil {
		p.onOpen(hit)
	}
}

func (p *Palette) span() float32 {
	w := palW
	win := p.Bounds()
	if win.Width() > 48 && win.Width() < w+32 {
		w = win.Width() - 32
	}
	if w < 320 {
		w = 320
	}
	return w
}

func (p *Palette) card() geometry.Rect {
	w := p.span()
	h := palPad + palQuery + 12 + palPad
	start, end := p.window()
	for i := start; i < end; i++ {
		h += p.blockH(i)
		if i+1 < end {
			h += palGap
		}
	}
	win := p.Bounds()
	x := win.Min.X + (win.Width()-w)/2
	y := win.Min.Y + 64
	if x < win.Min.X+16 {
		x = win.Min.X + 16
	}
	return geometry.NewRect(x, y, w, h)
}

func (p *Palette) blockH(i int) float32 {
	if i == p.at {
		return 6 + palTitle + palMeta + palMeta + 6
	}
	return palTitle
}

type palPlace struct {
	i    int
	y, h float32
}

func (p *Palette) rows(card geometry.Rect) []palPlace {
	start, end := p.window()
	y := card.Min.Y + palPad + palQuery + 12
	out := make([]palPlace, 0, end-start)
	for i := start; i < end; i++ {
		h := p.blockH(i)
		out = append(out, palPlace{i: i, y: y, h: h})
		y += h + palGap
	}
	return out
}

func (p *Palette) mark(canvas widget.Canvas, path string, x, y float32) {
	box := geometry.NewRect(x, y, palIcon, palIcon)
	if doc.IsTypst(path) {
		icon.Draw(canvas, iconFile, box, rgb(p.theme.Muted))
		return
	}
	icon.DrawMulti(canvas, icon.FileMD, box, icon.Palette{
		icon.KeyPrimary:  rgb(p.theme.Muted),
		icon.KeyMarkdown: rgb(p.theme.Accent),
	})
}

func (p *Palette) window() (int, int) {
	if len(p.hits) == 0 {
		return 0, 0
	}
	if p.at < 0 {
		p.at = 0
	}
	if p.at >= len(p.hits) {
		p.at = len(p.hits) - 1
	}
	start := 0
	if p.at >= palShow {
		start = p.at - palShow + 1
	}
	end := start + palShow
	if end > len(p.hits) {
		end = len(p.hits)
	}
	return start, end
}

func (p *Palette) hitAt(y float32) (int, bool) {
	for _, row := range p.rows(p.card()) {
		if y >= row.y && y < row.y+row.h {
			return row.i, true
		}
	}
	return 0, false
}

func fitTail(canvas widget.Canvas, text string, width, size float32, col widget.Color, family string) string {
	if text == "" || textWidth(canvas, text, size, col, false, family) <= width {
		return text
	}
	r := []rune(text)
	for n := len(r) - 1; n > 4; n-- {
		s := string(r[:n]) + "…"
		if textWidth(canvas, s, size, col, false, family) <= width {
			return s
		}
	}
	return "…"
}
