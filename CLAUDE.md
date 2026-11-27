# Project instructions for AI agents

Guidance for people and coding agents working in this repository. Do not push unless the person who asked for the work has said to push.

## Build and test

macOS is the documented platform. You need Go 1.27.4 or later and Make.

```sh
make binary
./build/pairti-lense
```

Run from source with `CGO_ENABLED=0 go run ./cmd/pairti-lense`. Install the app bundle in `~/Applications/Lens.app` with `make app`.

Serve the shared browser workspace with

```sh
make wasm-serve
```

Open `http://127.0.0.1:8081/library/`. `make wasm` builds without serving.

```sh
CGO_ENABLED=0 go test ./...
```

Some export tests need `typst`, `pdftoppm` or `pandoc` on `PATH`. They skip when the tool is missing.

## Layout

- `cmd/pairti-lense` builds the desktop and browser workspace.
- `cmd/lens-library` serves the Git-backed Markdown library.
- `internal/ui` draws the window with GoGPU.
- `internal/edit` is the Markdown buffer.
- `internal/doc` and `internal/library` handle documents and the project library.
- `internal/index` is the lexical search index.
- `internal/watch` reloads files after writes settle.
- `internal/gitlog` is Git history.
- `internal/report` exports PDF with Typst and `.docx` with Pandoc.
- `internal/agent` is the local MCP server (`document` and `apply_edit`).
- `doc/editor-keymap.md` is the editor keymap.

## Style

Build with `CGO_ENABLED=0`. Keep module versions pinned in `go.mod`. The Makefile already sets this for `make binary` and `make wasm`.

## Shell

Use non-interactive flags so commands do not hang on prompts. Prefer `cp -f`, `mv -f` and `rm -f`. For `scp` and `ssh`, use `-o BatchMode=yes`.
