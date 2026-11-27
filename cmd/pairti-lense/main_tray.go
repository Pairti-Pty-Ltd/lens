//go:build !js

package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"runtime"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/systray"
)

// startTray puts Pairti Lens in the menu bar. The window loop already pumps
// the platform events, so this does not call SystemTray.Run. That call would
// take the main thread and, on macOS, switch the app to an accessory with no
// Dock icon.
func startTray(app *gogpu.App, onMCP, onQuit func()) *systray.SystemTray {
	menu := systray.NewMenu()
	menu.Add("Show Lens", func() { showLens(app) })
	menu.Add("MCP connection", func() {
		if onMCP != nil {
			onMCP()
		}
	})
	menu.AddSeparator()
	menu.Add("Quit", func() {
		if onQuit != nil {
			onQuit()
		}
	})

	icon := lensIcon()
	tray := systray.New()
	tray.SetAppName("Lens").SetIcon(icon).SetTooltip("Lens").SetMenu(menu)
	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(icon)
	}
	tray.OnClick(func() { showLens(app) })
	tray.Show()
	return tray
}

func showLens(app *gogpu.App) {
	if app == nil {
		return
	}
	w := app.PrimaryWindow()
	if w == nil {
		return
	}
	w.Show()
}

// lensIcon is a 22px monochrome lens. On macOS only the alpha is used.
func lensIcon() []byte {
	const s = 22
	img := image.NewNRGBA(image.Rect(0, 0, s, s))
	ink := color.NRGBA{A: 255}
	cx, cy, r := 8.5, 8.5, 6.2
	for y := 0; y < s; y++ {
		for x := 0; x < s; x++ {
			d := math.Hypot(float64(x)+0.5-cx, float64(y)+0.5-cy)
			if d <= r && d >= r-1.8 {
				img.SetNRGBA(x, y, ink)
			}
		}
	}
	for i := 0; i < 6; i++ {
		x, y := 13+i, 13+i
		img.SetNRGBA(x, y, ink)
		if x+1 < s {
			img.SetNRGBA(x+1, y, ink)
		}
		if y+1 < s {
			img.SetNRGBA(x, y+1, ink)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}
