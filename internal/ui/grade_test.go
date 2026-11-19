package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestCalculateGrade(t *testing.T) {
	tests := []struct {
		name       string
		duration   time.Duration
		difficulty string
		wantGrade  Grade
		wantTitle  string
	}{
		{"zero duration", 0, "easy", "", ""},
		{"negative duration", -1 * time.Second, "easy", "", ""},

		{"easy S-rank boundary", 4000 * time.Millisecond, "easy", GradeS, "GODSPEED"},
		{"easy S-rank fast", 1500 * time.Millisecond, "easy", GradeS, "GODSPEED"},
		{"easy A-rank boundary", 8000 * time.Millisecond, "easy", GradeA, "MASTER"},
		{"easy A-rank mid", 6000 * time.Millisecond, "easy", GradeA, "MASTER"},
		{"easy B-rank boundary", 15000 * time.Millisecond, "easy", GradeB, "SKILLED"},
		{"easy C-rank", 15001 * time.Millisecond, "easy", GradeC, "CLEARED"},

		{"medium S-rank boundary", 12000 * time.Millisecond, "medium", GradeS, "GODSPEED"},
		{"medium A-rank boundary", 25000 * time.Millisecond, "medium", GradeA, "MASTER"},
		{"medium B-rank boundary", 50000 * time.Millisecond, "medium", GradeB, "SKILLED"},
		{"medium C-rank", 55000 * time.Millisecond, "medium", GradeC, "CLEARED"},

		{"hard S-rank boundary", 25000 * time.Millisecond, "hard", GradeS, "GODSPEED"},
		{"hard A-rank boundary", 50000 * time.Millisecond, "hard", GradeA, "MASTER"},
		{"hard B-rank boundary", 90000 * time.Millisecond, "hard", GradeB, "SKILLED"},
		{"hard C-rank", 95000 * time.Millisecond, "hard", GradeC, "CLEARED"},

		{"uppercase MEDIUM", 10000 * time.Millisecond, "MEDIUM", GradeS, "GODSPEED"},
		{"unknown difficulty defaults to easy", 3000 * time.Millisecond, "custom", GradeS, "GODSPEED"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotGrade, gotTitle := CalculateGrade(tt.duration, drill.Drill{Difficulty: tt.difficulty})
			if gotGrade != tt.wantGrade {
				t.Errorf("CalculateGrade() gotGrade = %v, want %v", gotGrade, tt.wantGrade)
			}
			if gotTitle != tt.wantTitle {
				t.Errorf("CalculateGrade() gotTitle = %v, want %v", gotTitle, tt.wantTitle)
			}
		})
	}
}

func TestRenderGradeBanner(t *testing.T) {
	banner := RenderGradeBanner(GradeS, "GODSPEED")
	if !strings.Contains(banner, "S-RANK") || !strings.Contains(banner, "GODSPEED") {
		t.Errorf("expected banner with rank and title, got %q", banner)
	}

	emptyBanner := RenderGradeBanner("", "")
	if emptyBanner != "" {
		t.Errorf("expected empty banner for empty grade, got %q", emptyBanner)
	}
}

func TestNextPaceGoal(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Second} {
		if got, top := NextPaceGoal(d, drill.Drill{Difficulty: "easy"}); got != "" || top {
			t.Errorf("NextPaceGoal(%v) = %q, %v; want nothing", d, got, top)
		}
	}
	if _, top := NextPaceGoal(3*time.Second, drill.Drill{Difficulty: "easy"}); !top {
		t.Error("an S time should report the top rank")
	}
	for _, tc := range []struct {
		d          time.Duration
		difficulty string
		want       []string
	}{
		{6 * time.Second, "easy", []string{"S-RANK at 4.000s", "2.000s to cut"}},
		{10 * time.Second, "easy", []string{"A-RANK at 8.000s", "2.000s to cut"}},
		{20 * time.Second, "easy", []string{"B-RANK at 15.000s", "5.000s to cut"}},
		{20 * time.Second, "medium", []string{"S-RANK at 12.000s"}},
		{40 * time.Second, "hard", []string{"S-RANK at 25.000s"}},
	} {
		got, top := NextPaceGoal(tc.d, drill.Drill{Difficulty: tc.difficulty})
		for _, w := range tc.want {
			if top || !strings.Contains(got, w) {
				t.Errorf("NextPaceGoal(%v, %s) = %q; want it to contain %q", tc.d, tc.difficulty, got, w)
			}
		}
	}
}

