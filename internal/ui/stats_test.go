package ui

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
	"github.com/muesli/termenv"
)

func TestRenderStatsDashboard(t *testing.T) {
	hist := &storage.HistoryStore{
		Attempts: []storage.Attempt{},
		Stats:    make(map[string]*storage.DrillStats),
	}

	drills := []drill.Drill{
		{ID: "ft_strlen", Title: "ft_strlen", Language: "c"},
	}

	emptyView := RenderStatsDashboard(hist, drills, 90)
	if strings.Contains(emptyView, "SPEEDRUNNER") {
		t.Errorf("profile should not shout a SPEEDRUNNER title")
	}
	if !strings.Contains(emptyView, "level 1") {
		t.Errorf("expected level line in stats dashboard")
	}
	if !strings.Contains(emptyView, "XP") {
		t.Errorf("expected XP in stats dashboard")
	}
	if !strings.Contains(emptyView, "total sprints") {
		t.Errorf("expected total sprints stat in empty stats dashboard")
	}
	if !strings.Contains(emptyView, "win streak") {
		t.Errorf("expected win streak stat in empty stats dashboard")
	}
	if !strings.Contains(emptyView, "today") {
		t.Errorf("expected today stat in empty stats dashboard")
	}
	if !strings.Contains(emptyView, "No runs yet") {
		t.Errorf("expected notice in empty stats dashboard")
	}

	hist.Attempts = append(hist.Attempts, storage.Attempt{
		DrillID:   "ft_strlen",
		Duration:  2 * time.Second,
		Passed:    true,
		Timestamp: time.Now(),
		Attempts:  1,
	})
	hist.Stats["ft_strlen"] = &storage.DrillStats{
		DrillID:      "ft_strlen",
		PersonalBest: 2 * time.Second,
		TotalRuns:    1,
		Successful:   1,
		LastAttempt:  time.Now(),
	}

	title := storage.GetLevelInfo(hist.GetTotalXP()).Title
	narrowView := RenderStatsDashboard(hist, drills, 80)
	if !strings.Contains(narrowView, title) {
		t.Errorf("expected level title %q in narrow view", title)
	}
	if !strings.Contains(narrowView, "practice time") {
		t.Errorf("expected practice time in narrow view")
	}
	if !strings.Contains(narrowView, "today") {
		t.Errorf("expected today in narrow view")
	}

	wideView := RenderStatsDashboard(hist, drills, 140)
	if !strings.Contains(wideView, title) {
		t.Errorf("expected level title %q in wide view", title)
	}
	if !strings.Contains(wideView, "cleared") {
		t.Errorf("expected cleared in wide view")
	}
	if !strings.Contains(wideView, "today") {
		t.Errorf("expected today in wide view")
	}

	ultraNarrowView := RenderStatsDashboard(hist, drills, 60)
	if !strings.Contains(ultraNarrowView, title) {
		t.Errorf("expected level title %q in ultra-narrow view", title)
	}
	if !strings.Contains(ultraNarrowView, "drill") {
		t.Errorf("expected drill header in ultra-narrow view")
	}

}

// statsTestHistory builds a history with a long drill title, a drill with a
// long run streak and a drill that was never passed.
func statsTestHistory() (*storage.HistoryStore, []drill.Drill) {
	hist := &storage.HistoryStore{
		Attempts: []storage.Attempt{},
		Stats:    make(map[string]*storage.DrillStats),
	}
	drills := []drill.Drill{
		{ID: "long", Title: "a very long drill title that goes on and on", Language: "python", Difficulty: "medium"},
		{ID: "short", Title: "ft_strlen", Language: "c", Difficulty: "easy"},
		{ID: "failed", Title: "ft_never", Language: "c", Difficulty: "easy"},
	}
	now := time.Now()
	hist.Stats["long"] = &storage.DrillStats{
		DrillID:       "long",
		PersonalBest:  95 * time.Second,
		TotalRuns:     12,
		Successful:    12,
		CurrentStreak: 12,
		LastAttempt:   now,
	}
	hist.Stats["short"] = &storage.DrillStats{
		DrillID:       "short",
		PersonalBest:  1560 * time.Millisecond,
		TotalRuns:     3,
		Successful:    3,
		CurrentStreak: 3,
		LastAttempt:   now,
	}
	hist.Stats["failed"] = &storage.DrillStats{
		DrillID:     "failed",
		TotalRuns:   1,
		LastAttempt: now,
	}
	hist.Attempts = append(hist.Attempts,
		storage.Attempt{DrillID: "long", Duration: 95 * time.Second, Passed: true, Timestamp: now, Attempts: 1},
		storage.Attempt{DrillID: "short", Duration: 1560 * time.Millisecond, Passed: true, Timestamp: now, Attempts: 1},
		storage.Attempt{DrillID: "failed", Duration: 4 * time.Second, Passed: false, Timestamp: now, Attempts: 1},
	)
	hist.CurrentStreak = 3
	hist.BestStreak = 12
	return hist, drills
}

