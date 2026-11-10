# Configuration

Settings are stored in `config.json` in the codedrill config directory. Change them with `codedrill config` subcommands or edit the file:

```json
{
  "editor": "hx",
  "sound": true,
  "countdown": 3,
  "theme": "legacy",
  "mouse": true,
  "shell": "bash"
}
```

Every key is optional. The defaults are an auto-detected editor, sound and mouse support on, a 3-second countdown, the `legacy` theme, and bash as the pane shell. A countdown of `0` starts sprints immediately.

`o` also switches the pane shell between bash and zsh and toggles mouse support. For another shell, run `codedrill config shell <name or path>`.

Pick a theme with `o` in the menu or `codedrill config theme <id>`. The IDs are `legacy` (the default), `cyan`, `gruvbox`, `tokyo-night`, `catppuccin-mocha`, `nord`, `dracula`, `rose-pine`, `kanagawa`, `cyberpunk`, `monochrome` and `light`. Use `light` on a light terminal background. In settings, press Enter on the Themes row to expand or collapse the list. The PASS/FAIL and rank colour sample appears when the list is expanded.

On Linux the config directory is `$XDG_CONFIG_HOME/codedrill` (usually `~/.config/codedrill`). On macOS it is `~/Library/Application Support/codedrill`. `codedrill config` prints the paths in use.

## Editor

codedrill uses the first of these that it finds:

1. `editor` in config.json, if it is on `PATH`
2. `$CODEDRILL_EDITOR` (arguments are allowed, e.g. `code --wait`)
3. The first of `hx`, `nvim`, `vim`, `emacs`, `micro` on `PATH`
4. `$EDITOR`, then `$VISUAL`
5. `nano`, then `vim`
