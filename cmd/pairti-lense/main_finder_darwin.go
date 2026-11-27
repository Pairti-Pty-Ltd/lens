package main

import (
	"fmt"
	"path/filepath"
	"runtime"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"golang.org/x/sys/unix"
)

// LaunchServices delivers both cold and warm opens as 'aevt'/'odoc' Apple
// events. Install before GoGPU calls finishLaunching. The event target is
// independent of GoGPU's NSApplication delegate, which owns menu callbacks.
func installFinderOpen(enqueue func(string)) error {
	if _, err := ffi.LoadLibrary("/System/Library/Frameworks/AppKit.framework/AppKit"); err != nil {
		return err
	}
	r, err := newMenuRuntime()
	if err != nil {
		return err
	}
	pool := r.send(r.class("NSAutoreleasePool"), "new")
	defer r.send(pool, "drain")
	r.send(r.class("NSApplication"), "sharedApplication")
	lib, err := ffi.LoadLibrary("/usr/lib/libobjc.A.dylib")
	if err != nil {
		return err
	}
	alloc, err := ffi.GetSymbol(lib, "objc_allocateClassPair")
	if err != nil {
		return err
	}
	addMethod, err := ffi.GetSymbol(lib, "class_addMethod")
	if err != nil {
		return err
	}
	register, err := ffi.GetSymbol(lib, "objc_registerClassPair")
	if err != nil {
		return err
	}
	class := r.withString("LensDocumentOpenHandler", func(name uintptr) uintptr { return r.call(alloc, r.class("NSObject"), name, 0) })
	if class == 0 {
		return fmt.Errorf("could not create document open handler")
	}
	sel := r.withString("openDocuments:reply:", func(name uintptr) uintptr { return r.call(r.register, name) })
	callback := ffi.NewCallback(func(_, _, event, _ uintptr) uintptr {
		list := r.send(event, "paramDescriptorForKeyword:", 0x2d2d2d2d) // keyDirectObject ('----')
		count := r.send(list, "numberOfItems")
		var path [unix.PathMax]byte
		var pin runtime.Pinner
		pin.Pin(&path[0])
		defer pin.Unpin()
		for i := uintptr(1); i <= count; i++ {
			url := r.send(r.send(list, "descriptorAtIndex:", i), "fileURLValue")
			if url == 0 || r.send(url, "isFileURL")&1 == 0 {
				continue
			}
			if r.send(url, "getFileSystemRepresentation:maxLength:", uintptr(unsafe.Pointer(&path[0])), uintptr(len(path)))&1 != 0 {
				enqueue(unix.ByteSliceToString(path[:]))
			}
		}
		return 0
	})
	if r.withString("v@:@@", func(sig uintptr) uintptr { return r.call(addMethod, class, sel, callback, sig) })&1 == 0 {
		return fmt.Errorf("could not register document open callback")
	}
	manager := r.send(r.class("NSAppleEventManager"), "sharedAppleEventManager")
	if manager == 0 {
		return fmt.Errorf("no Apple event manager")
	}
	install := func(target uintptr) {
		r.send(manager, "setEventHandler:andSelector:forEventClass:andEventID:", target, sel, 0x61657674, 0x6f646f63)
	}
	// NSApplication installs its own standard event handlers while launching.
	// Register at will-finish to receive the initial open before did-finish.
	startSel := r.withString("willFinishLaunching:", func(name uintptr) uintptr { return r.call(r.register, name) })
	start := ffi.NewCallback(func(target, _, _ uintptr) uintptr { install(target); return 0 })
	if r.withString("v@:@", func(sig uintptr) uintptr { return r.call(addMethod, class, startSel, start, sig) })&1 == 0 {
		return fmt.Errorf("could not register launch observer")
	}
	r.call(register, class)
	handler := r.send(class, "new")
	center := r.send(r.class("NSNotificationCenter"), "defaultCenter")
	r.send(center, "addObserver:selector:name:object:", handler, startSel, r.nsString("NSApplicationWillFinishLaunchingNotification"), 0)
	// The class and observer remain alive for the application lifetime.
	return nil
}

func confirmFileClose(path string) closeChoice {
	r, err := newMenuRuntime()
	if err != nil {
		return cancelClose
	}
	dialog := r.send(r.class("NSAlert"), "new")
	defer r.send(dialog, "release")
	r.send(dialog, "setMessageText:", r.nsString("Save changes to "+filepath.Base(path)+"?"))
	r.send(dialog, "setInformativeText:", r.nsString("Closing without saving will discard your edits."))
	for _, name := range []string{"Save", "Cancel", "Discard Changes"} {
		r.send(dialog, "addButtonWithTitle:", r.nsString(name))
	}
	switch r.send(dialog, "runModal") {
	case 1000:
		return saveClose
	case 1002:
		return discardClose
	default:
		return cancelClose
	}
}
