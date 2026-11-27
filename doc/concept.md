# Pairti Lens

Working name, taken from this repo. A desktop tool for moving between projects, writing Markdown, and reading the report Typst laid out from a template.

Four features, taken from Zettlr and narrowed.

1. **Multi-project navigation.** Several project folders stay open. You move between them in one window.
2. **Native Markdown.** The file you write is Markdown on disk. The app does not edit a converted copy.
3. **Typst viewing.** The report you read is the Typst page. The view and the exported file are one compile.
4. **Template-driven reports.** A Typst template fixes the page. Same sources and same template produce the same pixels.

The product decision is: which report, in which project, and do the pages match the template.

Zettlr (zettlr.com) is the reference for navigation and for publishing a folder through a chosen template. It is not the spec.

## Multi-project navigation

A library is the folders this install has open, stored as paths. Opening a folder adds it. Removing it drops the path and does not delete the folder. A missing path stays listed and reads as absent.

A project is one folder. Its name comes from `lens.toml` if that file exists, otherwise from the folder name. The window lists every open project. Inside a project the rows are documents, labelled by the first Markdown heading, otherwise the filename. A path segment longer than about 40 characters is cut. Filter matches project names, titles, and relative paths. It does not scan bodies.

You do not close one project to see another.

## Native Markdown

Documents are `.md` and `.markdown`. They stay CommonMark plus the small set the template bind knows: headings, lists, tables, fenced code, links, images, footnotes. That set is written down when the first template bind exists.

The browser shows the Markdown. Editing, when it exists, writes those same bytes back. There is no shadow document, no HTML round-trip, and no Typst source standing in for the note.

## Typst viewing

Typst is how a report is seen, not a second language you write the report in.

A view is the pages of one Typst compile. The app runs the Typst compiler and shows those pages. Export writes the PDF of that same compile. What you see is that file. A web-view reflow is not a report view.

We do not reimplement Typst. Math, type, and page breaking stay in the compiler. `.typ` files are templates and partials. They are not documents in the navigation list.

## Template-driven reports

A report is a Typst template applied to the project's Markdown.

The template owns page size, type, margins, headings, tables, and where the body goes. The project points at one template in `lens.toml`. The template may live in the project or in a shared template set. Swapping the template and compiling again is how a second report design is made. The UI does not restyle a report by itself.

Pixel-perfect means one compile feeds the view and the file. Two renders are a bug.

Zettlr's export profile is the analogy: a named way out, bound to a template, run on the project. The layout engine is Typst. A converter may turn Markdown into the compiler's input. Pandoc is not the layout engine. It may be that converter later, if we need it. Not decided here.

One project, one active report template. Several profiles can wait. Outputs go in `out/`. Compiler errors stay with the job.

## What we do not copy

- Electron, or a browser widget as the shell.
- Pandoc as the thing that decides where ink goes.
- Writing the report as Typst. Typst source is the template.
- Zettelkasten IDs, tags, snippets, and writing statistics. They do not define the product.
- A viewer per output kind. The report view is the Typst page. HTML and source-code browsers can be added later. They are not reports.

## Stack

Go, module `pairti-lense`. UI on [GoGPU](https://github.com/gogpu): `gogpu/ui` for widgets, `gogpu/gogpu` for the window, `gogpu/gg` for drawing. Pure Go, `CGO_ENABLED=0`. The report pages are drawn on that same surface.

This is a bet. As of November 2026, `gogpu/ui` is v0.1.x (v0.1.55). `gogpu/editor` is early and is not a dependency. Non-Latin text, IME, and Wayland input are known weak spots, and `@latest` has failed to compile for other people. Versions are pinned. The Markdown buffer is ours.

The Typst CLI is the compiler. Gio is the fallback shell if build 0 rejects GoGPU. That is written down before any template work.

## Builds

| Build | Proves |
| --- | --- |
| 0 | Multi-project navigation. Markdown files listed by title. The selected source readable. GoGPU accepted or rejected. |
| 1 | Edit and save the Markdown in place. |
| 2 | One Typst template on one project. One compile. Pages in the window, and the same PDF in `out/`. |
| 3 | Swap the template. The pages and the file change together. |

Build 0 is only the first row. Details are in `doc/build-0.md`.
