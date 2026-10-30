package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeRawConfig(t *testing.T, body string) {
	t.Helper()
	path, err := GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func readRawConfig(t *testing.T) string {
	t.Helper()
	path, err := GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestConfigMouseMissingKeyMeansOn(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if !(Config{}).MouseOn() {
		t.Fatal("zero config has mouse off")
	}
	writeRawConfig(t, `{"editor": "vim", "theme": "nord"}`)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.MouseOn() {
		t.Fatal("config without the mouse key has mouse off")
	}
}

func TestConfigMouseOffRoundTrips(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	off := false
	if err := SaveConfig(Config{Mouse: &off}); err != nil {
		t.Fatal(err)
	}
	if data := readRawConfig(t); !strings.Contains(data, `"mouse": false`) {
		t.Fatalf("config.json lacks mouse false:\n%s", data)
	}
	cfg, err := LoadConfig()
	if err != nil || cfg.MouseOn() {
		t.Fatalf("LoadConfig() mouse on = %v, %v; want off", cfg.MouseOn(), err)
	}
}

func TestConfigThemeSavePreservesMouseOff(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeRawConfig(t, `{"mouse": false}`)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Theme = "gruvbox"
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig()
	if err != nil || cfg.MouseOn() || cfg.Theme != "gruvbox" {
		t.Fatalf("after theme save: mouse on %v, theme %q, err %v; want off and gruvbox", cfg.MouseOn(), cfg.Theme, err)
	}
}

func TestConfigMouseSavePreservesTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeRawConfig(t, `{"theme": "nord"}`)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	off := false
	cfg.Mouse = &off
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadConfig()
	if err != nil || cfg.MouseOn() || cfg.Theme != "nord" {
		t.Fatalf("after mouse save: mouse on %v, theme %q, err %v; want off and nord", cfg.MouseOn(), cfg.Theme, err)
	}
}
