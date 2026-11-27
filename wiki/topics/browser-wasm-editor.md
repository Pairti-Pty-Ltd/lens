---
title: Browser WASM Editor
type: topic
sources: [beads]
related: [entities/gogpu.md, entities/lens-editor.md, entities/pairti-lens.md, topics/single-compile-rendering.md, topics/report-brand-packs.md]
updated: 2026-11-28
---

# Browser WASM Editor

Pairti Lens includes a standalone WebAssembly build (`cmd/lens-wasm`) that runs the existing Raw [[Lens Editor]] locally in a browser via `make wasm-serve`. Introduced in `lens-vij`, this experiment establishes which parts of the desktop editor work directly on `GOOS=js GOARCH=wasm` and defines the boundary between reusable editing logic and desktop-only services.

## Details

### Reuse vs. Desktop-Only Boundary
- **Reusable core**: `internal/edit/buffer.go` constructs text buffers purely in memory, and `internal/ui/editor.go` draws the buffer, handles modal input, and exposes a save callback (`ui.NewEditor(nil, save)`) plus Ex command hook (`editor.SetCommand(...)` for `:w`).
- **Excluded desktop dependencies**: `cmd/pairti-lense/main.go` (library, theme, and session loading plus project folder rescans), `internal/edit/file.go` (OS path writes), `internal/watch` (`fsnotify`), `internal/gitlog` (Git CLI), `internal/term` (PTY), `internal/report` (external export tools), and native menus/tray stay outside the browser startup path.
- **Build-tag isolation**: `internal/ui/console.go` requires `//go:build !js` so importing `internal/ui` from `cmd/lens-wasm` does not pull in desktop terminal/PTY dependencies.

### GoGPU Browser Runtime & JS Bridge
- **GoGPU support**: Pinned `gogpu` `v0.54.1` provides `app_run_browser.go`, `renderer_browser.go`, and `internal/platform/platform_browser.go`, locating or creating a `<canvas>` and attaching keyboard listeners. `cmd/lens-wasm/main_js.go` runs the editor through `desktop.Run(host, window)`.
- **JS-Go interop (`syscall/js`)**: `main_js.go` exposes `importMarkdown(name, text)`, `saveMarkdown()`, and `hasEdits()` on `js.Global()`, and invokes JS `downloadMarkdown(name, text)` and `showError(msg)`.
- **File I/O without host filesystem APIs**: `cmd/lens-wasm/index.html` reads user-selected `.md`/`.markdown` files into memory with `TextDecoder('utf-8', {fatal: true, ignoreBOM: true})` and exports edits via a UTF-8 `Blob` download. Direct folder APIs (`showDirectoryPicker`) were intentionally avoided due to uneven browser support and permission prompts.
- **Dirty-buffer protection**: `hasEdits()` (`editor.Buffer().Dirty()`) gates both re-import confirmation (`confirm(...)`) and tab close (`beforeunload`).

### Build & Verification
- `make wasm` compiles `./cmd/lens-wasm` with `GOOS=js GOARCH=wasm CGO_ENABLED=0` into `build/wasm/lens.wasm` and copies `$(go env GOROOT)/lib/wasm/wasm_exec.js` and `cmd/lens-wasm/index.html`.
- `make wasm-serve` serves `build/wasm/` on `http://127.0.0.1:8080/` using `python3 -m http.server`.
- Verified on macOS 26.6.2 (arm64), Chrome 158.0.8127.88, and Go 1.27.4: ASCII edits, Visual selection (`v`, `l`), delete (`d`), undo (`u`), insert (`i`, `Escape`), and UTF-8 round-trip (en dash and line breaks) all worked.

