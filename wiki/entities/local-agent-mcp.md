---
title: Local Agent MCP
type: entity
sources: [doc:///concept.md, doc:///viewer-plan.md]
related: [entities/pairti-lens.md, entities/lens-editor.md, topics/viewer-verification-plan.md]
updated: 2026-11-28
---

# Local Agent MCP

On desktop, [[Pairti Lens]] exposes a local MCP server (`internal/agent`) so coding or writing agents can inspect and revise the passage a writer is currently viewing without bypassing the open draft.

## Key Facts
- **Two tools**:
  - `document` — exposes the focused buffer, including unsaved text, the current selection, and its file path.
  - `apply_edit` — applies an edit given the target path and expected text.
- **Stale-edit protection**: If the writer has switched documents or revised the target passage since `document` was read, `apply_edit` refuses the stale edit.
- **Draft-first integration**: A successful `apply_edit` appears in the open focused buffer as a single undoable step. Disk contents stay untouched until the writer explicitly saves.
- **Coexistence with direct disk writes**: Agents or external editors may also write directly to disk. Clean buffers reload after the write settles; dirty buffers preserve the writer's draft and surface the external change for reconciliation.

## How It Works
An agent revising a paragraph reads the focused buffer via `document`, then calls `apply_edit` with the document's path and expected text. Lens validates that the path and expected text still match the focused buffer before mutating the in-memory [[Lens Editor]] buffer in one undo step, leaving final save control with the writer. Both the MCP path and direct-to-disk external writes are tested against the same note.

## Relationships
- Mutates the in-memory buffer of [[Lens Editor]] as a single undo step.
- Part of the desktop [[Pairti Lens]] workflow in a shared Git checkout.
- Verified under **Agent edits on desktop** and **External changes** in [[Viewer Verification Plan]].
