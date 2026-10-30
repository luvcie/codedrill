package storage

import (
	"os"
	"strings"
	"testing"
)

func TestConfigThemeRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SaveConfig(Config{Theme: "gruvbox"}); err != nil {
		t.Fatal(err)
	}
	path, err := GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"theme": "gruvbox"`) {
		t.Fatalf("config.json lacks the theme:\n%s", data)
	}
	cfg, err := LoadConfig()
	if err != nil || cfg.Theme != "gruvbox" {
		t.Fatalf("LoadConfig() theme = %q, %v; want gruvbox", cfg.Theme, err)
	}
}

func TestConfigWithoutThemeOmitsIt(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SaveConfig(Config{Editor: "vim"}); err != nil {
		t.Fatal(err)
	}
	path, _ := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "theme") {
		t.Fatalf("empty theme was written:\n%s", data)
	}
}
