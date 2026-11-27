---
title: Lens Editor
type: entity
sources: [doc:///build-1.md, doc:///editor-keymap.md, doc:///viewer-plan.md, doc:///todo.md, beads]
related: [entities/pairti-lens.md, entities/goldmark-and-markdown.md, topics/editor-keymap-and-modes.md, topics/viewer-verification-plan.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# Lens Editor

Lens Editor is the custom, app-owned modal text editor in [[Pairti Lens]] that allows in-place editing of Markdown and source files without importing `gogpu/editor`. Both **Formatted** and **Raw** views share the same underlying buffer, cursor, selection, undo history, and editing mode.

## Key Facts
- **Shared state**: Raw (Markdown source) and Formatted (editable rendered note or highlighted source) operate on one buffer; each mode remembers its own line-number gutter preference.
- **Two primary modes**: **View** (Vim Normal mode) and **Edit** (Vim Insert mode), plus characterwise and linewise Visual selections.
- **Dirty protection**: unsaved edits mark the buffer dirty (`Unsaved` on the status/format line), block switching or closing a dirty tab, and surface a Save button; `:q!` or Escape (in Build 1) discards changes.
- **Save behavior**: `:w`, `Cmd-S`/`Ctrl-S`, or the Save button writes the buffer back to the same file path and refreshes the sidebar title from the first heading.
- **Caret visibility**: opening a file places a caret on line 1 with line highlight; the caret remains visible on blank lines.
- **Status vs roadmap gap**: `doc/editor-keymap.md` specifies full Vim-like motions, operators, Visual mode, undo/redo (`u`/`Ctrl-R`/`Cmd-Z`), and mouse drag selection, while `doc/todo.md` and `doc/viewer-plan.md` Phase 2 track undo, pointer selection drag, and markup inserts (code, link, image, table, footnote) as verification/implementation items.
- **In-memory reuse boundary (`lens-vij`)**: `internal/edit/buffer.go` and `internal/ui/editor.go` are decoupled from OS filesystem I/O (`internal/edit/file.go`), allowing `cmd/lens-wasm` to run the Raw editor in a browser by supplying UTF-8 text and a download save callback (`ui.NewEditor(nil, save)`). `internal/ui/console.go` is excluded on `js` via `//go:build !js`.

## How It Works
The editor owns the text buffer in Go and processes keyboard and mouse events across Formatted and Raw surfaces. Operators (`c`, `d`, `y`) combine with motions, counts, text objects (`iw`/`aw`/`iW`/`aW`), Visual selections (started via `v`, `V`, or Lens's `Ctrl-W` shortcut), or Ex line ranges (`:12,18d`, `:'<,'>y`).

## Relationships
- Full keybindings and Vim comparison are documented in [[Editor Keymap and Modes]].
- Edits files managed by [[Goldmark and Markdown]].
- Checked against canvas-level UI tests and manual window checks in [[Viewer Verification Plan]].
- Runs standalone in a browser via WebAssembly in [[Browser WASM Editor]].
