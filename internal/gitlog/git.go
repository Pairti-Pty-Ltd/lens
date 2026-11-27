// Package gitlog reads a repository with the git on the machine.
// It does not write, stage, or switch branches.
package gitlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ErrNone means the directory is not a git work tree.
var ErrNone = errors.New("not a git repository")

// Commit is one revision. Hash is for git, not for a title.
type Commit struct {
	Hash    string
	Date    string
	Subject string
}

// Top is the repository root that contains dir.
func Top(dir string) (string, error) {
	out, err := git(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Branch is the short branch name. A detached head has no name.
func Branch(dir string) (string, error) {
	out, err := git(dir, "symbolic-ref", "-q", "--short", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Log is the newest n commits. rel limits the log to one file when set.
func Log(dir, rel string, n int) ([]Commit, error) {
	if n < 1 {
		n = 1
	}
	args := []string{"--literal-pathspecs", "log", "-n", fmt.Sprint(n), "--date=format:%d %b", "--pretty=format:%H%x1f%ad%x1f%s"}
	if rel != "" {
		args = append(args, "--", rel)
	}
	out, err := git(dir, args...)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(out) == "" {
		return nil, nil
	}
	var commits []Commit
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		parts := strings.SplitN(line, "\x1f", 3)
		if len(parts) != 3 || parts[0] == "" {
			continue
		}
		commits = append(commits, Commit{Hash: parts[0], Date: parts[1], Subject: parts[2]})
	}
	return commits, nil
}

// Files are the paths touched by hash.
func Files(dir, hash string) ([]string, error) {
	out, err := git(dir, "show", "--format=", "--root", "--diff-merges=first-parent", "--find-renames", "--name-only", "-z", hash)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(strings.TrimSuffix(out, "\x00"), "\x00"), nil
}

// Diff is the patch for hash. rel limits it to one file.
func Diff(dir, hash, rel string) (string, error) {
	args := []string{"show", "--format=", "--diff-merges=first-parent", "--no-color", "-p", hash}
	if rel != "" {
		args = append(args, "--", rel)
	}
	return git(dir, args...)
}

func git(dir string, args ...string) (string, error) {
	if dir == "" {
		return "", ErrNone
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if strings.Contains(msg, "not a git repository") || strings.Contains(msg, "Needed a single revision") {
			return "", ErrNone
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s", msg)
	}
	return string(out), nil
}
