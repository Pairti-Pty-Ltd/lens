package ui

import (
	"strconv"
	"strings"
)

// Heading is one outline row. Number is the dotted index, such as 1.3.1.
type Heading struct {
	Number string
	Text   string
	Level  int
	Line   int
}

// Ref is a link or an image in the open document.
type Ref struct {
	Label string
	URL   string
	Line  int
}

// Outline numbers the headings in reading order.
func (p *Page) Outline() []Heading {
	var stack []int
	var out []Heading
	for _, b := range p.blocks {
		level := headingLevel(b.Kind)
		if level == 0 {
			continue
		}
		for len(stack) < level {
			stack = append(stack, 0)
		}
		stack = stack[:level]
		stack[level-1]++
		text := strings.TrimSpace(b.Text)
		if text == "" {
			text = "Heading"
		}
		out = append(out, Heading{
			Number: headingNumber(stack),
			Text:   text,
			Level:  level,
			Line:   b.Line,
		})
	}
	return out
}

// Links lists the destinations in the open document.
func (p *Page) Links() []Ref {
	var out []Ref
	for _, b := range p.blocks {
		for _, sp := range b.Spans {
			if sp.Image || sp.Link == "" {
				continue
			}
			label := strings.TrimSpace(sp.Text)
			if label == "" {
				label = imageName(sp.Link)
			}
			out = append(out, Ref{Label: label, URL: sp.Link, Line: b.Line})
		}
	}
	return out
}

// Pictures lists the images in the open document.
func (p *Page) Pictures() []Ref {
	var out []Ref
	for _, b := range p.blocks {
		for _, sp := range b.Spans {
			if !sp.Image {
				continue
			}
			out = append(out, Ref{Label: imageLabel(sp.Text, sp.Link), URL: sp.Link, Line: b.Line})
		}
	}
	return out
}

// HeadingLine is the source line of a heading with this text, or 0.
func (p *Page) HeadingLine(text string) int {
	want := strings.ToLower(strings.TrimSpace(text))
	if want == "" {
		return 0
	}
	for _, h := range p.Outline() {
		if strings.ToLower(strings.TrimSpace(h.Text)) == want {
			return h.Line
		}
	}
	return 0
}

// LineY is the scroll offset of a source line. Layout must have run.
// The first line and an unknown line both return 0.
func (p *Page) LineY(line int) float32 {
	if line <= 1 {
		return 0
	}
	var y float32
	for _, ln := range p.lines {
		if ln.num >= line || ln.src >= line {
			return y
		}
		y += ln.before + p.rowHeight(ln)
	}
	return y
}

func headingLevel(kind string) int {
	if len(kind) < 2 || kind[0] != 'h' {
		return 0
	}
	n := int(kind[1] - '0')
	if n < 1 || n > 6 {
		return 0
	}
	return n
}

func headingNumber(stack []int) string {
	var b strings.Builder
	for i, n := range stack {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(strconv.Itoa(n))
	}
	return b.String()
}
