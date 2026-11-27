package ui

import (
	"strings"

	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/theme"
)

func rgb(v uint32) widget.Color { return widget.Hex(v) }

func mixHex(a, b uint32, towardB float32) widget.Color {
	ch := func(shift uint) uint32 {
		av := float32((a >> shift) & 0xff)
		bv := float32((b >> shift) & 0xff)
		v := av + (bv-av)*towardB
		if v < 0 {
			v = 0
		}
		if v > 255 {
			v = 255
		}
		return uint32(v) << shift
	}
	return widget.Hex(ch(16) | ch(8) | ch(0))
}

func soften(th theme.Theme) widget.Color {
	return mixHex(th.Ink, th.Muted, 0.35)
}

func alertMark(kind string) string {
	switch kind {
	case "tip":
		return "✓"
	case "note", "important":
		return "ℹ"
	default:
		return "⚠"
	}
}

func alertColor(th theme.Theme, kind string) widget.Color {
	switch kind {
	case "tip":
		return rgb(th.String)
	case "note", "important":
		return rgb(th.Accent)
	default:
		return rgb(th.Number)
	}
}

func rgba(v uint32) widget.Color { return widget.HexA(v) }

func textWidth(canvas widget.Canvas, text string, size float32, col widget.Color, bold bool, family string) float32 {
	if text == "" {
		return 0
	}
	if family != "" {
		if sd, ok := canvas.(widget.StyledTextDrawer); ok {
			if w := sd.MeasureStyledText(text, widget.TextStyle{
				FontFamily: family, FontSize: size, Bold: bold, Color: col,
			}); textWidthOK(text, w, size) {
				return w
			}
		}
	}
	if w := canvas.MeasureText(text, size, bold); textWidthOK(text, w, size) {
		return w
	}
	return float32(len([]rune(text))) * size * 0.5
}

// textWidthOK keeps a real measurement. A space in Inter is about a quarter
// of an em. The old floor was half an em per rune, so every space, and any
// short word under that floor, was drawn wider than its glyphs.
func textWidthOK(text string, w, size float32) bool {
	if w <= 0 {
		return false
	}
	if strings.TrimSpace(text) == "" {
		return true
	}
	return w >= size*0.08
}

func drawFace(canvas widget.Canvas, text string, rect geometry.Rect, size float32, col widget.Color, bold bool, family string) {
	if family != "" {
		if sd, ok := canvas.(widget.StyledTextDrawer); ok {
			sd.DrawStyledText(text, rect, widget.TextStyle{
				FontFamily: family,
				FontSize:   size,
				Bold:       bold,
				Color:      col,
				Align:      widget.TextAlignLeft,
			})
			return
		}
	}
	canvas.DrawText(text, rect, size, col, bold, widget.TextAlignLeft)
}

func fitRun(canvas widget.Canvas, text string, size float32, col widget.Color, family string, width float32) string {
	if text == "" || width < 8 || textWidth(canvas, text, size, col, false, family) <= width {
		return text
	}
	rs := []rune(text)
	for len(rs) > 1 {
		rs = rs[:len(rs)-1]
		s := string(rs) + "…"
		if textWidth(canvas, s, size, col, false, family) <= width {
			return s
		}
	}
	return "…"
}

func faceOf(th theme.Theme, code bool) string {
	if code {
		return th.CodeFont
	}
	return th.UIFont
}
