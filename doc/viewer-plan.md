# Document viewer plan

The shell is parked. Its prompt font is wrong and command output is not shown. That work starts only after the viewer below has passed its checks. A passing `go test` is not enough on its own for anything the user sees. An item is finished when the check named here has been run and the result matches the intended behaviour. Say what was looked at. Do not call an item done before that.

Two kinds of check:

- **Test.** A test drives the widget and asserts what is on the canvas: the string, the colour, the caret, the bounds. A test that only checks a return value does not pass a view item.
- **Window.** Launch the app, do the action, look at the window. Record the file used and what was on screen.

## Phase 1 — the viewer has to work

These are already written. None of them is signed off on the window.

| Item | Intended behaviour | Check |
| --- | --- | --- |
| Search | The field under the sidebar shows each character as it is typed. The tree narrows to names, titles, and paths that contain that text. Clearing the field restores the tree. | Window: type into Search on a real project and watch the rows change. Test: the field paints the typed string inside its box, not clipped. |
| Hidden and Ignored | Two icons, same 28px slots as the main toolbar. Grey means hidden. Blue means shown. The eye toggles dot-leading names. The slashed circle toggles gitignored names. `.git` and `node_modules` stay hidden. | Window: toggle each icon on a repo that has a dotfile and a gitignored file. Test: `TestProjectTogglesChangeTheTree` plus the icon row drawing no "Hidden" or "Ignored" label. |
| Markdown page | Headings, paragraphs, lists, quotes, tables, fenced code, and links render from the file bytes. The title is the first heading, shown once. A long line wraps inside the column. | Test: existing page tests stay green on `testdata/membrane-assay.md`. Window: open that file and one long note from a real project. Look at a table and a link. |
| Images and diagrams | A local image sits in the paragraph at its own size, shrunk only when wider than the column. A mermaid fence becomes the picture. A missing `mmdc` leaves the fence and says `mmdc not installed`. | Test: picture and mermaid tests. Window: one note that contains an image and one mermaid fence. |
| Source files | A `.ts`, `.go`, or other known source file is code, not Markdown. Keywords are blue, comments grey, strings green. A `*` inside a comment stays a star. | Test: `TestPageHighlightsSource`. Window: open `poll-retry.ts` or another real source file and read one comment and one keyword. |
| Caret and vi | Opening a file shows a caret on line 1 and highlights that line. `h j k l` move it. `i` inserts. Escape returns to view. `:w` writes the file. `:q` closes the line. The caret is visible on a blank line. | Test: click, `i`, type, escape, and assert the caret rect and the buffer. Window: open a note, move with `j`, type one word, Escape, and see the word in the preview. |
| Formatted and Raw | The toolbar says Formatted and Raw. Formatted is the editable rendered note, or coloured source for code. Raw shows the Markdown source. Each mode remembers its line-number setting. Save appears when the buffer is dirty and writes the same path. | Window: switch Formatted and Raw on a Markdown file and on a source file. Change one word, Save, and confirm the file on disk. |
| Tabs and split | Each opened file is a tab. Dragging a tab right, or the Split icon, opens a second pane. Split again closes it. Closing a dirty tab refuses and the status line says Unsaved. | Test: tab and pair tests. Window: open two files, split, click each side, close the split. |
| Outline | The right-hand icon opens the contents. A heading click scrolls the preview to that line. | Window: open a note with three headings and jump to the last one. |
| Footer | One status row from the first frame. It does not grow when the status text arrives and it does not wait for a resize. | Test: `TestFooterStaysOneLine`. Window: launch, open a file, do not resize, and read the status line. |
| Scroll | The file list and the document scroll in the Mac natural direction, one short step per tick. | Window: scroll both surfaces with the trackpad. |

Phase 1 is finished only when every Window row above has been looked at and the mismatches are fixed. Until then the viewer is not done.

## Phase 2 — viewer features still missing

Start these after phase 1 is finished. Same exit rule.

| Item | Intended behaviour | Check |
| --- | --- | --- |
| Search inside files | A query searches the body of the open file, or of the project, and jumps to the hit. The tree filter stays the sidebar field. | Test: a known string in a fixture is the only hit. Window: search a word that is not in the filename. |
| New document | Creates a Markdown file in the selected folder and opens it. | Window: the new file is on disk and in a tab. |
| Undo and selection | Undo reverses the last edit. The pointer can drag a selection. | Test: type two characters, undo once, the second character is gone. Window: drag across a word and delete it. |
| Inserts | Code, link, image, table, and footnote insert that markup at the caret. They do not only write their name on the status line. | Test: each insert changes the buffer. Window: insert a table and see it in Formatted. |
| Back and forward | Back returns to the previously opened document. Forward reverses that. | Window: open three files, go back twice, forward once. |
| Settings | The gear opens the choices that are not the two sidebar icons. Theme and font still load from `theme.json` until that screen exists. | Window: open the gear and change one setting that survives a restart. |
| Tags | A tag can be set on a document and filtered. | Window: tag a file and show only that tag. |
| Report | Export compiles the open note with Typst to `out/report.pdf` and shows pages of that file. Formatted returns to the note. This waits on phase 1. `typst` is not on PATH yet. | Test: compile tests with a stand-in compiler. Window: install Typst, Export a real note, open `out/report.pdf` and compare it with the pages in the window. |

## Phase 3 — shell, after the viewer

Do not start this while a phase 1 window check is still open.

| Item | Intended behaviour | Check |
| --- | --- | --- |
| Prompt font | The shell uses a font that contains the prompt glyphs. The line is not a row of question marks. | Window: open Shell and read the prompt. |
| Command output | Typing a command and Return shows that command’s output in the panel. | Test: a session runs `echo` and the transcript contains the token. Window: run `pwd` and see the directory. |

Language-model agents, Word, and HTML stay behind the report. They are not part of the viewer gate.
