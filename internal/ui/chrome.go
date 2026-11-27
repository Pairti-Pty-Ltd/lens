package ui

import (
	"github.com/gogpu/ui/core/button"
	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// FieldPainter draws a flat dark field. The toolkit default is a white box.
func FieldPainter(t theme.Theme) textfield.Painter { return fieldPainter{t} }

// QuietButton draws a text button. emphasis is the header action (Edit, Save, Done).
func QuietButton(t theme.Theme, emphasis bool) button.Painter {
	return quietButton{theme: t, emphasis: emphasis}
}

type fieldPainter struct{ theme theme.Theme }

func (fieldPainter) ContentPadding() (float32, float32) { return 8, 3 }
func (p fieldPainter) TextFieldFontSize() float32       { return p.theme.Meta }
func (fieldPainter) TextFieldCursorWidth() float32      { return 1 }
func (fieldPainter) TextFieldCornerRadius() float32     { return 6 }

func (p fieldPainter) PaintTextField(canvas widget.Canvas, st *textfield.PaintState) {
	if st.Bounds.IsEmpty() {
		return
	}
	th := p.theme
	bg := rgb(th.CodeFill)
	border := rgb(th.Rule)
	stroke := float32(1)
	if st.Focused {
		border = rgb(th.Accent)
	}
	canvas.DrawRoundRect(st.Bounds, bg, 6)
	canvas.StrokeRoundRect(st.Bounds, border, 6, stroke)
	canvas.PushClip(st.ContentRect)
	defer canvas.PopClip()
	size := st.FontSize
	if size <= 0 {
		size = th.Meta
	}
	if st.DisplayText == "" && !st.Focused {
		canvas.DrawText(st.Placeholder, st.ContentRect, size, rgb(th.Muted), false, widget.TextAlignLeft)
		return
	}
	if st.ShowSelection {
		canvas.DrawRect(st.SelectionRect, rgba(th.Selection))
	}
	canvas.DrawText(st.DisplayText, st.TextRect, size, rgb(th.Ink), false, widget.TextAlignLeft)
	if st.ShowCursor {
		top := geometry.Pt(st.CursorRect.Min.X, st.CursorRect.Min.Y)
		bot := geometry.Pt(st.CursorRect.Min.X, st.CursorRect.Max.Y)
		canvas.DrawLine(top, bot, rgb(th.Accent), st.CursorRect.Width())
	}
}

type quietButton struct {
	theme    theme.Theme
	emphasis bool
}

func (quietButton) ButtonHeight(button.Size) float32 { return 26 }
func (quietButton) ButtonPadding(button.Size) (float32, float32) {
	return 10, 4
}
func (p quietButton) ButtonFontSize(button.Size) float32 { return p.theme.Meta }
func (quietButton) ButtonRadius() float32                { return 4 }

func (p quietButton) PaintButton(canvas widget.Canvas, st button.PaintState) {
	if st.Bounds.IsEmpty() {
		return
	}
	fg := rgb(p.theme.Muted)
	if p.emphasis {
		fg = rgb(p.theme.Ink)
	}
	if st.Hovered || st.Pressed {
		canvas.DrawRoundRect(st.Bounds, rgba(p.theme.Selection), 4)
		fg = rgb(p.theme.Ink)
	}
	if st.Focused {
		canvas.StrokeRoundRect(st.Bounds, rgb(p.theme.Accent), 4, 1)
	}
	canvas.DrawText(st.Text, st.Bounds, p.theme.Meta, fg, false, widget.TextAlignCenter)
}

var (
	_ textfield.Painter       = fieldPainter{}
	_ textfield.LayoutMetrics = fieldPainter{}
	_ button.Painter          = quietButton{}
	_ button.LayoutMetrics    = quietButton{}
)
