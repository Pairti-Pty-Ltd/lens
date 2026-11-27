package main

import (
	"fmt"
	"github.com/gogpu/gpucontext"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/gogpu/gogpu"
)

// GoGPU pins main to AppKit's thread at init. As in GoGPU's own Darwin
// tests, run testing.M separately so AppKit test operations remain on main.
var menuTestTasks = make(chan func())

func TestMain(m *testing.M) {
	done := make(chan int, 1)
	go func() {
		done <- m.Run()
		close(menuTestTasks)
	}()
	for task := range menuTestTasks {
		task()
	}
	os.Exit(<-done)
}

func TestNativeMenuBinding(t *testing.T) {
	done := make(chan struct{})
	menuTestTasks <- func() {
		defer close(done)
		if _, err := ffi.LoadLibrary("/System/Library/Frameworks/AppKit.framework/AppKit"); err != nil {
			t.Error(err)
			return
		}
		r, err := newMenuRuntime()
		if err != nil {
			t.Error(err)
			return
		}
		pool := r.send(r.class("NSAutoreleasePool"), "new")
		defer r.send(pool, "drain")
		r.send(r.class("NSApplication"), "sharedApplication")
		bar := r.send(r.class("NSMenu"), "new")
		defer r.send(bar, "release")
		add := func(group, title string) (uintptr, uintptr) {
			menu := r.send(r.send(r.class("NSMenu"), "alloc"), "initWithTitle:", r.nsString(group))
			defer r.send(menu, "release")
			header := r.send(r.class("NSMenuItem"), "new")
			defer r.send(header, "release")
			r.send(header, "setTitle:", r.nsString(group))
			r.send(header, "setSubmenu:", menu)
			r.send(bar, "addItem:", header)
			item := r.send(r.class("NSMenuItem"), "new")
			defer r.send(item, "release")
			r.send(item, "setTitle:", r.nsString(title))
			r.send(menu, "addItem:", item)
			return menu, item
		}
		lens, pref := add("Application", "Open in Raw")
		view, lines := add("View", "Line Numbers")
		window, _ := add("Window", "Minimize")
		on := false
		const cmdShift = 1<<20 | 1<<17
		refresh, err := r.bind(bar, window, []menuCommand{
			{group: "Lens", title: "Open in Raw", checked: func() bool { return !on }},
			{group: "Window", title: "Minimize"},
			{group: "View", title: "Line Numbers", key: "l", mods: cmdShift,
				enabled: func() bool { return on }, checked: func() bool { return on }},
		})
		if err != nil {
			t.Error(err)
			return
		}
		if r.send(lines, "isEnabled")&1 != 0 || r.send(lines, "state") != 0 || r.send(pref, "state") != 1 {
			t.Error("initial native item states mismatched")
		}
		if r.send(lens, "autoenablesItems")&1 == 0 || r.send(window, "autoenablesItems")&1 == 0 || r.send(view, "autoenablesItems")&1 != 0 {
			t.Error("expected AppKit validation for application/window and Lens validation for View")
		}
		actualKey := r.send(lines, "keyEquivalent")
		if r.send(actualKey, "isEqualToString:", r.nsString("l"))&1 == 0 || r.send(lines, "keyEquivalentModifierMask") != cmdShift {
			t.Error("native shortcut is not Command-Shift-L")
		}
		on = true
		refresh()
		r.send(view, "update")
		if r.send(lines, "isEnabled")&1 == 0 || r.send(lines, "state") != 1 || r.send(pref, "state") != 0 {
			t.Error("native menu did not reflect changes after validation")
		}
	}
	<-done
}

