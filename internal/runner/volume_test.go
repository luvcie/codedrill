package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
)

func TestDrillVolumeShortcut(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	stub := filepath.Join(t.TempDir(), "codedrill with spaces")
	socketFile := filepath.Join(t.TempDir(), "socket")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\nprintf '%s' \"$CODEDRILL_MUSIC_SOCKET\" > "+shellQuote(socketFile)+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	previous := executablePath
	executablePath = func() string { return stub }
	t.Cleanup(func() { executablePath = previous })
	s, err := NewSession(drill.Drill{ID: "volume-test", EntryFile: "main.c"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	for _, name := range []string{"v", "volume"} {
		out, err := exec.Command(filepath.Join(s.Workspace, name)).CombinedOutput()
		if err != nil || string(out) != "volume\n" {
			t.Fatalf("%s: %q, %v", name, out, err)
		}
		socket, err := os.ReadFile(socketFile)
		if err != nil || string(socket) != s.MusicSocket() {
			t.Fatalf("%s did not connect to its drill's player: %q, %v", name, socket, err)
		}
	}
	banner, err := os.ReadFile(filepath.Join(s.Workspace, "start-terminal.sh"))
	if err != nil || !strings.Contains(string(banner), "v  volume (music / SFX)") {
		t.Fatal("terminal banner has no mixer command")
	}
	for _, name := range []string{"v", "volume", ".codedrill-music.sock"} {
		if _, err := NewSession(drill.Drill{ID: "collision", EntryFile: name}, 0); err == nil {
			t.Fatalf("student file could overwrite %s", name)
		}
	}
}
