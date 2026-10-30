package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/ui"
)

// TestReadOnlyCommandsDoNotCreateConfigDir checks that help and version, which
// only read the config to pick a theme, leave no codedrill directory behind.
func TestReadOnlyCommandsDoNotCreateConfigDir(t *testing.T) {
	useDefaultTheme(t)
	for _, args := range [][]string{{"--help"}, {"version"}} {
		t.Run(args[0], func(t *testing.T) {
			xdg := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", xdg)
			if code, _, errOut := runCLI(t, args...); code != 0 {
				t.Fatalf("exit %d, stderr %q", code, errOut)
			}
			if _, err := os.Stat(filepath.Join(xdg, "codedrill")); !os.IsNotExist(err) {
				t.Fatalf("config dir exists after %v (stat error %v)", args, err)
			}
		})
	}
}

// TestStartupAppliesSavedTheme checks that a saved theme is applied by a
// read-only command without the command rewriting the config.
func TestStartupAppliesSavedTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	if err := storage.SaveConfig(storage.Config{Theme: "gruvbox"}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(mustConfigPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runCLI(t, "version"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if got := ui.CurrentTheme().ID; got != "gruvbox" {
		t.Errorf("theme after startup = %q, want gruvbox", got)
	}
	after, err := os.ReadFile(mustConfigPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("config changed by a read-only command:\n%s\n%s", before, after)
	}
}

func mustConfigPath(t *testing.T) string {
	t.Helper()
	path, err := storage.GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}
