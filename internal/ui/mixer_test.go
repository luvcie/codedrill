package ui

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/music"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestMixerSavesIndependentVolumes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	on, musicVolume, sfxVolume := true, 35, 90
	if err := storage.SaveConfig(storage.Config{
		Editor: "hx", Music: &storage.MusicSettings{Enabled: true, Volume: &musicVolume, Tracks: []string{"chk-016"}},
		Typing: &storage.TypingSettings{Enabled: &on, Volume: &sfxVolume, Pack: "prototype"},
	}); err != nil {
		t.Fatal(err)
	}
	m, err := NewMixerModel()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	music := storage.FollowVolume(ctx, .35, storage.Config.MusicVolume)
	sfx := storage.FollowVolume(ctx, .9, storage.Config.TypingVolume)
	checkVolumes := func(want [2]int) {
		t.Helper()
		reopened, err := NewMixerModel()
		if err != nil || m.volumes != want || reopened.volumes != want {
			t.Fatalf("mixer volumes = %v, want %v: %v", m.volumes, want, err)
		}
		deadline := time.Now().Add(2 * time.Second)
		for music.Gain() != float64(want[0])/100 || sfx.Gain() != float64(want[1])/100 {
			if time.Now().After(deadline) {
				t.Fatalf("live volumes = %g, %g, want %v", music.Gain(), sfx.Gain(), want)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "35%") || !strings.Contains(view, "90%") {
		t.Fatal(view)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MusicVolume() != .36 || cfg.TypingVolume() != .89 || !cfg.Music.Enabled || !cfg.TypingSoundsOn() || cfg.Editor != "hx" || cfg.Music.Tracks[0] != "chk-016" || cfg.Typing.Pack != "prototype" {
		t.Fatalf("mixer changed unrelated settings: %+v", cfg)
	}
	want := [2]int{36, 89}
	checkVolumes(want)
	for cursor := range want {
		m.cursor = cursor
		for _, step := range []struct {
			key     tea.KeyType
			presses int
			want    int
		}{
			{tea.KeyRight, 101, 100},
			{tea.KeyRight, 1, 100},
			{tea.KeyLeft, 1, 99},
			{tea.KeyLeft, 101, 0},
			{tea.KeyLeft, 1, 0},
			{tea.KeyRight, 1, 1},
		} {
			for range step.presses {
				m.Update(tea.KeyMsg{Type: step.key})
			}
			want[cursor] = step.want
			checkVolumes(want)
		}
	}
	if _, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc}); cmd == nil {
		t.Fatal("Esc did not return to the terminal")
	}
}

