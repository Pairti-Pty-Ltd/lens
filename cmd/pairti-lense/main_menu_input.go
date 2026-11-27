package main

import (
	"runtime"
	"strings"

	"github.com/gogpu/gpucontext"
)

// GoGPU's Darwin event loop queues a physical key and printable text before
// forwarding the same NSEvent to AppKit. Keep native menu equivalents out of
// the widget event bridge, which would otherwise process both copies.
type desktopInput interface {
	gpucontext.EventSource
	gpucontext.PointerEventSource
	gpucontext.ScrollEventSource
	gpucontext.GestureEventSource
}

type menuInput struct {
	desktopInput
	shortcuts map[menuShortcut]bool
	skipText  bool
}

type menuShortcut struct {
	key  string
	mods uintptr
}

func (s *shell) menuEventSource(source gpucontext.EventSource) gpucontext.EventSource {
	if runtime.GOOS != "darwin" {
		return source
	}
	input := &menuInput{desktopInput: source.(desktopInput), shortcuts: make(map[menuShortcut]bool)}
	for _, c := range s.menuCommands() {
		if c.key != "" {
			input.shortcuts[menuShortcut{c.key, c.mods}] = true
		}
	}
	return input
}

func (in *menuInput) OnKeyPress(fn func(gpucontext.Key, gpucontext.Modifiers)) {
	in.desktopInput.OnKeyPress(func(key gpucontext.Key, mods gpucontext.Modifiers) {
		// Command-modified text is never document input, including when the
		// currently focused target disables the command.
		in.skipText = mods.HasSuper()
		if mods.HasSuper() || mods.HasControl() {
			mask := uintptr(0)
			if mods.HasSuper() {
				mask |= menuCommandKey
			}
			if mods.HasControl() {
				mask |= menuControl
			}
			if mods.HasShift() {
				mask |= menuShift
			}
			if mods.HasAlt() {
				mask |= 1 << 19
			}
			if in.shortcuts[menuShortcut{menuKey(key), mask}] {
				in.skipText = true
				return
			}
		}
		fn(key, mods)
	})
}

func menuKey(key gpucontext.Key) string {
	switch key {
	case gpucontext.KeyTab:
		return "\t"
	case gpucontext.KeyGrave:
		return "`"
	case gpucontext.KeyEqual:
		return "="
	case gpucontext.KeyMinus:
		return "-"
	case gpucontext.KeyLeftBracket:
		return "["
	case gpucontext.KeyRightBracket:
		return "]"
	case gpucontext.KeyComma:
		return ","
	}
	return strings.ToLower(key.String())
}

func (in *menuInput) OnKeyRelease(fn func(gpucontext.Key, gpucontext.Modifiers)) {
	in.desktopInput.OnKeyRelease(func(key gpucontext.Key, mods gpucontext.Modifiers) {
		in.skipText = false
		fn(key, mods)
	})
}

func (in *menuInput) OnTextInput(fn func(string)) {
	in.desktopInput.OnTextInput(func(text string) {
		if !in.skipText {
			fn(text)
		}
	})
}

func (in *menuInput) OnFocus(fn func(bool)) {
	in.desktopInput.OnFocus(func(focused bool) {
		in.skipText = false
		fn(focused)
	})
}
