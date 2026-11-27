//go:build ignore

// Command makeicon generates the Lens launcher icon from vector geometry.
// Run go run assets/makeicon.go from the repository root.
package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"os"

	"golang.org/x/image/vector"
)

func main() {
	const size = 1024
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	paper := color.RGBA{0xed, 0xf3, 0xfd, 0xff}
	ink := color.RGBA{0x17, 0x2c, 0x45, 0xff}
	blue := color.RGBA{0x72, 0xc9, 0xff, 0xff}
	r := vector.NewRasterizer(size, size)
	paint := func(c color.RGBA) {
		r.Draw(dst, dst.Bounds(), image.NewUniform(c), image.Point{})
		r.Reset(size, size)
	}
	// Rounded dark tile.
	r.MoveTo(256, 72)
	r.LineTo(768, 72)
	r.QuadTo(952, 72, 952, 256)
	r.LineTo(952, 768)
	r.QuadTo(952, 952, 768, 952)
	r.LineTo(256, 952)
	r.QuadTo(72, 952, 72, 768)
	r.LineTo(72, 256)
	r.QuadTo(72, 72, 256, 72)
	r.ClosePath()
	paint(ink)

	// Folded sheet with two lines of source text.
	r.MoveTo(245, 196)
	r.LineTo(565, 196)
	r.LineTo(689, 320)
	r.LineTo(689, 773)
	r.LineTo(245, 773)
	r.ClosePath()
	paint(paper)
	r.MoveTo(565, 196)
	r.LineTo(689, 320)
	r.LineTo(565, 320)
	r.ClosePath()
	paint(blue)
	for _, line := range []image.Rectangle{image.Rect(313, 359, 542, 387), image.Rect(313, 429, 472, 457)} {
		draw.Draw(dst, line, image.NewUniform(ink), image.Point{}, draw.Src)
	}

	// Magnifier handle and lens. Cubic arcs preserve smooth edges at icon sizes.
	r.MoveTo(679, 673)
	r.LineTo(846, 840)
	r.QuadTo(868, 862, 846, 884)
	r.QuadTo(824, 906, 802, 884)
	r.LineTo(635, 717)
	r.ClosePath()
	paint(blue)
	circle := func(x, y, radius float32, c color.RGBA) {
		k := radius * 0.55228475
		r.MoveTo(x+radius, y)
		r.CubeTo(x+radius, y+k, x+k, y+radius, x, y+radius)
		r.CubeTo(x-k, y+radius, x-radius, y+k, x-radius, y)
		r.CubeTo(x-radius, y-k, x-k, y-radius, x, y-radius)
		r.CubeTo(x+k, y-radius, x+radius, y-k, x+radius, y)
		r.ClosePath()
		paint(c)
	}
	circle(605, 610, 157, ink)
	circle(605, 610, 142, blue)
	circle(605, 610, 99, ink)

	f, err := os.Create("assets/AppIcon.png")
	if err != nil {
		log.Fatal(err)
	}
	if err := png.Encode(f, dst); err != nil {
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		log.Fatal(err)
	}
}
