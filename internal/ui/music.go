package ui

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/download"
	"github.com/luvcie/codedrill/internal/music"
	"github.com/luvcie/codedrill/internal/storage"
)

type musicSettings struct {
	musicPrefs   storage.MusicSettings
	musicVolume  int
	musicSource  string
	musicTracks  []music.Track
	musicCursor  int
	musicNotice  string
	musicErr     error
	musicFolder  textinput.Model
	musicEditing bool
}

func (m *MainModel) openMusic() {
	cfg, err := storage.LoadConfig()
	m.musicSettings = musicSettings{musicVolume: int(cfg.MusicVolume()*100 + .5), musicErr: err}
	if cfg.Music != nil {
		m.musicPrefs = *cfg.Music
	}
	m.state = StateMusic
}

func (m *MainModel) saveMusic() {
	cfg, err := storage.LoadConfig()
	if err == nil {
		volume := m.musicVolume
		m.musicPrefs.Volume = &volume
		cfg.Music = &m.musicPrefs
		err = storage.SaveConfig(cfg)
	}
	m.musicErr = err
}

func (m *MainModel) openMusicSource(source string) {
	m.musicSource = source
	m.musicCursor = 0
	m.musicErr = nil
	m.musicTracks = music.Tracks(source)
	if source == music.Local && m.musicPrefs.Folder != "" {
		m.musicTracks, m.musicErr = music.LocalTracks(m.musicPrefs.Folder)
	}
}

func (m *MainModel) musicTrackBase() int {
	switch m.musicSource {
	case music.Local:
		return 4
	case music.Holidays:
		return 2
	}
	return 5
}

func (m *MainModel) musicRowCount() int {
	if m.musicSource == "" {
		return 7
	}
	return m.musicTrackBase() + len(m.musicTracks)
}

func (m *MainModel) musicLayout(w, h int) typingLayout {
	notice := m.musicNotice
	if m.musicErr != nil {
		notice = "Could not save/load: " + m.musicErr.Error()
	}
	if m.download != nil || (notice == "" && m.downloadNotice != "") {
		notice = m.downloadStatus(w)
	}
	credit := ""
	if m.musicSource == music.SprintZen || m.musicSource == music.TetraLeague {
		credit = "  HURT RECORD"
	}
	return m.audioSettingsLayout(w, h, m.musicCursor, m.musicRowCount(), notice, credit)
}

func (m *MainModel) viewMusic() string {
	w, h := m.screenSize()
	layout := m.musicLayout(w, h)
	lines := strings.Split(layout.header, "\n")
	if layout.blankAbove {
		lines = append(lines, "")
	}
	for row := layout.start; row < layout.start+layout.shown; row++ {
		lines = append(lines, cursorRow(m.musicCursor == row, w, m.musicRowLabel(row)))
	}
	for _, footer := range []string{layout.credit, layout.notice, layout.help} {
		if footer != "" {
			lines = append(lines, strings.Split(footer, "\n")...)
		}
	}
	return strings.Join(lines[:min(len(lines), h)], "\n")
}

func (m *MainModel) musicRowLabel(row int) string {
	if m.musicSource == "" {
		enabled := "off"
		if m.musicPrefs.Enabled {
			enabled = "on"
		}
		shuffle := "off"
		if m.musicPrefs.Shuffle {
			shuffle = "on"
		}
		return []string{"Music  " + enabled, fmt.Sprintf("Volume  %d%%", m.musicVolume), "Shuffle  " + shuffle, "Calm (Sprint / Zen)  →", "Battle (Tetra League)  →", "TETR.IO Holidays 2023  →", "Local MP3 folder  →"}[row]
	}
	if row == 0 {
		return "← Music"
	}
	if m.musicSource == music.Local {
		if row == 1 {
			if m.musicEditing {
				return m.musicFolder.View()
			}
			folder := m.musicPrefs.Folder
			if folder == "" {
				folder = "choose a folder"
			}
			return "Folder  " + folder
		}
		if row == 2 {
			return "Select all"
		}
		if row == 3 {
			return "Select none"
		}
	} else if m.musicSource == music.Holidays {
		if row == 1 {
			return "Download ZIP · 7.6 MiB (both tracks)"
		}
	} else {
		switch row {
		case 1:
			return "Select all"
		case 2:
			return "Select none"
		case 3:
			return "Download selected" + m.musicDownloadSize(false)
		case 4:
			return "Download all" + m.musicDownloadSize(true)
		}
	}
	track := m.musicTracks[row-m.musicTrackBase()]
	checked := "[ ] "
	if slices.Contains(m.musicPrefs.Tracks, track.ID) {
		checked = "[x] "
	}
	label := checked + track.Name
	if track.Artist != "" {
		label += " — " + track.Artist
	}
	if mood := music.TrackLabel(track.ID); mood != "" {
		label += " · " + mood
	}
	if m.musicSource == music.Local {
		return label
	}
	if music.Cached(track) {
		label += " · cached"
	} else if (m.musicSource == music.SprintZen || m.musicSource == music.TetraLeague) && track.Size > 0 {
		label += " · " + byteSize(track.Size)
	}
	return label
}

