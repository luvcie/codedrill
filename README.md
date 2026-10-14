# codedrill

A terminal tool for timed coding drills. Each drill opens a workspace with a stopwatch, your editor and a test command. A run passes when the tests pass.

## Install

Needs Go 1.26.5+ and bash. [Zellij](https://zellij.dev/) is optional; with it, a terminal pane opens beside the editor.

```bash
git clone https://github.com/luvcie/codedrill.git
cd codedrill && go build -o codedrill ./cmd/codedrill
```

## Usage

```
codedrill                        Open the menu
codedrill <id>                   Open the briefing for a drill (ID or title)
codedrill start <id>             Same as above
codedrill daily [info]           Briefing for today's daily drill, or print it
codedrill random [filter]        Briefing for a random drill
codedrill new                    Create a custom drill
codedrill import <file>          Save a drill from a JSON file as a custom drill
codedrill export <id>            Print a drill as JSON
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
codedrill config sound on|off     Turn the bell and chimes on or off
codedrill config mouse on|off     Turn mouse support on or off

codedrill config countdown <0-10|off>   Seconds before a sprint starts
codedrill config theme [id]      List themes, or set one
codedrill reset <id>             Clear runs and PB for a drill
codedrill reset all              Clear all runs, PBs and streaks
codedrill version
codedrill help
```

`filter` matches a category, language or difficulty name.

## Config

Settings are in `~/.config/codedrill/config.json`. Change them with `codedrill config` or edit the file:

```json
{
  "editor": "hx",
  "sound": true,
  "countdown": 3,
  "theme": "legacy",
  "mouse": true
}
```

Every key is optional. Defaults are an auto-detected editor, sound on, mouse on, a 3 second countdown and the default theme, `legacy`; a countdown of `0` starts sprints at once.

Pick a theme with `o` in the menu or `codedrill config theme <id>`. The IDs are `legacy` (the default), `cyan`, `gruvbox`, `tokyo-night`, `catppuccin-mocha`, `nord`, `dracula`, `rose-pine`, `kanagawa`, `cyberpunk`, `monochrome` and `light`. Use `light` on a light terminal background.

On Linux the config directory is `$XDG_CONFIG_HOME/codedrill` (usually `~/.config/codedrill`). On macOS it is `~/Library/Application Support/codedrill`. `codedrill config` prints the paths in use.

## Editor

codedrill uses the first of these that it finds:

1. `editor` in config.json, if it is on `PATH`
2. `$CODEDRILL_EDITOR` (arguments are allowed, e.g. `code --wait`)
3. The first of `hx`, `nvim`, `vim`, `emacs`, `micro` on `PATH`
4. `$EDITOR`, then `$VISUAL`
5. `nano`, then `vim`

## Sprint workspace

A sprint copies the drill into a temporary directory and opens its entry file. In the Zellij terminal pane, these commands are available:

- `r` or `run`: run `./pace`, then the test command. Prints PASS or FAIL.
- `p` or `pace`: show elapsed time and current rank. `p -w` updates it live.
- `d` or `diff`: show changes against the starter code. `d -r` resets the entry file after a y/N prompt; `d -r -y` skips the prompt.
- `i` or `info`: show the problem statement.
- `s` or `submit`: close the Zellij session. The app then runs the tests.

Closing the editor does not end the sprint unless the drill sets `auto_submit`. Otherwise, use `s`.

## Ranks and XP

A passed run gets an S, A, B or C rank from its time, against cut-offs that depend on the drill's difficulty (defined in `internal/drill/pace.go`; `codedrill info <id>` shows them). XP comes from passing, from speed, from passing on the first test run, and from the first clear of the daily drill. Forfeited runs earn a little XP too. Levels run from Syntax Novice to Grandmaster.

## Custom drills

A custom drill is a folder in `~/.config/codedrill/drills/<id>/` with a `drill.json` and any other files the drill needs. Every file except `drill.json` is copied into the sprint workspace. Create one with `codedrill new`, or write the JSON and run `codedrill import drill.json`.

```json
{
  "title": "ft_putchar",
  "category": "42",
  "language": "c",
  "difficulty": "easy",
  "description": "Write a function that prints one character.",
  "entry_file": "ft_putchar.c",
  "starter_code": "void ft_putchar(char c)\n{\n}\n",
  "test_command": "gcc -Wall -Wextra -Werror -c ft_putchar.c",
  "auto_submit": false
}
```

- `codedrill import` requires `title` and `entry_file`; `entry_file` must be a relative path.
- `id` defaults to the folder name.
- If `test_command` is empty, codedrill picks one from the file extension (C, C++, Go, Rust, Python, JavaScript, TypeScript, shell).
- `auto_submit: true` submits the run when the editor closes.
- `extra_files` is a map of file name to contents, written into the workspace.

`codedrill rm` on a custom drill deletes its folder and runs. On a built-in drill it only hides it; the ID is kept in `~/.config/codedrill/hidden.json`. Use `codedrill restore <id>` or `codedrill restore all` to bring it back.

## Keys

Menu keys:

| Key | Action |
|-----|--------|
| Up/k, Down/j | Move |
| Enter, Space | Open briefing (on "+ Add a drill", open the form) |
| d, r | Daily drill briefing; random drill briefing |
| Tab/l/Right, Shift+Tab/h/Left | Next or previous category |
| / | Search (Enter keeps the search, Esc cancels) |
| n, N | Next or previous match |
| v, s | Hall of Fame for the selected drill; stats |
| o | Settings (theme, mouse) |
| x | Delete a custom drill or hide a built-in (y to confirm) |
| q, Esc | Quit |

With mouse support on (the default), click a drill to select it and click it again to open it, and click a category tab to switch. Footer actions are clickable too; the item flashes like a pressed button before the action runs. The wheel moves the selection and scrolls other screens. Shift+drag still selects text in most terminals; turn mouse support off with `o` or `codedrill config mouse off` to get normal selection back.

Search matches a substring of the drill ID or title and jumps to the first match as you type, across all category tabs. Matching is case-insensitive unless the pattern contains an uppercase letter.

Other screens show their keys at the bottom.

MIT licensed, see LICENSE.
