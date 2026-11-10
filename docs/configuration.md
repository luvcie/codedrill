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

## Factory Reset

Open Settings with `o`, select **Factory Reset**, and press Enter through the
three confirmations. The fourth Enter resets the data. Escape or moving to
another setting cancels the sequence.

Reset removes custom drills, local reference attachments, history and progress,
practice preferences, hidden-drill selections, menu layout, settings, old local
backups, corrupt-history backups, and interrupted settings saves. Bundled drills
and references remain available, and hidden built-ins are restored. The app
returns to its default menu with a success message and the backup location.

Before removal, Codedrill copies all affected data to a private sibling directory
named `codedrill-backup-<UTC timestamp>-<suffix>`. It verifies the copied contents,
permissions, and symlink targets against the original before clearing anything.
If backup or verification fails, the data is not reset. Reset refuses linked
config roots and source repositories. It preserves unrecognized files and never
follows exercise symlinks into external collections.

To restore, quit Codedrill and copy the contents of the backup's `data/` directory
back into the config directory, preserving permissions and symlinks. `RESTORE.txt`
inside the backup records the original path. Later resets do not remove these
sibling backups. Do not run another Codedrill instance while resetting or restoring.

## Editor

codedrill uses the first of these that it finds:

1. `editor` in config.json, if it is on `PATH`
2. `$CODEDRILL_EDITOR` (arguments are allowed, e.g. `code --wait`)
3. The first of `hx`, `nvim`, `vim`, `emacs`, `micro` on `PATH`
4. `$EDITOR`, then `$VISUAL`
5. `nano`, then `vim`
