
## [2026-11-27T01:55:04.987182+00:00] scan
New: build-0.md, build-1.md, concept.md, editor-keymap.md, export-plan.md, todo.md, viewer-plan.md


## [2026-11-27T12:56:00+11:00] maintain
Read all 7 source documents (build-0.md, build-1.md, concept.md, editor-keymap.md, export-plan.md, todo.md, viewer-plan.md).
Created 8 entity pages (pairti-lens, gogpu, typst, lens-toml, library-and-projects, goldmark-and-markdown, lens-editor, embedded-shell), 5 topic pages (single-compile-rendering, build-roadmap, viewer-verification-plan, editor-keymap-and-modes, ui-layout-and-surfaces), overview.md, and index.md.

## [2026-11-27T17:23:00+11:00] learn
Reviewed 1 bead (`lens-qr5`).
Wiki-worthy: `lens-qr5` (executable entry point and command tests moved from repo root into `cmd/pairti-lense/`).
Updated: `wiki/entities/pairti-lens.md`.
Git evidence used: `79c786a` (`Makefile`, `README.md`, and file move summary).

## [2026-11-27T17:46:00+11:00] learn
Reviewed 5 beads (`lens-iqx`, `lens-iqx.1`, `lens-iqx.2`, `lens-iqx.3`, `lens-iqx.4`).
Wiki-worthy: `lens-iqx`, `lens-iqx.1`, `lens-iqx.2`, `lens-iqx.3`, `lens-iqx.4` (pure-Go `CGO_ENABLED=0` AppKit menu bar and Finder `Open With` via `goffi`, `OnUpdate` platform-thread ownership for menu refreshes, `menuInput` Darwin shortcut deduplication, and `(project path, report ID)` Reports submenu).
Created: `wiki/topics/macos-native-integration.md`.
Updated: `wiki/entities/gogpu.md`, `wiki/index.md`.
Git evidence used: `14a53a2`, `b7bd8f3`, `492a578`, `f70b5e6`, `c7c3a5a`, `e056a82`.


## [2026-11-28T09:08:00+11:00] learn
Reviewed 1 bead (`lens-vij`).
Wiki-worthy: `lens-vij` (standalone `GOOS=js GOARCH=wasm` Raw editor in `cmd/lens-wasm`, reuse vs. desktop-only boundary, `//go:build !js` isolation on `internal/ui/console.go`, GoGPU WebGPU adapter requirement despite CPU paint fallback, canvas focus restoration, and Blob-based file I/O).
Created: `wiki/topics/browser-wasm-editor.md`.
Updated: `wiki/entities/gogpu.md`, `wiki/entities/lens-editor.md`, `wiki/index.md`.
Git evidence used: `db63d44` (`Makefile`, `README.md`, `cmd/lens-wasm/index.html`, `cmd/lens-wasm/main_js.go`, `internal/ui/console.go`).

## [2026-11-27T22:29:45.168395+00:00] scan
Changed: build-0.md

## [2026-11-28T09:31:00+11:00] maintain
Read all 7 source documents (`build-0.md`, `build-1.md`, `concept.md`, `editor-keymap.md`, `export-plan.md`, `todo.md`, `viewer-plan.md`) and verified all 15 entity/topic pages against `doc/build-0.md` changes (all existing entity/topic pages already match current source facts).
Updated: `wiki/overview.md` (added links to `macOS Native Integration` and `Browser WASM Editor`, refreshed `updated` date).


## [2026-11-28T09:38:00+11:00] learn
Reviewed 5 beads (`lens-dws`, `lens-ezv`, `lens-79c`, `lens-8hn`, `lens-c9n`).
Wiki-worthy: `lens-dws` (loopback Git-backed Markdown library server `cmd/lens-library` with `git ls-files --stage -z` discovery, SHA-256 `ETag`/`If-Match` optimistic concurrency, and atomic writes), `lens-ezv` (shared desktop shell compiled to WASM in `cmd/pairti-lense` with `startup_js.go`/`startup_native.go` split and `browserFiles` async HTTP bridge), `lens-79c` (Canvas2D paint overlay via `app.RenderModeHostManaged` + `widget.RegisterSceneRecorder(nil)` and `Editor.lineColors` syntax span caching on resize), `lens-c9n` (user-configured versioned `brand.json` report brand packs under `os.UserConfigDir()/pairti-lense/brands/<id>/` with neutral `default` fallback and generalized `brand.typ`/`docx.lua`).
Skipped: `lens-8hn` (routine public fixture and internal config sanitization).
Created: `wiki/topics/report-brand-packs.md`.
Updated: `wiki/topics/browser-wasm-editor.md`, `wiki/entities/gogpu.md`, `wiki/topics/single-compile-rendering.md`, `wiki/index.md`.
Git evidence used: `203ebda`, `9a0513e`, `0fd28a9`, `7654874`.
