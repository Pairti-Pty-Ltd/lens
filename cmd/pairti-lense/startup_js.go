//go:build js && wasm

package main

import (
	"encoding/json"
	"path"
	"sort"
	"strings"
	"syscall/js"

	"github.com/gogpu/gogpu"
	"github.com/gogpu/ui/app"

	"github.com/gogpu/ui/theme/material3"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/index"
	"pairti-lense/internal/library"
	"pairti-lense/internal/project"
	"pairti-lense/internal/theme"
	"pairti-lense/internal/ui"
)

const webRoot = "/Library"

func main() {
	page := js.Global()
	look := theme.Pairti()
	s := &shell{
		lib:  &library.Library{Paths: []string{webRoot}},
		look: look, base: look, zoomIndex: theme.ZoomIndex("100%"),
		active: -1, splitAt: -1, histAt: -1,
		idx:        index.New(),
		expanded:   map[string]bool{webRoot: true},
		cache:      map[string][]doc.DirEntry{},
		projects:   []doc.Project{{Name: path.Base(webRoot), Root: webRoot}},
		reportRoot: webRoot,
		remote:     &browserFiles{},
	}
	libraryMode := page.Get("libraryMode").Bool()
	s.lib.OpenRaw = !libraryMode
	if libraryMode {
		s.remote.history = &libraryHistory{base: page.Get("location").Get("origin").String(), pending: make(chan func(), 8)}
	}
	host := gogpu.NewApp(gogpu.DefaultConfig().WithTitle("Lens"))
	window := app.New(app.WithWindowProvider(host), app.WithPlatformProvider(host), app.WithEventSource(host.EventSource()), app.WithTheme(material3.NewDark(widget.Hex(look.Accent)).AsTheme()))
	s.wake = host.RequestRedraw
	s.menuContext = window.Window().Context()
	s.clipboard = host
	s.files.confirmClose = func(name string) closeChoice {
		if page.Call("confirm", "Discard unsaved changes to "+path.Base(name)+"?").Bool() {
			return discardClose
		}
		return cancelClose
	}
	s.initWidgets()
	window.SetRoot(ui.NewPulse(s.build(), s.tick))

	var names []string
	s.remote.list = func() { s.populateBrowserTree(names) }
	s.remote.load = func(it ui.Item, open func(ui.Item)) {
		request := s.remote.request
		browserResult(page.Call("readMarkdown", it.Rel), func(result js.Value) {
			if request != s.remote.request {
				return
			}
			s.remote.readyPath = it.Path
			s.remote.data = []byte(result.Get("text").String())
			open(it)
			s.remote.readyPath = ""
			s.remote.data = nil
			page.Call("openedMarkdown", it.Rel, result.Get("revision"))
			host.RequestRedraw()
		}, func(msg string) {
			if request == s.remote.request {
				s.showMessage(msg)
				host.RequestRedraw()
			}
		})
	}
	s.remote.save = func(name string, b *edit.Buffer) {
		text := b.Text()
		browserResult(page.Call("writeMarkdown", strings.TrimPrefix(name, webRoot+"/"), text), func(js.Value) {
			s.documentSaved(name, b, text)
			host.RequestRedraw()
		}, func(msg string) { s.showMessage(msg); host.RequestRedraw() })
	}
	expose := func(name string, fn func([]js.Value) any) {
		page.Set(name, js.FuncOf(func(_ js.Value, args []js.Value) any { return fn(args) }))
	}
	expose("setReportCatalog", func(a []js.Value) any {
		var catalog struct {
			Book    project.Book
			Outputs []ui.ExportFormat
		}
		if err := json.Unmarshal([]byte(a[0].String()), &catalog); err != nil {
			return false
		}
		s.book = catalog.Book
		s.remote.formats = catalog.Outputs
		s.remote.export = func(id, name, output string) { page.Call("exportReport", id, name, output) }
		s.refresh()
		host.RequestRedraw()
		return true
	})
	expose("setDocuments", func(a []js.Value) any {
		names = nil
		for i := 0; i < a[0].Length(); i++ {
			names = append(names, a[0].Index(i).String())
		}
		s.remote.list()
		return nil
	})
	expose("openImported", func(a []js.Value) any {
		name := a[0].String()
		it := ui.Item{Path: path.Join(webRoot, name), Rel: name, ProjectRoot: webRoot, ProjectName: "Library"}
		if i := s.indexOf(it.Path); i >= 0 {
			if !s.canCloseFile(i) {
				return false
			}
			s.dropDoc(i)
		}
		s.openReplacing(it)
		return true
	})
	expose("activePath", func([]js.Value) any { return strings.TrimPrefix(s.docPath, webRoot+"/") })
	expose("markdownText", func([]js.Value) any {
		if s.buf == nil {
			return ""
		}
		return s.buf.Text()
	})
	expose("hasEdits", func([]js.Value) any {
		for _, d := range s.docs {
			if d.buf != nil && d.buf.Dirty() {
				return true
			}
		}
		return false
	})
	expose("menuItems", func([]js.Value) any {
		var items []any
		for _, c := range s.menuCommands() {
			items = append(items, map[string]any{"group": c.group, "title": c.title, "enabled": s.browserCommand(c) && (c.enabled == nil || c.enabled()), "checked": c.checked != nil && c.checked()})
		}
		return items
	})
	expose("invokeMenu", func(a []js.Value) any {
		for _, c := range s.menuCommands() {
			if c.group == a[0].String() && c.title == a[1].String() && s.browserCommand(c) {
				c.invoke()
				host.RequestRedraw()
				break
			}
		}
		return nil
	})
	if err := runBrowser(host, window); err != nil {
		page.Call("showError", err.Error())
	}
}

