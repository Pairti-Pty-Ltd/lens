package main

import (
	"strings"
	"unicode/utf8"

	"github.com/gogpu/ui/core/textfield"
	"github.com/gogpu/ui/event"
	"github.com/gogpu/ui/widget"

	"pairti-lense/internal/doc"
	"pairti-lense/internal/edit"
)

func (s *shell) editMenuCommands() []menuCommand {
	var commands []menuCommand
	for _, action := range []struct{ title, key string }{
		{"Undo", "z"}, {"Redo", "z"}, {"Cut", "x"}, {"Copy", "c"}, {"Paste", "v"}, {"Select All", "a"}, {"Find", "f"},
	} {
		title := action.title
		mods := uintptr(menuCommandKey)
		if title == "Redo" {
			mods |= menuShift
		}
		commands = append(commands, menuCommand{group: "Edit", title: title, key: action.key, mods: mods,
			run: func() { s.performEdit(title) }, enabled: func() bool { return s.canEdit(title) }})
	}
	for _, title := range []string{"Code", "Link", "Image", "Table", "Footnote", "Tag"} {
		commands = append(commands, menuCommand{group: "Insert", title: title, run: func() { s.onTool(title) },
			enabled: func() bool { return s.editableDocument() && !s.menuModal() && s.toolAvailable(title) }})
	}
	return commands
}

func (s *shell) editableDocument() bool {
	return s.buf != nil && s.docPath != "" && s.diff == nil && s.pending == "" && s.viewing == ""
}

func (s *shell) menuFocus() widget.Widget {
	if s.menuContext == nil {
		return nil
	}
	return s.menuContext.FocusedWidget()
}

func (s *shell) canEdit(title string) bool {
	focus := s.menuFocus()
	field, isField := focus.(*textfield.Widget)
	if s.menuModal() && !(isField && s.rename != nil && s.rename.Shown() && s.rename.Field() == field) {
		return false
	}
	if isField {
		from, to := field.Selection()
		switch title {
		case "Copy", "Cut":
			return from != to
		case "Select All":
			return field.Text() != ""
		case "Paste":
			return true
		}
		return false
	}
	if s.console != nil && focus == s.console {
		return s.console.Open() && title == "Paste"
	}
	if !s.editableDocument() || (focus != nil && focus != s.currentPage() && focus != s.currentEditor()) {
		return false
	}
	if s.buf.Modal.Mode == edit.ModeCommand {
		return title == "Paste"
	}
	switch title {
	case "Undo":
		return s.buf.CanUndo()
	case "Redo":
		return s.buf.CanRedo()
	case "Cut", "Copy":
		_, _, ok := s.buf.Selection()
		return ok
	case "Select All":
		return s.buf.Text() != ""
	}
	return true
}

func (s *shell) performEdit(title string) {
	if field, ok := s.menuFocus().(*textfield.Widget); ok {
		// The pinned textfield uses Ctrl for its editing commands on all platforms.
		key := map[string]event.Key{"Copy": event.KeyC, "Cut": event.KeyX, "Paste": event.KeyV, "Select All": event.KeyA}[title]
		field.Event(s.menuContext, event.NewKeyEvent(event.KeyPress, key, 0, event.ModCtrl))
		return
	}
	var paste string
	if title == "Paste" {
		if s.clipboard == nil {
			return
		}
		text, err := s.clipboard.ClipboardRead()
		if err != nil {
			s.showMessage(err.Error())
			return
		}
		if text == "" {
			return
		}
		if s.console != nil && s.menuFocus() == s.console {
			s.console.Paste(text)
			return
		}
		paste = strings.ReplaceAll(text, "\r\n", "\n")
	}
	b := s.buf
	switch title {
	case "Undo", "Redo":
		b.Modal.Handle(b, edit.Key{Rune: 'z', Super: true, Shift: title == "Redo"})
	case "Select All":
		b.Modal.Drag(b, 0, utf8.RuneCountInString(b.Text()))
	case "Cut", "Copy":
		from, to, ok := b.Selection()
		if !ok || s.clipboard == nil {
			return
		}
		if err := s.clipboard.ClipboardWrite(string([]rune(b.Text())[from:to])); err != nil {
			s.showMessage(err.Error())
			return
		}
		if title == "Copy" {
			return
		}
		b.EndGroup()
		b.DeleteSelection()
		b.Modal.Reset()
	case "Paste":
		if b.Modal.Mode == edit.ModeCommand {
			b.Modal.Cmd += paste
			s.syncCommand()
			return
		}
		inEdit := b.Modal.Mode == edit.ModeEdit
		b.EndGroup()
		b.BeginGroup()
		b.DeleteSelection()
		b.InsertText(paste)
		b.EndGroup()
		b.Modal.Reset()
		if inEdit {
			b.Modal.Mode = edit.ModeEdit
			b.BeginGroup()
		}
	case "Find":
		b.Modal.StartCommand('/')
	}
	if s.editing {
		s.currentEditor().RevealCaret()
		s.noteEdit()
	} else {
		p := s.currentPage()
		if title != "Select All" && title != "Find" {
			p.SetBlocks(doc.Render(s.docPath, []byte(b.Text())))
		}
		p.Reopen(b.CaretLine() + 1)
		s.notePage(s.focus)
	}
	s.focusDoc()
}
