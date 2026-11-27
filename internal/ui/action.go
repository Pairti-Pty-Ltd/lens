package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// Action is the header control. With no choices it is Edit, Save, or Done.
// With choices it is the disk-change decision.
type Action struct {
	widget.WidgetBase
	label   string
	active  string
	choices []string
	theme   theme.Theme
	on      func(string)
	spans   []actionSpan
}

type actionSpan struct {
	label string
	x, w  float32
}

// NewAction returns the header control. on receives the clicked label.
func NewAction(on func(string)) *Action {
	a := &Action{label: "Edit", on: on, theme: theme.Pairti()}
	a.SetVisible(true)
	a.SetEnabled(true)
	return a
}

// SetTheme replaces the semantic theme.
func (a *Action) SetTheme(t theme.Theme) {
	a.theme = t
	a.MarkNeedsLayout()
	a.SetNeedsRedraw(true)
}

// SetActive paints that label in the accent colour.
func (a *Action) SetActive(label string) {
	if a.active == label {
		return
	}
	a.active = label
	a.SetNeedsRedraw(true)
}

// SetLabel sets the single action. It is hidden while choices are set.
func (a *Action) SetLabel(s string) {
	if a.label == s {
		return
	}
	a.label = s
	a.MarkNeedsLayout()
	a.SetNeedsRedraw(true)
}

// SetChoices shows a decision, or clears it when nil.
func (a *Action) SetChoices(labels []string) {
	a.choices = append([]string(nil), labels...)
	a.MarkNeedsLayout()
	a.SetNeedsRedraw(true)
}

// Choices is the current decision, or nil.
func (a *Action) Choices() []string {
	if len(a.choices) == 0 {
		return nil
	}
	return append([]string(nil), a.choices...)
}

// Layout sizes the label or the choice row.
func (a *Action) Layout(widget.Context, geometry.Constraints) geometry.Size {
	a.spans = a.measure()
	var w float32
	for _, sp := range a.spans {
		if sp.x+sp.w > w {
			w = sp.x + sp.w
		}
	}
	if len(a.spans) == 0 {
		return geometry.Sz(0, 0)
	}
	return geometry.Sz(w, 26)
}

// Draw paints the label or the choices.
func (a *Action) Draw(_ widget.Context, canvas widget.Canvas) {
	if !a.IsVisible() {
		return
	}
	b := a.Bounds()
	ink := rgb(a.theme.Ink)
	accent := rgb(a.theme.Accent)
	for _, sp := range a.spans {
		color := ink
		if sp.label == a.active {
			color = accent
		}
		drawFace(canvas, sp.label, geometry.NewRect(b.Min.X+sp.x, b.Min.Y, sp.w, b.Height()), a.theme.Meta, color, false, a.theme.UIFont)
	}
}

// Event hits one label.
func (a *Action) Event(ctx widget.Context, e event.Event) bool {
	me, ok := e.(*event.MouseEvent)
	if !ok || me.MouseType != event.MousePress || me.Button != event.ButtonLeft {
		return false
	}
	if !a.Bounds().Contains(me.Position) {
		return false
	}
	x := me.Position.X - a.Bounds().Min.X
	for _, sp := range a.spans {
		if x >= sp.x && x < sp.x+sp.w {
			if a.on != nil {
				a.on(sp.label)
			}
			ctx.Invalidate()
			return true
		}
	}
	return false
}

// Children returns nil.
func (a *Action) Children() []widget.Widget { return nil }

func (a *Action) measure() []actionSpan {
	labels := a.choices
	if len(labels) == 0 && a.label != "" {
		labels = []string{a.label}
	}
	var spans []actionSpan
	var x float32
	for i, label := range labels {
		w := float32(len([]rune(label)))*a.theme.Meta*0.6 + 8
		spans = append(spans, actionSpan{label: label, x: x, w: w})
		x += w
		if i < len(labels)-1 {
			x += 10
		}
	}
	return spans
}
