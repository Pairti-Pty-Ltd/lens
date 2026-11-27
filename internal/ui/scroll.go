package ui

import (
	"github.com/gogpu/ui/core/scrollview"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/primitives"
	"github.com/gogpu/ui/widget"
)

// ScrollGutter is the empty strip at the right of a document. The toolkit
// paints the pill over the viewport, so the text ends before that strip.
const ScrollGutter float32 = 20

// ScrollInset keeps child clear of the vertical scrollbar.
func ScrollInset(child widget.Widget) widget.Widget {
	return primitives.Box(child).PaddingRight(ScrollGutter)
}

// ScrollPainter draws the document scrollbar as a pill on a clear track.
// The toolkit track is a light bar, and on the dark page it hides the pill.
type ScrollPainter struct{}

// PaintScrollbar draws the thumb only. The track stays transparent.
func (ScrollPainter) PaintScrollbar(canvas widget.Canvas, ps scrollview.PaintState) {
	if ps.Bounds.IsEmpty() {
		return
	}
	if ps.VScrollVisible {
		paintScrollPill(canvas, ps.VThumbRect, ps)
	}
	if ps.HScrollVisible {
		paintScrollPill(canvas, ps.HThumbRect, ps)
	}
}

func paintScrollPill(canvas widget.Canvas, thumb geometry.Rect, ps scrollview.PaintState) {
	if thumb.IsEmpty() {
		return
	}
	canvas.DrawRoundRect(thumb, scrollPillColor(ps), 4)
}

// Same pill the toolkit draws. Only the track changes.
func scrollPillColor(ps scrollview.PaintState) widget.Color {
	if ps.Dragging {
		return widget.RGBA(0.4, 0.4, 0.4, 0.9)
	}
	if ps.Hovered {
		return widget.RGBA(0.5, 0.5, 0.5, 0.8)
	}
	return widget.RGBA(0.6, 0.6, 0.6, 0.7)
}

var _ scrollview.Painter = ScrollPainter{}
