# Templated export

Typst is the only layout. One compile writes `out/report.pdf`. The window shows images of that file. A second renderer is not part of this.

Word and HTML are not this compile. Typst's HTML export is still experimental and the browser reflows it. Word has no Typst target. They stay later, and they are not the same pixels.

## Profile

`lens.toml` in the project root names the template and the fields the template may read:

```
template = "brief.typ"
title = "Assay report"
author = "Pairti"
date = "2026-11-22"
```

A missing file, or `template = "default"`, uses the built-in A4 page. A value that is a file name is resolved under `templates/` and then the project root. `templates/*.typ` and a root `template.typ` are the project's templates.

The note becomes `body.typ`. The chosen template is the compile root. If it does not include `body.typ`, the include is added. Fields are passed as `typst compile --input title=... --input author=... --input date=...`. The built-in template prints title, author, and date, then the body. A project template may read the same inputs with `sys.inputs.at`.

An empty title falls back to the note's first heading. An empty date falls back to the day of the compile.

## Swap

`:template` lists the names. `:template brief` writes `lens.toml` and compiles again. Export uses whatever `lens.toml` currently names. The pages in the window and `out/report.pdf` both come from that run. Formatted returns to the note.

## Check

A stand-in `typst` records the arguments and the template text. The test asserts the selected template, the `--input` fields, the PDF path, and that swapping the name changes the next compile. The window comparison needs a real `typst` on PATH and is not claimed until that has been looked at.
