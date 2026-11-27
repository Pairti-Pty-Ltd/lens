package ui

import (
	"strings"
	"unicode"

	"pairti-lense/internal/gitlog"
)

// Wrap across syntax and word-highlight boundaries. The run index retains each
// token's style while tabs are expanded against the logical source column.
func wrapDiffLine(line *gitlog.DiffLine, columns int) [][]gitlog.DiffRun {
	if line == nil {
		return nil
	}
	runs := line.Runs
	if len(runs) == 0 {
		runs = []gitlog.DiffRun{{Text: line.Text}}
	}
	type glyph struct {
		char rune
		run  int
	}
	var chars []glyph
	for i, run := range runs {
		for _, ch := range run.Text {
			if ch == '\t' {
				for n := 4 - len(chars)%4; n > 0; n-- {
					chars = append(chars, glyph{' ', i})
				}
			} else {
				chars = append(chars, glyph{ch, i})
			}
		}
	}
	if len(chars) == 0 {
		return [][]gitlog.DiffRun{nil}
	}
	columns = max(1, columns)
	var rows [][]gitlog.DiffRun
	for start := 0; start < len(chars); {
		end := min(start+columns, len(chars))
		if end < len(chars) {
			for i := end; i > start; i-- {
				if unicode.IsSpace(chars[i-1].char) {
					end = i
					break
				}
			}
		}
		var pieces []gitlog.DiffRun
		for i := start; i < end; {
			style := chars[i].run
			piece := runs[style]
			var buf strings.Builder
			for i < end && chars[i].run == style {
				buf.WriteRune(chars[i].char)
				i++
			}
			piece.Text = buf.String()
			pieces = append(pieces, piece)
		}
		rows = append(rows, pieces)
		start = end
	}
	return rows
}
