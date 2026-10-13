# Real-terminal visual testing

Proved on 2026-10-26 with Kitty 0.49.2, Xvfb, Zellij 0.45.1, and real Neovim.

## Start a session

```sh
python3 qa/visual.py start --editor nvim
```

The first output line is the session directory. Sessions go under the system temp directory, or `$CODEDRILL_QA_TMP` if set. Assign that exact path to `S` in the shell issuing subsequent commands. For example:

```sh
S=/tmp/cdvis-hwn600uv
python3 qa/visual.py text --session "$S" './codedrill'
python3 qa/visual.py key --session "$S" enter
python3 qa/visual.py wait --session "$S" CODEDRILL
python3 qa/visual.py shot --session "$S" menu
```

Use a new session, not the example directory, for a new test run. Separate tool calls may not preserve shell variables; pass the path explicitly when necessary.

The controller builds the current working tree into the session directory. It creates a normal Kitty window on an automatically allocated Xvfb display, forces Kitty's X11 backend, and uses software rendering. Your desktop is not the test display. HOME, XDG directories, editor configuration, and Zellij configuration/socket paths are disposable. The private socket uses `allow_remote_control=socket-only`.

The default terminal is 120 columns by 40 rows. The app starts in a clean Bash shell; type `./codedrill` to launch it. Use `--editor hx` for a separate Helix run; the proof below used Neovim only.

## Controller operations

| Command after `python3 qa/visual.py` | Purpose |
| --- | --- |
| `key --session "$S" enter` | Press a key |
| `key --session "$S" ctrl+p right enter` | Press three keys in sequence |
| `text --session "$S" 'literal text'` | Type text without adding Enter |
| `screen --session "$S"` | Read currently visible text |
| `wait --session "$S" '[PASS]'` | Wait for visible text, with a deadline |
| `shot --session "$S" checkpoint` | Save PNG, plain text, ANSI text, and terminal metadata |
| `resize --session "$S" 80 24` | Resize and wait for the reported dimensions to match |
| `status --session "$S"` | Read dimensions and foreground-process metadata |
| `stop --session "$S"` | Stop this session's Zellij sessions, Kitty, and Xvfb |

For multiline input, pipe the text to the `text` action without a text argument. It sends literal stdin using automatic bracketed paste. Individual key events use Kitty's keyboard encoder. Unlike passing multiple keys to `kitten @ send-key`, the controller's `key` arguments are sequential, not one combined chord.

Use the image-reading tool to OPEN the PNG returned by `shot`. Text files and image filenames are not visual inspection. Kitty control calls do not prove that input reached the intended application; check the resulting screen and relevant files.

Actions are recorded in `actions.jsonl`. A failed text wait automatically takes a timeout screenshot. Inspect it before retrying: the target may be clipped, hidden behind a modal, or visible in a different pane.

## Observed interaction details

1. **Fresh Zellij setup blocks the editor.** The initial screenshot showed the setup wizard over the actual source buffer. Seeing `your code here` in the visible text did not mean the editor had focus. Read the modal and apply the Default binding scheme with Enter before continuing. Include both first-run and already-configured behavior in the coverage map.
2. **Submission can resurrect suspended panes on resume.** After `./submit`, evaluation, and `e`, both panes displayed `Waiting to run`. Start the editor pane with Enter. The shell pane may need its own Enter later. Treat this as observed product friction, not something to hide in fixture configuration.
3. **Pane focus is separate from editor mode.** With Default Zellij bindings, `key ctrl+p left enter` focuses the editor and exits pane mode; use `right` for the shell. Inspect the active pane border/title. Do not type code or shell commands based only on text elsewhere in the window.
4. **Resize completion is asynchronous.** The controller now polls reported columns/rows. At 80x24, the briefing's heading was clipped above the visible area. At 60x20, more content disappeared. A missing heading did not mean the program had hung. Screenshots, metadata, and neighboring visible controls identified the state.
5. **Titles and timings need context.** Kitty's window title retained an old Zellij session name after codedrill restarted. Check foreground-process metadata and the screen instead. This visual run includes time spent inspecting screenshots and handling setup; its duration is not a typing benchmark.

The application requires lowercase shortcut keys even where help labels use uppercase letters. Literal `s` and Shift+S are different events. Send keys according to actual bindings and report misleading labels separately.

The right-pane shell used the system Bash prompt despite the initial shell's `qa$` prompt. Recognize the visible prompt and active pane rather than hard-coding one prompt for both shells.

## Proven Neovim workflow

The session selected the first drill, `ft_strlen`, from the menu. After the briefing/countdown and Zellij setup, it focused the editor and replaced the starter through real Neovim keystrokes:

```sh
python3 qa/visual.py key --session "$S" escape g g shift+v shift+g c
python3 qa/visual.py text --session "$S" 'int ft_strlen(char *s) { (void)s; return 999; }'
python3 qa/visual.py key --session "$S" escape colon
python3 qa/visual.py text --session "$S" w
python3 qa/visual.py key --session "$S" enter
```

After focusing the shell pane, it typed `./run`, observed the assertion failure, and typed `./submit`. Back in codedrill it pressed `e`, handled suspended-pane resurrection, and replaced the code in Neovim with:

```c
int ft_strlen(char *s) { int n=0; while(s[n]) n++; return n; }
```

It saved with `:w`, ran `./run` in the shell, observed `PASS`, submitted, and accepted the result. It then quit codedrill, launched a fresh process from the same shell, and inspected the persisted PB and profile. It did not write the solution directly into the workspace from the testing process.
