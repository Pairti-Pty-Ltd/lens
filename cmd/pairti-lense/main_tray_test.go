package main

import (
	"bytes"
	"image/png"
	"testing"
)

func TestLensIconIsASmallMark(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(lensIcon()))
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	if b.Dx() != 22 || b.Dy() != 22 {
		t.Fatalf("size %v", b)
	}
	n := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				n++
			}
		}
	}
	if n < 20 || n > 180 {
		t.Fatalf("opaque pixels %d", n)
	}
}

func TestShowLensWithNoWindow(t *testing.T) {
	showLens(nil)
}
