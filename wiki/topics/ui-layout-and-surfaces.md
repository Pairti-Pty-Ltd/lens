---
title: UI Layout and Surfaces
type: topic
sources: [doc:///build-0.md, doc:///build-1.md, doc:///viewer-plan.md, doc:///todo.md, beads]
related: [entities/gogpu.md, entities/library-and-projects.md, entities/goldmark-and-markdown.md, topics/viewer-verification-plan.md]
updated: 2026-11-28
---

# UI Layout and Surfaces

[[Pairti Lens]] uses a minimalist native layout on [[GoGPU]] designed to answer one question at a glance: *which document in which open project*. Because `gogpu/ui`'s built-in tree and split widgets failed hit testing and resize behavior in Build 0, those core surfaces are custom-implemented.

## Details
- **Information budget & 4-line header**:
  - Displays at most four lines for the selected document: project name, document title (shown once—not repeated between list and body), middle-truncated relative path (segments >40 chars cut), and format (with `Unsaved` appended when dirty).
  - File size, mtime, line count, and raw `#` heading syntax are excluded from the main view.
  - Empty states are single lines: `open a project folder` (empty library) and `no Markdown files` (empty project).
- **Sidebar (project tree & filter)**:
  - Lists open projects and their document titles.
  - Includes a single filter field underneath (matching project names, titles, and paths) and two 28px icon toggles without text labels: an eye icon (dot-leading Hidden files) and a slashed circle (gitignored files), grey when off and blue when on. `.git` and `node_modules` always remain hidden.
- **Document surfaces**:
  - **Formatted / Raw toggle**: Formatted renders Markdown blocks (with inline local images and `mmdc` Mermaid diagrams) or syntax-highlighted source code (blue keywords, grey comments, green strings); Raw shows Markdown source. Each remembers its line-number gutter state.
  - **Tabs and split pane**: every opened file is a tab; dragging a tab right or clicking the Split icon opens a second pane (clicking Split again closes it). Closing a dirty tab is blocked with `Unsaved` in the status line.
  - **Outline panel**: toggled by the right-hand icon; clicking a heading scrolls the preview to that line.
- **Footer & scrolling**:
  - Footer occupies strictly one status row from the first frame without resizing when text arrives.
  - Both file list and document surfaces scroll in macOS natural direction with one short step per trackpad tick.
- **Theming**: theme and fonts currently load from `theme.json` until the Settings gear screen is implemented.

## Decisions & Tradeoffs
- **Custom tree and split surfaces**: Replacing `gogpu/ui`'s tree and split widgets added custom surface code in `main.go`/`internal/ui` but fixed hit testing and window resize without abandoning GoGPU for Gio.
- **No metadata clutter**: Stripping sizes, timestamps, and duplicate titles keeps navigation focused on project + title selection.
- **Heading navigation lives in the outline rail, not tab strips** (`lens-c8t`): Section breadcrumbs and disclosure controls were removed from both document tab strips (`internal/ui/tabs.go`, `cmd/pairti-lense/main_docs.go`, and `Rail.SectionName` in `internal/ui/rail.go`). Tab strips render only open document tabs; heading navigation belongs exclusively in the right-hand Table of contents panel.

## See Also
- [[GoGPU]]
- [[Library and Projects]]
- [[Viewer Verification Plan]]
