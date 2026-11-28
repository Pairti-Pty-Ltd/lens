
## [2026-11-27T23:52:55.076474+00:00] scan
Changed: concept.md, export-plan.md, viewer-plan.md

## [2026-11-28T10:55:00+11:00] maintain
Synced wiki pages with updated `doc/concept.md`, `doc/export-plan.md`, `doc/viewer-plan.md`, and `doc/editor-keymap.md`; removed stale references to deleted `doc/build-0.md`, `doc/build-1.md`, `doc/todo.md`, and `wiki/topics/build-roadmap.md`.
Updated: `wiki/overview.md`, `wiki/index.md`, `wiki/entities/pairti-lens.md`, `wiki/entities/gogpu.md`, `wiki/entities/typst.md`, `wiki/entities/lens-toml.md`, `wiki/entities/library-and-projects.md`, `wiki/entities/goldmark-and-markdown.md`, `wiki/entities/lens-editor.md`, `wiki/entities/embedded-shell.md`, `wiki/topics/single-compile-rendering.md`, `wiki/topics/viewer-verification-plan.md`, `wiki/topics/editor-keymap-and-modes.md`, `wiki/topics/ui-layout-and-surfaces.md`, `wiki/topics/browser-wasm-editor.md`, `wiki/topics/report-brand-packs.md`.

## [2026-11-27T23:58:12.512915+00:00] scan
Changed: concept.md, viewer-plan.md


## [2026-11-28T10:59:00+11:00] maintain
Synced wiki with updated `doc/concept.md` and `doc/viewer-plan.md` (local desktop MCP agent `document`/`apply_edit` flow, external disk-write reload/reconciliation, and 13-surface viewer verification matrix).
Created: `wiki/entities/local-agent-mcp.md`.
Updated: `wiki/overview.md`, `wiki/index.md`, `wiki/entities/lens-editor.md`, `wiki/topics/viewer-verification-plan.md`.


## [2026-11-28T11:01:00+11:00] learn
Reviewed 3 closed beads (`lens-19f`, `lens-3h3`, `lens-zub`).
- Wiki-worthy: `lens-19f` (Spotlight/LaunchServices duplicate bundle pitfall), `lens-3h3` (macOS launch-to-first-frame benchmarks and per-directory Git ignore batching).
- Skipped: `lens-zub` (documentation consolidation with no recorded notes/design insights).
- Git evidence used: commits `864be61` (`internal/doc/list.go`) and `6a9e838` (`Makefile`).
- Created: `wiki/topics/macos-startup-performance.md`.
- Updated: `wiki/topics/macos-native-integration.md`, `wiki/entities/library-and-projects.md`, `wiki/index.md`.


## [2026-11-28T16:02:00+11:00] learn
Reviewed 1 closed bead (`lens-idf`).
- Wiki-worthy: `lens-idf` (custom Goldmark parsers for single-column Pandoc grid cells and `[...]{.mark}` spans, preserving document-relative source positions and Formatted-view highlight/wrap behavior).
- Git evidence used: commit `345e4d1` (`internal/doc/doc.go`, `internal/doc/render.go`, `internal/ui/page.go`, `internal/ui/table.go`).
- Updated: `wiki/entities/goldmark-and-markdown.md`.


## [2026-11-28T16:32:00+11:00] learn
Reviewed 2 closed beads (`lens-ckd`, `lens-ry4`).
- Wiki-worthy: `lens-ckd` (continuous multi-paragraph blockquote panels keyed by `Block.QuoteLine`, plus `quotePad` parity in `pointSource`/`moveScreen`), `lens-ry4` (source-whitespace `word`/`gap` wrapping across inline span boundaries replacing `tightPunct`/`stuckTo`, and unified `pieceWidth` advance measurement between layout and `Draw`).
- Git evidence used: commit `e366827` (`internal/doc/doc.go`, `internal/doc/render.go`, `internal/ui/page.go`, `internal/ui/raw.go`).
- Updated: `wiki/entities/goldmark-and-markdown.md`, `wiki/topics/ui-layout-and-surfaces.md`.
