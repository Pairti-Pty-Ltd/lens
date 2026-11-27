---
title: Viewer Verification Plan
type: topic
sources: [doc:///viewer-plan.md, doc:///concept.md, beads]
related: [entities/pairti-lens.md, entities/lens-editor.md, entities/embedded-shell.md, topics/ui-layout-and-surfaces.md, topics/single-compile-rendering.md]
updated: 2026-11-28
---

# Viewer Verification Plan

The Viewer Verification Plan (`doc/viewer-plan.md`) defines how changes to [[Pairti Lens]] are judged: by what the writer can see, save, reopen, or publish. It separates automated canvas-level assertions from live window checks and requires recording file, platform, action, and visible result (marking unrun checks pending).

## Details
- **Verification discipline**:
  - Use a temporary copy for save checks.
  - **Canvas assertions** catch clipped lines or misplaced carets in automated tests (`CGO_ENABLED=0 go test ./...`).
  - **Window checks** exercise focus, input, layout, and platform integration together; record automated and window results separately, naming the fixture, exact action, and what appeared on screen.
- **11-surface verification matrix** (`doc/viewer-plan.md`):
  1. **Files sidebar**: open two folders with similarly named notes, filter by title/path, clear field, toggle hidden and ignored separately → typed text fits, matching rows keep project context, clearing restores tree, `.git` stays hidden.
  2. **Markdown page**: open `testdata/membrane-assay.md` and a long note → text wraps inside column, headings appear once in body, tables/links/local images/Mermaid fences render (or show visible fallback when renderer is unavailable).
  3. **Source view**: open a Go or TypeScript file with a comment containing `*` → syntax colours distinguish comments and keywords; `*` survives as source text.
  4. **Caret and modes**: move through a blank line, enter Edit, type a word, press Escape, select text, switch between Raw and Formatted, undo → both views show the same edit and selection, caret stays visible, caret bounds verified in canvas test.
  5. **Saving**: edit one word in a temporary note with unfamiliar markup, save and reopen from disk, try closing a second dirty note → file contains only the intended change; close protects the unsaved draft.
  6. **Navigation**: open two tabs, split, change focus, follow an outline heading, use back/forward, search for a body-only word → commands affect focused pane, selected heading or search hit is visible, other pane keeps its place.
  7. **Editing controls**: create a note, insert a table, undo, drag-select a word and replace it → inspect buffer and rendered note; new file appears at selected location.
  8. **Project controls**: tag a file and filter by tag; change a persistent setting and reopen Lens → filter selects intended file; saved setting survives restart.
  9. **Layout and scrolling**: launch, open a note, scroll sidebar and document, resize → footer fits from frame 1, each scroll acts on intended surface, check clipping and caret visibility at narrower width.
  10. **Terminal on desktop**: focus terminal, run `pwd`, return to document → prompt is legible, `pwd` prints working directory, subsequent typing goes to focused surface.
  11. **Publishing**: export an ordered report and compare preview with saved PDF → section order, images, and page breaks match the same output file.

## Decisions & Tradeoffs
- **Canvas + window split**: Canvas tests catch geometry, clipping, and caret bugs reproducibly in CI, while live window checks verify focus, platform menus, and input routing end to end.
- **macOS live-window automation permission gate** (`lens-c8t`): Automated native window checks against `Lens.app` fail when macOS Screen Recording and Accessibility permissions are absent (`CGPreflightScreenCaptureAccess`, `CGPreflightPostEventAccess`, `AXIsProcessTrusted` return `false`). In headless runs, `CGO_ENABLED=0 go test ./...` verifies canvas rendering while live window checks remain pending until run in a permitted session.

## See Also
- [[UI Layout and Surfaces]]
- [[Single-Compile Rendering]]
- [[Embedded Shell]]
- [[Lens Editor]]
