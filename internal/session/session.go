// Package session is the window the user had open: tabs, the tree, and the panes.
// A missing file is an empty session. A file that cannot be read is an error
// for the caller to ignore.
package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Doc is one open note.
type Doc struct {
	Path string `json:"path"`
	Raw  bool   `json:"raw,omitempty"`
}

// State is one saved window. Split is -1 when the document is not split.
type State struct {
	Recent           []string `json:"recent,omitempty"`
	HideRawNumbers   bool     `json:"hideRawNumbers,omitempty"`
	FormattedNumbers bool     `json:"formattedNumbers,omitempty"`
	Docs             []Doc    `json:"docs,omitempty"`
	Active           int      `json:"active"`
	Split            int      `json:"split"`
	Focus            int      `json:"focus"`
	Expanded         []string `json:"expanded,omitempty"`
	Scroll           float32  `json:"scroll,omitempty"`
	ScrollB          float32  `json:"scrollB,omitempty"`
	Sidebar          float32  `json:"sidebar,omitempty"`
	HideSide         bool     `json:"hideSide,omitempty"`
	Rail             bool     `json:"rail,omitempty"`
	RestoreTree      bool     `json:"-"`
}

// Load reads a session. A missing file is an empty session and a nil error.
func Load(path string) (State, error) {
	empty := State{Split: -1}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return empty, nil
		}
		return empty, err
	}
	if len(bytesTrim(b)) == 0 {
		return empty, nil
	}
	var st State
	if err := json.Unmarshal(b, &st); err != nil {
		return empty, err
	}
	st.RestoreTree = true
	if st.Docs == nil {
		st.Docs = []Doc{}
	}
	return st, nil
}

// Save writes the session. The directory is created if needed.
func Save(path string, st State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func bytesTrim(b []byte) []byte {
	i := 0
	j := len(b)
	for i < j && (b[i] == ' ' || b[i] == '\n' || b[i] == '\t' || b[i] == '\r') {
		i++
	}
	for j > i && (b[j-1] == ' ' || b[j-1] == '\n' || b[j-1] == '\t' || b[j-1] == '\r') {
		j--
	}
	return b[i:j]
}
