package ui

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
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
