package gitlog

import (
	"strings"
	"unicode"
)

// Line pairing uses ordered matches. Insertions inside a replacement block
// leave a gap, so an extra line does not offset every later comparison.
func alignChanges(old, new []*DiffLine) []DiffRow {
	var out []DiffRow
	var i, j int
	emitOld := func() { out = append(out, DiffRow{Old: old[i]}); i++ }
	emitNew := func() { out = append(out, DiffRow{New: new[j]}); j++ }
	pair := func() { out = append(out, DiffRow{Old: old[i], New: new[j]}); i++; j++ }
	switch {
	case len(old) == 1 && len(new) == 1:
		pair()
	case len(old) > 0 && len(new) > 0 && len(old)*len(new) <= 4096:
		a, b := lineWords(old), lineWords(new)
		stride := len(b) + 1
		scores := make([]float32, (len(a)+1)*stride)
		directions := make([]byte, len(scores))
		for x := len(a) - 1; x >= 0; x-- {
			for y := len(b) - 1; y >= 0; y-- {
				at := x*stride + y
				best := scores[at+stride]
				directions[at] = '-'
				if scores[at+1] > best {
					best = scores[at+1]
					directions[at] = '+'
				}
				match := similarity(a[x], b[y])
				if match >= 0.25 && scores[at+stride+1]+match >= best {
					best = scores[at+stride+1] + match
					directions[at] = '='
				}
				scores[at] = best
			}
		}
		for i < len(old) && j < len(new) {
			switch directions[i*stride+j] {
			case '=':
				pair()
			case '+':
				emitNew()
			default:
				emitOld()
			}
		}
	}
	for i < len(old) {
		emitOld()
	}
	for j < len(new) {
		emitNew()
	}
	return out
}
func lineWords(lines []*DiffLine) []map[string]bool {
	out := make([]map[string]bool, len(lines))
	for i, line := range lines {
		out[i] = make(map[string]bool)
		for _, word := range strings.FieldsFunc(line.Text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' }) {
			out[i][word] = true
		}
		if len(out[i]) == 0 {
			out[i][strings.TrimSpace(line.Text)] = true
		}
	}
	return out
}
func similarity(a, b map[string]bool) float32 {
	if len(a) > len(b) {
		a, b = b, a
	}
	var hits int
	for word := range a {
		if b[word] {
			hits++
		}
	}
	return float32(2*hits) / float32(len(a)+len(b))
}

// Tokens retain their exact text, including punctuation and whitespace. The
// common prefix and suffix are peeled first for long single-paragraph edits.
type word struct {
	text  string
	equal bool
}

func words(s string) []word {
	var out []word
	start, last := 0, -1
	for i, r := range s {
		kind := 0
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsMark(r) || r == '_' {
			kind = 1
		} else if unicode.IsSpace(r) {
			kind = 2
		}
		if i > 0 && (kind != last || kind == 0) {
			out = append(out, word{text: s[start:i]})
			start = i
		}
		last = kind
	}
	if start < len(s) {
		out = append(out, word{text: s[start:]})
	}
	return out
}
func changedWords(a, b []word) {
	for len(a) > 0 && len(b) > 0 && a[0].text == b[0].text {
		a[0].equal = true
		b[0].equal = true
		a = a[1:]
		b = b[1:]
	}
	for len(a) > 0 && len(b) > 0 && a[len(a)-1].text == b[len(b)-1].text {
		a[len(a)-1].equal = true
		b[len(b)-1].equal = true
		a = a[:len(a)-1]
		b = b[:len(b)-1]
	}
	if len(a) == 0 || len(b) == 0 || len(a)*len(b) > 65536 {
		return
	}
	stride := len(b) + 1
	dp := make([]int, (len(a)+1)*stride)
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			p := i*stride + j
			if a[i].text == b[j].text {
				dp[p] = dp[p+stride+1] + 1
			} else {
				dp[p] = max(dp[p+stride], dp[p+1])
			}
		}
	}
	for i, j := 0, 0; i < len(a) && j < len(b); {
		if a[i].text == b[j].text {
			a[i].equal = true
			b[j].equal = true
			i++
			j++
		} else if dp[(i+1)*stride+j] >= dp[i*stride+j+1] {
			i++
		} else {
			j++
		}
	}
}
func markWords(line *DiffLine, tokens []word) {
	var runs []DiffRun
	idx, offset, pos := 0, 0, 0
	for _, run := range line.Runs {
		rest := run.Text
		for len(rest) > 0 && idx < len(tokens) {
			n := min(len(rest), len(tokens[idx].text)-offset)
			piece := run
			piece.Text = rest[:n]
			piece.Changed = !tokens[idx].equal
			if len(runs) > 0 && runs[len(runs)-1].Changed == piece.Changed && runs[len(runs)-1].Tone == piece.Tone && runs[len(runs)-1].Bold == piece.Bold && runs[len(runs)-1].Italic == piece.Italic {
				// Adjacent slices can be merged without copying the source.
				last := &runs[len(runs)-1]
				last.Text = line.Text[pos-len(last.Text) : pos+n]
			} else {
				runs = append(runs, piece)
			}
			rest = rest[n:]
			offset += n
			pos += n
			if offset == len(tokens[idx].text) {
				idx++
				offset = 0
			}
		}
	}
	line.Runs = runs
}
