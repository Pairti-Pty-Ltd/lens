# Lens north stars

Assume a project already has several local editors, a Git workflow, and agents that change files while someone is reading them. Open Lens when you need to read the result in context, revise a passage with an agent, or see exactly what a report will publish.

## Work in the same checkout

A writer should be able to edit a note in their preferred editor, let an agent update a neighbouring file, and inspect both in Lens. Keep Markdown readable on disk and changes reviewable in Git. A library remembers folders. Removing a folder from the library leaves its files intact.

## Give the agent the passage the writer is looking at

On desktop, the local MCP tool `document` exposes the focused buffer, including unsaved text, the selection, and its path. An agent revising a paragraph should read it first, then pass the path and expected text to `apply_edit`. If the writer has switched documents or revised the passage, refuse the stale edit. A successful edit appears in the open draft as one undo step. The writer decides when to save.

An agent or another editor can also write directly to disk. Reload a clean buffer after the write settles. If the writer has a draft, surface the change and let them choose how to reconcile the two versions. Test both paths with the same note.

## Make switching projects cheap

Keep several folders in one window. Show enough of the project and relative path to distinguish two notes with the same title. The tree filter helps writers find a file by name, title, or path. Body search takes them to a passage. Opening a second note or splitting the view must preserve the first draft and its place. Measure responsiveness from the writer’s action to the visible result.

## Give each note one editing history

Markdown is the source. Raw and Formatted share the buffer, cursor, selection, and undo history. Use Goldmark to understand document structure. Opening a note or changing views must preserve unfamiliar markup. If the file changes underneath a draft, keep the writer’s edits available until they choose how to resolve it. [Editor keymap](editor-keymap.md)

## Publish the report the writer selected

The project, report, ordered sources, and chosen template determine the publication. Typst owns PDF layout. When a writer compares preview and download, every page break, table, and image should match because both come from the same PDF. Word has its own exporter and styles. Keep organisation-specific assets in installable brand packs. [Publishing design](export-plan.md)

## Share the workspace across desktop and browser

Use the same Go editor, sidebar, menus, and pane layout wherever the interaction is the same. Put the differences at the storage and platform boundaries. Save writes to a local file on desktop or to the served checkout in browser library mode. A browser save against a stale revision must leave the draft recoverable. [Browser workspace](../wiki/topics/browser-wasm-editor.md)

Judge a change by what the writer can see, save, reopen, or publish. [Viewer verification](viewer-plan.md)
