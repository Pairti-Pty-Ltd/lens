# Lens editor keymap

Raw and Formatted share the document’s cursor, selection, undo history, and editing mode. **View** is Vim’s Normal mode. **Edit** is Insert mode. Keys are case-sensitive. In the tables below, `gg` means press `g` twice, and `Ctrl-W e` means press Control and W together, release them, then press `e`.

## Entering and leaving Edit

| Keys from View | Where typing begins |
| --- | --- |
| `i` | Before the character under the cursor. |
| `a` | After the character under the cursor. |
| `I` | Before the first nonblank character on the line, preserving indentation. |
| `0i` | At the very beginning of the line, ahead of any indentation. |
| `A` | After the last character on the line. |
| `o` / `O` | On a new line below / above the current line. |
| `s` / `3s` | Replace the current character / three characters with what you type. |
| `S` or `cc` | Replace the current line. |
| `C` or `c$` | Replace from the cursor through the end of the line. |
| `cw` | Replace from inside a word through its end. |
| `ciw` | Replace the entire word, including the part to the left of the cursor. |

In Edit, printable characters insert text. Enter splits the line, Backspace deletes backward, Delete deletes forward, and the arrows move the insertion point. Home and End move to the beginning and end of the source line. **Ctrl-W in Edit** erases backward over a word. **Esc** finishes the change and returns to View. A second Esc cancels a pending command or selection.

## Movement and search

| Keys | Destination |
| --- | --- |
| `h` / `l` or Left / Right | Previous / next character on this line. |
| `j` / `k` or Down / Up in View | Next / previous source line. A long Markdown paragraph can occupy one source line. |
| `gj` / `gk` | Next / previous displayed row, including inside a wrapped paragraph. |
| `w` / `b` / `e` | Next word start / previous word start / word end. Punctuation forms a separate group. |
| `W` / `B` / `E` | Same motions with whitespace-delimited words. For example, `go-to-market` is one WORD. |
| `0` / `^` / `$` | First column / first nonblank character / last character of the line. Home and End also reach the line boundaries. |
| `gg` / `G` | First / last source line. |
| `42G`, `42gg`, or `:42` Enter | Source line 42. Line numbering is one-based and works with the gutter hidden. |
| `{` / `}` | Previous / next paragraph boundary at an empty line or the edge of the document. |
| `%` | Partner of a parenthesis, square bracket, or brace. Starts at the cursor or the next delimiter on this line and follows nesting of that pair. |
| `f,` / `F,` | Next / previous comma on this line. Substitute any character for the comma. |
| `t,` / `T,` | Stop immediately before the next comma / immediately after the previous comma. |
| `;` / `,` | Repeat the last character find in its original / opposite direction. |
| `/text` Enter / `?text` Enter | Search forward / backward for literal, case-sensitive text in the current document, wrapping at the end. Esc cancels the prompt. |
| `n` / `N` | Next match in the search direction / the opposite direction. |
| `*` / `#` | Search forward / backward for the whole word under the cursor. Continue with `n`. |
| `H` / `M` / `L` | Move the cursor to the top / middle / bottom of the visible pane. |
| `zt` / `zz` / `zb` | Scroll the current line to the top / middle / bottom, as far as the document allows. |
| Ctrl-F / Ctrl-B | Page forward / backward. |
| Ctrl-D / Ctrl-U | Half a page down / up. |
| Ctrl-E / Ctrl-Y | Scroll one line down / up. |

A count gives a distance. `5j` moves five source lines, `3w` advances three word starts, and `2f,` lands on the second following comma.

## Editing a range

Choose the range by motion when you know its endpoint, by text object when you know its contents, or visually when you want to inspect it first.

