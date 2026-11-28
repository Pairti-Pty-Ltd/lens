---
title: macOS Startup Performance
type: topic
sources: [beads]
related: [entities/pairti-lens.md, entities/library-and-projects.md, topics/macos-native-integration.md]
updated: 2026-11-28
---

# macOS Startup Performance

Instrumented measurements of [[Pairti Lens]] native macOS startup from executable launch (`cmd/pairti-lense/startup_native.go`) to the first presented GPU frame identify where cold and repeat launch latency actually goes.

## Details

### Measurement Baseline
- **Environment**: Apple M5 Pro, macOS 26.6.2, Go 1.27.4.
- **Methodology**: Six interleaved launches per profile (first launch tracked separately; next five reported as median) across three isolated temporary profiles so user sessions are unaffected:
  1. **Empty** workspace.
  2. **Five projects**: 5 Git repositories, 12 expanded directories, mixed ignored files and folders.
  3. **Projects + tab**: Same 5 projects plus a restored 49,152-byte formatted Markdown document (based on `testdata/membrane-assay.md`).

### Latency Breakdown & Git Ignore Batching (`864be61`)
- **Pre-`main` cold launch cost**: The largest first-binary-launch interval (~838–880 ms out of ~977–1041 ms for an empty profile) happens *before* Go `main` begins (executable loading and macOS launch assessment).
- **Repeat core startup**: Repeat median empty launch is ~184–189 ms. Native renderer initialization takes ~2 ms on repeats.
- **Document restore**: `restoreDocs` / `loadPane` (`cmd/pairti-lense/main_docs.go`) takes ~6.9–7.3 ms. The active tab renders 4 times during open and restore (~6.6–6.9 ms total).
- **Synchronous Git ignore filtering (largest avoidable repeat cost)**: Previously, `readDir` in `internal/doc/list.go` called `dropIgnored` twice per directory—once for `dirs` and once for `files`. Batching both into a single `dropIgnored(projectRoot, out)` call per directory (`864be61`) halved synchronous Git subprocesses and filtering time:

| Profile (repeat median) | Before (`list.go` split) | After (`864be61` batched) |
|---|---|---|
| Empty launch | 189 ms (0 Git procs) | 184 ms (0 Git procs) |
| Five projects launch | 318 ms (25 Git procs / 123 ms) | 261 ms (13 Git procs / 66 ms) |
| Projects + tab launch | 340 ms (24 Git procs / 117 ms) | 282 ms (12 Git procs / 59 ms) |

*(Initial comparison before adding branch-query instrumentation also showed project directory listing dropping from 114 ms to 63 ms and rescan from 51 ms to 28 ms.)*

## Pitfalls

- **Optimizing Go code for cold-install latency**: Because ~838–880 ms of a fresh binary's first launch occurs before Go `main` executes, Go-level optimizations primarily improve repeat launches. Fresh-install investigations need to isolate executable loading and macOS launch assessment separately.
- **Splitting per-directory Git checks**: Invoking `git check-ignore` separately for subdirectories and files doubles subprocess overhead during sidebar expansion and rescanning (`internal/doc/list.go`, `internal/doc/ignore.go`). Keep entries combined into one `dropIgnored` call per directory.
- **Redundant active-tab renders**: Restoring a saved document triggers 4 renders across open and restore; though small (~6.6–6.9 ms for a 49 KB document), keep this in mind if document render cost grows.

## See Also
- [[Library and Projects]]
- [[macOS Native Integration]]
- [[Pairti Lens]]