// This check opens a real AppKit window and exercises the installed GoGPU
// callbacks. Keep it opt-in for build hosts without a window server.
func TestInstalledDarwinMenus(t *testing.T) {
	if os.Getenv("LENS_WINDOW_TEST") != "1" {
		t.Skip("set LENS_WINDOW_TEST=1 in a logged-in macOS session")
	}
	s := testShell()
	dir := t.TempDir()
	first := treeNote(t, dir, "first.md", "# First\n")
	second := treeNote(t, dir, "second.md", "# Second\n")
	s.open(first)
	s.open(second)

	done := make(chan struct{})
	menuTestTasks <- func() {
		defer close(done)
		app := gogpu.NewApp(gogpu.DefaultConfig().WithAppName("Lens").WithSize(320, 240).WithContinuousRender(true))
		s.installMenus(app)
		var nextFrame func()
		quitRequested := false
		app.OnUpdate(func(float64) {
			if quitRequested {
				t.Error("native Quit did not end the GoGPU event loop")
				app.Quit()
				return
			}
			if nextFrame != nil {
				nextFrame()
				return
			}
			defer func() {
				if !quitRequested && nextFrame == nil {
					app.Quit()
				}
			}()
			s.refreshMenu()
			if s.refreshMenu == nil {
				t.Error("menu binding failed during startup")
				return
			}
			r, err := newMenuRuntime()
			if err != nil {
				t.Error(err)
				return
			}
			nsApp := r.send(r.class("NSApplication"), "sharedApplication")
			bar := r.send(nsApp, "mainMenu")
			submenu := func(group string) uintptr {
				switch group {
				case "Lens":
					return r.send(r.send(bar, "itemAtIndex:", 0), "submenu")
				case "Window":
					return r.send(nsApp, "windowsMenu")
				}
				return r.send(r.send(bar, "itemWithTitle:", r.nsString(group)), "submenu")
			}
			item := func(group, title string) uintptr {
				return r.send(submenu(group), "itemWithTitle:", r.nsString(title))
			}
			choose := func(group, title string) {
				it := item(group, title)
				if it == 0 {
					t.Errorf("missing %s > %s", group, title)
					return
				}
				menu := submenu(group)
				r.send(menu, "performActionForItemAtIndex:", r.send(menu, "indexOfItem:", it))
			}
			back, forward := item("Go", "Back"), item("Go", "Forward")
			if back == 0 || forward == 0 || r.send(back, "isEnabled")&1 == 0 || r.send(forward, "isEnabled")&1 != 0 {
				t.Error("installed Go menu gave incorrect Back/Forward limits")
				return
			}
			choose("Go", "Back")
			s.refreshMenu()
			if s.docPath != first.Path || r.send(back, "isEnabled")&1 != 0 || r.send(forward, "isEnabled")&1 == 0 {
				t.Error("native Back did not navigate and update both history limits")
			}
			if err := performMenuKey(r, bar, 19, menuCommandKey); err != nil {
				t.Error(err)
			}
			s.refreshMenu()
			if !s.editing || r.send(item("View", "Raw"), "state") != 1 {
				t.Error("Command-2 did not activate Raw")
			}
			s.pick("Formatted")
			s.refreshMenu()
			if r.send(item("View", "Formatted"), "state") != 1 || r.send(item("View", "Raw"), "state") != 0 {
				t.Error("toolbar view change did not update both AppKit checkmarks")
			}
			choose("Go", "Next Tab")
			if s.docPath != second.Path {
				t.Errorf("Next Tab opened %s", s.docPath)
			}
			window := r.send(nsApp, "keyWindow")
			if window == 0 {
				t.Error("expected a key window before exercising Window menu")
				return
			}
			// AppKit completes window transitions on subsequent turns of the
			// event loop. Observe each actual transition before starting another.
			steps := []struct {
				name    string
				act     func()
				reached func() bool
			}{
				{"minimize", func() { choose("Window", "Minimize") }, func() bool { return r.send(window, "isMiniaturized")&1 != 0 }},
				{"restore minimized window", func() { showLens(app) }, func() bool { return r.send(window, "isMiniaturized")&1 == 0 }},
				{"hide", func() { choose("Lens", "Hide Lens") }, func() bool { return r.send(nsApp, "isHidden")&1 != 0 }},
				{"show hidden window", func() { showLens(app) }, func() bool { return r.send(nsApp, "isHidden")&1 == 0 && r.send(window, "isVisible")&1 != 0 }},
			}
			deadline := time.Now().Add(2 * time.Second)
			steps[0].act()
			nextFrame = func() {
				if !steps[0].reached() {
					if time.Now().Before(deadline) {
						return
					}
					t.Errorf("timed out waiting for %s", steps[0].name)
				}
				steps = steps[1:]
				if len(steps) == 0 {
					checkBundledHelp(t, s, choose)
					quitRequested = true
					choose("Lens", "Quit Lens")
					return
				}
				deadline = time.Now().Add(2 * time.Second)
				steps[0].act()
			}
		})
		if err := app.Run(); err != nil {
			t.Error(err)
		}
	}
	<-done
}

func checkBundledHelp(t *testing.T, s *shell, choose func(group, title string)) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Error(err)
		return
	}
	dir := filepath.Dir(exe)
	if filepath.Base(dir) != "MacOS" {
		t.Log("packaged Help check requires running the test binary inside Contents/MacOS")
		return
	}
	keymap := filepath.Join(dir, "..", "Resources", "editor-keymap.md")
	if !isFile(keymap) {
		t.Errorf("missing bundled keymap %s", keymap)
		return
	}
	choose("Help", "Editor Keymap")
	if strings.HasPrefix(s.line3.Get(), "Editor keymap:") {
		t.Error(s.line3.Get())
	}
}

// Deliver a keyboard equivalent directly to this test process's menu.
// No global event posting or Accessibility permission is involved.
func performMenuKey(r *menuRuntime, bar, keyCode, modifiers uintptr) error {
	key, err := testMenuKey(r, keyCode, modifiers)
	if err != nil {
		return err
	}
	if r.send(bar, "performKeyEquivalent:", key)&1 == 0 {
		return fmt.Errorf("installed menu did not recognize key %d with modifiers %#x", keyCode, modifiers)
	}
	return nil
}