// tableRows returns the stripped lines of the drill table: everything from
// the header row to the end of the output. The best column is the one every
// width keeps.
func tableRows(t *testing.T, out string) []string {
	t.Helper()
	lines := strings.Split(ansi.Strip(out), "\n")
	for i, line := range lines {
		if strings.Contains(line, "drill") && strings.Contains(line, "best") {
			return lines[i:]
		}
	}
	t.Fatalf("no drill table header in:\n%s", ansi.Strip(out))
	return nil
}

func TestStatsTableRowsDoNotWrap(t *testing.T) {
	hist, drills := statsTestHistory()
	for _, width := range []int{60, 80, 120} {
		t.Run(fmt.Sprintf("width%d", width), func(t *testing.T) {
			out := RenderStatsDashboard(hist, drills, width)
			for _, line := range strings.Split(ansi.Strip(out), "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Fatalf("line is %d columns at width %d: %q", w, width, line)
				}
			}
			rows := tableRows(t, out)
			if len(rows) < 5 {
				t.Fatalf("expected header, divider and three drill rows, got %q", rows)
			}
			if !strings.Contains(rows[2], "a very long") || !strings.Contains(rows[2], "…") {
				t.Fatalf("long title is not truncated on one row: %q", rows[2])
			}
			if !strings.Contains(rows[2], "12, 12 in a row") {
				t.Fatalf("long streak is not on the same row as its drill: %q", rows[2])
			}
			if !strings.Contains(rows[3], "ft_strlen") || !strings.Contains(rows[3], "1.560s S") {
				t.Fatalf("short drill row is split or wrong: %q", rows[3])
			}
			if !strings.Contains(rows[4], "ft_never") || !strings.Contains(rows[4], "--") {
				t.Fatalf("unpassed drill row is split or wrong: %q", rows[4])
			}
		})
	}
}

func TestStatsLinesStartAtColumnTwo(t *testing.T) {
	hist, drills := statsTestHistory()
	for _, width := range []int{0, 60, 80, 120} {
		out := RenderStatsDashboard(hist, drills, width)
		for _, line := range strings.Split(ansi.Strip(out), "\n") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if indent := len(line) - len(strings.TrimLeft(line, " ")); indent != 2 {
				t.Fatalf("width %d: line starts at column %d, want 2: %q", width, indent, line)
			}
		}
	}

	empty := &storage.HistoryStore{Stats: make(map[string]*storage.DrillStats)}
	for _, line := range strings.Split(ansi.Strip(RenderStatsDashboard(empty, drills, 80)), "\n") {
		if strings.Contains(line, "No runs yet") && !strings.HasPrefix(line, "  No runs yet.") {
			t.Fatalf("empty notice is not at column 2: %q", line)
		}
	}
}

func TestStatsHasNoFillOrPurple(t *testing.T) {
	useTheme(t, "cyan")
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })

	hist, drills := statsTestHistory()
	out := RenderStatsDashboard(hist, drills, 100)
	if !strings.Contains(out, "\x1b[") {
		t.Fatal("no colour was rendered, the check below would pass vacuously")
	}
	if !strings.Contains(out, "38;2;0;240;255") {
		t.Fatal("cyan accent is missing")
	}
	if hasBackgroundSGR(out) {
		t.Fatalf("background colour in output: %q", out)
	}
	if strings.Contains(out, "38;2;125;86;244") {
		t.Fatal("purple accent is still used")
	}
}

// checkCellsKept fails when a shown column lost part of its value without an
// ellipsis. header is the stripped header row, cells maps each column name to
// its full value.
func checkCellsKept(t *testing.T, header, row string, cells map[string]string) {
	t.Helper()
	for name, value := range cells {
		if !strings.Contains(header, name) || strings.Contains(row, value) {
			continue
		}
		if !strings.Contains(row, "…") {
			t.Fatalf("%s is cut without an ellipsis in row %q", name, row)
		}
	}
}

