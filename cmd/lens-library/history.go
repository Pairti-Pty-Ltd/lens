package main

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"time"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/gitlog"
)

var objectID = regexp.MustCompile("^(?:[0-9a-f]{40}|[0-9a-f]{64})$")

func (l *library) history(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()
	q := r.URL.Query()
	path := q.Get("path")
	if path != "" && (!fs.ValidPath(path) || !doc.IsMarkdown(path)) {
		http.NotFound(w, r)
		return
	}
	paths, err := l.documents(ctx, path)
	if err != nil {
		http.Error(w, "Cannot read the Git index.", 500)
		return
	}
	if path != "" && !slices.Contains(paths, path) {
		http.NotFound(w, r)
		return
	}
	root := l.root.Name()
	hash := q.Get("commit")
	var result any
	switch r.PathValue("action") {
	case "log":
		skip := 0
		if q.Has("skip") {
			skip, err = strconv.Atoi(q.Get("skip"))
			if err != nil || skip < 0 || skip > 1<<30 {
				http.Error(w, "Invalid history offset.", 400)
				return
			}
		}
		var commits []gitlog.Commit
		if len(paths) > 0 {
			commits, err = gitlog.LogPaths(ctx, root, paths, skip, 81)
		}
		result = commits
	case "files", "diff":
		if !objectID.MatchString(hash) || exec.CommandContext(ctx, "git", "-C", root, "merge-base", "--is-ancestor", hash, "HEAD").Run() != nil {
			http.NotFound(w, r)
			return
		}
		if r.PathValue("action") == "diff" {
			result, err = gitlog.ComparePaths(ctx, root, hash, paths)
		} else {
			if len(paths) > 0 {
				result, err = gitlog.FilesPaths(ctx, root, hash, paths)
			} else {
				result = []string{}
			}
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, "Cannot read Git history.", 500)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
