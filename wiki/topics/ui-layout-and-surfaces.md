---
title: UI Layout and Surfaces
type: topic
sources: [doc:///concept.md, doc:///viewer-plan.md, beads]
related: [entities/gogpu.md, entities/library-and-projects.md, entities/goldmark-and-markdown.md, topics/viewer-verification-plan.md, topics/browser-wasm-editor.md]
updated: 2026-11-28
---

# UI Layout and Surfaces

[[Pairti Lens]] shares one minimalist Go workspace layout across desktop ([[GoGPU]]) and browser ([[Browser WASM Editor]]), designed to keep project context visible, switching cheap, and scrolling/focus predictable. Because `gogpu/ui`'s built-in tree and split widgets failed hit testing and resize behavior, those surfaces are custom-implemented.

## Details
- **Files sidebar**:
  - Keeps multiple project folders open in one window and shows enough project and relative path context to distinguish two notes with the same title.
  - Tree filter matches file names, titles, and paths while preserving project context on matching rows; clearing restores the full tree.
  - Separate toggles control hidden (dot-leading) and gitignored files while keeping generated folders such as `.git` hidden.
  - Supports tagging files and filtering by tag, alongside body search that jumps straight to a matching passage.
- **Document surfaces**:
  - **Raw & Formatted views**: share one buffer, cursor, selection, and undo history; Formatted renders Markdown blocks (headings shown once, wrapped lines, tables, links, local images, Mermaid fences with fallback) or syntax-highlighted source code; Raw shows Markdown source.
  - **Tabs & split panes**: opening a second note or splitting the view preserves the first draft and its scroll position; commands affect the focused pane while the other pane keeps its place; closing a dirty note protects unsaved edits.
  - **Outline panel**: right-hand Table of contents panel jumps directly to headings and integrates with back/forward navigation.
- **Footer, scrolling & resize**:
  - Footer fits on a single row from the first frame.
  - Sidebar and document surfaces scroll independently on the targeted surface, and narrower window widths preserve text wrapping and caret visibility without clipping.

## Decisions & Tradeoffs
- **Custom tree and split surfaces**: Replacing `gogpu/ui`'s built-in tree and split widgets fixed hit testing and window resize behavior while keeping the pure-Go stack.
- **Heading navigation lives in the outline rail, not tab strips** (`lens-c8t`): Section breadcrumbs and disclosure controls were removed from both document tab strips (`internal/ui/tabs.go`, `cmd/pairti-lense/main_docs.go`, and `Rail.SectionName` in `internal/ui/rail.go`). Tab strips render only open document tabs; heading navigation belongs exclusively in the right-hand Table of contents panel.

## See Also
- [[GoGPU]]
- [[Library and Projects]]
- [[Viewer Verification Plan]]
- [[Browser WASM Editor]]
