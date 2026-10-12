# codedrill

A terminal speedrun drill tool to master algorithms and build code muscle memory.

Inspired by TETR.IO 40-line sprints and typing tests, `codedrill` lets you practice coding exercises under the pressure of a live millisecond clock, comparing every run against your personal best.

## Features

- **Speedrun Engine**: High-precision stopwatch measuring elapsed wall-clock time down to the millisecond.
- **Zellij Workspace**: Automatically spins up an isolated split layout with your editor (`hx`, `nvim`, `vim`, etc.) on the left and a test terminal on the right.
- **Instant Test Verification**: Pre-configured `./run` script for live compilation and feedback.
- **Customizable Submission Modes**: Submit via `./submit` or configure exercises to auto-submit when saving and closing your editor.
- **Local History & Personal Bests**: Automatically tracks best times and calculates deltas (`-00:04.250s` faster).
- **Extensible**: Create custom drills on the fly with custom languages, entry files, and test commands.

## Requirements

- [Go](https://go.dev/) 1.22+
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
  "editor": "hx"
}
```

## Usage

```bash
./codedrill
```

- Navigate exercises with arrow keys or `j`/`k`.
- Press `Enter` to view the briefing, then `Enter` or `Space` to start the countdown.
- Inside the sprint:
  - Write your solution in the left pane.
  - Test anytime in the right pane with `./run`.
  - Submit with `./submit` when finished.
