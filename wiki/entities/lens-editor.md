---
title: Lens Editor
type: entity
sources: [doc:///concept.md, doc:///editor-keymap.md, doc:///viewer-plan.md, beads]
related: [entities/pairti-lens.md, entities/goldmark-and-markdown.md, entities/local-agent-mcp.md, topics/editor-keymap-and-modes.md, topics/viewer-verification-plan.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# Lens Editor

Lens Editor is the custom, app-owned modal text editor in [[Pairti Lens]] (`internal/edit` and `internal/ui`). Both **Raw** and **Formatted** views share one underlying Markdown buffer, cursor, selection, undo history, and editing mode.

## Key Facts
- **Single editing history**: switching between Raw and Formatted preserves the same edit, cursor position, selection, and undo stack; unfamiliar Markdown markup is preserved intact.
- **Two primary modes**: **View** (Vim Normal mode) and **Edit** (Vim Insert mode), plus characterwise and linewise Visual selections and mouse drag selection.
- **Dirty & conflict protection**:
  - Closing a dirty note protects the unsaved draft; `:q!` discards unsaved changes by reloading from disk.
  - External disk writes reload a clean buffer after the write settles; if the writer has an unsaved draft, Lens surfaces the change and keeps the writer's edits available for reconciliation.
  - On desktop, [[Local Agent MCP]] (`document` and `apply_edit`) can read unsaved buffer text/selection and apply an edit as one undo step while refusing stale edits and leaving saving to the writer.
  - In browser library mode, saving against a stale server revision (`412 Precondition Failed`) leaves the draft recoverable.
- **Save & creation controls**: `:w`, `Cmd-S`/`Ctrl-S`, or the Save action writes directly to the local file (desktop) or served Git checkout (browser). Editing controls also support creating new notes at a chosen location and inserting structures such as tables.
- **Caret visibility**: moving through blank lines keeps the caret visible; caret bounds are checked via canvas assertions.
- **WASM reuse boundary (`lens-vij`, `lens-ezv`)**: `internal/edit/buffer.go` and `internal/ui/editor.go` stay decoupled from OS filesystem I/O, allowing `cmd/pairti-lense` to run the same editor in desktop and WebAssembly builds.

## How It Works
The editor owns the Markdown buffer in Go (`internal/edit`) and processes modal keyboard and pointer input across Formatted and Raw surfaces (`internal/ui`). Operators (`c`, `d`, `y`) combine with motions, counts, text objects (`iw`/`aw`/`iW`/`aW`), Visual selections (`v`, `V`, `Ctrl-W`, or mouse drag), and Ex line ranges (`:12,18d`, `:'<,'>y`).

## Relationships
- Full keybindings and Vim comparison are documented in [[Editor Keymap and Modes]].
- Edits files parsed by [[Goldmark and Markdown]].
- Verified under Caret and modes, Saving, and Editing controls in [[Viewer Verification Plan]].
- Shared directly with [[Browser WASM Editor]].
