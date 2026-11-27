//go:build js && wasm

package main

import (
	"fmt"
	"image"
	"image/draw"
	"math"
	"strconv"
	"strings"
	"syscall/js"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"
	"github.com/gogpu/ui/geometry"
	"github.com/gogpu/ui/widget"
)

// GoGPU owns the focusable input surface. Canvas2D paints the same logical
// coordinates on a pointer-transparent surface, using the browser rasterizer.
func runBrowser(host *gogpu.App, a *app.App) error {
	surface := js.Global().Get("document").Call("getElementById", "paint")
	c := &browserCanvas{ctx: surface.Call("getContext", "2d")}
	win := a.Window()
	win.SetRenderMode(app.RenderModeHostManaged)
	widget.RegisterClipboardProvider(host)
	// Draw boundary children directly. Desktop scene recording rasterizes the
	// entire boundary in software when running under WebAssembly.
	widget.RegisterSceneRecorder(nil)
	host.OnDraw(func(_ *gogpu.Context) {
		a.Frame()
		size := win.WindowSize()
		if size.Width <= 0 || size.Height <= 0 {
			return
		}
		scale := js.Global().Get("devicePixelRatio").Float()
		w := int(math.Round(float64(size.Width) * scale))
		h := int(math.Round(float64(size.Height) * scale))
		if surface.Get("width").Int() != w {
			surface.Set("width", w)
		}
		if surface.Get("height").Int() != h {
			surface.Set("height", h)
		}
		c.ctx.Call("setTransform", float64(w)/float64(size.Width), 0, 0, float64(h)/float64(size.Height), 0, 0)
		c.clips = []geometry.Rect{geometry.FromPointSize(geometry.Point{}, size)}
		c.Clear(widget.ColorBlack)
		win.ClearAnimationFrame()
		win.ClearDirtyBoundaries()
		win.DrawTo(c)
	})
	host.OnClose(win.Close)
	return host.Run()
}

type browserCanvas struct {
	ctx     js.Value
	offsets []geometry.Point
	clips   []geometry.Rect // screen coordinates
}

