---
title: Embedded Shell
type: entity
sources: [doc:///viewer-plan.md, doc:///todo.md]
related: [entities/pairti-lens.md, topics/viewer-verification-plan.md, topics/build-roadmap.md]
updated: 2026-11-27
---

# Embedded Shell

The Embedded Shell is an in-app command runner panel in [[Pairti Lens]] that runs a real subprocess with ANSI colour codes stripped rather than emulating a full terminal. It is explicitly parked until all Phase 1 document viewer window checks pass.

## Key Facts
- **Current status**: parked (Phase 3 of [[Viewer Verification Plan]]).
- **Known defects blocking release**:
  1. **Prompt font**: currently uses a font missing prompt glyphs, rendering a row of question marks.
  2. **Command output**: typing a command and pressing Return does not yet display command output in the panel.
- **Design scope**: runs a real OS process and strips ANSI escape/colour codes; intentionally not a full VT100/xterm terminal emulator.
- **Exit checks**:
  - Window check for prompt font: open Shell and read a properly rendered prompt.
  - Automated + window check for command output: test asserts `echo` token appears in transcript; window check runs `pwd` and verifies the directory appears.

## How It Works
The shell panel spawns a subprocess, captures stdout/stderr, strips ANSI colour sequences, and appends the output to a scrollable transcript panel inside the [[Pairti Lens]] window.

## Relationships
- Gated behind Phase 1 of [[Viewer Verification Plan]].
- Tracked alongside future features (agents, extra diagram renderers) in [[Build Roadmap]].