| Method | Example | Effect |
| --- | --- | --- |
| Operator and motion | `dt,` | Delete from the cursor up to the comma, leaving the comma. |
| Operator and motion | `df,` | Include the comma in the deletion. |
| Operator and motion | `y$` | Copy from the cursor through the end of the line. |
| Operator and text object | `ciw` / `diw` / `yiw` | Change / delete / copy the word containing the cursor. |
| Outer text object | `daw` | Delete the word and adjacent whitespace. `iW` and `aW` use whitespace-delimited words. |
| Doubled operator | `dd` / `yy` / `cc` | Delete / copy / change the entire current line. `3dd` deletes three lines including this one. |
| Counted operator | `d2w` / `2d2w` | Delete over two / four word motions. |
| Character selection | **Ctrl-W**, then a motion, or `v` then a motion | Start at the cursor. Each motion extends or shrinks the inclusive selection. |
| Whole-line selection | `Vjj` | Select the current line and the next two lines. |
| Adjust selection | `o` in Visual | Exchange the moving end and the anchor. Use this to adjust the other boundary. |
| Select a word | `viw` or Ctrl-W then `iw` | Select the whole word from anywhere inside it. `aw` includes adjacent whitespace. |
| Act on selection | `c`, `d` or `x`, `y`, Esc | Type a replacement, delete, copy, or cancel, respectively. |
| Mouse selection | Drag across text | Select a range. Use `c`, `d`, or `y` as with a keyboard selection. |
| Addressed lines | `:12,18d` / `:12,18y` Enter | Delete / copy lines 12 through 18 inclusive. |
| Relative endpoints | `:.,$y` Enter | Copy from the current line through the final line. `%` addresses all lines, as in `:%y`. |
| Visual line addresses | Select text, then type `:` | Lens fills in `'<,'>`. Append `d` or `y` and press Enter to act on all **lines** touched by the selection. |

**Example.** From the first letter of “investment management”, press Ctrl-W, then `2e`. Both words are selected through the final “t”. Press `c`, type “asset management”, and press Esc. Press `u` to undo the replacement in one step.

## Replace, paste, repeat, and save

| Keys | Action |
| --- | --- |
| `x` / `X` | Delete the character under / before the cursor. |
| `D` | Delete through the end of the line. |
| `rX` | Replace the current character with a capital X and stay in View. `3r-` replaces three characters with hyphens. |
| `J` | Join the next line to this one. |
| `~` | Toggle the case of an ASCII letter. |
| `p` / `P` | Put copied or deleted text after / before the cursor. A linewise copy goes below / above the current line. The register belongs to the open document. |
| `.` | Repeat the last change made from View, including the text entered by a change such as `cw`. |
| `u` / Ctrl-R | Undo / redo. On macOS, Cmd-Z and Cmd-Shift-Z also work. |
| `:w` Enter / Cmd-S / Ctrl-S | Save the current file. |
| `:q` Enter | Leave the editor for the formatted reading surface when the document is clean. |
| `:wq` or `:x` Enter | Save and leave the editing surface. |
| `:q!` Enter | Discard unsaved changes by reloading the file. |
| `:sp` or `:vsp` Enter | Open Lens’s split layout. |
| Ctrl-W then Ctrl-H / Ctrl-L / Ctrl-W | Focus the left pane / right pane / other pane. Keep Control held for the second key when targeting panes. |

## How Lens relates to Vim

The editor uses Vim’s operator-and-motion convention and its characterwise and linewise Visual modes. Lens adds **Ctrl-W as a selection starter in View**. In standard Vim that prefix opens window commands; in Insert, Ctrl-W deletes the preceding word. Search in this Lens build is literal. Vim’s regular-expression search, Visual Block, named registers, macros, and general Ex scripting are broader facilities.

Manual references: [motions and operators](https://vimhelp.org/motion.txt.html), [Visual selections](https://vimhelp.org/visual.txt.html), [line addresses](https://vimhelp.org/cmdline.txt.html#cmdline-ranges), [Insert](https://vimhelp.org/insert.txt.html), and [window commands](https://vimhelp.org/windows.txt.html).
