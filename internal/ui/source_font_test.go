package ui

import (
	"testing"

	"github.com/gogpu/gg"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/plugin"
	"github.com/gogpu/ui/render"
	"github.com/gogpu/ui/widget"
	"golang.org/x/image/font/gofont/gomono"

	"pairti-lense/internal/theme"
)

// GoGPU rounds advances at the current font size. A 100pt measurement
// scaled to 14pt can leave a visible gap after long syntax-highlighted runs.
func TestSourceFontUsesActualGlyphAdvances(t *testing.T) {
	const custom = "Test configured mono"
	if err := plugin.NewDefaultPluginContext().Assets.LoadFont(custom, gomono.TTF); err != nil {
		t.Fatal(err)
	}
	dc := gg.NewContext(650, 80)
	defer dc.Close()
	canvas := render.NewCanvas(dc, 650, 80)
	styled := canvas.(widget.StyledTextDrawer)
	for _, family := range []string{"", custom, "Missing font"} {
		t.Run(family, func(t *testing.T) {
			ed := NewEditor(nil, nil)
			th := theme.Pairti()
			th.RawFont = family
			ed.SetTheme(th)
			ed.SetBounds(geometry.NewRect(0, 0, 650, 80))
			ed.SetBuffer("Its analytics treat the **organisation** as the unit of interest.")
			ed.Draw(nil, canvas)
			if (ed.customFace == custom) != (family == custom) {
				t.Fatalf("configured %q, used %q", family, ed.customFace)
			}
			for _, bold := range []bool{false, true} {
				for _, italic := range []bool{false, true} {
					run := "**organisation**"
					got := styled.MeasureStyledText(run, widget.TextStyle{FontFamily: ed.sourceFamily(bold, italic), FontSize: th.Code, Bold: bold, Italic: italic})
					want := float32(len(run)) * ed.charW()
					if got > want+0.25 || got < want-0.25 {
						t.Fatalf("rendered run width %.2f, editor reserved %.2f", got, want)
					}
				}
			}
		})
	}
}
