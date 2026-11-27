---
title: GoGPU
type: entity
sources: [doc:///concept.md, doc:///build-0.md, beads]
related: [entities/pairti-lens.md, topics/build-roadmap.md, topics/ui-layout-and-surfaces.md, topics/macos-native-integration.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# GoGPU

GoGPU is the pure-Go GPU UI stack chosen for [[Pairti Lens]], consisting of `gogpu/ui` for widgets, `gogpu/gogpu` for windowing, and `gogpu/gg` for 2D drawing. It keeps the application free of Electron, web views, and C dependencies (`CGO_ENABLED=0`).

## Key Facts
- **Go version requirement**: Go 1.25 or newer (required by `gogpu/ui`).
- **Pinned versions**: `github.com/gogpu/ui` is pinned at `v0.1.55` along with the matching `gogpu` and `gg` versions; floating `@latest` is forbidden because it has broken builds for other users.
- **Excluded dependency**: `gogpu/editor` is early and explicitly not imported; the Markdown editing buffer is owned by Pairti Lens.
- **Known weak spots** (as of November 2026): non-Latin text, IME, and Wayland input; the toolkit lacks a rich-text widget, so Markdown blocks are drawn as text in a scroll view.
- **Fallback shell**: Gio was designated as the fallback if Build 0 rejected GoGPU.
- **Build 0 decision — Stay**: The window opened on macOS, selected the M5 Pro Metal adapter, and listed real project folders. Because the toolkit's built-in tree and split widgets failed hit testing and window resize behavior, those two surfaces are custom-built while the rest of the shell stays on `gogpu/ui` `v0.1.55`.
- **Darwin thread & event loop behavior** (`lens-iqx.3`, `lens-iqx.4`):
  - `gogpuApp.OnUpdate` executes on the platform event thread before `OnDraw` dispatches the UI frame to the render thread; AppKit menu rebuilds must run in `OnUpdate`, not in `Pulse` render ticks (otherwise signed bundles hit an AppKit main-thread exception).
  - The Darwin event loop queues a physical key event and printable text before forwarding the same `NSEvent` to AppKit, so native menu shortcuts must be filtered out of `gogpuApp.EventSource()` (`menuInput`) to avoid duplicate dispatch.
  - Native macOS menu state, key equivalents, and Finder `Open With` handlers use pinned `goffi` alongside GoGPU callbacks, keeping `CGO_ENABLED=0`.
- **Browser/WASM target (`lens-vij`, `lens-79c`)**: `gogpu` `v0.54.1` includes browser platform/renderer support (`app_run_browser.go`, `renderer_browser.go`, `internal/platform/platform_browser.go`) that binds to a `<canvas>` and runs under `GOOS=js GOARCH=wasm CGO_ENABLED=0`. Startup requires a WebGPU adapter (`navigator.gpu.requestAdapter()`), even though GoGPU logged a CPU rendering fallback during painting in Chrome testing. To reduce click-to-paint latency in the shared WASM shell (`cmd/pairti-lense/canvas_js.go`), `runBrowser` uses `app.RenderModeHostManaged` with `widget.RegisterSceneRecorder(nil)` (disabling software boundary scene rasterization in WASM), keeping GoGPU's `<canvas>` for input while painting the UI onto a stacked Canvas2D overlay via `win.DrawTo`.

## How It Works
`gogpu/gogpu` opens the native OS window and binds the GPU backend (Metal on macOS). `gogpu/ui` supplies standard shell widgets while custom surfaces (project tree, split panes, scrollable Markdown page, and report page images) draw directly onto the shared `gogpu/gg` surface. macOS folder picking uses `osascript` so the binary remains `CGO_ENABLED=0`.

## Relationships
- Powers the desktop shell of [[Pairti Lens]].
- Evaluated and accepted during Build 0 in [[Build Roadmap]].
- Custom tree and split surfaces are detailed in [[UI Layout and Surfaces]].
- macOS menu bar, shortcut deduplication, and `OnUpdate` thread ownership are detailed in [[macOS Native Integration]].
- Browser WASM runtime behavior and JS bridge are detailed in [[Browser WASM Editor]].
