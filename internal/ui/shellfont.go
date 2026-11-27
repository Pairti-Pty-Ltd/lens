package ui

import (
	"os"
	"path/filepath"

	"github.com/gogpu/gg/text"
	"github.com/gogpu/ui/plugin"
)

// shellFamily is the mono face the shell cells are drawn with.
// Empty means the code font, which falls back to a proportional face.
var shellFamily string

// ShellFace is the registered mono family, or Courier New when none loaded.
func ShellFace() string {
	if shellFamily != "" {
		return shellFamily
	}
	return "Courier New"
}

// RegisterShellFace loads the first mono font it can parse.
// JetBrains Mono is preferred. Courier New and SF Mono are the system fallbacks.
// A nerd font is not used: its outlines fill in on this renderer.
func RegisterShellFace() (string, error) {
	home, _ := os.UserHomeDir()
	candidates := []struct{ name, path string }{
		{"JetBrains Mono", filepath.Join(home, "Library", "Fonts", "JetBrainsMono-Regular.ttf")},
		{"Courier New", "/System/Library/Fonts/Supplemental/Courier New.ttf"},
		{"SF Mono", "/System/Library/Fonts/SFNSMono.ttf"},
	}
	var last error
	ctx := plugin.NewDefaultPluginContext()
	for _, c := range candidates {
		b, err := os.ReadFile(c.path)
		if err != nil {
			last = err
			continue
		}
		src, err := text.NewFontSource(b)
		if err != nil || !monoFace(src) {
			last = err
			continue
		}
		if err := ctx.Assets.LoadFont(c.name, b); err != nil {
			last = err
			continue
		}
		shellFamily = c.name
		return c.name, nil
	}
	if last == nil {
		last = os.ErrNotExist
	}
	return "", last
}

func monoFace(src *text.FontSource) bool {
	face := src.Face(14)
	iw, mw := advanceOf(face, "i"), advanceOf(face, "M")
	if iw < 1 || mw < 1 {
		return false
	}
	d := iw - mw
	if d < 0 {
		d = -d
	}
	return d < 0.5
}

func advanceOf(face text.Face, s string) float64 {
	var w float64
	for g := range face.Glyphs(s) {
		w = g.Advance
	}
	return w
}
