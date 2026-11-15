# Background music

Run `go run ./cmd/codedrill`, press `o` for Settings, then Tab twice to Music
(or click the Music tab).

1. Open **TETR.IO Sprint / Zen** to browse the 26 Calm tracks from the
   [source spreadsheet](https://docs.google.com/spreadsheets/d/1R_tpfLo_qUfNG2vQG3uqQ5fbjT01VefZVgB7RH2lOzM/edit).
   Space/Enter toggles a song. Use **Download selected** or **Download all**.
   Opening a playlist does not download or enable anything.
2. **TETR.IO Holidays 2023** uses the
   [official free MP3 download](https://merch.tetr.io/holidays-2023-ost/).
   **Download ZIP** retrieves both songs in one 7.6 MiB download;
   the checkboxes decide which songs play.
3. For **Local MP3 folder**, select **Folder**, enter its path, then select songs.
   `~/Music` works. The list includes MP3s directly in that folder.

Press Esc to return to the Music page. Turn **Music** on and adjust its own
volume with Left/Right (35% initially). Music stays off until enabled.
Selections from all three sources play in catalog order and repeat during the
drill. Uncached songs are skipped; playback never downloads music.

Start a drill. Switch files in Helix, quit Helix and run `hx` again in the same
Zellij session: music keeps playing. Finishing or leaving the drill stops it.
Reopening the editor from evaluation starts music again.

In the drill's terminal pane, run `v` or `volume` to open the live mixer.
Use Up/Down to choose **Music** or **SFX**, and Left/Right to adjust it in 5%
steps. Set a bar to 0% to mute it. Esc, Enter or q returns to the terminal.
Changes apply to the running audio and are saved for later
drills. The mixer leaves the enabled/off switches in Settings unchanged.
You can also open it outside a drill with `codedrill volume`.

Music and SFX downloads run in the background. Their bars show downloaded
bytes, a percentage when the length is known, and completed tracks for a
playlist. Without a length, the bar shows activity and downloaded bytes. You
can navigate Settings while a download runs; one download runs at a time.

Audio uses the same PulseAudio/PipeWire server as typing sounds. No new player
or audio dependency is required. If playback fails, evaluation shows the error.
Codedrill bundles no music. Downloads are stored under
`$XDG_CACHE_HOME/codedrill/music` (normally `~/.cache/codedrill/music`);
SFX keep their existing `typing-sounds` cache. Local songs stay in their folder.
