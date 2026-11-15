package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luvcie/codedrill/internal/drill"
)

func TestPaceUsesPublicRanks(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "date"), []byte("#!/bin/sh\nprintf '%s\\n' \"$PACE_NOW_MS\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	drills, err := drill.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	drills = append(drills, drill.Drill{ID: "custom", Difficulty: "hard", EntryFile: "solution.sh"})
	for _, d := range drills {
		t.Run(d.ID, func(t *testing.T) {
			sess, err := NewSession(d, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer sess.Cleanup()
			for _, name := range []string{"pace", "start-terminal.sh"} {
				data, err := os.ReadFile(filepath.Join(sess.Workspace, name))
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(data), "X+") {
					t.Fatalf("%s reveals the hidden rank", name)
				}
			}
			if err := os.WriteFile(filepath.Join(sess.Workspace, ".sprint_start"), []byte("0"), 0644); err != nil {
				t.Fatal(err)
			}
			check := func(elapsed time.Duration, want string) {
				t.Helper()
				cmd := exec.Command("./pace")
				cmd.Dir = sess.Workspace
				cmd.Env = append(os.Environ(), fmt.Sprintf("PACE_NOW_MS=%d", elapsed.Milliseconds()))
				out, err := cmd.CombinedOutput()
				if err != nil || !strings.Contains(string(out), "  "+want+" rank  ") || strings.Contains(string(out), "X+") {
					t.Fatalf("pace at %v = %s (%v), want %s rank", elapsed, out, err, want)
				}
			}
			targets := drill.RankTargets(d)
			for i, target := range targets[1:] {
				for _, offset := range []time.Duration{-time.Millisecond, 0, time.Millisecond} {
					want := targets[i].Rank
					if offset < 0 || target.Inclusive && offset == 0 {
						want = target.Rank
					}
					check(target.Time+offset, want)
				}
			}
			check(time.Second, targets[len(targets)-1].Rank)
		})
	}
}
