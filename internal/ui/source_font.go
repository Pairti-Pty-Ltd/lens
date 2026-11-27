package ui

import (
	"math"
	"sync"

	"github.com/gogpu/gg/text"
	"github.com/gogpu/ui/plugin"
	"github.com/gogpu/ui/widget"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/gomonobold"
	"golang.org/x/image/font/gofont/gomonobolditalic"
	"golang.org/x/image/font/gofont/gomonoitalic"
)

// Load the four actual fixed-pitch faces. The toolkit asset loader registers
// each resource at regular weight, so address each face by its own family name.
var loadSourceFont = sync.OnceValues(func() (float32, error) {
	ctx := plugin.NewDefaultPluginContext()
	fonts := map[string][]byte{
		"Go Mono":             gomono.TTF,
		"Go Mono Bold":        gomonobold.TTF,
		"Go Mono Italic":      gomonoitalic.TTF,
		"Go Mono Bold Italic": gomonobolditalic.TTF,
	}
	for name, data := range fonts {
		if err := ctx.Assets.LoadFont(name, data); err != nil {
			return 0.6, err
		}
	}
	src, err := text.NewFontSource(gomono.TTF)
	if err != nil {
		return 0.6, err
	}
	return float32(advanceOf(src.Face(100), "M") / 100), nil
})

// A configured code face can be used if its rendered variants share a cell
// advance. Unavailable font families resolve to proportional Inter in GoGPU.
func (e *Editor) resolveSourceFace(canvas widget.Canvas) {
	if !e.checkFace {
		return
	}
	e.checkFace = false
	family, width := measuredSourceFont(canvas, e.theme.RawFont, e.theme.Code)
	if width <= 0 || e.theme.Code <= 0 {
		return
	}
	anchor := e.Passage()
	e.customFace = family
	e.cellRatio = width / e.theme.Code
	e.restoreRows(anchor, e.visualRows())
	e.clampScroll()
	e.MarkNeedsLayout()
}

// measuredSourceFont returns a verified custom family or the bundled Go Mono,
// together with the rendered advance of one source cell at this size.
func measuredSourceFont(canvas widget.Canvas, custom string, size float32) (string, float32) {
	styled, ok := canvas.(widget.StyledTextDrawer)
	if !ok {
		return "", 0
	}
	base := widget.TextStyle{FontFamily: custom, FontSize: size}
	if custom != "" {
		width := styled.MeasureStyledText("MMMMMMMM", base)
		if width > 0 && fixedPitch(styled, base, width) {
			return custom, width / 8
		}
	}
	base.FontFamily = sourceFamily("", false, false)
	return "", styled.MeasureStyledText("MMMMMMMM", base) / 8
}

func fixedPitch(styled widget.StyledTextDrawer, base widget.TextStyle, width float32) bool {
	for _, bold := range []bool{false, true} {
		for _, italic := range []bool{false, true} {
			style := base
			style.Bold, style.Italic = bold, italic
			for _, sample := range []string{"iiiiiiii", "MMMMMMMM"} {
				if math.Abs(float64(styled.MeasureStyledText(sample, style)-width)) > 0.25 {
					return false
				}
			}
		}
	}
	return true
}

func (e *Editor) sourceFamily(bold, italic bool) string {
	return sourceFamily(e.customFace, bold, italic)
}

func sourceFamily(custom string, bold, italic bool) string {
	if custom != "" {
		return custom
	}
	name := "Go Mono"
	if bold {
		name += " Bold"
	}
	if italic {
		name += " Italic"
	}
	return name
}
