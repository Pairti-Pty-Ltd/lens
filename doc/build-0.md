# Build 0 — multi-project Markdown viewer

One Mac window. Several project folders. The selected Markdown file is rendered. No HTML, no CSS, no web view.

Typst viewing, templates, the terminal, and agents are later builds. No editor, no compile, no PDF. The GoGPU stay-or-switch note is written after this window has been used on a real project.

## Done when

1. `CGO_ENABLED=0 go run .` opens one window on this machine.
2. You can add more than one folder. The list survives a restart. A missing folder stays listed and reads as absent.
3. Each folder shows only `*.md` and `*.markdown`, recursively, skipping `.git` and `out/`. Typst files are templates, not documents, and are not listed.
4. A row’s label is the document title: the first ATX heading, otherwise the filename. No YAML front matter parser.
5. Selecting a row renders that file: headings, paragraphs, lists, and fenced code. The source bytes are not rewritten. A file over 1 MiB is refused with its size.
6. The GoGPU decision is one paragraph at the bottom of this file: stay, or switch the shell to Gio before build 1. Written after the window has been used on a real project, not after a hello-world.

## Out of scope

Editing, save, syntax colour, Typst compile, templates, PDF, HTML, search inside files, tags, links, split view, drag-and-drop, menus beyond what the window needs to open a folder.

## The screen

The decision on screen: which document in which open project.

Budget: project name, the selected title once, a short path, the format. At most four lines. A path segment longer than about 40 characters is cut in the middle. The title is not repeated in the list and in a header. The list shows the title only. The header is the four lines below, for the selection.

Reading with a long title:

```
Grove lab notes
Write-up of the cohort run, with the plate map and the calls we will not repeat
grove/notes/membrane-assay.md
Markdown
```

The tree is projects, then titles. File size, modified time, line count, and the raw heading line stay off this view. Nothing else is on the first screen. An empty library is one line: open a project folder. An empty project is one line: no Markdown files.

## Behaviour

- Add project: OS folder picker. The path is stored absolute.
- Remove project: drops it from the library. Does not delete the folder.
- Select: read the file, sniff the title, render the blocks. The heading in the document is the title, so the header does not quote it again.
- Titles are sniffed once per path and size-and-mtime. Restart may sniff again. No index database.
- Filter: a single field over project names, titles, and relative paths. Clearing it restores the full list. Bodies are not scanned.
- Quit: write the library paths. No document state to save, because build 0 does not edit.

Config file: one JSON or TOML list of absolute paths, in the user config dir (`os.UserConfigDir()/pairti-lense/library.json`). Not inside any project.

## Stack

- Go 1.25 or newer, because `gogpu/ui` requires it.
- Pin `github.com/gogpu/ui` at v0.1.55 and the `gogpu` and `gg` versions that release was built with. Do not float `@latest`. `CGO_ENABLED=0`.
- Do not import `gogpu/editor`.
- Markdown parse: `github.com/yuin/goldmark`. The toolkit has no rich-text widget. Blocks are drawn as text in a scroll view.
- Folder add: the macOS folder panel via `osascript`, so the binary stays CGO-free. Other systems get a path field.

Fixture used for the spike, checked in under `testdata/`: a Markdown file whose first heading is the long title above, body at least a few hundred lines. A test asserts the sniffed title equals that heading and does not equal the filename.

## Module

```
pairti-lense/
  go.mod
  main.go            window, project tree, header, folder panel
  internal/library/  add, remove, persist paths
  internal/doc/      walk, title sniff, Goldmark blocks
  internal/ui/       scrollable Markdown page
  testdata/          the long-title fixture
  doc/               these notes
```

`internal/doc` walks, sniffs titles, caps the read, and turns Markdown into blocks with Goldmark. It does not import GoGPU.

## Order of work

1. Pin the module and open an empty window. Stop if that does not build with CGO off.
2. Library list, persisted. Two folders. Restart. Remove one.
3. Walk and title sniff, with the fixture test.
4. Selection loads the body. Cap the read. Show the four-line header from a real file in one of the folders, and confirm the title is not repeated.
5. Filter.
6. Use it on one real project folder you already have. Write the stay-or-switch paragraph.

## Checks

- `go test ./...` passes the sniff test, including a file with no heading (title is the filename) and a heading longer than 40 characters (title is kept whole; only a path segment is cut).
- The absent-folder case is tested without a window: a path that does not exist is returned as absent and is still saved.
- The window check is manual, once, on this Mac: two projects, select the long Markdown file, restart, filter. That pass is the evidence for the GoGPU paragraph.

## GoGPU decision

Stay. The window opened on this Mac, selected the M5 Pro's Metal adapter, and listed real project folders. The toolkit tree and split were wrong for hit testing and for window resize, so those two surfaces are ours. The rest of the shell stays on `gogpu/ui` v0.1.55.
