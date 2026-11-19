# codedrill

codedrill is a terminal app for timed coding practice. Pick a drill, solve it in your editor, and run its tests.

![codedrill terminal demo](docs/images/codedrill.gif)

## Install

You need bash, an editor, and the tools your drills use, such as a C compiler or Python. With [Zellij](https://zellij.dev/), codedrill opens your editor beside a terminal for tests and drill commands.

With Nix and flakes enabled:

```bash
nix profile add github:luvcie/codedrill
codedrill
```

Or build from source with Go 1.26.5 or newer:

```bash
git clone https://github.com/luvcie/codedrill.git
cd codedrill
go build -o codedrill ./cmd/codedrill
./codedrill
```

## Quick start

1. Choose a drill and open its briefing.
2. Select **Study** (`1`) or **Exam** (`2`), then press Enter.
3. Write your solution. In Zellij, run `r` to test and `s` to submit. Without Zellij, close the editor to submit.

Study lets you use a reference while practicing. Pass the tests, or enable **Copy reference exactly** to pass by matching the reference. Exam hides the reference and grades by tests. Your selection is saved per drill.

For C and C++ drills with a reference, use `m -v` to peek or `m -p` to pin it beside the editor in Study.

Use `back` in the Zellij terminal to discard the run and return to the same briefing with your mode and reference selected. After submission, you can resume editing or finish the run.

## Common commands

```sh
codedrill list                 # list drills
codedrill start ft_strlen      # open a drill's briefing
codedrill daily                # open today's drill
codedrill random               # choose a drill at random
codedrill new                  # create a custom drill
codedrill import drill.json    # import a drill
codedrill solution <id> <path>  # attach a reference solution
codedrill --help               # all commands
```

## Settings

Press `o` in the menu to change your editor, theme, music, and typing sounds. Run `codedrill config` to see settings and paths.

In a Zellij drill terminal, `v` opens the music and typing-sound mixer. Left/Right adjusts volume by one percentage point. Typing sounds are off by default and need a PulseAudio-compatible server such as PipeWire.
