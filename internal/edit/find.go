package edit

import "strings"

// FindLine returns the 1-based line of query, searching from the 1-based
// line after `after` and wrapping once. Zero means there is no hit.
func FindLine(text, query string, after int) int {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return 0
	}
	lines := strings.Split(text, "\n")
	n := len(lines)
	if n == 0 {
		return 0
	}
	start := after
	if start < 0 {
		start = 0
	}
	if start >= n {
		start = 0
	}
	for step := 0; step < n; step++ {
		i := (start + step) % n
		if strings.Contains(strings.ToLower(lines[i]), query) {
			return i + 1
		}
	}
	return 0
}
