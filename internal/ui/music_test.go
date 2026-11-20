package ui

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/download"
	"github.com/luvcie/codedrill/internal/music"
	"github.com/luvcie/codedrill/internal/storage"
)

func openMusicTestModel(t *testing.T) *MainModel {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	m := newMenuTestModel(t)
	m.openSettings()
	m.openMusic()
	return m
}

func TestMusicTrackLabelsFollowPlaylistMembership(t *testing.T) {
	m := openMusicTestModel(t)
	for _, tc := range []struct {
		source string
		tracks []music.Track
		label  string
	}{
		{music.SprintZen, music.Calm, "Calm"},
		{music.TetraLeague, music.Battle, "Battle"},
	} {
		m.openMusicSource(tc.source)
		for i, track := range tc.tracks {
			want := track.Name + " — " + track.Artist + " · " + tc.label + " · "
			if row := m.musicRowLabel(m.musicTrackBase() + i); !strings.Contains(row, want) || strings.Contains(row, "cached") {
				t.Fatalf("label is missing or out of order: %s", row)
			}
		}
		track := tc.tracks[0]
		if err := os.MkdirAll(filepath.Dir(music.Path(track)), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(music.Path(track), []byte("cached"), 0600); err != nil {
			t.Fatal(err)
		}
		if row := m.musicRowLabel(m.musicTrackBase()); !strings.HasSuffix(row, " · "+tc.label+" · cached") {
			t.Fatalf("cached label is out of order: %s", row)
		}
	}
	for _, id := range []string{music.HolidayTracks[0].ID, "/tmp/local.mp3"} {
		if label := music.TrackLabel(id); label != "" {
			t.Fatalf("unrelated track acquired label %q", label)
		}
	}
}

func TestMusicSelectionDoesNotDownloadOrEnable(t *testing.T) {
	m := openMusicTestModel(t)
	if m.musicPrefs.Enabled || m.musicVolume != 35 {
		t.Fatal("music defaults changed")
	}
	m.musicCursor = 3
	if cmd := m.activateMusicRow(); cmd != nil || m.download != nil {
		t.Fatal("opening playlist downloaded music")
	}
	if len(m.musicTracks) != 26 {
		t.Fatalf("Calm catalog: %d", len(m.musicTracks))
	}
	if view := ansi.Strip(m.viewMusic()); !strings.Contains(view, "Piercing Wind — KVK") || !strings.Contains(view, "Download selected") || !strings.Contains(view, "Download all") {
		t.Fatal(view)
	}
	m.musicCursor = m.musicTrackBase()
	m.activateMusicRow()
	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Music == nil || len(cfg.Music.Tracks) != 1 || cfg.Music.Tracks[0] != music.Calm[0].ID || cfg.Music.Enabled {
		t.Fatalf("selection: %+v", cfg.Music)
	}
	m.updateMusic(tea.KeyMsg{Type: tea.KeyRight})
	cfg, _ = storage.LoadConfig()
	if cfg.MusicVolume() != .36 || cfg.TypingSoundsOn() {
		t.Fatal("music volume was not independent")
	}
	m.openMusicSource(music.Holidays)
	if m.download != nil || !strings.Contains(m.musicRowLabel(1), "7.6 MiB (both tracks)") {
		t.Fatal("holiday download behavior")
	}
}

func TestMusicVolumeStepsAndLimits(t *testing.T) {
	m := openMusicTestModel(t)
	m.musicCursor = 1
	for _, step := range []struct {
		key     tea.KeyType
		presses int
		want    int
	}{
		{tea.KeyRight, 1, 36},
		{tea.KeyLeft, 1, 35},
		{tea.KeyLeft, 101, 0},
		{tea.KeyLeft, 1, 0},
		{tea.KeyRight, 1, 1},
		{tea.KeyRight, 101, 100},
		{tea.KeyRight, 1, 100},
		{tea.KeyLeft, 1, 99},
	} {
		for range step.presses {
			m.Update(tea.KeyMsg{Type: step.key})
		}
		if m.musicVolume != step.want || m.musicErr != nil {
			t.Fatalf("music volume = %d, want %d: %v", m.musicVolume, step.want, m.musicErr)
		}
		m.openMusic()
		if m.musicVolume != step.want || m.musicErr != nil {
			t.Fatalf("saved music volume = %d, want %d: %v", m.musicVolume, step.want, m.musicErr)
		}
	}
}

func TestMusicShufflePersists(t *testing.T) {
	m := openMusicTestModel(t)
	if !strings.Contains(ansi.Strip(m.View()), "Shuffle  off") {
		t.Fatal("music settings have no Shuffle toggle")
	}
	m.musicCursor = 2
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.openMusic()
	cfg, err := storage.LoadConfig()
	if err != nil || !m.musicPrefs.Shuffle || !cfg.Music.Shuffle || cfg.Music.Enabled || cfg.MusicVolume() != .36 {
		t.Fatalf("saved Shuffle changed other settings: %+v, %v", cfg.Music, err)
	}
	w, h := m.screenSize()
	layout := m.musicLayout(w, h)
	m.Update(leftClick(4, layout.contentTop+2-layout.start))
	m.openMusic()
	if m.musicPrefs.Shuffle || !strings.Contains(ansi.Strip(m.View()), "Shuffle  off") {
		t.Fatal("mouse did not save Shuffle off")
	}
}

func TestBattlePlaylistReusesMusicControls(t *testing.T) {
	m := openMusicTestModel(t)
	t.Cleanup(m.Cleanup)
	for _, source := range []struct {
		row       int
		id, label string
	}{
		{3, music.SprintZen, "Calm (Sprint / Zen)"},
		{4, music.TetraLeague, "Battle (Tetra League)"},
		{5, music.Holidays, "TETR.IO Holidays 2023"},
		{6, music.Local, "Local MP3 folder"},
	} {
		m.openMusic()
		if !strings.Contains(m.musicRowLabel(source.row), source.label) {
			t.Fatalf("playlist label: %s", m.musicRowLabel(source.row))
		}
		m.musicCursor = source.row
		if cmd := m.activateMusicRow(); cmd != nil || m.download != nil || m.musicPrefs.Enabled || m.musicSource != source.id {
			t.Fatalf("opening %s changed playback or started a download", source.label)
		}
	}
	m.musicPrefs.Tracks = []string{music.Calm[0].ID}
	m.saveMusic()
	m.openMusicSource(music.TetraLeague)
	if len(m.musicTracks) != 13 || m.musicTracks[0].ID != music.Battle[0].ID {
		t.Fatalf("wrong Battle track list: %v", m.musicTracks)
	}
	view := ansi.Strip(m.viewMusic())
	for _, label := range []string{"Backwater — Kamoking", "Download selected", "Download all", "HURT RECORD", "3.1 MiB"} {
		if !strings.Contains(view, label) {
			t.Fatalf("Battle is missing %q:\n%s", label, view)
		}
	}
	for _, track := range append([]music.Track{music.Calm[0], music.Battle[0]}, music.HolidayTracks...) {
		if err := os.MkdirAll(filepath.Dir(music.Path(track)), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(music.Path(track), []byte("cached"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if !strings.Contains(m.musicRowLabel(m.musicTrackBase()), "cached") {
		t.Fatal("cached Battle track is not marked")
	}
	w, h := m.screenSize()
	layout := m.musicLayout(w, h)
	m.updateMusicMouse(leftClick(4, layout.contentTop+m.musicTrackBase()-layout.start))
	cfg, _ := storage.LoadConfig()
	if len(cfg.Music.Tracks) != 2 || cfg.Music.Tracks[0] != music.Calm[0].ID || cfg.Music.Tracks[1] != music.Battle[0].ID || cfg.Music.Enabled {
		t.Fatalf("Battle selection changed Calm or enabled playback: %+v", cfg.Music)
	}
	m.musicCursor = 3
	cmd := m.activateMusicRow()
	if cmd == nil || m.download == nil || m.download.tracks != 1 || m.download.name != "Battle (Tetra League)" {
		t.Fatal("Download selected did not use only the Battle selection")
	}
	job := m.download
	m.openMusicSource(music.Holidays)
	m.Update(cmd().(tea.BatchMsg)[0]())
	if m.download != nil || job.completed != 1 {
		t.Fatal("changing playlists changed the queued download")
	}
	m.openMusicSource(music.TetraLeague)
	for _, row := range []int{1, 2} {
		m.musicCursor = row
		m.activateMusicRow()
	}
	cfg, _ = storage.LoadConfig()
	if len(cfg.Music.Tracks) != 1 || cfg.Music.Tracks[0] != music.Calm[0].ID {
		t.Fatal("Battle bulk selection changed Calm selections")
	}
	m.musicCursor = 4
	if cmd := m.activateMusicRow(); cmd == nil || m.download == nil || m.download.tracks != 13 {
		t.Fatal("Download all did not include the whole Battle playlist")
	}
	if m.musicPrefs.Enabled || len(m.musicPrefs.Tracks) != 1 {
		t.Fatal("Download all changed playback settings")
	}
}

func TestMusicLocalFolderAndMouse(t *testing.T) {
	m := openMusicTestModel(t)
	folder := t.TempDir()
	if err := os.WriteFile(filepath.Join(folder, "My song.mp3"), []byte("local"), 0600); err != nil {
		t.Fatal(err)
	}
	m.openMusicSource(music.Local)
	m.musicCursor = 1
	m.activateMusicRow()
	m.musicFolder.SetValue(folder)
	m.updateMusic(tea.KeyMsg{Type: tea.KeyEnter})
	if m.musicEditing || m.musicErr != nil || len(m.musicTracks) != 1 {
		t.Fatalf("folder: %+v", m.musicSettings)
	}
	w, h := m.screenSize()
	layout := m.musicLayout(w, h)
	row := m.musicTrackBase()
	m.updateMusicMouse(leftClick(4, layout.contentTop+row-layout.start))
	cfg, _ := storage.LoadConfig()
	if len(cfg.Music.Tracks) != 1 || cfg.Music.Folder != folder {
		t.Fatalf("local settings: %+v", cfg.Music)
	}
	m.openMusic()
	m.musicCursor = 0
	m.activateMusicRow()
	cfg, _ = storage.LoadConfig()
	if !cfg.Music.Enabled {
		t.Fatal("toggle did not persist")
	}
}

func TestMusicViewFitsAndScrolls(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {60, 12}, {40, 8}, {30, 6}} {
		m := openMusicTestModel(t)
		m.width, m.height = size[0], size[1]
		m.openMusicSource(music.SprintZen)
		m.updateMusic(runeKey("G"))
		view := m.View()
		if len(strings.Split(view, "\n")) > size[1] || !strings.Contains(ansi.Strip(view), "│") {
			t.Fatal(view)
		}
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("wide line: %q", line)
			}
		}
		layout := m.musicLayout(size[0], size[1])
		if layout.start+layout.shown != m.musicRowCount() {
			t.Fatal("last track scrolled off screen")
		}
	}
}

func TestAudioDownloadProgressStaysResponsive(t *testing.T) {
	for _, total := range []int64{100, 0} {
		m := openMusicTestModel(t)
		started, release := make(chan struct{}), make(chan struct{})
		cmd := m.startDownload("test", 2, func(ctx context.Context, report download.Report, completed func()) error {
			report(download.Progress{Bytes: 25, Total: total})
			completed()
			report(download.Progress{Bytes: 50, Total: total})
			close(started)
			<-release
			return nil
		}, nil)
		batch := cmd().(tea.BatchMsg)
		finished := make(chan tea.Msg, 1)
		go func() { finished <- batch[0]() }()
		<-started
		m.Update(runeKey("j"))
		if m.musicCursor != 1 {
			t.Fatal("UI blocked while downloading")
		}
		status := m.downloadStatus(80)
		if !strings.Contains(status, "50 B") || !strings.Contains(status, "1/2 tracks completed") {
			t.Fatal(status)
		}
		if total > 0 && !strings.Contains(status, "50%") {
			t.Fatal(status)
		}
		if total == 0 && strings.Contains(status, "%") {
			t.Fatal(status)
		}
		m.Update(downloadTickMsg{m.download})
		m.Update(tea.KeyMsg{Type: tea.KeyTab})
		if m.state != StateSettings {
			t.Fatal("could not leave music during download")
		}
		close(release)
		m.Update(<-finished)
		if m.download != nil || !strings.Contains(m.downloadNotice, "Downloaded") {
			t.Fatal("download did not finish")
		}
	}
}

func TestHolidayPlaylistHasOneDownloadAndIndependentSongChoices(t *testing.T) {
	m := openMusicTestModel(t)
	m.openMusicSource(music.Holidays)
	m.width, m.height = 60, 10
	view := ansi.Strip(m.View())
	if strings.Count(view, "Download ") != 1 || !strings.Contains(view, "both tracks") {
		t.Fatal(view)
	}
	for _, track := range music.HolidayTracks {
		if !strings.Contains(view, track.Name) {
			t.Fatalf("song is hidden by controls: %s\n%s", track.Name, view)
		}
	}
	for _, row := range []int{2, 3, 2} {
		m.musicCursor = row
		m.activateMusicRow()
	}
	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Music.Tracks) != 1 || cfg.Music.Tracks[0] != music.HolidayTracks[1].ID {
		t.Fatalf("songs were not selected independently: %v", cfg.Music.Tracks)
	}
	m.musicCursor = 1
	if cmd := m.activateMusicRow(); cmd == nil || m.download == nil || m.download.tracks != 2 {
		t.Fatal("ZIP download did not include both tracks")
	}
	m.Cleanup()
	cfg, err = storage.LoadConfig()
	if err != nil || len(cfg.Music.Tracks) != 1 || cfg.Music.Tracks[0] != music.HolidayTracks[1].ID || cfg.Music.Enabled {
		t.Fatalf("downloading changed playback choices: %+v, %v", cfg.Music, err)
	}
}

func TestMusicDownloadAllKeepsPlaybackSelection(t *testing.T) {
	m := openMusicTestModel(t)
	m.openMusicSource(music.SprintZen)
	m.musicCursor = 3
	if cmd := m.activateMusicRow(); cmd != nil || m.musicNotice == "" {
		t.Fatal("empty selection did not explain why the download was skipped")
	}
	m.musicCursor = m.musicTrackBase()
	m.activateMusicRow()
	if m.musicNotice != "" {
		t.Fatal("old selection warning survived selecting a track")
	}
	m.musicCursor = 4
	if cmd := m.activateMusicRow(); cmd == nil || m.download == nil || m.download.tracks != len(music.Calm) {
		t.Fatal("download all did not include the whole playlist")
	}
	m.Cleanup()
	cfg, err := storage.LoadConfig()
	if err != nil || len(cfg.Music.Tracks) != 1 || cfg.Music.Tracks[0] != music.Calm[0].ID || cfg.Music.Enabled {
		t.Fatalf("download all changed playback choices: %+v, %v", cfg.Music, err)
	}
}

func TestMusicStopsWhenLeavingDrill(t *testing.T) {
	m := openMusicTestModel(t)
	m.startMusic()
	if m.musicPlayer != nil {
		t.Fatal("music started by default")
	}
	m.musicPrefs.Enabled = true
	m.saveMusic()
	m.startMusic()
	if m.musicPlayer == nil {
		t.Fatal("enabled music did not start")
	}
	m.state = StateSprinting
	m.sprintStartTime = time.Now()
	m.onSprintFinished(sprintFinishedMsg{err: errors.New("editor did not start")})
	if m.musicPlayer != nil || m.musicPlaybackErr == nil {
		t.Fatal("music not stopped / missing-track error lost")
	}
	m.startMusic()
	m.Cleanup()
	if m.musicPlayer != nil {
		t.Fatal("music survived cleanup")
	}
}
