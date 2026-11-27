// Package diagram is the registry of fenced-code renderers.
// A language renders when a renderer is registered for it. An unregistered
// fence stays source. There is no plugin host: a diagram type is a type in
// this process, registered at startup.
package diagram

import (
	"context"
	"image"
	"strings"
	"sync"
)

// NotInstalledError means the renderer has no local tool.
type NotInstalledError struct {
	Name string
}

func (e *NotInstalledError) Error() string {
	if e.Name == "" {
		return "renderer not installed"
	}
	return e.Name + " not installed"
}

// Renderer turns one fence language into a picture.
type Renderer interface {
	Languages() []string
	Render(ctx context.Context, source string) (image.Image, error)
}

var (
	mu   sync.RWMutex
	regs = map[string]Renderer{}
)

// Register enables r for each of its languages. A later register replaces one.
func Register(r Renderer) {
	if r == nil {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	for _, lang := range r.Languages() {
		lang = norm(lang)
		if lang == "" {
			continue
		}
		regs[lang] = r
	}
}

// Reset drops every renderer. Tests use it.
func Reset() {
	mu.Lock()
	regs = map[string]Renderer{}
	mu.Unlock()
}

// Known reports whether lang has a renderer.
func Known(lang string) bool {
	_, ok := lookup(lang)
	return ok
}

// Render draws source with the renderer for lang.
func Render(ctx context.Context, lang, source string) (image.Image, error) {
	r, ok := lookup(lang)
	if !ok {
		return nil, &NotInstalledError{Name: norm(lang)}
	}
	return r.Render(ctx, source)
}

func lookup(lang string) (Renderer, bool) {
	mu.RLock()
	defer mu.RUnlock()
	r, ok := regs[norm(lang)]
	return r, ok
}

func norm(lang string) string {
	return strings.ToLower(strings.TrimSpace(lang))
}
