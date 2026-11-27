package doc

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// dropIgnored removes entries git would ignore. A folder that is not a
// repository, or a machine without git, keeps every entry.
func dropIgnored(root string, entries []DirEntry) []DirEntry {
	if len(entries) == 0 {
		return entries
	}
	rels := make([]string, len(entries))
	for i, e := range entries {
		rels[i] = e.Rel
	}
	ignored := gitIgnored(root, rels)
	if len(ignored) == 0 {
		return entries
	}
	out := make([]DirEntry, 0, len(entries))
	for _, e := range entries {
		if ignored[e.Rel] {
			continue
		}
		out = append(out, e)
	}
	return out
}

func gitIgnored(root string, rels []string) map[string]bool {
	if root == "" || len(rels) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "-C", root, "check-ignore", "-z", "--stdin")
	var in bytes.Buffer
	for _, rel := range rels {
		in.WriteString(rel)
		in.WriteByte(0)
	}
	cmd.Stdin = &in
	out, err := cmd.Output()
	if err != nil && len(out) == 0 {
		return nil
	}
	ignored := map[string]bool{}
	for _, p := range bytes.Split(out, []byte{0}) {
		if len(p) == 0 {
			continue
		}
		ignored[filepathToSlash(string(p))] = true
	}
	return ignored
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
