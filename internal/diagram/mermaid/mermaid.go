// Package mermaid renders a ```mermaid fence in this process.
// The renderer is mermaid-go-renderer (mmdg), a pure Go port of Mermaid
// 12.2.0. The diagram source is not sent to a network service.
package mermaid

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"strings"
	"time"

	mmdg "github.com/bvolpato/mermaid-go-renderer"
)

// Renderer is the mermaid fence.
type Renderer struct{}

// New is the in-process renderer.
func New() Renderer { return Renderer{} }

// Languages is the fence info string this renderer claims.
func (Renderer) Languages() []string { return []string{"mermaid"} }

// Render draws source for the dark window. Layout stops when ctx is done.
func (Renderer) Render(ctx context.Context, source string) (image.Image, error) {
	data, err := renderPNG(ctx, source, screenOpts)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return img, nil
}

// PaperPNG draws source for a white page. Word and PDF use it. The window
// keeps Render, whose ink is for the dark canvas and would vanish on paper.
func PaperPNG(ctx context.Context, source string) ([]byte, error) {
	return renderPNG(ctx, source, paperOpts)
}

func renderPNG(ctx context.Context, source string, opts []mmdg.Option) ([]byte, error) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, fmt.Errorf("empty diagram")
	}
	if len(source) > 256<<10 {
		return nil, fmt.Errorf("diagram is over 256KiB")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 20*time.Second)
		defer cancel()
	}
	res, err := mmdg.RenderContext(ctx, source, opts...)
	if err != nil {
		return nil, err
	}
	data, err := mmdg.SVGToPNG(res.SVG, opts...)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("diagram render wrote an empty image")
	}
	return data, nil
}

// renderOpts keep the Pairti ink on a transparent ground, at the column
// width, and at scale 2 so the page can fit the picture without softening
// the labels. The page is the mermaid-cli page, so icon fonts resolve.
// htmlLabels stays off: the rasterizer paints SVG text with the fonts that
// measured the layout.
var screenOpts = []mmdg.Option{
	mmdg.WithPage(mmdg.PageMermaidCLI),
	mmdg.WithConfigJSON([]byte(mermaidConfig)),
	mmdg.WithViewportWidth(960),
	mmdg.WithScale(2),
	mmdg.WithBackground(color.Transparent),
}

// paperOpts is the default Mermaid theme on white. A printed page is white,
// and the window theme's light lines disappear on it.
var paperOpts = []mmdg.Option{
	mmdg.WithPage(mmdg.PageMermaidCLI),
	mmdg.WithConfigJSON([]byte(paperConfig)),
	mmdg.WithViewportWidth(960),
	mmdg.WithScale(2),
	mmdg.WithBackgroundColor("white"),
}

const paperConfig = `{
  "theme": "default",
  "flowchart": {"htmlLabels": false, "useMaxWidth": true, "padding": 16},
  "sequence": {"useMaxWidth": true}
}`

// mermaidConfig keeps SVG text, a light ink on the window ground, and a
// width close to the reading column.
const mermaidConfig = `{
  "theme": "base",
  "themeVariables": {
    "darkMode": true,
    "background": "transparent",
    "fontFamily": "Inter, sans-serif",
    "fontSize": "18px",
    "primaryColor": "#161616",
    "primaryTextColor": "#ffffff",
    "primaryBorderColor": "#2979ff",
    "secondaryColor": "#111111",
    "secondaryTextColor": "#ffffff",
    "secondaryBorderColor": "#3a3a3a",
    "tertiaryColor": "#0a0a0a",
    "tertiaryTextColor": "#ffffff",
    "lineColor": "#d0d0d0",
    "textColor": "#ffffff",
    "mainBkg": "#161616",
    "nodeBorder": "#2979ff",
    "clusterBkg": "#111111",
    "clusterBorder": "#333333",
    "titleColor": "#ffffff",
    "edgeLabelBackground": "#111111",
    "nodeTextColor": "#ffffff"
  },
  "flowchart": {"htmlLabels": false, "useMaxWidth": true, "padding": 16},
  "sequence": {"useMaxWidth": true}
}`