func (m *MainModel) musicDownloadSize(all bool) string {
	var size int64
	for _, track := range m.musicTracks {
		if (all || slices.Contains(m.musicPrefs.Tracks, track.ID)) && !music.Cached(track) {
			size += track.Size
		}
	}
	if size == 0 {
		return ""
	}
	return " · " + byteSize(size)
}

func (m *MainModel) toggleMusicTrack(track music.Track, selected bool) {
	at := slices.Index(m.musicPrefs.Tracks, track.ID)
	if selected && at < 0 {
		m.musicPrefs.Tracks = append(m.musicPrefs.Tracks, track.ID)
	}
	if !selected && at >= 0 {
		m.musicPrefs.Tracks = slices.Delete(m.musicPrefs.Tracks, at, at+1)
	}
}

func (m *MainModel) activateMusicRow() tea.Cmd {
	row := m.musicCursor
	m.musicNotice = ""
	if m.musicSource == "" {
		switch row {
		case 0:
			m.musicPrefs.Enabled = !m.musicPrefs.Enabled
			m.saveMusic()
		case 2:
			m.musicPrefs.Shuffle = !m.musicPrefs.Shuffle
			m.saveMusic()
		case 3:
			m.openMusicSource(music.SprintZen)
		case 4:
			m.openMusicSource(music.TetraLeague)
		case 5:
			m.openMusicSource(music.Holidays)
		case 6:
			m.openMusicSource(music.Local)
		}
		return nil
	}
	if m.musicSource == music.Holidays && row == 1 {
		return m.startDownload("Holidays 2023 ZIP", len(music.HolidayTracks), func(ctx context.Context, report download.Report, completed func()) error {
			return music.Download(ctx, music.Holidays, music.HolidayTracks, report, completed)
		}, nil)
	}
	if row == 0 {
		m.musicSource = ""
		m.musicCursor = 3
		return nil
	}
	if m.musicSource == music.Local && row == 1 {
		m.musicFolder = textinput.New()
		m.musicFolder.Prompt = "Folder: "
		m.musicFolder.SetValue(m.musicPrefs.Folder)
		w, _ := m.screenSize()
		m.musicFolder.Width = max(1, w-14)
		m.musicEditing = true
		return m.musicFolder.Focus()
	}
	if row >= m.musicTrackBase() {
		track := m.musicTracks[row-m.musicTrackBase()]
		m.toggleMusicTrack(track, !slices.Contains(m.musicPrefs.Tracks, track.ID))
		m.saveMusic()
		return nil
	}
	selectAll, selectNone := 1, 2
	if m.musicSource == music.Local {
		selectAll, selectNone = 2, 3
	}
	if row == selectAll || row == selectNone {
		for _, track := range m.musicTracks {
			m.toggleMusicTrack(track, row == selectAll)
		}
		m.saveMusic()
	} else if (m.musicSource == music.SprintZen || m.musicSource == music.TetraLeague) && (row == 3 || row == 4) {
		var tracks []music.Track
		for _, track := range m.musicTracks {
			if row == 4 || slices.Contains(m.musicPrefs.Tracks, track.ID) {
				tracks = append(tracks, track)
			}
		}
		if len(tracks) == 0 {
			m.musicNotice = "No tracks selected."
			return nil
		}
		source, name := m.musicSource, "Calm (Sprint / Zen)"
		if source == music.TetraLeague {
			name = "Battle (Tetra League)"
		}
		return m.startDownload(name, len(tracks), func(ctx context.Context, report download.Report, completed func()) error {
			return music.Download(ctx, source, tracks, report, completed)
		}, nil)
	}
	return nil
}

