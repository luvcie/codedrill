# codedrill

A terminal speedrun drill tool to master algorithms and build code muscle memory.

Inspired by TETR.IO 40-line sprints and typing tests, `codedrill` lets you practice coding exercises under the pressure of a live millisecond clock, comparing every run against your personal best.

## Features

- **Speedrun Engine**: High-precision stopwatch measuring elapsed wall-clock time down to the millisecond.
- **Arcade Ranks & Streaks**: Performance grading (S/A/B/C ranks) with target pace hints, live PB deltas, and consecutive win streak tracking.
- **XP & Levels**: Every run earns XP: more for faster clears and first-try passes, a little for a failed attempt. Titles go from Syntax Novice at level 1 to Grandmaster at level 40.
- **Daily Challenges**: One drill per day, picked from your drill list by date, with +50 bonus XP for the first clear.
- **Embedded Drill Pack**: Includes classic 42 exam drills (`ft_strlen`, `ft_swap`, `ft_strrev`, `ft_list_size`, `ft_split`, `ft_atoi`), algorithms (`reverse_bits`, `binary_search`), and LeetCode drills out-of-the-box.
- **Zellij Workspace**: Automatically spins up an isolated split layout with your editor (`hx`, `nvim`, `vim`, etc.) on the left and a test terminal on the right.
- **Instant Workspace Helpers**:
  - `r` / `./run`: Compile, run test suite, and check PB pace.
  - `p` / `./pace`: Instant elapsed stopwatch and grade pace check (`p -w` for continuous live ticker).
  - `d` / `./diff`: Colored diff against original starter code (`d -r` to reset).
  - `i` / `./info`: View problem constraints and description in terminal.
  - `s` / `./submit`: Lock time and submit sprint.
- **Category Filter Tabs**: Press `Tab`, or `h`/`l`, to cycle between categories (`42`, `Algorithms`, `LeetCode`, `Custom`, plus any you create).
- **Leaderboards & Stats**: Hall of Fame tracking top runs per drill, plus aggregate speedrunner stats dashboard (`S` key).
- **Audio Feedback**: Optional terminal bell (`\a`) when the countdown hits GO and on new personal bests.

## Requirements

- [Go](https://go.dev/) 1.26.5+
- [Zellij](https://zellij.dev/)
- Your preferred terminal editor (`hx`, `nvim`, `vim`, `emacs`, `micro`, `nano`)

## Installation

```bash
git clone https://github.com/luvcie/codedrill.git
cd codedrill
go build -o codedrill ./cmd/codedrill
```

Optionally move it to your `$PATH`:

```bash
mv codedrill ~/.local/bin/
```

## Configuration

Preferences can be set in `~/.config/codedrill/config.json`:

```json
{
  "editor": "hx",
  "sound": true,
  "countdown": 3
}
```

## Usage

```bash
codedrill                  # Launch interactive speedrun menu
codedrill <id>             # Shortcut to sprint a drill (e.g. codedrill ft_strlen)
codedrill start <id>       # Jump directly into a sprint
codedrill daily            # Sprint today's deterministic daily challenge (+50 XP bonus)
codedrill random [filter]  # Jump into a random drill (e.g. random c, random 42)
codedrill new              # Create a new custom drill interactively
codedrill export <id>      # Export drill definition as JSON to stdout
codedrill import <file>    # Import drill definition from a JSON file
codedrill rm <id>          # Delete a custom drill, or hide a built-in one
codedrill restore [id|all] # List hidden built-in drills, or bring them back
codedrill info <id>        # View drill briefing, description & target paces
codedrill top <id>         # View Hall of Fame leaderboard for a drill
codedrill recent <id>      # View recent sprint attempts for a drill
codedrill list [filter]    # List available drills (e.g. list c, list leetcode)
codedrill search <query>   # Search drills by title, category, or language
codedrill stats            # View aggregate statistics & speedrunner profile
codedrill config           # View active editor and audio configuration
codedrill config editor hx # Set editor preference (hx, nvim, vim, etc.)
codedrill config sound on  # Toggle audio bell (on / off)
codedrill config countdown 0 # Set countdown seconds (0 or off for instant start)
codedrill reset <id>       # Reset PB and attempts for a specific drill
codedrill reset all        # Reset all personal bests, attempts, and streaks
codedrill version          # Show version
```

### Hotkeys

- `↑`/`k`, `↓`/`j`: Navigate exercises
- `Enter`: Select drill / Confirm
- `D`: Sprint today's Daily Challenge (+50 bonus XP)
- `R`: Quick play / Random drill sprint (or retry in Results)
- `N`: Next drill (in Briefing, Results, or Hall of Fame)
- `P`: Previous drill (in Briefing, Results, or Hall of Fame)
- `Tab` / `l`, `Shift+Tab` / `h`: Cycle category filters (`ALL`, `42`, `Algorithms`, `LeetCode`, `Custom`, your own) / `Tab` toggles recent runs in Hall of Fame
- `/`: Search and filter exercises
- `v`: Open Hall of Fame leaderboard from the menu (`l` in Briefing and Results)
- `x`: Delete the selected custom drill, or hide a built-in one (asks to confirm)
- `S`: Open Speedrunner Profile & Stats dashboard
- `Esc` / `q`: Back / Quit
