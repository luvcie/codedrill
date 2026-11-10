# Commands and shortcuts

## CLI

```
codedrill                        Open the menu
codedrill <id>                   Open the briefing for a drill (ID or title)
codedrill start <id>             Same as above
codedrill daily [info]           Briefing for today's daily drill, or print it
codedrill random [filter]        Briefing for a random drill
codedrill new                    Create a custom drill
codedrill import <file>          Save a drill from a JSON file as a custom drill
codedrill export <id>            Print a drill as JSON
codedrill solution <id> [path...]  List attached reference files, or attach files and folders
codedrill match <id> [--workspace DIR] [--json]  Compare a workspace against its reference solution
codedrill rm <id>                Delete a custom drill, or hide a built-in one
codedrill restore [id|all]       List hidden drills, or restore one or all
codedrill list [filter]          List drills
codedrill search <query>         Search id, title, language, category, difficulty, description
codedrill info <id>              Show target paces, PB, entry file and description
codedrill top <id>               Hall of Fame for a drill
codedrill recent <id>            Recent runs for a drill
codedrill stats                  Level, XP, win rate, streaks and personal bests
codedrill config                 Show config and current settings
codedrill config editor <name>   Set the editor
codedrill config shell <name>    Set the Zellij pane shell (bash, zsh or a path)
codedrill config sound on|off     Turn the bell and chimes on or off
codedrill config mouse on|off     Turn mouse support on or off

codedrill config countdown <0-10|off>   Seconds before a sprint starts
codedrill config theme [id]      List themes, or set one
codedrill reset <id>             Clear runs and PB for a drill
codedrill reset all              Clear all runs, PBs and streaks
codedrill version
codedrill help
```

`filter` matches a category, language, or difficulty name.

## Sprint helpers

A sprint creates a temporary workspace and opens an empty student entry file. Exercise-provided resources remain available. In the Zellij terminal pane, these commands are available:

- `r` or `run`: run `./pace`, then the test command. Prints PASS or FAIL.
- `p` or `pace`: show elapsed time and current rank. `p -w` updates it live.
- `d` or `diff`: show entry-file changes against the empty baseline. `d -r` empties all declared required student files and removes declared optional student files after a y/N prompt; `d -r -y` skips the prompt. Provided resources are preserved.
- `m` or `match`: show the reference similarity score. `m -w` updates it live.
- `m -v`: open the C or C++ reference in a floating, read-only pane. `Alt+v` opens it from either pane in Zellij's normal mode.
- `m -p`: toggle the reference beside the editor. `Alt+b` toggles it from either pane in Zellij's normal mode.
- `m -d`: view normalized token differences in the terminal. `-` marks missing reference tokens and `+` marks extra student tokens, with source line numbers. Scroll with arrows/PgUp/PgDn, switch files with Tab, and close with q/Esc. Available only when the sprint's Show Reference option is enabled.
- `i` or `info`: show the problem statement.
- `s` or `submit`: close the Zellij session. The app then runs the tests.

In Zellij, closing the editor does not end the sprint unless the drill sets `auto_submit`. Otherwise, use `s`. Without Zellij, closing the editor returns to evaluation.

## Menu keys

| Key | Action |
|-----|--------|
| Up/k, Down/j | Move |
| Shift+Up/Down | Reorder the selected drill or subcategory |
| Enter, Space | Open briefing (on a subcategory header, collapse or expand it; on "+ Add a drill", open the form) |
| d, r | Daily drill briefing; random drill briefing |
| Tab/l/Right, Shift+Tab/h/Left | Next or previous category |
| / | Search (Enter keeps the search, Esc cancels) |
| n, N | Next or previous match |
| v, s | Hall of Fame for the selected drill; stats |
| o | Settings (theme, shell, mouse) |
| x | Delete a custom drill or hide a built-in (y to confirm) |
| e | Edit a custom drill |
| q, Esc | Quit |

With mouse support on (the default), click a drill to select it and click it again to open it, click a category tab to switch, and click a subcategory header to collapse or expand it. Drag a drill or subcategory header to rearrange the current category; drills inside a subcategory can be reordered there, and subcategory blocks can move among the loose drills. Hold a drag at the top or bottom of the list to page through a long category. Without a mouse, Shift+Up/Down reorders the selected row the same way. The arrangement is saved in `layout.json` beside the config. Footer actions are clickable too. The wheel moves the selection and scrolls other screens. Shift+drag still selects text in most terminals; turn mouse support off with `o` or `codedrill config mouse off` to get normal selection back.

Search matches a substring of the drill ID or title and jumps to the first match as you type, across all category tabs. Matching is case-insensitive unless the pattern contains an uppercase letter.

Other screens show their keys at the bottom.

## Ranks and XP

A passed run gets an S, A, B or C rank from its time, against cut-offs that depend on the drill's difficulty (defined in `internal/drill/pace.go`; `codedrill info <id>` shows them). XP comes from passing, from speed, from passing on the first test run, and from the first clear of the daily drill. Forfeited runs earn a little XP too. Levels run from Syntax Novice to Grandmaster.