func TestStatsTableNarrowWidths(t *testing.T) {
	hist, drills := statsTestHistory()
	for _, width := range []int{30, 40, 50, 60, 80, 120} {
		t.Run(fmt.Sprintf("width%d", width), func(t *testing.T) {
			out := RenderStatsDashboard(hist, drills, width)
			for _, line := range strings.Split(ansi.Strip(out), "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Fatalf("line is %d columns at width %d: %q", w, width, line)
				}
			}
			rows := tableRows(t, out)
			if len(rows) != 5 {
				t.Fatalf("expected header, divider and three drill rows, got %d lines: %q", len(rows), rows)
			}
			header := rows[0]
			checkCellsKept(t, header, rows[2], map[string]string{
				"drill":    "a very long drill title",
				"lang":     "PYTHON",
				"best":     "01m 35.000s",
				"sprints":  "12, 12 in a row",
				"last run": time.Now().Format("2006-01-02"),
			})
			checkCellsKept(t, header, rows[3], map[string]string{
				"drill":   "ft_strlen",
				"lang":    "C",
				"best":    "1.560s S",
				"sprints": "3, 3 in a row",
			})
			checkCellsKept(t, header, rows[4], map[string]string{
				"drill":   "ft_never",
				"lang":    "C",
				"best":    "--",
				"sprints": "1",
			})
		})
	}
}

func TestStatsTableDropsColumnsInOrder(t *testing.T) {
	hist, drills := statsTestHistory()
	header := func(width int) string {
		return tableRows(t, RenderStatsDashboard(hist, drills, width))[0]
	}
	if h := header(120); !strings.Contains(h, "last run") || !strings.Contains(h, "sprints") || !strings.Contains(h, "lang") {
		t.Fatalf("wide table should show every column: %q", h)
	}
	if h := header(80); strings.Contains(h, "last run") || !strings.Contains(h, "sprints") {
		t.Fatalf("width 80 should drop only the last run column: %q", h)
	}
	if h := header(40); strings.Contains(h, "sprints") || !strings.Contains(h, "lang") {
		t.Fatalf("width 40 should drop sprints before lang: %q", h)
	}
	if h := header(30); strings.Contains(h, "lang") || !strings.Contains(h, "best") {
		t.Fatalf("width 30 should drop lang and keep best: %q", h)
	}
}

func TestLevelLineKeepsDimTextAfterBar(t *testing.T) {
	forceTrueColor(t)
	hist := &storage.HistoryStore{Stats: make(map[string]*storage.DrillStats)}
	lvl := storage.GetLevelInfo(hist.GetTotalXP())
	xp := fmt.Sprintf("%d / %d XP · %d to level %d", lvl.TotalXP, lvl.NextLevelXP, lvl.NextLevelXP-lvl.TotalXP, lvl.Level+1)

	line := renderLevelLines(hist, 120)[0]
	if !strings.Contains(line, renderProgressBar(lvl.Progress, 10)) {
		t.Fatalf("level line does not carry the bar as returned by renderProgressBar: %q", line)
	}
	if !strings.Contains(line, SubtextStyle.Render(xp)) {
		t.Fatalf("text after the bar is not in the dim colour, want %q in %q", SubtextStyle.Render(xp), line)
	}
	if !strings.Contains(ansi.Strip(line), xp) {
		t.Fatalf("XP text missing from level line: %q", ansi.Strip(line))
	}
}

func TestLevelLineFitsNarrowWidths(t *testing.T) {
	forceTrueColor(t)
	// 260 clears at 50 XP each put the player in level 21, Keyboard Virtuoso,
	// the longest level title.
	attempts := make([]storage.Attempt, 260)
	for i := range attempts {
		attempts[i] = storage.Attempt{Passed: true, Attempts: 2}
	}
	hist := &storage.HistoryStore{Attempts: attempts, Stats: make(map[string]*storage.DrillStats)}
	lvl := storage.GetLevelInfo(hist.GetTotalXP())
	if lvl.Title != "Keyboard Virtuoso" {
		t.Fatalf("test history is in %q, want Keyboard Virtuoso", lvl.Title)
	}
	pct := fmt.Sprintf("] %d%%", int(math.Round(lvl.Progress*100)))

	for _, width := range []int{30, 40, 50, 60, 74, 80, 120} {
		lines := renderLevelLines(hist, width)
		for _, line := range lines {
			if w := lipgloss.Width(line); w > width-len(statsIndent) {
				t.Errorf("width %d: line is %d cols wide: %q", width, w, ansi.Strip(line))
			}
		}
		if plain := ansi.Strip(strings.Join(lines, "\n")); !strings.Contains(plain, pct) {
			t.Errorf("width %d: percent %q missing from level lines:\n%s", width, pct, plain)
		}
	}
}
