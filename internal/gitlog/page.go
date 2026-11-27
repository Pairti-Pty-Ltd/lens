package gitlog

import "strings"

// Page is the read-only diff document. The subject is cut so a long
// commit line does not become the page. The hash is not a heading.
func Page(subject, path, date, patch string) string {
	var b strings.Builder
	b.WriteString("# Diff\n\n")
	b.WriteString(cut(subject, 72))
	b.WriteString("\n\n")
	if path != "" {
		b.WriteString(path)
		b.WriteString("\n\n")
	}
	if date != "" {
		b.WriteString(date)
		b.WriteString("\n\n")
	}
	b.WriteString("```diff\n")
	if len(patch) > 120000 {
		patch = patch[:120000] + "\n…\n"
	}
	b.WriteString(patch)
	if patch != "" && !strings.HasSuffix(patch, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("```\n")
	return b.String()
}

func cut(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}
