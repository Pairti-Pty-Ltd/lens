package doc

import (
	"os"
	"strings"
	"sync"
)

// frontMatter is a leading YAML block, the text between the fences, and how
// many source lines the fences and the text cover. A later rule is left alone.
// A plain sentence between two rules is not front matter.
// Meta is the flat YAML front matter of a note. A missing block is an empty map.
// Nested values and folded blocks are left unread.
func Meta(src []byte) map[string]string {
	out := map[string]string{}
	body, _, ok := frontMatter(src)
	if !ok {
		return out
	}
	for _, line := range strings.Split(body, "\n") {
		key, val, found := strings.Cut(line, ":")
		if !found {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || strings.Contains(key, " ") {
			continue
		}
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		if val == "" || val == "|" || val == ">" {
			continue
		}
		out[key] = val
	}
	return out
}

// Tags are the note's own labels: tags, tag, keywords, and type.
// A flow list and a dash list are both read. The library's tags are separate.
func Tags(src []byte) []string {
	body, _, ok := frontMatter(src)
	if !ok {
		return nil
	}
	var out []string
	lines := strings.Split(body, "\n")
	for i := 0; i < len(lines); i++ {
		key, val, found := strings.Cut(lines[i], ":")
		if !found {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		if !tagKey(key) {
			continue
		}
		val = strings.TrimSpace(val)
		if val == "" || val == "|" || val == ">" {
			for i+1 < len(lines) {
				next := strings.TrimSpace(lines[i+1])
				if !strings.HasPrefix(next, "-") {
					break
				}
				i++
				item := trimYAML(strings.TrimPrefix(next, "-"))
				if item != "" {
					out = append(out, item)
				}
			}
			continue
		}
		out = append(out, splitFlow(val)...)
	}
	return out
}

func tagKey(key string) bool {
	switch key {
	case "tags", "tag", "keywords", "keyword", "type":
		return true
	default:
		return false
	}
}

func splitFlow(val string) []string {
	val = strings.TrimSpace(val)
	if strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]") {
		val = strings.TrimSpace(val[1 : len(val)-1])
	}
	var out []string
	for _, part := range strings.Split(val, ",") {
		part = trimYAML(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func trimYAML(s string) string {
	s = strings.TrimSpace(s)
	return strings.Trim(s, `"'`)
}

type tagHit struct {
	mtime int64
	size  int64
	tags  []string
}

var tagCache sync.Map

// TagsOf reads the labels from the front of a note. A repeat with the same
// size and modification time does not read the file again.
func TagsOf(path string) []string {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil
	}
	mtime := info.ModTime().UnixNano()
	if v, ok := tagCache.Load(path); ok {
		hit := v.(tagHit)
		if hit.mtime == mtime && hit.size == info.Size() {
			return hit.tags
		}
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	buf := make([]byte, sniffBytes)
	n, _ := f.Read(buf)
	tags := Tags(buf[:n])
	tagCache.Store(path, tagHit{mtime: mtime, size: info.Size(), tags: tags})
	return tags
}

func frontMatter(src []byte) (body string, lines int, ok bool) {
	s := strings.TrimPrefix(string(src), "\uFEFF")
	line, i, ok := frontLine(s, 0)
	if !ok || strings.TrimSpace(line) != "---" {
		return "", 0, false
	}
	lines = 1
	var b strings.Builder
	keyed := false
	for {
		line, next, ok := frontLine(s, i)
		if !ok {
			return "", 0, false
		}
		lines++
		if strings.TrimSpace(line) == "---" {
			if !keyed {
				return "", 0, false
			}
			return strings.TrimRight(b.String(), "\n"), lines, true
		}
		if lines > 200 || !yamlLine(line) {
			return "", 0, false
		}
		if strings.Contains(line, ":") {
			keyed = true
		}
		b.WriteString(line)
		b.WriteByte('\n')
		i = next
	}
}

func frontLine(s string, i int) (line string, next int, ok bool) {
	if i >= len(s) {
		return "", i, false
	}
	j := strings.IndexAny(s[i:], "\r\n")
	if j < 0 {
		return s[i:], len(s), true
	}
	line = s[i : i+j]
	next = i + j
	if next < len(s) && s[next] == '\r' {
		next++
	}
	if next < len(s) && s[next] == '\n' {
		next++
	}
	return line, next, true
}

func yamlLine(line string) bool {
	trim := strings.TrimSpace(line)
	if trim == "" || strings.HasPrefix(trim, "#") || strings.HasPrefix(trim, "-") {
		return true
	}
	if strings.Contains(line, ":") {
		return true
	}
	return len(line) > 0 && (line[0] == ' ' || line[0] == '\t')
}
