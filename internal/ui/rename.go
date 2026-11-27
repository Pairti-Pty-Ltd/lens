package ui

import (
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// Rename asks for a report name. Enter saves. Escape cancels.
type Rename struct {
	widget.WidgetBase
	theme  theme.Theme
	field  *textfield.Widget
	on     bool
	commit func(string)
}

// NewRename returns a hidden card. commit receives the trimmed name.
func NewRename(commit func(string)) *Rename {
	r := &Rename{theme: theme.Pairti(), commit: commit}
	r.field = textfield.New(
		textfield.PainterOpt(FieldPainter(theme.Pairti())),
		textfield.OnSubmit(func(v string) {
			r.Hide()
			if r.commit != nil {
				r.commit(v)
			}
		}),
	)
	r.field.SetParent(r)
	r.SetVisible(true)
	r.SetEnabled(true)
	return r
}

// SetTheme replaces the colours.
func (r *Rename) SetTheme(t theme.Theme) {
	r.theme = t
	r.SetNeedsRedraw(true)
}

// Field is the editor, so the window can focus it.
func (r *Rename) Field() widget.Widget { return r.field }

// Show opens the card with the current name.
func (r *Rename) Show(name string) {
	r.on = true
	r.field.SetText(name)
	r.SetNeedsRedraw(true)
}

// Hide closes the card.
func (r *Rename) Hide() {
	if !r.on {
		return
	}
	r.on = false
	r.SetNeedsRedraw(true)
}

// Shown reports whether the card is open.
func (r *Rename) Shown() bool { return r.on }

// Layout fills the window and places the field in the card.
func (r *Rename) Layout(ctx widget.Context, c geometry.Constraints) geometry.Size {
	size := c.BiggestFinite(800, 600)
	r.placeField(ctx)
	return size
}

// Draw paints the title and the field.
func (r *Rename) Draw(ctx widget.Context, canvas widget.Canvas) {
	if !r.on || canvas == nil {
		return
	}
	card := r.card()
	canvas.DrawRoundRect(card, rgb(exportCard), 12)
	canvas.StrokeRoundRect(card, rgb(r.theme.Rule), 12, 1)
	drawFace(canvas, "Rename", geometry.NewRect(card.Min.X+20, card.Min.Y+16, card.Width()-40, 22), 16, rgb(r.theme.Ink), false, r.theme.UIFont)
	r.placeField(ctx)
	if r.field != nil {
		r.field.Draw(ctx, canvas)
	}
}

// placeField puts the editor inside the card. Layout can run before the
// window bounds exist, which previously left the field on the left of the card.
func (r *Rename) placeField(ctx widget.Context) {
	if r.field == nil {
		return
	}
	card := r.card()
	box := geometry.NewRect(card.Min.X+20, card.Min.Y+46, card.Width()-40, 28)
	widget.LayoutChild(r.field, ctx, geometry.Constraints{
		MinWidth: box.Width(), MaxWidth: box.Width(),
		MinHeight: box.Height(), MaxHeight: box.Height(),
	})
	r.field.SetBounds(box)
}

// Event edits the name. Escape and a click outside cancel.
func (r *Rename) Event(ctx widget.Context, e event.Event) bool {
	if !r.on {
		return false
	}
	if ke, ok := e.(*event.KeyEvent); ok {
		if ke.IsPress() && ke.Key == event.KeyEscape {
			r.Hide()
			invalidate(ctx)
			return true
		}
		if r.field != nil {
			return r.field.Event(ctx, e)
		}
		return true
	}
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress {
		return true
	}
	if r.field != nil && r.field.Bounds().Contains(me.Position) {
		return r.field.Event(ctx, e)
	}
	if !r.card().Contains(me.Position) {
		r.Hide()
		invalidate(ctx)
	}
	return true
}

// Children is the field while the card is open.
func (r *Rename) Children() []widget.Widget {
	if !r.on || r.field == nil {
		return nil
	}
	return []widget.Widget{r.field}
}

func (r *Rename) card() geometry.Rect {
	win := r.Bounds()
	const width = float32(360)
	const height = float32(96)
	x := win.Min.X + (win.Width()-width)/2
	y := win.Min.Y + 72
	if x < win.Min.X+16 {
		x = win.Min.X + 16
	}
	return geometry.NewRect(x, y, width, height)
}
