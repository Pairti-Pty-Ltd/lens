package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// quarantine moves a settings file that could not be read aside, so the next
// launch does not open the same bytes. A missing file is left alone.
// The new path is returned, or "" when the file stays put.
func quarantine(path string, reason error) string {
	if path == "" || reason == nil || errors.Is(reason, os.ErrNotExist) {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	dest := path + ".bad"
	for i := 1; i < 100; i++ {
		if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
			break
		}
		dest = fmt.Sprintf("%s.bad.%d", path, i)
	}
	if err := os.Rename(path, dest); err != nil {
		log.Printf("settings: %s stays (%v): %v", filepath.Base(path), reason, err)
		return ""
	}
	log.Printf("settings: moved %s to %s (%v)", filepath.Base(path), dest, reason)
	return dest
}