// Release the per-request callbacks on both resolution and rejection.
func browserResult(promise js.Value, done func(js.Value), failed func(string)) {
	var ok, bad js.Func
	release := func() { ok.Release(); bad.Release() }
	ok = js.FuncOf(func(_ js.Value, args []js.Value) any { defer release(); done(args[0]); return nil })
	bad = js.FuncOf(func(_ js.Value, args []js.Value) any {
		defer release()
		failed(args[0].Get("message").String())
		return nil
	})
	promise.Call("then", ok, bad)
}

func (s *shell) browserCommand(c menuCommand) bool {
	if c.group == "Reports" || strings.HasPrefix(c.group, "Reports\x00") {
		return s.remote.export != nil && (c.title == "Export Report…" || strings.HasPrefix(c.group, "Reports\x00Open Report\x00"))
	}
	switch c.group {
	case "File":
		return c.title == "Save" || c.title == "Close Tab"
	case "View", "Insert", "Lens":
		return true
	case "Edit":
		return c.title != "Cut" && c.title != "Copy" && c.title != "Paste"
	case "Go":
		return c.title != "History" || s.remote.history != nil
	}
	return false
}

func (s *shell) populateBrowserTree(names []string) {
	s.cache = map[string][]doc.DirEntry{webRoot: nil}
	view := s.lib.View(webRoot)
	seen := map[string]bool{}
	for _, name := range names {
		parent := webRoot
		parts := strings.Split(name, "/")
		for i, part := range parts {
			if !view.ShowDot && strings.HasPrefix(part, ".") {
				break
			}
			full := path.Join(parent, part)
			dir := i < len(parts)-1
			if !seen[full] {
				seen[full] = true
				s.cache[parent] = append(s.cache[parent], doc.DirEntry{Name: part, Title: part, Rel: strings.TrimPrefix(full, webRoot+"/"), Abs: full, IsDir: dir})
			}
			parent = full
		}
	}
	for _, entries := range s.cache {
		sort.Slice(entries, func(i, j int) bool {
			if entries[i].IsDir != entries[j].IsDir {
				return entries[i].IsDir
			}
			return entries[i].Name < entries[j].Name
		})
	}
	s.refresh()
	s.redraw()
}