func testMenuKey(r *menuRuntime, keyCode, modifiers uintptr) (uintptr, error) {
	lib, err := ffi.LoadLibrary("/System/Library/Frameworks/CoreGraphics.framework/CoreGraphics")
	if err != nil {
		return 0, err
	}
	create, err := ffi.GetSymbol(lib, "CGEventCreateKeyboardEvent")
	if err != nil {
		return 0, err
	}
	flags, err := ffi.GetSymbol(lib, "CGEventSetFlags")
	if err != nil {
		return 0, err
	}
	release, err := ffi.GetSymbol(lib, "CFRelease")
	if err != nil {
		return 0, err
	}
	event := r.call(create, 0, keyCode, 1)
	if event == 0 {
		return 0, fmt.Errorf("CGEventCreateKeyboardEvent returned nil")
	}
	defer r.call(release, event)
	r.call(flags, event, modifiers)
	return r.send(r.class("NSEvent"), "eventWithCGEvent:", event), nil
}

// Drive the installed Edit items with the standard macOS key equivalents.
func TestInstalledEditShortcuts(t *testing.T) {
	if os.Getenv("LENS_WINDOW_TEST") != "1" {
		t.Skip("set LENS_WINDOW_TEST=1 in a macOS GUI session")
	}
	s := testShell()
	openNote(t, s, "# Clipboard ☕\n")
	done := make(chan struct{})
	menuTestTasks <- func() {
		defer close(done)
		app := gogpu.NewApp(gogpu.DefaultConfig().WithAppName("Lens").WithSize(320, 240).WithContinuousRender(true))
		s.installMenus(app)
		var verify func()
		app.OnUpdate(func(float64) {
			if verify != nil {
				verify()
				return
			}
			s.refreshMenu()
			r, err := newMenuRuntime()
			if err != nil {
				t.Error(err)
				app.Quit()
				return
			}
			nsApp := r.send(r.class("NSApplication"), "sharedApplication")
			checkNativeEditShortcuts(t, s, r, r.send(nsApp, "mainMenu"))
			verify = checkQueuedMenuInput(t, s, app, r, nsApp)
		})
		if err := app.Run(); err != nil {
			t.Error(err)
		}
	}
	<-done
}

func checkNativeEditShortcuts(t *testing.T, s *shell, r *menuRuntime, bar uintptr) {
	t.Helper()
	cb := &menuClipboard{}
	s.clipboard = cb
	press := func(key, mods uintptr) {
		t.Helper()
		s.refreshMenu()
		if err := performMenuKey(r, bar, key, mods); err != nil {
			t.Error(err)
		}
	}
	original := s.buf.Text()
	press(0, menuCommandKey) // A
	press(8, menuCommandKey) // C
	if cb.text != original {
		t.Errorf("Cmd-C copied %q", cb.text)
	}
	press(7, menuCommandKey) // X
	if s.buf.Text() != "" {
		t.Error("Cmd-X did not cut the selection")
	}
	press(6, menuCommandKey) // Z
	if s.buf.Text() != original {
		t.Error("Cmd-Z did not undo the cut")
	}
	press(6, menuCommandKey|menuShift)
	if s.buf.Text() != "" {
		t.Error("Cmd-Shift-Z did not redo the cut")
	}
	press(9, menuCommandKey) // V
	if s.buf.Text() != original {
		t.Error("Cmd-V did not restore clipboard text")
	}
	press(3, menuCommandKey) // F
	if !s.page.CommandOn() || s.buf.Modal.Prompt() != '/' {
		t.Error("Cmd-F did not open document search")
	}
}

// Post keyboard events only to this test process. Observe the original GoGPU
// window callbacks and the stream forwarded to widgets by the Lens bridge.
func checkQueuedMenuInput(t *testing.T, s *shell, app *gogpu.App, r *menuRuntime, nsApp uintptr) func() {
	t.Helper()
	var raw, delivered []string
	recordKey := func(log *[]string) func(gpucontext.Key, gpucontext.Modifiers) {
		return func(key gpucontext.Key, _ gpucontext.Modifiers) { *log = append(*log, "key "+key.String()) }
	}
	recordText := func(log *[]string) func(string) { return func(text string) { *log = append(*log, "text "+text) } }
	app.PrimaryWindow().SetOnKeyPress(recordKey(&raw))
	app.PrimaryWindow().SetOnTextInput(recordText(&raw))
	bridge := s.menuEventSource(app.EventSource())
	bridge.OnKeyPress(recordKey(&delivered))
	bridge.OnTextInput(recordText(&delivered))
	for _, input := range []struct{ key, mods uintptr }{{6, menuCommandKey}, {0, 0}} {
		e, err := testMenuKey(r, input.key, input.mods)
		if err != nil {
			t.Error(err)
			return app.Quit
		}
		r.send(nsApp, "postEvent:atStart:", e, 0)
	}
	deadline := time.Now().Add(3 * time.Second)
	return func() {
		if len(raw) < 4 && time.Now().Before(deadline) {
			return
		}
		if !slices.Equal(raw, []string{"key Z", "text z", "key A", "text a"}) || !slices.Equal(delivered, []string{"key A", "text a"}) {
			t.Errorf("GoGPU received %v, Lens widgets received %v", raw, delivered)
		}
		t.Logf("queued Cmd-Z then A: GoGPU %v, Lens widget bridge %v", raw, delivered)
		app.Quit()
	}
}
