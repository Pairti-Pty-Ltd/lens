---
title: Viewer Verification Plan
type: topic
sources: [doc:///viewer-plan.md, doc:///todo.md, beads]
related: [entities/pairti-lens.md, entities/lens-editor.md, entities/embedded-shell.md, topics/ui-layout-and-surfaces.md, topics/build-roadmap.md]
updated: 2026-11-28
---

# Viewer Verification Plan

The Viewer Verification Plan defines a strict release gate for [[Pairti Lens]]: passing `go test` alone is insufficient for visual features. Every user-visible item requires explicit canvas-level tests and/or manual window inspection with recorded evidence before advancing to later phases.

## Details
- **Two verification types**:
  - **Test**: drives the widget and asserts what is drawn on the canvas (text string, colour, caret rect, bounds)—never just a return value.
  - **Window**: launches the real app, performs the user action on real files, and records the file used and what appeared on screen.
- **Phase 1 — Core viewer sign-off (implemented, awaiting window sign-off)**:
  1. **Search**: sidebar field paints unclipped characters and filters project names, titles, and paths; clearing restores tree.
  2. **Hidden and Ignored**: two 28px icon slots (eye for dotfiles, slashed circle for gitignored; grey = hidden, blue = shown; `.git` and `node_modules` stay hidden; no text labels; `TestProjectTogglesChangeTheTree`).
  3. **Markdown page**: headings, paragraphs, lists, quotes, tables, fenced code, links; title shown once; long lines wrap (`testdata/membrane-assay.md` + real note).
  4. **Images and diagrams**: local images inline (shrunk only if wider than column); `mermaid` fences render via `mmdc` or show `mmdc not installed`.
  5. **Source files**: `.ts`, `.go`, etc. render as syntax-highlighted code (blue keywords, grey comments with `*` preserved, green strings; `TestPageHighlightsSource`).
  6. **Caret and vi**: caret on line 1 with line highlight, visible on blank lines; `h j k l`, `i`, Escape, `:w`, `:q`.
  7. **Formatted and Raw**: mode toggle with per-mode line-number memory; Save appears when dirty and writes same path.
  8. **Tabs and split**: tab per open file; drag-right or Split icon toggles second pane; closing dirty tab refused with `Unsaved`.
  9. **Outline**: right-hand icon opens table of contents; clicking a heading scrolls preview to that line.
  10. **Footer**: single status row from frame 1 without growing or waiting for resize (`TestFooterStaysOneLine`).
  11. **Scroll**: macOS natural scroll direction, one short step per trackpad tick on both file list and document.
- **Phase 2 — Missing viewer features (starts only after Phase 1 window sign-off)**:
  - Search inside files, New document, Undo & pointer selection drag, Markup inserts (code/link/image/table/footnote), Back/Forward navigation, Settings screen (beyond `theme.json`), Tags, and Typst Report export (requires `typst` on `PATH`).
- **Phase 3 — Embedded Shell (starts only after Phase 1 window checks are closed)**:
  - Fix prompt font glyphs and wire command output display (`echo` test + `pwd` window check).
- **Out of viewer gate**: language-model agents, Word export, and HTML export remain behind the report milestone.

## Decisions & Tradeoffs
- **Parking the shell**: Halting [[Embedded Shell]] work prevents splitting effort while core viewer regressions (rendering, hit testing, footer height, scroll feel) remain unverified on screen.
- **Canvas-level test assertions**: Requiring tests to inspect drawn strings, colours, and caret rects catches layout and clipping bugs that logic-only unit tests miss.
- **macOS live-window automation permission gate** (`lens-c8t`): Automated native window checks against `Lens.app` fail when macOS screen capture and input permissions are absent (`CGPreflightScreenCaptureAccess`, `CGPreflightPostEventAccess`, and `AXIsProcessTrusted` all return `false`, producing *"Computer Use permissions are not granted"*). In headless/unprivileged runs, canvas-level `CGO_ENABLED=0 go test ./...` suites can verify rendering and rail heading navigation, while live native window sign-off requires an environment with macOS Screen Recording and Accessibility permissions granted.

## See Also
- [[UI Layout and Surfaces]]
- [[Build Roadmap]]
- [[Embedded Shell]]
