package storage

import "testing"

func TestTypingSettingsDefaults(t *testing.T) {
	cfg := Config{}
	if cfg.TypingSoundsOn() {
		t.Fatal("typing sounds are on by default")
	}
	if cfg.TypingPackID() != "" {
		t.Fatalf("pack %q, want empty", cfg.TypingPackID())
	}
	if got := cfg.TypingVolume(); got != 0.7 {
		t.Fatalf("volume %v, want 0.7", got)
	}
	if cfg.TypingWordSoundsOn() {
		t.Fatal("word completion sounds are on by default")
	}
	lock, combo, quad := cfg.TypingKeySettings()
	if lock != "" || combo != "" || quad != "" {
		t.Fatalf("key settings %q %q %q, want empty defaults", lock, combo, quad)
	}
}

func TestTypingSettingsPersist(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	on := true
	volume := 42
	words := true
	want := Config{
		Editor: "hx",
		Typing: &TypingSettings{
			Enabled: &on, Pack: "sfx-test-pack", Volume: &volume, WordCompletion: &words,
			LockKeys: "{,enter", ComboKeys: ";", QuadKeys: "}",
		},
	}
	if err := SaveConfig(want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !got.TypingSoundsOn() || got.TypingPackID() != "sfx-test-pack" {
		t.Fatalf("typing settings lost: %+v", got.Typing)
	}
	if v := int(got.TypingVolume()*100 + 0.5); v != 42 {
		t.Fatalf("volume %d, want 42", v)
	}
	if !got.TypingWordSoundsOn() {
		t.Fatalf("word completion setting lost: %+v", got.Typing)
	}
	lock, combo, quad := got.TypingKeySettings()
	if lock != "{,enter" || combo != ";" || quad != "}" {
		t.Fatalf("keys %q %q %q", lock, combo, quad)
	}
	// Other fields survive a typing-only change.
	typing := got.Typing
	enabled := false
	typing.Enabled = &enabled
	if err := SaveConfig(got); err != nil {
		t.Fatal(err)
	}
	again, _ := LoadConfig()
	if again.Editor != "hx" || again.TypingSoundsOn() {
		t.Fatalf("editor %q enabled %v after toggle", again.Editor, again.TypingSoundsOn())
	}
}

func TestTypingVolumeClamp(t *testing.T) {
	low, high := -20, 240
	if got := (Config{Typing: &TypingSettings{Volume: &low}}).TypingVolume(); got != 0 {
		t.Fatalf("negative volume %v", got)
	}
	if got := (Config{Typing: &TypingSettings{Volume: &high}}).TypingVolume(); got != 1 {
		t.Fatalf("high volume %v", got)
	}
}
