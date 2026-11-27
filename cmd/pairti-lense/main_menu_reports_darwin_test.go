package main

import (
	"os"
	"sync/atomic"
	"testing"

	"github.com/gogpu/gogpu"
	"pairti-lense/internal/doc"
)

func TestInstalledReportMenu(t *testing.T) {
	if os.Getenv("LENS_WINDOW_TEST") != "1" {
		t.Skip("requires a macOS window server")
	}
	s := testShell()
	root := t.TempDir()
	s.projects = []doc.Project{{Name: "Work", Root: root}}
	done := make(chan struct{})
	menuTestTasks <- func() {
		defer close(done)
		app := gogpu.NewApp(gogpu.DefaultConfig().WithAppName("Lens").WithSize(320, 240))
		s.installMenus(app)
		app.OnUpdate(func(float64) {
			defer app.Quit()
			r, err := newMenuRuntime()
			if err != nil {
				t.Error(err)
				return
			}
			nsApp := r.send(r.class("NSApplication"), "sharedApplication")
			item := func(path ...string) uintptr {
				menu := r.send(nsApp, "mainMenu")
				var it uintptr
				for _, title := range path {
					it = r.send(menu, "itemWithTitle:", r.nsString(title))
					menu = r.send(it, "submenu")
				}
				return it
			}
			choose := func(path ...string) {
				it := item(path...)
				if it == 0 {
					t.Errorf("missing menu item %v", path)
					return
				}
				menu := r.send(it, "menu")
				r.send(menu, "performActionForItemAtIndex:", r.send(menu, "indexOfItem:", it))
				s.refreshMenu()
			}
			s.refreshMenu()
			projectItem := func(title string) uintptr { return item("Reports", "Open Report", "Work", title) }
			if empty := projectItem("No Reports"); empty == 0 || r.send(empty, "isEnabled")&1 != 0 {
				t.Error("missing disabled empty-project placeholder")
			}
			choose("Reports", "New Report")
			if created := projectItem("Report"); created == 0 || r.send(created, "state") != 1 {
				t.Error("new report missing its checkmark")
			}
			choose("Reports", "Open Report", "Work", "Report")
			if s.viewing != "report-1" || s.reportRoot != root {
				t.Error("did not open selected report")
			}
			choose("Reports", "Delete Report")
			if projectItem("Report") != 0 || projectItem("No Reports") == 0 {
				t.Error("deleted report remained in native submenu")
			}
		})
		if err := app.Run(); err != nil {
			t.Error(err)
		}
	}
	<-done
}

// A report change can arrive during the Pulse render tick (for example via
// MCP). Rebuilding NSMenu there raises an NSInternalInconsistencyException.
func TestReportMenuRefreshAfterRender(t *testing.T) {
	if os.Getenv("LENS_WINDOW_TEST") != "1" {
		t.Skip("requires a macOS window server")
	}
	s := testShell()
	root := t.TempDir()
	s.projects = []doc.Project{{Name: "Work", Root: root}}
	done := make(chan struct{})
	menuTestTasks <- func() {
		defer close(done)
		app := gogpu.NewApp(gogpu.DefaultConfig().WithAppName("Lens").WithSize(320, 240))
		s.wake = app.RequestRedraw
		s.installMenus(app)
		r, err := newMenuRuntime()
		if err != nil {
			t.Error(err)
			return
		}
		var rendered atomic.Bool
		app.OnDraw(func(ctx *gogpu.Context) {
			ctx.Clear(0, 0, 0, 1)
			if rendered.Swap(true) {
				return
			}
			s.newReport()
			s.tick(nil)
		})
		app.OnUpdate(func(dt float64) {
			s.updateNativeCommands(dt)
			if !rendered.Load() {
				return
			}
			defer app.Quit()
			menu := r.send(r.send(r.class("NSApplication"), "sharedApplication"), "mainMenu")
			var item uintptr
			for _, title := range []string{"Reports", "Open Report", "Work", "Report"} {
				item = r.send(menu, "itemWithTitle:", r.nsString(title))
				menu = r.send(item, "submenu")
			}
			if item == 0 || r.send(item, "state") != 1 {
				t.Error("render-time report missing from native menu on next update")
			}
		})
		if err := app.Run(); err != nil {
			t.Error(err)
		}
	}
	<-done
}