func (m *MainModel) updateMusic(key tea.KeyMsg) tea.Cmd {
	if m.musicEditing {
		switch key.String() {
		case "esc":
			m.musicEditing = false
			return nil
		case "enter":
			folder := strings.TrimSpace(m.musicFolder.Value())
			tracks, err := music.LocalTracks(folder)
			m.musicErr = err
			if err == nil {
				m.musicTracks = tracks
				m.musicPrefs.Folder = folder
				m.saveMusic()
				m.musicEditing = false
			}
			return nil
		}
		var cmd tea.Cmd
		m.musicFolder, cmd = m.musicFolder.Update(key)
		return cmd
	}
	w, h := m.screenSize()
	switch key.String() {
	case "up", "k":
		m.musicCursor = max(0, m.musicCursor-1)
	case "down", "j":
		m.musicCursor = min(m.musicRowCount()-1, m.musicCursor+1)
	case "home", "g":
		m.musicCursor = 0
	case "end", "G":
		m.musicCursor = m.musicRowCount() - 1
	case "pgup":
		m.musicCursor = max(0, m.musicCursor-m.musicLayout(w, h).contentRows)
	case "pgdown":
		m.musicCursor = min(m.musicRowCount()-1, m.musicCursor+m.musicLayout(w, h).contentRows)
	case "left", "h", "right", "l":
		delta := 1
		if key.String() == "left" || key.String() == "h" {
			delta = -1
		}
		m.musicVolume = min(100, max(0, m.musicVolume+delta))
		m.saveMusic()
	case "enter", " ":
		return m.activateMusicRow()
	case "tab", "t":
		m.state = StateSettings
	case "shift+tab":
		m.openTypingSounds()
	case "esc", "q":
		if m.musicSource != "" {
			m.musicSource = ""
			m.musicCursor = 3
		} else {
			m.closeSettings()
		}
	}
	return nil
}

func (m *MainModel) updateMusicMouse(msg tea.MouseMsg) tea.Cmd {
	if m.musicEditing {
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.musicCursor = max(0, m.musicCursor-1)
	case tea.MouseButtonWheelDown:
		m.musicCursor = min(m.musicRowCount()-1, m.musicCursor+1)
	case tea.MouseButtonLeft:
		if tab, ok := m.settingsTabAt(msg.X, msg.Y); ok {
			m.openSettingsTab(tab)
			return nil
		}
		w, h := m.screenSize()
		layout := m.musicLayout(w, h)
		if msg.X >= 0 && msg.X < w && msg.Y >= layout.contentTop && msg.Y < layout.contentTop+layout.shown {
			m.musicCursor = layout.start + msg.Y - layout.contentTop
			return m.activateMusicRow()
		}
	}
	return nil
}

func (m *MainModel) startMusic() {
	m.stopMusic()
	m.musicPlaybackErr = nil
	cfg, _ := storage.LoadConfig()
	if cfg.Music != nil && cfg.Music.Enabled {
		socket := ""
		if m.freeWork != nil {
			socket = m.freeWork.MusicSocket()
		} else if m.session != nil {
			socket = m.session.MusicSocket()
		}
		m.musicPlayer = music.Start(music.SelectedSongs(cfg.Music.Tracks, cfg.Music.Folder), cfg.MusicVolume(), cfg.Music.Shuffle, socket)
	}
}

func (m *MainModel) stopMusic() {
	if m.musicPlayer != nil {
		m.musicPlaybackErr = m.musicPlayer.Close()
		m.musicPlayer = nil
	}
}
