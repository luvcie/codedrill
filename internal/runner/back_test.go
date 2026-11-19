package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
)

func TestBackHelper(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command string
		session string
		fail    bool
		want    bool
	}{
		{"back", "back", "drill-session", false, true},
		{"failed exit", "back", "drill-session", true, false},
		{"outside zellij", "back", "", false, false},
		{"submit", "s", "drill-session", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			bin := t.TempDir()
			stub := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$BACK_LOG\"\n"
			if tc.command == "back" {
				stub += "test -f " + backMarker + " || exit 2\n"
			}
			if tc.fail {
				stub += "exit 1\n"
			}
			if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte(stub), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			log := filepath.Join(t.TempDir(), "zellij.log")
			t.Setenv("BACK_LOG", log)
			t.Setenv("ZELLIJ_SESSION_NAME", tc.session)
			sess, err := NewSession(drill.Drill{ID: "back-test", EntryFile: "main.c", TestCommand: "true"}, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer sess.Cleanup()
			if sess.BackRequested() {
				t.Fatal("new session already requested back")
			}
			out, err := exec.Command(filepath.Join(sess.Workspace, tc.command)).CombinedOutput()
			if (err != nil) != (tc.fail || tc.session == "") {
				t.Fatalf("helper returned %v: %s", err, out)
			}
			if got := sess.BackRequested(); got != tc.want {
				t.Fatalf("back requested = %v, want %v", got, tc.want)
			}
			if tc.session != "" {
				if data, err := os.ReadFile(log); err != nil || string(data) != "kill-session\n"+tc.session+"\n" {
					t.Fatalf("wrong Zellij command: %q, %v", data, err)
				}
			} else if _, err := os.Stat(log); !os.IsNotExist(err) {
				t.Fatalf("back outside Zellij invoked Zellij: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(sess.Workspace, "exit")); !os.IsNotExist(err) {
				t.Fatalf("shell exit was replaced: %v", err)
			}
			data, err := os.ReadFile(filepath.Join(sess.Workspace, "start-terminal.sh"))
			if err != nil || !strings.Contains(string(data), "back  briefing (quit without saving)") {
				t.Fatalf("terminal help lacks back: %v", err)
			}
		})
	}
}

func TestBackPathsAreReserved(t *testing.T) {
	for _, name := range []string{"back", backMarker} {
		if sess, err := NewSession(drill.Drill{ID: "collision", EntryFile: name}, 0); err == nil {
			sess.Cleanup()
			t.Fatalf("student file %q can overwrite the back command", name)
		}
	}
}