func TestMixerPlaybackControls(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := storage.SaveConfig(storage.Config{Music: &storage.MusicSettings{Enabled: true}}); err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join(t.TempDir(), "music.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEDRILL_MUSIC_SOCKET", socket)
	commands := make(chan music.Command, 32)
	done := make(chan struct{})
	go func() {
		defer close(done)
		state := music.PlaybackState{Title: "First song"}
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(time.Second))
				var command music.Command
				if json.NewDecoder(conn).Decode(&command) != nil {
					return
				}
				commands <- command
				switch command {
				case music.Next:
					state.Title = "Second song"
				case music.Previous:
					state.Title = "First song"
				case music.ToggleLoop:
					state.Loop = !state.Loop
				case music.ToggleShuffle:
					state.Shuffle = !state.Shuffle
				}
				_ = json.NewEncoder(conn).Encode(struct {
					State music.PlaybackState `json:"state"`
				}{state})
			}()
		}
	}()
	t.Cleanup(func() { listener.Close(); <-done })
	m, err := NewMixerModel()
	if err != nil {
		t.Fatal(err)
	}
	init := m.Init()().(tea.BatchMsg)
	m.Update(init[0]())
	if command := <-commands; command != music.GetStatus {
		t.Fatalf("mixer did not request the current song: %s", command)
	}
	view := ansi.Strip(m.View())
	for _, label := range []string{"Song: First song", "Loop current song: off", "Shuffle: off", "[p] previous", "[n] next", "[r] loop current song", "[s] shuffle", "[←/→] volume"} {
		if !strings.Contains(view, label) {
			t.Fatalf("mixer is missing %q:\n%s", label, view)
		}
	}
	_, next := m.Update(runeKey("n"))
	if _, previous := m.Update(runeKey("p")); previous != nil {
		t.Fatal("playback commands were not kept in keypress order")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.volumes[0] != 36 {
		t.Fatal("playback controls blocked precise volume adjustment")
	}
	_, previous := m.Update(next())
	if m.playback.Title != "Second song" || <-commands != music.Next || previous == nil {
		t.Fatal("Next was not applied before Previous")
	}
	m.Update(previous())
	if m.playback.Title != "First song" || <-commands != music.Previous {
		t.Fatal("queued Previous was lost")
	}
	for _, step := range []struct {
		key     string
		command music.Command
	}{{"r", music.ToggleLoop}, {"s", music.ToggleShuffle}} {
		_, cmd := m.Update(runeKey(step.key))
		m.Update(cmd())
		if <-commands != step.command {
			t.Fatalf("wrong playback command for %s", step.key)
		}
	}
	if !m.playback.Loop || !m.playback.Shuffle {
		t.Fatal("mixer did not display the player's loop and shuffle states")
	}
	if _, err := music.Control(socket, music.Next); err != nil || <-commands != music.Next {
		t.Fatalf("song did not advance outside the mixer: %v", err)
	}
	_, poll := m.Update(mixerTickMsg{})
	pollCommands := poll().(tea.BatchMsg)
	m.Update(pollCommands[0]())
	if m.playback.Title != "Second song" || <-commands != music.GetStatus {
		t.Fatal("mixer did not refresh the song title")
	}
	cfg, err := storage.LoadConfig()
	if err != nil || cfg.Music.Shuffle || cfg.MusicVolume() != .36 {
		t.Fatalf("session Shuffle changed the next drill's default: %+v, %v", cfg.Music, err)
	}
	for _, width := range []int{30, 50, 80} {
		m.Update(tea.WindowSizeMsg{Width: width})
		for _, line := range strings.Split(m.View(), "\n") {
			if ansi.StringWidth(line) > width {
				t.Fatalf("mixer exceeds %d columns: %q", width, line)
			}
		}
	}
}

func TestMixerWithoutPlayback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEDRILL_MUSIC_SOCKET", "")
	m, err := NewMixerModel()
	if err != nil {
		t.Fatal(err)
	}
	_, cmd := m.Update(runeKey("n"))
	m.Update(cmd())
	if !strings.Contains(ansi.Strip(m.View()), "No music playing") {
		t.Fatal("mixer did not explain that music is stopped")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.volumes[0] != 36 {
		t.Fatal("missing playback disabled volume controls")
	}
}

func TestMixerKeepsAudioOffAndReportsSaveErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMixerModel()
	if err != nil {
		t.Fatal(err)
	}
	m.changeVolume(5)
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m.changeVolume(5)
	cfg, err := storage.LoadConfig()
	if err != nil || cfg.Music.Enabled || cfg.TypingSoundsOn() {
		t.Fatalf("adjusting volume enabled audio: %+v, %v", cfg, err)
	}
	path, _ := storage.GetConfigPath()
	if err := os.WriteFile(path, []byte("broken config"), 0600); err != nil {
		t.Fatal(err)
	}
	before := m.volumes
	m.changeVolume(5)
	if m.err == nil || m.volumes != before || !strings.Contains(ansi.Strip(m.View()), "Could not save") {
		t.Fatal("failed save was not reported")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "broken config" {
		t.Fatal("mixer overwrote unreadable settings")
	}
	for _, w := range []int{30, 50, 80} {
		m.Update(tea.WindowSizeMsg{Width: w, Height: 20})
		for _, line := range strings.Split(m.View(), "\n") {
			if ansi.StringWidth(line) > w {
				t.Fatalf("mixer does not fit %d columns: %q", w, line)
			}
		}
	}
}
