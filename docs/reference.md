# Reference comparison

## Add and compare a reference

Run `codedrill solution <id> <path>...` to attach reference files or directories. Run `codedrill solution <id>` to list attached files. A local attachment takes precedence over a custom drill's `solution/` directory when it contains supported files.

Run `codedrill match <id> [--workspace DIR] [--json]` to compare a workspace. The default workspace is the current directory; `--json` prints the result as JSON.

## Comparison rules

Reference scoring, exact Copy/Recall checks, live scoring, and token diffs support C and C++ through the same comparison engine. For C, parentheses that wrap a complete return expression are ignored. C++ preserves return parentheses, literal spellings, and operator boundaries (including `>>` versus `> >`). Whitespace and comments outside literals are ignored where safe; directive spacing is retained conservatively. Malformed literals and unsupported lexical constructs fail comparison rather than passing an exact-copy gate. Stringifying macros, module directives, and non-ASCII identifier spellings are currently unsupported; macros are not expanded and templates are not parsed.

For C++ drills with ownership metadata, only declared student files in the selected reference are scored. Harnesses and supplied interfaces are excluded, and incomplete references cannot satisfy the required student-file list. Optional files participate when the selected reference contains them; a missing counterpart prevents an exact match. Extra workspace files are ignored. For C drills, all reference files are compared.

## Visibility and viewers

The reference viewer uses the sprint's snapshotted **Show reference** option. Recall and other hidden-reference configurations deny access; changing preferences after the sprint starts does not enable it. Missing or invalid session policies also deny access. Reference files stay outside the student's workspace.

After a completed submission, press `v` on the evaluation or results screen to inspect the reference diff when reference checking succeeded, including in Recall mode.

Use arrows or Page Up/Down to scroll, Left/Right to read long lines, and Tab/Shift+Tab to switch reference files. The header shows the current filename and, when reference checking is enabled, a match percentage calculated on opening. Press `q`, Esc, or Ctrl+C to close the viewer and return to the original editor and terminal panes. Syntax highlighting uses the existing libraries; no additional tools are required.

Side-by-side viewing requires Zellij 0.45 or newer. It temporarily replaces the right terminal with the read-only viewer while preserving the shell process, history, and scrollback. Opening it focuses the editor. Focus the reference with the mouse or Zellij pane controls to scroll or change files, then return focus to the editor to continue typing. Toggle again or press `q` in the viewer to restore the terminal. The floating viewer works independently. The 60/40 split remains at narrow widths; use horizontal scrolling or the floating viewer for more space.

`Alt+b` avoids the shifted-key encoding differences that can affect `Alt+Shift+v` across legacy and Kitty keyboard protocols, Zellij, and Helix. It needs no Ghostty key remapping. Shortcuts apply in Zellij's normal mode; unlock Zellij first if necessary.
