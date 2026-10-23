package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
	"github.com/muesli/termenv"
)

func TestRenderLeaderboardView(t *testing.T) {
	strlen := drill.Drill{ID: "ft_strlen", Title: "ft_strlen", Difficulty: "easy"}
	hist := &storage.HistoryStore{
		Attempts: []storage.Attempt{},
		Stats:    make(map[string]*storage.DrillStats),
	}

	view := RenderLeaderboardView(strlen, hist, false, 80)
	if !strings.Contains(view, "fastest runs") {
		t.Errorf("expected fastest runs mode indicator, got: %s", view)
	}
	if !strings.Contains(view, "No runs yet") {
		t.Errorf("expected empty notice, got: %s", view)
	}

	viewRecentEmpty := RenderLeaderboardView(strlen, hist, true, 80)
	if !strings.Contains(viewRecentEmpty, "recent runs") {
		t.Errorf("expected recent runs mode indicator, got: %s", viewRecentEmpty)
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
	if !strings.Contains(viewPop, "runs 2") {
		t.Errorf("expected stats summary box, got: %s", viewPop)
	}
	viewPopRecent := RenderLeaderboardView(strlen, hist, true, 80)
	if !strings.Contains(viewPopRecent, "status") {
		t.Errorf("expected STATUS column header, got: %s", viewPopRecent)
	}
	if !strings.Contains(viewPopRecent, "PASS") {
		t.Errorf("expected PASS badge in recent runs, got: %s", viewPopRecent)
	}
	if !strings.Contains(viewPopRecent, "FAIL") {
		t.Errorf("expected FAIL badge in recent runs, got: %s", viewPopRecent)
	}

	viewNarrow := RenderLeaderboardView(strlen, hist, false, 55)
	if !strings.Contains(viewNarrow, "rank") || !strings.Contains(viewNarrow, "time") {
		t.Errorf("expected essential columns in narrow view, got: %s", viewNarrow)
	}
	if strings.Contains(viewNarrow, "date") {
		t.Errorf("expected DATE column to be omitted in narrow view <72 cols, got: %s", viewNarrow)
	}

	hist.Stats["ft_strlen"].CurrentStreak = 3
	for _, line := range strings.Split(RenderLeaderboardView(strlen, hist, false, 60), "\n") {
		if w := lipgloss.Width(line); w > 60 {
			t.Fatalf("line is %d cols wide at width 60: %q", w, line)
		}
	}
}

// hallOfFameHist returns a drill with several runs, mixed pass and fail, so
// both modes and every stats item have something to draw.
func hallOfFameHist() (drill.Drill, *storage.HistoryStore) {
	d := drill.Drill{ID: "ft_strlen", Title: "ft_strlen", Difficulty: "easy"}
	now := time.Now()
	hist := &storage.HistoryStore{
		Attempts: []storage.Attempt{
			{DrillID: d.ID, Duration: 1500 * time.Millisecond, Timestamp: now, Passed: true, Attempts: 1},
			{DrillID: d.ID, Duration: 3500 * time.Millisecond, Timestamp: now.Add(-time.Hour), Passed: false, Attempts: 2},
			{DrillID: d.ID, Duration: 12 * time.Second, Timestamp: now.Add(-2 * time.Hour), Passed: true, Attempts: 1},
			{DrillID: d.ID, Duration: 95 * time.Second, Timestamp: now.Add(-3 * time.Hour), Passed: true, Attempts: 4},
		},
		Stats: map[string]*storage.DrillStats{
			d.ID: {
				DrillID:       d.ID,
				PersonalBest:  1500 * time.Millisecond,
				TotalRuns:     4,
				Successful:    3,
				CurrentStreak: 2,
				LastAttempt:   now,
			},
		},
	}
	return d, hist
}

// forceTrueColor makes lipgloss emit full colour SGR for the rest of the test,
// so the checks see the colours a real terminal would get.
func forceTrueColor(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

func TestHallOfFameLeftAligned(t *testing.T) {
	d, hist := hallOfFameHist()
	for _, recent := range []bool{false, true} {
		for _, width := range []int{60, 80, 120} {
			view := ansi.Strip(RenderLeaderboardView(d, hist, recent, width))
			for _, line := range strings.Split(view, "\n") {
				if line == "" {
					continue
				}
				if !strings.HasPrefix(line, "  ") {
					t.Errorf("recent=%v width %d: line does not start at column 2: %q", recent, width, line)
				}
			}
		}
	}
}

func TestHallOfFameNoFillOrPurple(t *testing.T) {
	forceTrueColor(t)
	d, hist := hallOfFameHist()
	out := []string{
		RenderLeaderboardView(d, hist, false, 80),
		RenderLeaderboardView(d, hist, true, 80),
		renderCountdown(3, 80, 24),
		renderCountdown(0, 80, 24),
	}
	for _, view := range out {
		if hasBackgroundSGR(view) {
			t.Errorf("found a background colour in output: %q", view)
		}
		if strings.Contains(view, "38;2;125;86;244") {
			t.Errorf("found the purple accent in output: %q", view)
		}
	}
}

func TestHallOfFameRowsDoNotWrap(t *testing.T) {
	d, hist := hallOfFameHist()
	for _, width := range []int{60, 72, 80} {
		for _, recent := range []bool{false, true} {
			view := ansi.Strip(RenderLeaderboardView(d, hist, recent, width))
			lines := strings.Split(view, "\n")
			divider := -1
			for i, line := range lines {
				if strings.Contains(line, "──") {
					divider = i
				}
			}
			if divider < 0 {
				t.Fatalf("width %d recent=%v: no table divider:\n%s", width, recent, view)
			}
			// One line per run after the divider. A wrapped cell would add a line.
			want := len(hist.GetTopAttempts(d.ID, 8))
			if recent {
				want = len(hist.GetRecentAttempts(d.ID, 8))
			}
			rows := lines[divider+1:]
			if len(rows) != want {
				t.Errorf("width %d recent=%v: expected %d rows, got %d:\n%s", width, recent, want, len(rows), view)
			}
		}
	}
}

func TestHallOfFameNarrowWidths(t *testing.T) {
	d, hist := hallOfFameHist()
	for _, width := range []int{30, 40, 60, 80} {
		for _, recent := range []bool{false, true} {
			view := ansi.Strip(RenderLeaderboardView(d, hist, recent, width))
			lines := strings.Split(view, "\n")
			for _, line := range lines {
				if w := lipgloss.Width(line); w > width {
					t.Fatalf("recent=%v width %d: line is %d columns: %q", recent, width, w, line)
				}
			}
			divider := -1
			for i, line := range lines {
				if strings.Contains(line, "──") {
					divider = i
				}
			}
			if divider < 0 {
				t.Fatalf("recent=%v width %d: no table divider:\n%s", recent, width, view)
			}
			attempts := hist.GetTopAttempts(d.ID, 8)
			if recent {
				attempts = hist.GetRecentAttempts(d.ID, 8)
			}
			rows := lines[divider+1:]
			if len(rows) != len(attempts) {
				t.Fatalf("recent=%v width %d: expected %d rows, got %d:\n%s", recent, width, len(attempts), len(rows), view)
			}
			header := lines[divider-1]
			for i, att := range attempts {
				grade, _ := CalculateGrade(att.Duration, d.Difficulty)
				gradeCell := "[" + string(grade) + "]"
				if recent && !att.Passed {
					gradeCell = "[-]"
				}
				checkCellsKept(t, header, rows[i], map[string]string{
					"time":    storage.FormatDuration(att.Duration),
					"grade":   gradeCell,
					"submits": fmt.Sprint(att.Attempts),
					"date":    att.Timestamp.Format("2006-01-02 15:04"),
				})
			}
		}
	}
}

func TestHallOfFameOnlyTopRankIsStyled(t *testing.T) {
	forceTrueColor(t)
	useTheme(t, "cyan")
	d, hist := hallOfFameHist()
	view := RenderLeaderboardView(d, hist, false, 80)

	top := AccentStyle.Render("★ #1")
	var first, second string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, top) {
			first = line
		}
		if strings.Contains(line, "#2") {
			second = line
		}
	}
	if first == "" {
		t.Fatalf("top rank is not bold cyan \"★ #1\": %q", view)
	}
	if second == "" {
		t.Fatalf("rank #2 missing: %q", view)
	}
	if !strings.HasPrefix(second, "  #2") {
		t.Fatalf("rank #2 should be plain text, got %q", second)
	}
	if strings.Contains(second, "38;2;0;240;255") {
		t.Fatalf("rank #2 is cyan: %q", second)
	}
}
