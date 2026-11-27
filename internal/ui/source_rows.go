package ui

import (
	"sort"
	"strings"
	"unicode"

	"pairti-lense/internal/doc"
)

// sourceRow maps one visual row back to the original rune positions. Byte
// slices of syntax spans can end at a wrap without changing the buffer.
type sourceRow struct {
	line, col, start, length int
	text                     string
	spans                    []doc.Span
}

func (e *Editor) sourceRows(width float32) []sourceRow {
	if width <= 0 {
		width = 400
	}
	cols := max(1, int((width-e.gutterWidth()-e.charW())/e.charW()))
	wrap := e.file == "" || doc.IsMarkdown(e.file)
	if !wrap {
		cols = 0
	}
	text := e.buf.Text()
	if e.rows != nil && e.cols == cols && e.rowFile == e.file && e.rowText == text {
		return e.rows
	}
	// Opening a sidebar or resizing the window only changes wrapping.
	if e.rowFile != e.file || e.rowText != text {
		switch {
		case wrap:
			e.lineColors = doc.MarkdownLines(text)
		case doc.IsSource(e.file):
			e.lineColors = doc.TokenLines(e.file, "", text)
		default:
			e.lineColors = nil
		}
	}
	e.cols, e.rowFile, e.rowText = cols, e.file, text
	lines := strings.Split(text, "\n")
	colors := e.lineColors
	e.rows = e.rows[:0]
	start := 0
	for line, s := range lines {
		runes := []rune(s)
		var spans []doc.Span
		if line < len(colors) && spansCover(colors[line], s) {
			spans = colors[line]
		}
		spanByte := 0
		for col := 0; ; {
			end := len(runes)
			if cols > 0 && end-col > cols {
				end = col + cols
				for j := end; j > col; j-- {
					if unicode.IsSpace(runes[j-1]) {
						end = j
						break
					}
				}
			}
			row := sourceRow{line: line, col: col, start: start + col, length: end - col, text: string(runes[col:end])}
			left := len(row.text)
			for len(spans) > 0 && left > 0 {
				sp := spans[0]
				n := min(left, len(sp.Text)-spanByte)
				sp.Text = sp.Text[spanByte : spanByte+n]
				if n > 0 {
					row.spans = append(row.spans, sp)
				}
				left -= n
				spanByte += n
				if spanByte == len(spans[0].Text) {
					spans = spans[1:]
					spanByte = 0
				}
			}
			e.rows = append(e.rows, row)
			if end == len(runes) {
				break
			}
			col = end
		}
		start += len(runes) + 1
	}
	return e.rows
}

func (e *Editor) visualRows() []sourceRow { return e.sourceRows(e.Bounds().Width()) }

func rowAt(rows []sourceRow, line, col int) int {
	next := sort.Search(len(rows), func(i int) bool {
		row := rows[i]
		return row.line > line || (row.line == line && row.col > col)
	})
	return max(0, next-1)
}

func (e *Editor) caretRow() int {
	return rowAt(e.visualRows(), e.buf.CaretLine(), e.buf.CaretCol())
}

func (e *Editor) placeRow(index, col int) {
	rows := e.visualRows()
	row := rows[max(0, min(index, len(rows)-1))]
	e.buf.Place(row.line, row.col+max(0, min(col, row.length)))
}

// A tab occupies one editor cell so every source rune remains addressable.
func rawDisplay(s string) string { return strings.ReplaceAll(s, "\t", " ") }
