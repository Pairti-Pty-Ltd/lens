package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"pairti-lense/internal/gitlog"
	"pairti-lense/internal/ui"
)

// Fetch off the UI thread; only install results during the shell tick.
type libraryHistory struct {
	base       string
	pending    chan func()
	generation uint64
}

func (s *shell) browserSideAvailable(mode int) bool {
	switch mode {
	case ui.SideFiles:
		return true
	case ui.SideHistory:
		return s.remote.history != nil
	case ui.SideReports:
		return s.remote.export != nil
	default:
		return false
	}
}

func (h *libraryHistory) get(ctx context.Context, action string, q url.Values, output any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.base+"/api/history/"+action+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return fmt.Errorf("library returned %s", res.Status)
	}
	return json.NewDecoder(res.Body).Decode(output)
}

func (h *libraryHistory) fetch(wake func(), action string, q url.Values, output any, done func(error)) {
	generation := h.generation
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		err := h.get(ctx, action, q, output)
		h.pending <- func() {
			if generation == h.generation {
				done(err)
			}
		}
		wake()
	}()
}
func (h *libraryHistory) receive() bool {
	changed := false
	for {
		select {
		case apply := <-h.pending:
			apply()
			changed = true
		default:
			return changed
		}
	}
}

func (s *shell) readLibraryHistory(skip int) {
	s.histMore = false
	if skip == 0 {
		s.commits = []gitlog.Commit{}
		s.histErr = "Loading history…"
	}
	var page []gitlog.Commit
	q := url.Values{"path": {s.histRel}, "skip": {strconv.Itoa(skip)}}
	s.remote.history.fetch(s.wake, "log", q, &page, func(err error) {
		if err != nil {
			s.histErr = "Could not read Git history"
		} else {
			s.histErr = ""
			s.histMore = len(page) > historyBatch
			s.commits = append(s.commits, page[:min(len(page), historyBatch)]...)
		}
		s.refresh()
	})
}

func (s *shell) readLibraryFiles(hash string) {
	var files []string
	s.remote.history.fetch(s.wake, "files", url.Values{"commit": {hash}}, &files, func(err error) {
		if err == nil {
			s.histFiles[hash] = files
		}
		s.refresh()
	})
}

func (h *libraryHistory) compare(ctx context.Context, _, hash, rel string) (gitlog.Comparison, error) {
	var model gitlog.Comparison
	err := h.get(ctx, "diff", url.Values{"path": {rel}, "commit": {hash}}, &model)
	return model, err
}
