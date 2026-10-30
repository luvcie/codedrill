package main

import (
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/storage"
)

func TestConfigMouseOffSaves(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	code, out, errOut := runCLI(t, "config", "mouse", "off")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "Mouse off.") {
		t.Fatalf("output %q lacks the confirmation", out)
	}
	cfg, err := storage.LoadConfig()
	if err != nil || cfg.MouseOn() {
		t.Fatalf("saved mouse on = %v, %v; want off", cfg.MouseOn(), err)
	}
}

func TestConfigMouseRejectsOtherValues(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	code, out, errOut := runCLI(t, "config", "mouse", "maybe")
	if code != 1 {
		t.Fatalf("exit %d, want 1", code)
	}
	if out != "" {
		t.Errorf("stdout %q, want nothing", out)
	}
	want := `codedrill: mouse must be on or off, not "maybe"` + "\n"
	if errOut != want {
		t.Errorf("stderr %q, want %q", errOut, want)
	}
	if cfg, _ := storage.LoadConfig(); !cfg.MouseOn() {
		t.Error("rejected value changed the setting")
	}
}

func TestConfigShowsMouse(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	useDefaultTheme(t)
	code, out, errOut := runCLI(t, "config")
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if !strings.Contains(out, "Mouse:          on") {
		t.Errorf("default config does not show mouse on:\n%s", out)
	}
	if !strings.Contains(out, "mouse <on|off>") {
		t.Errorf("change line does not list mouse:\n%s", out)
	}
	if code, _, errOut := runCLI(t, "config", "mouse", "off"); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	_, out, _ = runCLI(t, "config")
	if !strings.Contains(out, "Mouse:          off") {
		t.Errorf("config after mouse off does not show it:\n%s", out)
	}
}