### Git-Backed Library Server (`lens-dws`)
- **Loopback server**: `cmd/lens-library/main.go` serves a single Git working tree on `127.0.0.1:8081` (built and run by `make wasm-serve`, with `LIBRARY` defaulting to `$(CURDIR)`). It exposes `/library/` (Git-backed workspace) and `/local/` (standalone import/download), plus JSON/text endpoints `GET /api/documents`, `GET /api/document?path=...`, and `PUT /api/document?path=...`.
- **Tracked Markdown discovery**: Lists documents via `git ls-files --stage -z`, accepting only stage `0` regular file modes (`100644`/`100755`) that are valid UTF-8 Markdown paths and verified as non-symlink regular files through `os.OpenRoot` + `Lstat` so symlinks and conflicted index entries never become editable.
- **Optimistic concurrency & atomic writes**: `GET` returns a SHA-256 `ETag`; `PUT` requires a matching `If-Match` header or fails with `412 Precondition Failed` (`"File changed on the server. Download your edits before reopening."`). Writes are serialized under a mutex, validated as UTF-8 up to `1 MiB` (`doc.MaxBytes`), written to a sibling temp file (`.lens-<rand>`) with original permissions, and swapped via `Rename`.
- **Browser security**: Enforces `Host` allowlisting (`localhost:8081` / `127.0.0.1:8081`), `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Cache-Control: no-store`, and `http.NewCrossOriginProtection()`.

### Shared Desktop Shell in WASM (`lens-ezv`)
- **Unified entry point**: Replaced the standalone `cmd/lens-wasm` package by compiling `./cmd/pairti-lense` directly with `GOOS=js GOARCH=wasm CGO_ENABLED=0` (`make wasm` now copies `cmd/pairti-lense/web/index.html`). Desktop vs. browser startup is split via build tags (`startup_native.go` vs. `startup_js.go`).
- **Reused workspace**: Browser library mode runs the shared desktop sidebar, toolbar, tabs, Formatted/Raw views, editor buffers, and menu commands (filtered by `browserCommand` and rendered in the HTML `<nav>` via `menuItems`/`invokeMenu`). Standalone `/local/` mode preserves local file import/download and dirty-buffer protection.
- **Async HTTP file bridge**: `cmd/pairti-lense/browser_files.go` (`browserFiles`) routes document opens and saves through JS Promises (`readMarkdown`, `writeMarkdown`). Because browser reads finish after the input event returns, `awaitDocument` increments a monotonic `request` counter to drop stale responses and re-enters the normal tab-opening path so dirty-tab confirmation runs at completion. `browserResult` releases both `js.Func` callbacks on resolve and reject to avoid WASM callback leaks.

### Canvas2D Rendering & Resize Performance (`lens-79c`)
- **Canvas2D paint overlay**: `cmd/pairti-lense/canvas_js.go` (`runBrowser`) replaces `desktop.Run` in WASM mode. GoGPU's `<canvas>` remains the focusable input surface, while a pointer-transparent (`pointer-events: none`) `<canvas id="paint">` stacked on top paints the UI via `app.RenderModeHostManaged`, `win.DrawTo(c)`, and the browser's native Canvas2D rasterizer scaled by `devicePixelRatio`.
- **Disable software scene recording**: `runBrowser` calls `widget.RegisterSceneRecorder(nil)` so widgets draw directly; desktop boundary scene recording rasterizes entire boundaries in software under WebAssembly and degrades click-to-paint latency.
- **Syntax span caching on resize**: `internal/ui/source_rows.go` caches per-line highlight spans on `Editor.lineColors` keyed by `(file, text)` so window resizes and sidebar toggles only re-wrap lines instead of re-running `doc.MarkdownLines` or `doc.TokenLines`.

### Tracked Git History & Diffs (`lens-77q.1`)
- **Read-only history routes**: `cmd/lens-library/history.go` serves `GET /api/history/{log,files,diff}` scoped to tracked Markdown files (`doc.IsMarkdown`). For `files` and `diff`, it validates 40/64-character hex commit IDs and enforces `git merge-base --is-ancestor <hash> HEAD` so arbitrary or unreachable Git objects cannot be queried.
- **Off-thread async client**: `cmd/pairti-lense/browser_history.go` (`libraryHistory`) fetches history off the UI thread and applies results on the shell tick via a `pending` channel with a monotonic `generation` counter to drop stale responses.
- **Local mode isolation**: `browserSideAvailable` enables `ui.SideHistory` only when `s.remote.history != nil`, keeping standalone `/local/` imports independent of Git.

