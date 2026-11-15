package storage

import "testing"

func TestMusicSettingsRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Music != nil || cfg.MusicVolume() != .35 {
		t.Fatalf("music defaults: %+v", cfg)
	}
	enabled, volume := true, 60
	cfg.Typing = &TypingSettings{Enabled: &enabled, Volume: &volume, Pack: "existing-pack"}
	musicVolume := 0
	cfg.Music = &MusicSettings{Enabled: true, Volume: &musicVolume, Folder: "/my/music", Tracks: []string{"chk-016"}}
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Music.Enabled || cfg.MusicVolume() != 0 || cfg.Music.Folder != "/my/music" || cfg.Music.Tracks[0] != "chk-016" {
		t.Fatalf("music settings: %+v", cfg.Music)
	}
	if !cfg.TypingSoundsOn() || cfg.TypingVolume() != .6 || cfg.TypingPackID() != "existing-pack" {
		t.Fatal("music altered typing sounds")
	}
	for _, test := range []struct {
		volume int
		want   float64
	}{{-5, 0}, {125, 1}} {
		cfg.Music.Volume = &test.volume
		if cfg.MusicVolume() != test.want {
			t.Fatal("music volume not clamped")
		}
	}
}
