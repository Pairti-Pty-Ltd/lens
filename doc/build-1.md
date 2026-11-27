# Build 1 — edit and save

The open Markdown file can be changed and written back to the same path.

## Done when

1. Edit shows the source of the selected file. The rendered page is still there when you are not editing.
2. Typing, Enter, Backspace, Delete, and the arrow keys change a buffer the app owns.
3. Save writes that buffer to the same file. The sidebar title updates from the first heading.
4. A dirty file does not get replaced by selecting another one. Escape discards and returns to the rendered page.
5. `go test ./...` covers the buffer and a uitest that types into the editor and reads the drawn text back.

## Out of scope

Syntax colour, undo history, selection drag, Typst, the terminal, agents.

## Screen

The header stays the project, the path, and the format. One button sits on that header: Edit, then Save, then Done. Unsaved is a word on the format line, not a second copy of the document.
