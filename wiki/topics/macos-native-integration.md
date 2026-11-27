---
title: macOS Native Integration
type: topic
sources: [doc:///concept.md, doc:///viewer-plan.md]
related: [entities/pairti-lens.md, entities/gogpu.md, entities/library-and-projects.md, entities/lens-editor.md]
updated: 2026-11-28
---

# macOS Native Integration

[[Pairti Lens]] integrates with the macOS system menu bar, Finder `Open With`, and AppKit keyboard shortcuts while preserving a pure-Go `CGO_ENABLED=0` build.

## Details

- **Pure-Go AppKit binding**: Native menus (`Lens`, `File`, `Edit`, `Insert`, `View`, `Go`, `Reports`, `Window`, `Help`) combine [[GoGPU]] menu callbacks with AppKit menu state and key equivalents via pinned `goffi` (`cmd/pairti-lense/main_menu*.go`). Command handlers and enabled/checked state are shared with toolbar actions in `main.go`, `main_docs.go`, `main_files.go`, and `main_tray.go`. `doc/editor-keymap.md` is bundled via `Makefile` and opened from `Help > Editor Keymap`.
- **Platform event thread ownership**:
  - GoGPU runs `OnUpdate` on the platform event thread before `OnDraw` dispatches the UI frame to the render thread.
  - All AppKit menu rebuilds/refreshes, Finder open draining (`finderOpens`), and quit handling run in `gogpuApp.OnUpdate(s.updateNativeCommands)` (`cmd/pairti-lense/main_menu.go`), never inside a `ui.NewPulse` render tick.
  - Calling `invalidateMenus()` sets `s.menusDirty = true` and triggers `s.redraw()` so menu changes discovered during a demand-driven draw schedule another main-loop turn even without further user input.
- **Exactly-once shortcut dispatch & focus routing**:
  - `menuInput` (`cmd/pairti-lense/main_menu_input.go`) wraps `gogpuApp.EventSource()` to suppress registered native menu key equivalents (and their accompanying printable text via `skipText`) from the `gogpu/ui` widget event bridge.
  - Edit and clipboard commands (`cmd/pairti-lense/main_menu_edit.go`) route to the focused editor, text field, or PTY terminal, and remain enabled for the focused field even when modal overlays like Rename are open (while other menu groups disable when `s.menuModal()` is true).
- **Finder `Open With` & recent items**:
  - Markdown document types are registered in `Makefile` bundle metadata, and `cmd/pairti-lense/main_finder_darwin.go` registers an Apple Event handler via `goffi`.
  - Cold and warm LaunchServices opens deliver paths on AppKit's main thread into `s.finderOpens`, which `updateNativeCommands` drains through the same `openPath` path used by menu opens. Recent files and project folders persist newest-first in `internal/session/session.go`, disambiguating same-named files by full path and separating project removal from file deletion.
- **Hierarchical Reports menu**:
  - `Reports > Open Report > <project folder> > <report>` is built from `internal/library/library.go` and `internal/project/reports.go` (`cmd/pairti-lense/main_menu_reports.go`).
  - Entries are keyed by `(project path, report ID)` so duplicate report IDs or names across open projects remain distinct. Dynamic submenu paths use NUL (`\x00`) as a group separator (`menuGroup`) because NUL cannot appear in file paths.

## Pitfalls

- **Rebuilding AppKit menus during render crashes the signed app**: Running `refreshMenu()` inside `Pulse.tick` (render thread) throws an AppKit main-thread exception on cold open in the production bundle, even though standalone tests that drive `OnUpdate` directly pass. Always verify menu refreshes under an actual `Pulse` render frame (`TestReportMenuRefreshAfterRender`) and keep AppKit calls in `OnUpdate`.
- **Double-handled macOS shortcuts**: GoGPU's Darwin event loop queues both a physical key event and printable text before forwarding the same `NSEvent` to AppKit. Any native menu key equivalent not filtered by `menuInput` will fire twice (once in AppKit, once in `gogpu/ui`).
- **Demand-driven redraw starvation**: Marking `menusDirty = true` during a draw without requesting a redraw leaves the native menu stale until the next unrelated input event.
