package ui

import (
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

// tip is the one hover label for the window. Icon buttons set it from Event.
// Pulse draws it after the tree so a toolbar cannot clip it.
type tip struct {
	on     bool
	text   string
	anchor geometry.Rect
	theme  theme.Theme
}

var live tip

func showTip(text string, anchor geometry.Rect, th theme.Theme) bool {
	if text == "" {
		return clearTip()
	}
	if live.on && live.text == text && live.anchor == anchor {
		return false
	}
	live = tip{on: true, text: text, anchor: anchor, theme: th}
	return true
}

func clearTip() bool {
	if !live.on {
		return false
	}
	live = tip{}
	return true
}

func tipText() string {
	if !live.on {
		return ""
	}
	return live.text
}

func offerTip(ctx widget.Context, th theme.Theme, text string, anchor geometry.Rect) {
	changed := showTip(text, anchor, th)
	if changed && ctx != nil {
		ctx.Invalidate()
	}
}

// screenOrigin is the widget's top-left in window coordinates.
// Before the first draw, the parent-space bounds stand in for it.
func screenOrigin(w interface {
	IsScreenOriginValid() bool
	ScreenBounds() geometry.Rect
	Bounds() geometry.Rect
}) geometry.Point {
	if w.IsScreenOriginValid() {
		return w.ScreenBounds().Min
	}
	return w.Bounds().Min
}

// hoverPoint is the pointer in the widget's own coordinates.
// A move arrives in parent space. Enter and leave arrive in window space.
func hoverPoint(w interface {
	IsScreenOriginValid() bool
	ScreenBounds() geometry.Rect
	Bounds() geometry.Rect
}, me *event.MouseEvent) (geometry.Point, bool) {
	switch me.MouseType {
	case event.MouseEnter:
		return me.Position.Sub(screenOrigin(w)), true
	case event.MouseMove:
		if !w.Bounds().Contains(me.Position) {
			return geometry.Point{}, false
		}
		return me.Position.Sub(w.Bounds().Min), true
	default:
		return geometry.Point{}, false
	}
}

func drawTip(canvas widget.Canvas, window geometry.Rect) {
	if canvas == nil || !live.on || live.text == "" {
		return
	}
	th := live.theme
	size := th.Meta
	if size < 1 {
		size = 13
	}
	ink := rgb(th.Ink)
	tw := textWidth(canvas, live.text, size, ink, false, th.UIFont)
	w := tw + 16
	h := size*1.2 + 8
	x := live.anchor.Min.X
	y := live.anchor.Max.Y + 4
	if window.Width() > 0 && window.Height() > 0 {
		if x+w > window.Max.X-4 {
			x = window.Max.X - 4 - w
		}
		if x < window.Min.X+4 {
			x = window.Min.X + 4
		}
		if y+h > window.Max.Y-4 {
			y = live.anchor.Min.Y - 4 - h
		}
		if y < window.Min.Y+4 {
			y = window.Min.Y + 4
		}
	}
	rect := geometry.NewRect(x, y, w, h)
	canvas.DrawRoundRect(rect, rgb(th.Surface), 4)
	canvas.StrokeRoundRect(rect, rgb(th.Muted), 4, 1)
	drawFace(canvas, live.text, geometry.NewRect(x+8, y+4, tw+2, size*1.2), size, ink, false, th.UIFont)
}