func TestRenderPaceHints(t *testing.T) {
	for difficulty, want := range map[string][]string{
		"easy":   {"S ≤ 4s", "A ≤ 8s", "B ≤ 15s"},
		"medium": {"S ≤ 12s", "A ≤ 25s", "B ≤ 50s"},
		"hard":   {"S ≤ 25s", "A ≤ 50s", "B ≤ 90s"},
	} {
		hints := RenderPaceHints(drill.Drill{Difficulty: difficulty})
		plain := ansi.Strip(hints)
		if !strings.HasPrefix(plain, "  paces  ") {
			t.Errorf("%s pace hints %q do not start with the label at column 2", difficulty, plain)
		}
		for _, w := range want {
			if !strings.Contains(plain, w) {
				t.Errorf("%s pace hints %q lack %q", difficulty, plain, w)
			}
		}
		if strings.Contains(hints, "\x1b[4") {
			t.Errorf("%s pace hints fill a background: %q", difficulty, hints)
		}
	}
}

func TestVect2PaceHintsHideSecretRank(t *testing.T) {
	d := drill.Drill{ID: "rank05-vect2", Difficulty: "hard", IsPreset: true}
	hints := ansi.Strip(RenderPaceHints(d))
	if strings.Contains(hints, "X+") || strings.Contains(settingsPreview(), "X+") {
		t.Fatal("rank previews reveal the hidden rank")
	}
	for _, want := range []string{"D clear", "D+ < 30:00", "C- < 25:00", "S < 6:00", "U < 3:00", "X < 2:30"} {
		if !strings.Contains(hints, want) {
			t.Errorf("pace hints lack %q: %s", want, hints)
		}
	}
	for _, duration := range []time.Duration{150 * time.Second, 149 * time.Second, 120 * time.Second, 119 * time.Second} {
		goal, top := NextPaceGoal(duration, d)
		if strings.Contains(goal, "X+") || strings.Contains(goal, "02m 00") {
			t.Errorf("goal at %v reveals the hidden rank: %q", duration, goal)
		}
		if duration < 150*time.Second {
			if !top || goal != "Top rank reached." {
				t.Errorf("goal at %v = %q, %v; want top rank", duration, goal, top)
			}
		} else if top || goal != "next X-RANK under 02m 30.000s" {
			t.Errorf("goal at X boundary = %q, %v", goal, top)
		}
	}
}

func TestVect2NextPaceGoals(t *testing.T) {
	d := drill.Drill{ID: "rank05-vect2", IsPreset: true}
	for _, tc := range []struct {
		duration time.Duration
		want     string
	}{
		{time.Hour, "next D+-RANK under 30m 00.000s"},
		{30 * time.Minute, "next D+-RANK under 30m 00.000s"},
		{30*time.Minute - time.Nanosecond, "next C--RANK under 25m 00.000s"},
		{10 * time.Minute, "next A--RANK under 10m 00.000s"},
		{9 * time.Minute, "next A-RANK under 09m 00.000s"},
		{5 * time.Minute, "next S+-RANK under 05m 00.000s"},
	} {
		if got, top := NextPaceGoal(tc.duration, d); top || got != tc.want {
			t.Errorf("goal at %v = %q, %v; want %q", tc.duration, got, top, tc.want)
		}
	}
}

func TestBuiltinPaceHintsHideSecretRank(t *testing.T) {
	drills, err := drill.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range drills {
		t.Run(d.ID, func(t *testing.T) {
			hints := ansi.Strip(RenderPaceHints(d))
			if strings.Contains(hints, "X+") {
				t.Fatal("pace hints reveal hidden rank")
			}
			if !strings.Contains(hints, "D clear") {
				t.Fatal("pace hints omit the completion rank")
			}
			position := 0
			targets := drill.RankTargets(d)
			for _, target := range targets[1:] {
				next := strings.Index(hints, target.Rank+" <")
				if next <= position {
					t.Errorf("pace hints omit or misorder %s: %s", target.Rank, hints)
				}
				position = next
				goal, top := NextPaceGoal(target.Time, d)
				want := "next " + target.Rank + "-RANK under " + storage.FormatDuration(target.Time)
				if top || goal != want {
					t.Errorf("goal at %v = %q, %v; want %q", target.Time, goal, top, want)
				}
			}
			for _, duration := range []time.Duration{targets[len(targets)-1].Time - time.Nanosecond, time.Second} {
				if goal, top := NextPaceGoal(duration, d); !top || goal != "Top rank reached." {
					t.Errorf("top goal reveals another rank: %q, %v", goal, top)
				}
			}
		})
	}
}
