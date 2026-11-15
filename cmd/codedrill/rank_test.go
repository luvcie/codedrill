package main

import (
	"strings"
	"testing"
	"time"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestVect2InfoTargets(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	code, out, errOut := runCLI(t, "info", "vect2")
	if code != 0 {
		t.Fatalf("info: %s", errOut)
	}
	for _, want := range []string{"D on any successful completion", "D+ < 30m0s", "S < 6m0s", "X < 2m30s"} {
		if !strings.Contains(out, want) {
			t.Errorf("info lacks %q: %s", want, out)
		}
	}
	if strings.Contains(out, "X+") {
		t.Fatal("info reveals unearned hidden rank")
	}
}

func TestBuiltinRanksInCommands(t *testing.T) {
	drills, err := drill.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range drills {
		t.Run(d.ID, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			targets := drill.RankTargets(d)
			a, x := targets[9].Time, targets[16].Time
			history, err := storage.LoadHistory()
			if err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				duration time.Duration
				passed   bool
				pbRank   string
			}{
				{a, true, "A-"},
				{time.Second, false, "A-"},
				{x - time.Millisecond, true, "X"},
				{time.Second, true, "X+"},
				{a, true, "X+"},
			} {
				if _, _, err := history.RecordAttempt(d.ID, tc.duration, tc.passed, 1, false); err != nil {
					t.Fatal(err)
				}
				for _, args := range [][]string{
					{"info", d.ID}, {"list"}, {"search", d.ID},
					{"top", d.ID}, {"recent", d.ID}, {"stats"},
				} {
					code, out, errOut := runCLI(t, args...)
					want := "[" + tc.pbRank + "]"
					switch args[0] {
					case "info":
						want = "(" + tc.pbRank + "-RANK)"
					case "list", "search":
						want = "(" + tc.pbRank + ")"
					}
					if code != 0 || !strings.Contains(out, want) {
						t.Errorf("%v lacks PB rank %s: %s (%s)", args, tc.pbRank, out, errOut)
					}
					if tc.pbRank != "X+" && strings.Contains(out, "X+") {
						t.Errorf("%v reveals unearned X+: %s", args, out)
					}
					if args[0] == "info" && strings.Contains(strings.Split(out, "Personal best:")[0], "X+") {
						t.Fatal("target list reveals the hidden rank after earning it")
					}
					if args[0] == "recent" {
						for _, line := range strings.Split(out, "\n") {
							if strings.Contains(line, "FAIL") && strings.Contains(line, "[") {
								t.Errorf("failed attempt displays a rank: %s", line)
							}
						}
					}
				}
			}
		})
	}
}
