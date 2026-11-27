package main

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
)

// menuRuntime calls the small set of pointer/integer Objective-C methods that
// gogpu.MenuItem does not expose. All calls run on AppKit's main thread.
type menuRuntime struct {
	getClass, register, message unsafe.Pointer
	calls                       [7]types.CallInterface // objc_msgSend with up to four explicit arguments
	selectors                   map[string]uintptr
}

func newMenuRuntime() (*menuRuntime, error) {
	lib, err := ffi.LoadLibrary("/usr/lib/libobjc.A.dylib")
	if err != nil {
		return nil, err
	}
	r := &menuRuntime{selectors: make(map[string]uintptr)}
	for name, target := range map[string]*unsafe.Pointer{
		"objc_getClass": &r.getClass, "sel_registerName": &r.register, "objc_msgSend": &r.message,
	} {
		*target, err = ffi.GetSymbol(lib, name)
		if err != nil {
			return nil, err
		}
	}
	var args []*types.TypeDescriptor
	for n := 1; n < len(r.calls); n++ {
		args = append(args, types.PointerTypeDescriptor)
		if err := ffi.PrepareCallInterface(&r.calls[n], types.DefaultCall, types.PointerTypeDescriptor, args); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *menuRuntime) call(fn unsafe.Pointer, words ...uintptr) uintptr {
	ptrs := make([]unsafe.Pointer, len(words))
	for i := range words {
		ptrs[i] = unsafe.Pointer(&words[i])
	}
	var result uintptr
	if _, err := ffi.CallFunction(&r.calls[len(words)], fn, unsafe.Pointer(&result), ptrs); err != nil {
		panic(err) // All signatures are fixed and validated at startup.
	}
	return result
}

// A C string is borrowed only for the duration of the call.
func (r *menuRuntime) withString(s string, fn func(uintptr) uintptr) uintptr {
	p, err := syscall.BytePtrFromString(s)
	if err != nil {
		return 0
	}
	var pin runtime.Pinner
	pin.Pin(p)
	defer pin.Unpin()
	return fn(uintptr(unsafe.Pointer(p)))
}

func (r *menuRuntime) class(name string) uintptr {
	return r.withString(name, func(p uintptr) uintptr { return r.call(r.getClass, p) })
}

func (r *menuRuntime) send(id uintptr, name string, args ...uintptr) uintptr {
	if id == 0 {
		return 0
	}
	sel := r.selectors[name]
	if sel == 0 {
		sel = r.withString(name, func(p uintptr) uintptr { return r.call(r.register, p) })
		r.selectors[name] = sel
	}
	return r.call(r.message, append([]uintptr{id, sel}, args...)...)
}

func (r *menuRuntime) nsString(s string) uintptr {
	return r.withString(s, func(p uintptr) uintptr {
		return r.send(r.class("NSString"), "stringWithUTF8String:", p)
	})
}

func bindMenuItems(commands []menuCommand) (func(), error) {
	r, err := newMenuRuntime()
	if err != nil {
		return nil, err
	}
	app := r.send(r.class("NSApplication"), "sharedApplication")
	return r.bind(r.send(app, "mainMenu"), r.send(app, "windowsMenu"), commands)
}

type nativeMenuBinding struct {
	item                          uintptr
	command                       menuCommand
	initialized, enabled, checked bool
}

func (r *menuRuntime) bind(bar, window uintptr, commands []menuCommand) (func(), error) {
	if r.send(bar, "numberOfItems") == 0 {
		return nil, fmt.Errorf("AppKit main menu is empty")
	}
	var bindings []nativeMenuBinding
	for _, c := range commands {
		if c.title == "" {
			continue
		}
		var menu uintptr
		switch c.group {
		case "Lens":
			menu = r.send(r.send(bar, "itemAtIndex:", 0), "submenu")
		case "Window":
			menu = window
		default:
			menu = bar
			for _, title := range strings.Split(c.group, "\x00") {
				menu = r.send(r.send(menu, "itemWithTitle:", r.nsString(title)), "submenu")
			}
		}
		item := r.send(menu, "itemWithTitle:", r.nsString(c.title))
		if item == 0 {
			return nil, fmt.Errorf("missing native menu item %s > %s", c.group, c.title)
		}
		// Leave AppKit's own actions (Minimize, Hide Others, etc.) under
		// responder-chain validation. Lens owns validation of its submenus.
		if c.group != "Lens" && c.group != "Window" {
			r.send(menu, "setAutoenablesItems:", 0)
		}
		// GoGPU keeps the system Window > Close item across menu rebuilds.
		// File owns Command-W when its Close Tab command is installed.
		if c.group == "File" && c.key == "w" && c.mods == menuCommandKey {
			closeItem := r.send(window, "itemWithTitle:", r.nsString("Close"))
			r.send(closeItem, "setKeyEquivalent:", r.nsString(""))
		}
		if c.key != "" {
			r.send(item, "setKeyEquivalent:", r.nsString(c.key))
			r.send(item, "setKeyEquivalentModifierMask:", c.mods)
		}
		bindings = append(bindings, nativeMenuBinding{item: item, command: c})
	}
	refresh := func() {
		for i := range bindings {
			b := &bindings[i]
			if pred := b.command.enabled; pred != nil {
				if on := pred(); on != b.enabled || !b.initialized {
					r.send(b.item, "setEnabled:", menuBit(on))
					b.enabled = on
				}
			}
			if pred := b.command.checked; pred != nil {
				if on := pred(); on != b.checked || !b.initialized {
					r.send(b.item, "setState:", menuBit(on))
					b.checked = on
				}
			}
			b.initialized = true
		}
	}
	refresh()
	return refresh, nil
}

func menuBit(on bool) uintptr {
	if on {
		return 1
	}
	return 0
}
