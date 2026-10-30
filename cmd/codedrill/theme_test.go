package main

import (
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/ui"
)

// useDefaultTheme resets the palette after a test that runs CLI commands,
// because run applies the configured theme.
func useDefaultTheme(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { ui.ApplyTheme(ui.Themes[0]) })
}

func TestConfigThemeSaves(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	code, out, errOut := runCLI(t, "config", "theme", "GRUVBOX")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "Theme set to Gruvbox.") {
		t.Fatalf("output %q lacks the confirmation", out)
	}
	cfg, err := storage.LoadConfig()
	if err != nil || cfg.Theme != "gruvbox" {
		t.Fatalf("saved theme = %q, %v; want gruvbox", cfg.Theme, err)
	}
}

func TestConfigThemeRejectsUnknownAndListsIDs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	code, out, errOut := runCLI(t, "config", "theme", "nope")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if out != "" {
		t.Errorf("stdout %q, want nothing", out)
	}
	if !strings.HasPrefix(errOut, "codedrill: unknown theme") {
		t.Errorf("stderr %q lacks the codedrill prefix", errOut)
	}
	for _, th := range ui.Themes {
		if !strings.Contains(errOut, th.ID) {
			t.Errorf("error lacks theme %s: %q", th.ID, errOut)
		}
	}
	if cfg, _ := storage.LoadConfig(); cfg.Theme != "" {
		t.Errorf("unknown theme was saved: %q", cfg.Theme)
	}
}

func TestConfigThemeListsAllWithCurrentMarked(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)

	code, out, errOut := runCLI(t, "config", "theme")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	assertThemeList(t, out, "legacy")

	if code, _, errOut := runCLI(t, "config", "theme", "gruvbox"); code != 0 {
		t.Fatalf("set gruvbox: exit %d, stderr %q", code, errOut)
	}
	_, out, _ = runCLI(t, "config", "theme")
	assertThemeList(t, out, "gruvbox")
}

// assertThemeList checks that out has one row per theme and a * on only the
// current one.
func assertThemeList(t *testing.T, out, current string) {
	t.Helper()
	rows := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(line, "Set with") {
			continue
		}
		id := fields[0]
		if id == "*" {
			id = fields[1]
		}
		rows[id] = strings.HasPrefix(line, "* ")
	}
	marked := 0
	for _, th := range ui.Themes {
		isCurrent, listed := rows[th.ID]
		if !listed {
			t.Errorf("theme %s is not listed:\n%s", th.ID, out)
		}
		if isCurrent {
			marked++
			if th.ID != current {
				t.Errorf("%s is marked, want %s", th.ID, current)
			}
		}
	}
	if marked != 1 || len(rows) != len(ui.Themes) {
		t.Errorf("%d rows, %d marked; want %d rows and 1 mark:\n%s", len(rows), marked, len(ui.Themes), out)
	}
}

// themeListIDs returns the theme IDs in the order `config theme` lists them.
func themeListIDs(t *testing.T, out string) []string {
	t.Helper()
	var ids []string
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(line, "Set with") {
			continue
		}
		id := fields[0]
		if id == "*" {
			id = fields[1]
		}
		ids = append(ids, id)
	}
	return ids
}

func TestConfigThemeListsLegacyFirstThenCyan(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	_, out, _ := runCLI(t, "config", "theme")
	ids := themeListIDs(t, out)
	if len(ids) != 12 {
		t.Fatalf("listed %d themes, want 12:\n%s", len(ids), out)
	}
	if ids[0] != "legacy" || ids[1] != "cyan" {
		t.Errorf("first two themes are %v, want legacy then cyan:\n%s", ids[:2], out)
	}
	for _, id := range ids {
		if id == "default" {
			t.Errorf("config theme still lists the old default ID:\n%s", out)
		}
	}
}

func TestConfigShowsEffectiveTheme(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	if _, out, _ := runCLI(t, "config"); !strings.Contains(out, "Theme:          legacy (Legacy codedrill)") {
		t.Errorf("config output lacks the legacy default theme:\n%s", out)
	}

	if err := storage.SaveConfig(storage.Config{Theme: "bogus"}); err != nil {
		t.Fatal(err)
	}
	if _, out, _ := runCLI(t, "config"); !strings.Contains(out, "bogus (unknown, using legacy)") {
		t.Errorf("config output does not note the unknown theme:\n%s", out)
	}
}

func TestConfigOldDefaultThemeIsCyan(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	if err := storage.SaveConfig(storage.Config{Theme: "default"}); err != nil {
		t.Fatal(err)
	}
	_, out, _ := runCLI(t, "config")
	if !strings.Contains(out, "Theme:          cyan (Cyan)") {
		t.Errorf("config output for the old default ID lacks cyan:\n%s", out)
	}
	if got := ui.CurrentTheme().ID; got != "cyan" {
		t.Errorf("commands draw in %q, want cyan for the old default ID", got)
	}
	_, out, _ = runCLI(t, "config", "theme")
	assertThemeList(t, out, "cyan")
}

func TestConfigThemeDefaultSavesCyan(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	if code, _, errOut := runCLI(t, "config", "theme", "default"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	cfg, err := storage.LoadConfig()
	if err != nil || cfg.Theme != "cyan" {
		t.Fatalf("saved theme = %q, %v; want cyan, the real ID", cfg.Theme, err)
	}
}
