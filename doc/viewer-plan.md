# Viewer verification

Start with the writer’s action and record the file, platform, action, and visible result. Use a temporary copy for save checks. A canvas assertion can catch a clipped line or misplaced caret. A window check exercises focus, input, layout, and platform integration together. Mark each unrun check pending.

| Surface | Exercise it in the window | What to verify |
| --- | --- | --- |
| Files sidebar | Open two folders containing similarly named notes. Filter by title or path, then clear the field. Toggle hidden and ignored files separately. | Typed characters fit in the field. Matching rows keep their project context. Clearing restores the tree. Generated folders such as `.git` stay hidden. |
| Markdown page | Open [the assay fixture](../testdata/membrane-assay.md) and a long note. Inspect a heading, table, link, local image, and Mermaid fence. | Text wraps inside the column. Headings appear once in the body. Check the rendered image or the visible fallback when a renderer is unavailable. |
| Source view | Open a Go or TypeScript file with a comment containing an asterisk. | Syntax colours distinguish comments and keywords. The asterisk survives as source text. |
| Caret and modes | Move through a blank line, enter Edit, type a word, and press Escape. Select text, switch between Raw and Formatted, then undo. | Both views show the same edit and selection. The caret stays visible. Inspect the caret bounds in a canvas test. |
| Saving | Edit one word in a temporary note containing unfamiliar markup. Save and reopen from disk. Try closing a second dirty note. | The file contains exactly the intended change. The close action protects the unsaved draft. |
| External changes | While a clean note is open, edit it in another local editor. Repeat with an unsaved Lens draft. | The clean view reloads. The dirty view offers a choice and preserves the draft for comparison. |
| Agent edits on desktop | Select a passage. Read it through `document`, then use `apply_edit` with its path and expected text. Undo. Try again after changing the passage. | The first edit appears in the focused buffer as one undoable change. Disk contents stay intact until Save. A stale edit is refused. |
| Navigation | Open two tabs, split, change focus, follow an outline heading, then use back and forward. Search for a word found only in the body. | Commands affect the focused pane. The selected heading or search hit is visible. The other pane keeps its place. |
| Editing controls | Create a note, insert a table, undo, then drag-select a word and replace it. | Inspect the buffer and the rendered note. The new file appears at the selected location. |
| Project controls | Tag a file and filter by its tag. Change a persistent setting, then reopen Lens. | The filter selects the intended file. The saved setting survives restart. |
| Layout and scrolling | Launch, open a note, scroll the sidebar and document, then resize. | The footer fits from the first frame. Each scroll acts on the intended surface. Check clipping and caret visibility at the narrower width. |
| Terminal on desktop | Focus the terminal, run `pwd`, then return to the document. | The prompt is legible, the command prints the working directory, and subsequent typing goes to the focused surface. |
| Publishing | Export an ordered report and compare its preview with the saved PDF. | Check section order, images, and page breaks against the same output file. [Publishing design](export-plan.md) |

Record automated and window results separately. A useful window result names the fixture, the exact action, and what appeared on screen.
