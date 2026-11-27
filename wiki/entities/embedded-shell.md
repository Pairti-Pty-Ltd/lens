---
title: Embedded Shell
type: entity
sources: [doc:///viewer-plan.md]
related: [entities/pairti-lens.md, topics/viewer-verification-plan.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# Embedded Shell

The Embedded Shell (desktop terminal panel) is an in-app terminal surface in [[Pairti Lens]] on desktop (`internal/term` / `internal/ui/console.go`). It lets writers run shell commands without leaving the workspace and is excluded from browser WebAssembly builds (`//go:build !js`).

## Key Facts
- **Desktop-only surface**: runs a local PTY/subprocess on desktop; excluded from `GOOS=js GOARCH=wasm` builds so `internal/ui` remains WASM-safe.
- **Verification criteria** (`doc/viewer-plan.md`):
  - Focus the terminal, run `pwd`, then return focus to the document.
  - The prompt must be legible, `pwd` must print the working directory, and subsequent typing must route cleanly to whichever surface has focus.
- **Menu & clipboard routing**: on macOS, edit and clipboard commands route to the PTY terminal when it is the focused surface (`cmd/pairti-lense/main_menu_edit.go`).

## How It Works
The desktop terminal panel spawns an interactive shell process inside the Pairti Lens window and shares focus routing with the document editor and text fields so keyboard input always targets the active surface.

## Relationships
- Verified under Terminal on desktop in [[Viewer Verification Plan]].
- Excluded at the desktop/browser boundary in [[Browser WASM Editor]].
- Participates in focus-aware Edit menu routing in [[macOS Native Integration]].
