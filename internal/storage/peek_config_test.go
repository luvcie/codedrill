package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPeekConfigDoesNotCreateDir(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)
	cfg, err := PeekConfig()
	if err != nil {
		t.Fatalf("PeekConfig() error %v", err)
	}
	if cfg.Theme != "" {
		t.Errorf("theme %q, want the default", cfg.Theme)
	}
	if _, err := os.Stat(filepath.Join(xdg, "codedrill")); !os.IsNotExist(err) {
		t.Fatalf("PeekConfig created the config dir (stat error %v)", err)
	}
}

func TestPeekConfigReadsExistingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SaveConfig(Config{Theme: "gruvbox"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := PeekConfig()
	if err != nil || cfg.Theme != "gruvbox" {
		t.Fatalf("PeekConfig() theme = %q, %v; want gruvbox", cfg.Theme, err)
	}
}
