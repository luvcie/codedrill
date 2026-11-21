# codedrill

codedrill is a terminal app for timed coding practice. Pick a drill, solve it in your editor, and beat your personal best.

![codedrill terminal demo](docs/images/codedrill.gif)

## Install

You need Bash, an editor, and the tools your drills use, such as GCC, G++, or Python 3. Install Zellij to open the editor beside a terminal for drill commands. [Zellij](https://zellij.dev/)

**Nix with flakes**

```sh
nix profile add github:luvcie/codedrill
codedrill
```

**Release binaries**

Download the binary for Linux (x86-64 or ARM64) or Android (ARM64, run it in Termux). Rename it to `codedrill`, run `chmod +x codedrill`, then run `./codedrill`. [GitHub releases](https://github.com/luvcie/codedrill/releases)

**Build from source with Go 1.26.5 or newer**

```sh
git clone https://github.com/luvcie/codedrill.git
cd codedrill
go build -o codedrill ./cmd/codedrill
./codedrill
```

## Start a drill

1. Select a drill and press Enter to open its briefing. To open a specific briefing from the shell, run `codedrill start ft_strlen`.
2. Press `1` for Study, then Enter to start. Study shows the reference when available. Select **Copy reference exactly** to practice matching it on supported C and C++ drills.
3. Write your solution. In the Zellij terminal pane, run `t` to test and `s` to submit. Without Zellij, close the editor to submit.
4. After evaluation, press `e` to resume editing or Enter to save a passing run.

## Controls

| Where | Controls |
| --- | --- |
| Drill menu | `/` search, `*` toggle a persistent favorite, `o` settings, `w` Free Work |
| Drill terminal | `i` problem, `p` pace, `d` diff, `back` return to the briefing without saving |
| Drill terminal in Study | `m -v` peek at the reference, `m -p` toggle the reference pane, `m -d` view the normalized diff |
| Drill mixer | Run `v` in the drill terminal. Select a volume row and use Left or Right to adjust it. Press `f` to cycle All, Calm only, and Battle only. |

Free Work opens one to four Zellij terminals in a directory you choose. Leave the directory blank for home. Music and typing sounds use your saved settings.

In Settings, use Tab to reach Music or Typing Sounds. Run `codedrill config` to see settings and paths, `codedrill list` for drill IDs, or `codedrill --help` for all commands.
