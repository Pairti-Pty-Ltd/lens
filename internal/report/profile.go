package report

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Profile is the project's export: which template, and the fields it may read.
type Profile struct {
	Template string
	Title    string
	Author   string
	Date     string
}

// LoadProfile reads lens.toml. A missing file is the built-in template.
func LoadProfile(root string) Profile {
	p := Profile{Template: "default"}
	b, err := os.ReadFile(filepath.Join(root, "lens.toml"))
	if err != nil {
		return p
	}
	for _, line := range strings.Split(string(b), "\n") {
		key, val, ok := tomlPair(line)
		if !ok {
			continue
		}
		switch key {
		case "template":
			if val != "" {
				p.Template = val
			}
		case "title":
			p.Title = val
		case "author":
			p.Author = val
		case "date":
			p.Date = val
		}
	}
	return p
}

// Templates are default plus every .typ file in templates/ and a root template.typ.
func Templates(root string) []string {
	names := []string{"default"}
	if st, err := os.Stat(filepath.Join(root, "template.typ")); err == nil && st.Mode().IsRegular() {
		names = append(names, "template.typ")
	}
	matches, _ := filepath.Glob(filepath.Join(root, "templates", "*.typ"))
	sort.Strings(matches)
	for _, m := range matches {
		names = append(names, filepath.Base(m))
	}
	return names
}

// SetTemplate writes the template key. name must be default or one of Templates.
func SetTemplate(root, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("name a template")
	}
	ok := false
	for _, have := range Templates(root) {
		if have == name {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("no template %s", name)
	}
	return writeKeys(root, map[string]string{"template": name})
}

func writeKeys(root string, keys map[string]string) error {
	path := filepath.Join(root, "lens.toml")
	text := ""
	if b, err := os.ReadFile(path); err == nil {
		text = string(b)
	}
	lines := []string{}
	if text != "" {
		lines = strings.Split(strings.TrimRight(text, "\n"), "\n")
	}
	seen := map[string]bool{}
	for i, line := range lines {
		key, _, ok := tomlPair(line)
		if !ok {
			continue
		}
		if val, replace := keys[key]; replace {
			lines[i] = key + " = " + quote(val)
			seen[key] = true
		}
	}
	for key, val := range keys {
		if seen[key] {
			continue
		}
		lines = append(lines, key+" = "+quote(val))
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func tomlPair(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	key, val, ok = strings.Cut(line, "=")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	val = strings.Trim(strings.TrimSpace(val), `"'`)
	return key, val, key != ""
}

func quote(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, "") + `"`
}

func resolveTemplate(root, name string) string {
	if name == "" || name == "default" {
		return ""
	}
	candidates := []string{
		filepath.Join(root, "templates", name),
		filepath.Join(root, name),
	}
	if filepath.IsAbs(name) {
		candidates = []string{name}
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p
		}
	}
	return ""
}
