package ui

import "github.com/gogpu/ui/uitest"

type glyph struct {
	text string
	bold bool
	y    float32
}

func glyphs(c *uitest.MockCanvas) []glyph {
	var out []glyph
	for _, t := range c.Texts {
		out = append(out, glyph{t.Text, t.Bold, t.Bounds.Min.Y})
	}
	for _, t := range c.StyledTexts {
		out = append(out, glyph{t.Text, t.Style.Bold, t.Bounds.Min.Y})
	}
	return out
}
