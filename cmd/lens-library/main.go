// Command lens-library serves the browser editor and one Git working tree.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"pairti-lense/internal/doc"
)

type library struct {
	root   *os.Root
	writes sync.Mutex
}

// Read the index, including the entry mode, so links and conflicted entries
// never become editable documents. Read the actual text from the working tree.
func (l *library) documents(ctx context.Context, path string) ([]string, error) {
	args := []string{"-C", l.root.Name(), "--literal-pathspecs", "ls-files", "--stage", "-z", "--"}
	if path != "" {
		args = append(args, path)
	}
	out, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return nil, err
	}
	names := []string{}
	for entry := range bytes.SplitSeq(out, []byte{0}) {
		header, path, ok := strings.Cut(string(entry), "\t")
		if !ok || !strings.HasSuffix(header, " 0") || !(strings.HasPrefix(header, "100644 ") || strings.HasPrefix(header, "100755 ")) {
			continue
		}
		if doc.IsMarkdown(path) && fs.ValidPath(path) && utf8.ValidString(path) && l.regular(path) {
			names = append(names, path)
		}
	}
	return names, nil
}

func (l *library) regular(path string) bool {
	parent := "."
	for part := range strings.SplitSeq(path, "/") {
		parent = filepath.Join(parent, part)
		info, err := l.root.Lstat(parent)
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			return false
		}
		if parent == filepath.FromSlash(path) {
			return info.Mode().IsRegular()
		}
		if !info.IsDir() {
			return false
		}
	}
	return false
}

func revision(data []byte) string { return fmt.Sprintf("\"%x\"", sha256.Sum256(data)) }

func (l *library) api(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	path := r.URL.Query().Get("path")
	if r.URL.Path == "/api/document" && (!fs.ValidPath(path) || !doc.IsMarkdown(path)) {
		http.NotFound(w, r)
		return
	}
	var next []byte
	if r.Method == http.MethodPut {
		var err error
		next, err = io.ReadAll(http.MaxBytesReader(w, r.Body, doc.MaxBytes))
		if err != nil {
			http.Error(w, "Cannot read save (1 MiB maximum).", 413)
			return
		}
		if !utf8.Valid(next) {
			http.Error(w, "Expected UTF-8 Markdown.", 422)
			return
		}

		l.writes.Lock()
		defer l.writes.Unlock()
	}
	names, err := l.documents(r.Context(), path)
	if err != nil {
		http.Error(w, "Cannot read the Git index.", 500)
		return
	}
	if r.URL.Path == "/api/documents" {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(names)
		return
	}
	if !slices.Contains(names, path) {
		http.NotFound(w, r)
		return
	}
	f, err := l.root.Open(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, doc.MaxBytes+1))
	if err != nil {
		http.Error(w, "Cannot read document.", 500)
		return
	}
	if len(data) > doc.MaxBytes || !utf8.Valid(data) {
		http.Error(w, "Expected UTF-8 Markdown up to 1 MiB.", 422)
		return
	}
	tag := revision(data)
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		w.Header().Set("ETag", tag)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Write(data)
		return
	}
	if r.Header.Get("If-Match") != tag {
		http.Error(w, "File changed on the server. Download your edits before reopening.", http.StatusPreconditionFailed)
		return
	}
	info, err := f.Stat()
	if err != nil || l.replace(path, next, info.Mode().Perm()) != nil {
		http.Error(w, "Could not save document.", 500)
		return
	}
	w.Header().Set("ETag", revision(next))
	w.WriteHeader(http.StatusNoContent)
}

func (l *library) replace(path string, data []byte, mode fs.FileMode) error {
	tmp := filepath.Join(filepath.Dir(path), ".lens-"+rand.Text())
	f, err := l.root.OpenFile(tmp, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer l.root.Remove(tmp)
	if _, err = f.Write(data); err == nil {
		err = f.Chmod(mode)
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return l.root.Rename(tmp, path)
}

func (l *library) handler(assets string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/documents", l.api)
	mux.HandleFunc("GET /api/document", l.api)
	mux.HandleFunc("PUT /api/document", l.api)
	mux.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/library/", http.StatusFound) })
	for _, base := range []string{"/library/", "/local/"} {
		mux.HandleFunc("GET "+base+"{$}", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, filepath.Join(assets, "index.html"))
		})
		for _, name := range []string{"lens.wasm", "wasm_exec.js"} {
			mux.HandleFunc("GET "+base+name, func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, filepath.Join(assets, name))
			})
		}
	}
	protected := http.NewCrossOriginProtection().Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Host {
		case "localhost:8081", "127.0.0.1:8081":
		default:
			http.Error(w, "Use the localhost library address.", http.StatusForbidden)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		protected.ServeHTTP(w, r)
	})
}

func main() {
	repo := flag.String("repo", "", "root of the library's Git working tree")
	assets := flag.String("assets", "build/wasm", "built browser assets")
	flag.Parse()
	if *repo == "" {
		log.Fatal("Set LIBRARY to the Git working tree to serve.")
	}
	dir, err := filepath.Abs(*repo)
	if err != nil {
		log.Fatal(err)
	}
	prefix, err := exec.Command("git", "-C", dir, "rev-parse", "--show-prefix").Output()
	if err != nil || string(prefix) != "\n" {
		log.Fatal("Library must be the root of a Git working tree.")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		log.Fatal(err)
	}
	defer root.Close()
	l := &library{root: root}
	log.Printf("Serving %s at http://127.0.0.1:8081/library/", dir)
	server := http.Server{Addr: "127.0.0.1:8081", Handler: l.handler(*assets), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second}
	log.Fatal(server.ListenAndServe())
}
