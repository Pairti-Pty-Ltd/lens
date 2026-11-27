package edit

import "strings"

// Merge is a line-level 3-way merge of base, ours, and theirs.
// A conflict is wrapped in <<<<<<< ours / ======= / >>>>>>> theirs.
// The returned count is how many conflict regions were marked.
func Merge(base, ours, theirs string) (string, int) {
	b, o, t := splitLines(base), splitLines(ours), splitLines(theirs)
	if tooWide(b, o) || tooWide(b, t) {
		return conflictBlock(ours, theirs), 1
	}
	return joinLines(mergeLines(b, o, t))
}

func tooWide(a, b []string) bool {
	return len(a)*len(b) > 4_000_000
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func joinLines(lines []string, conflicts int) (string, int) {
	return strings.Join(lines, "\n"), conflicts
}

type anchor struct{ a, b int }

func lcsAnchors(a, b []string) []anchor {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var pairs []anchor
	i, j := 0, 0
	for i < n && j < m {
		switch {
		case a[i] == b[j]:
			pairs = append(pairs, anchor{i, j})
			i++
			j++
		case dp[i+1][j] >= dp[i][j+1]:
			i++
		default:
			j++
		}
	}
	return pairs
}

// seg is a slice of the base, either unchanged or replaced by lines.
type seg struct {
	b0, b1  int
	lines   []string
	changed bool
}

func tiles(base, other []string) []seg {
	pairs := lcsAnchors(base, other)
	pairs = append([]anchor{{-1, -1}}, pairs...)
	pairs = append(pairs, anchor{len(base), len(other)})
	var out []seg
	for i := 0; i < len(pairs)-1; i++ {
		prev, next := pairs[i], pairs[i+1]
		b0, b1 := prev.a+1, next.a
		o0, o1 := prev.b+1, next.b
		if b0 < b1 || o0 < o1 {
			out = append(out, seg{b0: b0, b1: b1, lines: append([]string(nil), other[o0:o1]...), changed: true})
		}
		if next.a < len(base) {
			out = append(out, seg{b0: next.a, b1: next.a + 1, lines: []string{base[next.a]}})
		}
	}
	return out
}

func mergeLines(base, ours, theirs []string) ([]string, int) {
	ot, tt := tiles(base, ours), tiles(base, theirs)
	var out []string
	conflicts := 0
	i, j := 0, 0
	guard := len(ot) + len(tt) + 2
	for i < len(ot) || j < len(tt) {
		guard--
		if guard < 0 {
			return conflictLines(ours, theirs), 1
		}
		if i >= len(ot) {
			out = append(out, tt[j].lines...)
			j++
			continue
		}
		if j >= len(tt) {
			out = append(out, ot[i].lines...)
			i++
			continue
		}
		a, bseg := ot[i], tt[j]
		switch {
		case !a.changed && !bseg.changed && a.b0 == bseg.b0:
			out = append(out, a.lines...)
			i++
			j++
		case a.changed && a.b0 == a.b1 && !(bseg.changed && bseg.b0 == bseg.b1 && bseg.b0 == a.b0):
			out = append(out, a.lines...)
			i++
		case bseg.changed && bseg.b0 == bseg.b1 && !(a.changed && a.b0 == a.b1 && a.b0 == bseg.b0):
			out = append(out, bseg.lines...)
			j++
		case a.changed && bseg.changed && a.b0 == bseg.b0 && a.b0 == a.b1 && bseg.b0 == bseg.b1:
			if strings.Join(a.lines, "\n") == strings.Join(bseg.lines, "\n") {
				out = append(out, a.lines...)
			} else {
				out = append(out, conflictLines(a.lines, bseg.lines)...)
				conflicts++
			}
			i++
			j++
		case a.changed && !bseg.changed && a.b0 <= bseg.b0:
			out = append(out, a.lines...)
			i++
			j = skipStable(tt, j, a.b0, a.b1)
		case bseg.changed && !a.changed && bseg.b0 <= a.b0:
			out = append(out, bseg.lines...)
			j++
			i = skipStable(ot, i, bseg.b0, bseg.b1)
		case a.changed && bseg.changed:
			end := a.b1
			if bseg.b1 > end {
				end = bseg.b1
			}
			oLines, tLines := append([]string{}, a.lines...), append([]string{}, bseg.lines...)
			i++
			j++
			for i < len(ot) && ot[i].b0 < end {
				oLines = append(oLines, ot[i].lines...)
				i++
			}
			for j < len(tt) && tt[j].b0 < end {
				tLines = append(tLines, tt[j].lines...)
				j++
			}
			if strings.Join(oLines, "\n") == strings.Join(tLines, "\n") {
				out = append(out, oLines...)
			} else {
				out = append(out, conflictLines(oLines, tLines)...)
				conflicts++
			}
		default:
			if a.b0 <= bseg.b0 {
				out = append(out, a.lines...)
				i++
			} else {
				out = append(out, bseg.lines...)
				j++
			}
		}
	}
	return out, conflicts
}

func skipStable(segs []seg, i, lo, hi int) int {
	for i < len(segs) && !segs[i].changed && segs[i].b0 >= lo && segs[i].b0 < hi {
		i++
	}
	return i
}

func conflictLines(ours, theirs []string) []string {
	out := []string{"<<<<<<< ours"}
	out = append(out, ours...)
	out = append(out, "=======")
	out = append(out, theirs...)
	out = append(out, ">>>>>>> theirs")
	return out
}

func conflictBlock(ours, theirs string) string {
	return strings.Join(conflictLines(splitLines(ours), splitLines(theirs)), "\n")
}
