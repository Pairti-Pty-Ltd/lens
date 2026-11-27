package main

import (
	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/ui"
)

// Browser reads complete after the input event returns. Re-enter the normal
// tab-opening path so the current tab's dirty state is checked at completion.
type browserFiles struct {
	load      func(ui.Item, func(ui.Item))
	save      func(string, *edit.Buffer)
	list      func()
	readyPath string
	data      []byte
	request   uint64
}

func (s *shell) awaitDocument(it ui.Item, open func(ui.Item)) bool {
	r := s.remote
	if r == nil || it.Path == r.readyPath {
		return false
	}
	r.request++
	if s.indexOf(it.Path) >= 0 {
		return false
	}
	r.load(it, open)
	return true
}

func (s *shell) readDocument(path string) ([]byte, error) {
	if s.remote != nil && s.remote.readyPath == path {
		return s.remote.data, nil
	}
	return doc.Read(path)
}
