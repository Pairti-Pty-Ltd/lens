package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/widget"
	"pairti-lense/internal/edit"
	"pairti-lense/internal/library"
	"pairti-lense/internal/ui"
)

type menuClipboard struct {
	text string
	err  error
}

func (c *menuClipboard) ClipboardRead() (string, error) { return c.text, c.err }
func (c *menuClipboard) ClipboardWrite(value string) error {
	if c.err == nil {
		c.text = value
	}
	return c.err
}

func TestEditMenuDocument(t *testing.T) {
	for _, view := range []string{"Raw", "Formatted"} {
		t.Run(view, func(t *testing.T) {
			s := testShell()
			const source = "# café\n\n<!-- lens keeps this -->\n"
			path := openNote(t, s, source)
			s.pick(view)
			cb := &menuClipboard{}
			s.clipboard = cb
			commands := s.menuCommands()
			run := func(title string) {
				c := menuByTitle(t, commands, title)
				if !c.enabled() {
					t.Fatalf("%s disabled", title)
				}
				c.invoke()
			}
			run("Select All")
			run("Copy")
			if cb.text != source {
				t.Fatalf("clipboard %q", cb.text)
			}
			cb.err = errors.New("clipboard unavailable")
			run("Cut")
			if s.buf.Text() != source {
				t.Fatal("failed Cut lost text")
			}
			cb.err = nil
			run("Cut")
			if s.buf.Text() != "" {
				t.Fatal("cut left text")
			}
			run("Undo")
			if s.buf.Text() != source {
				t.Fatal("undo cut")
			}
			run("Redo")
			if s.buf.Text() != "" {
				t.Fatal("redo cut")
			}
			run("Paste")
			run("Select All")
			cb.text = "changed\r\n"
			run("Paste")
			if s.buf.Text() != "changed\n" {
				t.Fatalf("paste: %q", s.buf.Text())
			}
			run("Undo")
			if s.buf.Text() != source {
				t.Fatal("replacing selection was not one undo step")
			}
			run("Find")
			cb.text = "café"
			run("Paste")
			if s.buf.Modal.Cmd != "café" || s.buf.Text() != source {
				t.Fatal("search input edited document")
			}
			s.buf.Modal.Handle(s.buf, edit.Key{Code: edit.CodeEnter})
			if s.buf.CaretCol() != 2 {
				t.Fatalf("find landed at %d", s.buf.CaretCol())
			}
			cb.text = "☕"
			run("Paste")
			s.saveDoc()
			reopened := testShell()
			reopened.open(ui.Item{Path: path, Rel: filepath.Base(path), ProjectRoot: filepath.Dir(path)})
			if reopened.buf.Text() != "# ☕café\n\n<!-- lens keeps this -->\n" {
				t.Fatalf("reopened %q", reopened.buf.Text())
			}
		})
	}
}

func TestEditMenuRoutesToFocusedFieldAndConsole(t *testing.T) {
	s := testShell()
	openNote(t, s, "keep this document")
	ctx := widget.NewContext()
	s.menuContext = ctx
	cb := &menuClipboard{text: "hello"}
	s.clipboard = cb
	widget.RegisterClipboardProvider(cb)
	t.Cleanup(func() { widget.RegisterClipboardProvider(nil) })
	commands := s.menuCommands()
	field := textfield.New()
	ctx.RequestFocus(field)
	verifyField := func(f *textfield.Widget) {
		cb.text = "hello"
		menuByTitle(t, commands, "Paste").invoke()
		menuByTitle(t, commands, "Select All").invoke()
		menuByTitle(t, commands, "Cut").invoke()
		if f.Text() != "" || cb.text != "hello" {
			t.Fatalf("field %q clipboard %q", f.Text(), cb.text)
		}
		if menuByTitle(t, commands, "Undo").enabled() {
			t.Fatal("document undo leaked into text field")
		}
	}
	verifyField(field)
	s.rename = ui.NewRename(nil)
	s.rename.Show("")
	ctx.RequestFocus(s.rename.Field())
	verifyField(s.rename.Field().(*textfield.Widget))
	s.rename.Hide()
	s.console = ui.NewConsole()
	dir := t.TempDir()
	receiver := filepath.Join(dir, "receiver")
	script := "#!/bin/sh\nIFS= read -r line\nprintf '%s' \"$line\" > result\n"
	if err := os.WriteFile(receiver, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", receiver)
	if err := s.console.Start(dir, nil); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.console.Close)
	ctx.RequestFocus(s.console)
	cb.text = "terminal clipboard ☕\n"
	menuByTitle(t, commands, "Paste").invoke()
	if s.buf.Text() != "keep this document" || menuByTitle(t, commands, "Select All").enabled() {
		t.Fatal("terminal focus routed edits to document")
	}
	deadline := time.Now().Add(3 * time.Second)
	for {
		got, _ := os.ReadFile(filepath.Join(dir, "result"))
		if string(got) == "terminal clipboard ☕" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("PTY received %q", got)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestInsertMenuMatchesToolbar(t *testing.T) {
	const source = "<!-- untouched -->"
	for _, view := range []string{"Raw", "Formatted"} {
		for _, kind := range []string{"Code", "Link", "Image", "Table", "Footnote", "Tag"} {
			t.Run(view+"/"+kind, func(t *testing.T) {
				run := func(menu bool) (string, bool) {
					s := testShell()
					path := openNote(t, s, source)
					s.pick(view)
					s.lib = &library.Library{}
					s.libPath = filepath.Join(t.TempDir(), "lib.json")
					s.filter = "#native"
					if menu {
						c := menuByTitle(t, s.menuCommands(), kind)
						if !c.enabled() {
							t.Fatal("disabled")
						}
						c.invoke()
					} else {
						s.onTool(kind)
					}
					s.saveDoc()
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					return string(data), len(s.lib.Tags[path]) > 0
				}
				got, tag := run(true)
				want, wantTag := run(false)
				if got != want || tag != wantTag || got != markup(kind)+source || tag != (kind == "Tag") {
					t.Fatalf("%q/%v vs %q/%v", got, tag, want, wantTag)
				}
			})
		}
	}
}
