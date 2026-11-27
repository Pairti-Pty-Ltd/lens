---
title: Editor Keymap and Modes
type: topic
sources: [doc:///editor-keymap.md, doc:///build-1.md, doc:///viewer-plan.md]
related: [entities/lens-editor.md, entities/pairti-lens.md, topics/viewer-verification-plan.md]
updated: 2026-11-27
---

# Editor Keymap and Modes

[[Lens Editor]] implements a case-sensitive, Vim-inspired modal keymap shared across both **Formatted** and **Raw** views. **View** corresponds to Vim's Normal mode, and **Edit** corresponds to Insert mode.

## Details
- **Entering and leaving Edit**:
  - Insert starters: `i`, `a`, `I` (first nonblank), `0i` (column 0), `A`, `o`/`O` (new line below/above).
  - Change starters: `s` / `<count>s`, `S`/`cc`, `C`/`c$`, `cw`, `ciw`.
  - In Edit: printable input, Enter, Backspace, Delete, arrows, Home/End, `Ctrl-W` (delete word backward), and `Esc` (return to View; second `Esc` cancels pending command or selection).
- **Movement and search** (counts supported, e.g. `5j`, `3w`, `2f,`):
  - Character/line: `h`/`l` (or Left/Right), `j`/`k` (source lines), `gj`/`gk` (wrapped visual rows), `0`/`^`/`$` (Home/End), `gg`/`G`, `<n>G`/`<n>gg`/`:<n>` (1-based line numbers even with gutter hidden).
  - Words & structure: `w`/`b`/`e` (punctuation separate), `W`/`B`/`E` (whitespace-delimited WORDs), `{`/`}` (paragraph boundaries), `%` (matching `()`, `[]`, `{}`).
  - Character find: `f`/`F`/`t`/`T` + char, repeated with `;` and `,`.
  - Literal case-sensitive search: `/text`, `?text` (wraps, `Esc` cancels), `n`/`N`, `*`/`#` (word under cursor).
  - Viewport: `H`/`M`/`L`, `zt`/`zz`/`zb`, `Ctrl-F`/`Ctrl-B`, `Ctrl-D`/`Ctrl-U`, `Ctrl-E`/`Ctrl-Y`.
- **Operators, text objects, Visual mode, and Ex ranges**:
  - Operators (`c`, `d`, `y`) combine with motions (`dt,`, `df,`, `y$`), text objects (`iw`/`aw`/`iW`/`aW`), doubled-line forms (`dd`, `yy`, `cc`, `3dd`), and counts (`d2w`, `2d2w`).
  - Visual selection: `v` or `Ctrl-W` (characterwise), `V` (linewise), `o` (swap anchor/cursor), mouse drag, then `c`, `d`/`x`, `y`, or `Esc`.
  - Line addresses: `:12,18d`, `:.,$y`, `:%y`, and visual `:` → `:'<,'>d`/`y`.
- **Replace, paste, repeat, undo, files, and splits**:
  - `x`/`X`, `D`, `r<char>` (`3r-`), `J`, `~`, `p`/`P` (per-document register), `.` (repeat last View change), `u`/`Ctrl-R` (`Cmd-Z`/`Cmd-Shift-Z` on macOS).
  - Ex commands: `:w` (`Cmd-S`/`Ctrl-S`), `:q` (return to formatted view if clean), `:wq`/`:x`, `:q!` (discard and reload), `:sp`/`:vsp` (open split).
  - Pane focus: `Ctrl-W` followed by `Ctrl-H` / `Ctrl-L` / `Ctrl-W` (Control held on second key).

## Decisions & Tradeoffs
- **`Ctrl-W` deviation from standard Vim**: In View mode, `Ctrl-W` followed by a motion or text object starts a characterwise selection (e.g. `Ctrl-W 2e`), whereas holding Control for the second key (`Ctrl-W Ctrl-H/L/W`) switches split panes. In Edit mode, `Ctrl-W` keeps Vim's word-erase behavior.
- **Scoped Vim subset**: Search is literal (not regex); Visual Block, named registers, macros, and general Ex scripting are intentionally omitted.

## See Also
- [[Lens Editor]]
- [[Viewer Verification Plan]]