func cssColor(c widget.Color) string {
	r, g, b, a := c.RGBA8()
	return fmt.Sprintf("#%02x%02x%02x%02x", r, g, b, a)
}
func (c *browserCanvas) Clear(col widget.Color) { c.DrawRect(c.clips[0], col) }
func (c *browserCanvas) DrawRect(r geometry.Rect, col widget.Color) {
	c.ctx.Set("fillStyle", cssColor(col))
	c.ctx.Call("fillRect", r.Min.X, r.Min.Y, r.Width(), r.Height())
}
func (c *browserCanvas) FillRectDirect(r geometry.Rect, col widget.Color) { c.DrawRect(r, col) }
func (c *browserCanvas) stroke(col widget.Color, w float32) {
	c.ctx.Set("strokeStyle", cssColor(col))
	c.ctx.Set("lineWidth", w)
	c.ctx.Call("stroke")
}
func (c *browserCanvas) fill(col widget.Color) {
	c.ctx.Set("fillStyle", cssColor(col))
	c.ctx.Call("fill")
}
func (c *browserCanvas) rectPath(r geometry.Rect, radius float32) {
	c.ctx.Call("beginPath")
	c.ctx.Call("roundRect", r.Min.X, r.Min.Y, r.Width(), r.Height(), max(0, radius))
}
func (c *browserCanvas) StrokeRect(r geometry.Rect, col widget.Color, w float32) {
	c.StrokeRoundRect(r, col, 0, w)
}
func (c *browserCanvas) DrawRoundRect(r geometry.Rect, col widget.Color, radius float32) {
	c.rectPath(r, radius)
	c.fill(col)
}
func (c *browserCanvas) StrokeRoundRect(r geometry.Rect, col widget.Color, radius, w float32) {
	c.rectPath(r, radius)
	c.stroke(col, w)
}
func (c *browserCanvas) arc(p geometry.Point, r float32, start, sweep float64) {
	c.ctx.Call("beginPath")
	c.ctx.Call("arc", p.X, p.Y, max(0, r), start, start+sweep, sweep < 0)
}
func (c *browserCanvas) DrawCircle(p geometry.Point, r float32, col widget.Color) {
	c.arc(p, r, 0, 2*math.Pi)
	c.fill(col)
}
func (c *browserCanvas) StrokeCircle(p geometry.Point, r float32, col widget.Color, w float32) {
	c.StrokeArc(p, r, 0, 2*math.Pi, col, w)
}
func (c *browserCanvas) StrokeArc(p geometry.Point, r float32, start, sweep float64, col widget.Color, w float32) {
	c.arc(p, r, start, sweep)
	c.stroke(col, w)
}
func (c *browserCanvas) DrawLine(a, b geometry.Point, col widget.Color, w float32) {
	c.ctx.Call("beginPath")
	c.ctx.Call("moveTo", a.X, a.Y)
	c.ctx.Call("lineTo", b.X, b.Y)
	c.stroke(col, w)
}
func (c *browserCanvas) setFont(s widget.TextStyle) {
	family := strconv.Quote(s.FontFamily) + ", system-ui"
	if strings.HasPrefix(s.FontFamily, "Go Mono") {
		family = "monospace"
	} else if s.FontFamily == "" {
		family = "system-ui"
	}
	weight, slant := "", ""
	if s.Bold {
		weight = "bold "
	}
	if s.Italic {
		slant = "italic "
	}
	c.ctx.Set("font", slant+weight+strconv.FormatFloat(float64(s.FontSize), 'f', -1, 32)+"px "+family)
}
func (c *browserCanvas) MeasureStyledText(t string, s widget.TextStyle) float32 {
	c.setFont(s)
	return float32(c.ctx.Call("measureText", t).Get("width").Float())
}
func (c *browserCanvas) MeasureText(t string, size float32, bold bool) float32 {
	return c.MeasureStyledText(t, widget.TextStyle{FontSize: size, Bold: bold})
}
func (c *browserCanvas) DrawStyledText(t string, r geometry.Rect, s widget.TextStyle) {
	c.setFont(s)
	c.ctx.Set("fillStyle", cssColor(s.Color))
	c.ctx.Set("textAlign", strings.ToLower(s.Align.String()))
	c.ctx.Set("textBaseline", "middle")
	c.ctx.Call("fillText", t, float64(r.Min.X)+s.Align.Float64()*float64(r.Width()), r.Center().Y)
}
func (c *browserCanvas) DrawText(t string, r geometry.Rect, size float32, col widget.Color, bold bool, align widget.TextAlign) {
	c.DrawStyledText(t, r, widget.TextStyle{FontSize: size, Color: col, Bold: bold, Align: align})
}
func (c *browserCanvas) DrawImage(img image.Image, at geometry.Point) {
	if img == nil || img.Bounds().Empty() {
		return
	}
	b := img.Bounds()
	pixels := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(pixels, pixels.Bounds(), img, b.Min, draw.Src)
	off := js.Global().Get("OffscreenCanvas").New(b.Dx(), b.Dy())
	ctx := off.Call("getContext", "2d")
	data := ctx.Call("createImageData", b.Dx(), b.Dy())
	js.CopyBytesToJS(data.Get("data"), pixels.Pix)
	ctx.Call("putImageData", data, 0, 0)
	c.ctx.Call("drawImage", off, at.X, at.Y)
}
func (c *browserCanvas) PushClip(r geometry.Rect) { c.PushClipRoundRect(r, 0) }
func (c *browserCanvas) PushClipRoundRect(r geometry.Rect, radius float32) {
	c.ctx.Call("save")
	c.rectPath(r, radius)
	c.ctx.Call("clip")
	c.clips = append(c.clips, c.clips[len(c.clips)-1].Intersection(r.Translate(c.TransformOffset())))
}
func (c *browserCanvas) PopClip() { c.ctx.Call("restore"); c.clips = c.clips[:len(c.clips)-1] }
func (c *browserCanvas) ClipBounds() geometry.Rect {
	return c.clips[len(c.clips)-1]
}
func (c *browserCanvas) PushTransform(p geometry.Point) {
	c.ctx.Call("save")
	c.ctx.Call("translate", p.X, p.Y)
	c.offsets = append(c.offsets, c.TransformOffset().Add(p))
}
func (c *browserCanvas) PopTransform() {
	c.ctx.Call("restore")
	c.offsets = c.offsets[:len(c.offsets)-1]
}
func (c *browserCanvas) TransformOffset() geometry.Point {
	if len(c.offsets) > 0 {
		return c.offsets[len(c.offsets)-1]
	}
	return geometry.Point{}
}
func (c *browserCanvas) ScreenOriginBase() geometry.Point { return geometry.Point{} }

// Scene recording is disabled at startup; every widget draws directly.
func (c *browserCanvas) ReplayScene(widget.SceneCache) {}
