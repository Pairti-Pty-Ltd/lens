package ui

import (
	"slices"
	"strings"

	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
)

func (p *Page) tableRuns(spans []doc.Span, header bool) []piece {
	var runs []piece
	for _, sp := range spans {
		pc := piece{text: sp.Text, size: p.theme.Body, family: p.theme.UIFont, color: rgb(p.theme.Ink),
			bold: header || sp.Bold, italic: sp.Italic, strike: sp.Strike, marked: sp.Marked, code: sp.Code, link: sp.Link}
		if pc.code {
			pc.family = p.theme.CodeFont
		}
		if sp.Link != "" {
			pc.color = rgb(p.theme.Accent)
		}
		if len(runs) == 0 {
			if dot, rest := splitBullet(pc.text); dot {
				runs = append(runs, piece{bullet: true})
				pc.text = rest
			}
		}
		if pc.text != "" {
			runs = append(runs, pc)
		}
	}
	return runs
}

// Wrap each styled run at a word boundary, falling back to rune boundaries
// when a single identifier exceeds the width of the column.
func (p *Page) wrapTableRuns(runs []piece, width float32) [][]piece {
	var lines [][]piece
	var line []piece
	var used float32
	flush := func() {
		lines = append(lines, line)
		line = nil
		used = 0
	}
	for _, run := range runs {
		if run.bullet {
			line = append(line, run)
			used += p.pieceWidth(run)
			continue
		}
		text := []rune(run.text)
		for len(text) > 0 {
			if len(line) == 0 {
				for len(text) > 0 && (text[0] == ' ' || text[0] == '\t') {
					text = text[1:]
				}
			}
			if len(text) == 0 {
				break
			}
			end := p.tablePrefix(run, text, width-used)
			if end < len(text) && (end == 0 || text[end] != ' ') {
				found := false
				for i, ch := range slices.Backward(text[:end]) {
					if ch == ' ' || ch == '\t' {
						end = i + 1
						found = true
						break
					}
				}
				if !found && len(line) > 0 {
					flush()
					continue
				}
			}
			end = max(1, end)
			pc := run
			pc.text = string(text[:end])
			text = text[end:]
			if len(text) > 0 {
				pc.text = strings.TrimRight(pc.text, " \t")
			}
			if pc.text != "" {
				line = append(line, pc)
				used += p.pieceWidth(pc)
			}
			if len(text) > 0 && len(line) > 0 {
				flush()
			}
		}
	}
	if len(line) > 0 {
		flush()
	}
	return lines
}

// tablePrefix bounds the measurement cost of wrapping a long identifier.
func (p *Page) tablePrefix(run piece, text []rune, width float32) int {
	fits := func(n int) bool {
		return p.advText(string(text[:n]), run.size, run.bold, run.family) <= width
	}
	lo, hi := 0, min(len(text), 32)
	for hi < len(text) && fits(hi) {
		lo = hi
		hi = min(len(text), hi*2)
	}
	for lo < hi {
		mid := lo + (hi-lo+1)/2
		if fits(mid) {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}

// Each paragraph flows independently inside the single cell. Keep the
// document-relative run positions when moving the lines into the table card.
func (p *Page) layoutGrid(b doc.Block, width float32) []rendered {
	var lines [][]piece
	for _, para := range b.GridParagraphs {
		if len(lines) > 0 {
			lines = append(lines, nil)
		}
		rows := p.layoutFlow(para, para.Spans, width-2*cellPad, p.theme.Body, 0, rgb(p.theme.Ink), strings.HasPrefix(para.Kind, "h"), widget.Color{}, p.theme.UIFont)
		if p.buf != nil {
			p.mapFlowSource(rows, para)
		}
		for _, row := range rows {
			lines = append(lines, row.pieces)
		}
	}
	row := rendered{gridCell: true, cols: []float32{width}, cellLines: [][][]piece{lines}, before: 10, num: b.Line, src: b.Line, tableEnd: true}
	row.tableH = p.rowHeight(row)
	return []rendered{row}
}