### Confined Server-Side Report Publishing & Preview (`lens-77q.2`)
- **Report endpoints**: `cmd/lens-library` exposes `GET /api/reports` (project book/report order and available export formats, passed to WASM via `setReportCatalog`) and `POST /api/report/export?report=<id>&output=<fmt>`, serialized under `exportJob sync.Mutex`.
- **Saved-documents-only & confined staging**: Server exports read saved tracked documents only (never unsaved browser buffer edits), stage inputs and relative images inside a private per-job directory (`LibrarySnapshot`), and reuse `internal/report` plus installed [[Report Brand Packs]] (supporting legacy settings and project profiles).
- **Sandboxed Pandoc DOCX media staging**: When `LibrarySnapshot` is set, `internal/report/docx.go` invokes Pandoc with `--sandbox --lua-filter=stage-media.lua`. Because Pandoc's sandbox disables filesystem reads by the DOCX writer while Lua filters retain host access, `internal/report/brand/snapshot-media.lua` pre-populates `pandoc.mediabag` exclusively from server-generated job files (`logo.png`, `img-<n>.<ext>`, `diag-<n>.png`, `diag-<n>-<m>.png`).
- **Single-blob preview and download**: `cmd/pairti-lense/web/index.html` (`exportReport`) creates one `URL.createObjectURL(blob)` from the `POST /api/report/export` response, rendering PDFs in `<dialog id="publication">`'s `<iframe>` and offering that exact blob URL for download (revoking the URL and restoring canvas focus on close).

### Browser vs. Desktop Capability Boundary (`lens-77q`)
- **Menu and mode gating**: `startup_js.go` (`browserCommand`) and `main.go` expose only backend-verified browser actions (`File/Save`, `File/Close Tab`, `Go/History` when `remote.history != nil`, and `Reports/Export Report…` / `Reports/Open Report` when `remote.export != nil`), marking platform-only commands unavailable in the matching mode.
- **Remaining desktop-only facilities**: Native file/folder pickers and destination reveal, terminal/PTY, system integration, relative attachment loading (requires a constrained library asset service), and clipboard integration (`Edit/Cut`, `Copy`, `Paste` stay disabled in browser mode).

## Pitfalls
- **WebGPU adapter required even with CPU fallback**: During testing, GoGPU logged a CPU rendering fallback while painting, yet startup still requires a working WebGPU adapter (`navigator.gpu.requestAdapter()`).
- **Canvas focus management**: Interacting with HTML controls (`<input type="file">`, download button, or `<dialog id="publication">`) steals keyboard focus from the `<canvas>`. Both JS (`canvas.focus()`) and Go (`window.Window().Context().RequestFocus(editor)` + `host.RequestRedraw()`) must restore focus after import/save/dialog close.
- **Synchronous vs. asynchronous clipboard**: Desktop widget clipboard callbacks are synchronous, whereas browser Clipboard API permissions and reads/writes are asynchronous; bridging them requires rethinking the synchronous widget contract.
- **Pandoc `--sandbox` drops writer file reads**: Running `pandoc --sandbox` for `.docx` prevents the DOCX writer from reading staged images from disk; media must be inserted into `pandoc.mediabag` via a pre-filter (`snapshot-media.lua`) limited to server-created job files.
- **Keep `internal/ui` WASM-safe**: Any new `internal/ui` file that imports OS-specific packages (PTY, `fsnotify`, `os/exec`) must be guarded with `//go:build !js` or isolated outside `internal/ui`, otherwise `make wasm` will break.
- **Untested areas**: IME input was not verified in the browser build; native project/workspace management remains desktop-only.

## See Also
- [[GoGPU]]
- [[Lens Editor]]
- [[Pairti Lens]]
- [[Single-Compile Rendering]]
- [[Report Brand Packs]]
