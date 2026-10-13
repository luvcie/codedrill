package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestRenderLeaderboardView(t *testing.T) {
	strlen := drill.Drill{ID: "ft_strlen", Title: "ft_strlen", Difficulty: "easy"}
	hist := &storage.HistoryStore{
		Attempts: []storage.Attempt{},
		Stats:    make(map[string]*storage.DrillStats),
	}

	view := RenderLeaderboardView(strlen, hist, false, 80)
	if !strings.Contains(view, "HALL OF FAME: ft_strlen") {
		t.Errorf("expected title in leaderboard, got: %s", view)
	}
	if !strings.Contains(view, "TOP FASTEST") {
		t.Errorf("expected TOP FASTEST mode indicator, got: %s", view)
	}
	if !strings.Contains(view, "No runs yet") {
		t.Errorf("expected empty notice, got: %s", view)
	}

	viewRecentEmpty := RenderLeaderboardView(strlen, hist, true, 80)
	if !strings.Contains(viewRecentEmpty, "RECENT RUNS") {
		t.Errorf("expected RECENT RUNS mode indicator, got: %s", viewRecentEmpty)
	}

	hist.Attempts = append(hist.Attempts, storage.Attempt{
		DrillID:   "ft_strlen",
		Duration:  1500 * time.Millisecond,
		Timestamp: time.Now(),
		Passed:    true,
		Attempts:  1,
	})
	hist.Attempts = append(hist.Attempts, storage.Attempt{
		DrillID:   "ft_strlen",
		Duration:  3500 * time.Millisecond,
		Timestamp: time.Now(),
		Passed:    false,
		Attempts:  2,
	})
	hist.Stats["ft_strlen"] = &storage.DrillStats{
		DrillID:      "ft_strlen",
		PersonalBest: 1500 * time.Millisecond,
		TotalRuns:    2,
		Successful:   1,
		LastAttempt:  time.Now(),
	}

	viewPop := RenderLeaderboardView(strlen, hist, false, 80)
	if !strings.Contains(viewPop, "#1") {
		t.Errorf("expected rank #1, got: %s", viewPop)
	}
	if !strings.Contains(viewPop, "[S]") {
		t.Errorf("expected S grade badge, got: %s", viewPop)
	}
	if !strings.Contains(viewPop, "RUNS: 2") {
		t.Errorf("expected stats summary box, got: %s", viewPop)
	}
	viewPopRecent := RenderLeaderboardView(strlen, hist, true, 80)
	if !strings.Contains(viewPopRecent, "STATUS") {
		t.Errorf("expected STATUS column header, got: %s", viewPopRecent)
	}
	if !strings.Contains(viewPopRecent, "PASS") {
		t.Errorf("expected PASS badge in recent runs, got: %s", viewPopRecent)
	}
	if !strings.Contains(viewPopRecent, "FAIL") {
		t.Errorf("expected FAIL badge in recent runs, got: %s", viewPopRecent)
	}

	viewNarrow := RenderLeaderboardView(strlen, hist, false, 55)
	if !strings.Contains(viewNarrow, "RANK") || !strings.Contains(viewNarrow, "TIME") {
		t.Errorf("expected essential columns in narrow view, got: %s", viewNarrow)
	}
	if strings.Contains(viewNarrow, "DATE") {
		t.Errorf("expected DATE column to be omitted in narrow view <72 cols, got: %s", viewNarrow)
	}

	hist.Stats["ft_strlen"].CurrentStreak = 3
	for _, line := range strings.Split(RenderLeaderboardView(strlen, hist, false, 60), "\n") {
		if w := lipgloss.Width(line); w > 60 {
			t.Fatalf("line is %d cols wide at width 60: %q", w, line)
		}
	}
}
