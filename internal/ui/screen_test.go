package ui

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestScreensFitAndKeepActionsVisible(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 60, Height: 20}, {Width: 80, Height: 24}, {Width: 120, Height: 40}} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.Update(size)
			m.lastDuration = 5 * time.Second
			for _, state := range []AppState{StateMenu, StateBriefing, StateEvaluating, StateResults} {
				m.state = state
				view := m.View()
				if lipgloss.Width(view) > size.Width || lipgloss.Height(view) > size.Height {
					t.Fatalf("state %v overflows %dx%d: %dx%d", state, size.Width, size.Height, lipgloss.Width(view), lipgloss.Height(view))
				}
				if !strings.Contains(view, "[Enter") {
					t.Fatalf("state %v hides primary action", state)
				}
			}
		})
	}
}

func TestLongEvaluationScrollsWithoutLosingActions(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m.state = StateEvaluating
	m.evalOutput = strings.Repeat("compiler diagnostic\n", 80) + "FINAL DIAGNOSTIC"
	if strings.Contains(m.View(), "FINAL DIAGNOSTIC") || !strings.Contains(m.View(), "scroll") {
		t.Fatal("long output is not presented as scrollable")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if !strings.Contains(m.View(), "FINAL DIAGNOSTIC") || !strings.Contains(m.View(), "[Enter/e]") {
		t.Fatal("last diagnostic or primary action is unreachable")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if !strings.Contains(m.View(), "ft_strlen · evaluation") {
		t.Fatal("home did not restore the heading")
	}
}

func TestEvaluationBorderStartsAtLeftWithStreak(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.state = StateEvaluating
	m.evalPassed = true
	m.history.CurrentStreak = 2
	m.evalOutput = "ok\tsolution\t0.01s"
	boxes := 0
	for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		idx := strings.Index(line, "╭")
		if idx < 0 {
			continue
		}
		boxes++
		if col := lipgloss.Width(line[:idx]); col != 2 {
			t.Fatalf("evaluation border starts at column %d, want 2: %q", col, line)
		}
	}
	if boxes == 0 {
		t.Fatal("evaluation output box is missing")
	}
}

func TestRenderHelpIndentsKeysAndWraps(t *testing.T) {
	forceTrueColor(t)

	out := renderHelp("[Enter] start  [n/p] next/prev\n[l] history", 80, "")
	for _, line := range strings.Split(ansi.Strip(out), "\n") {
		if !strings.HasPrefix(line, "  ") {
			t.Fatalf("help row is not indented two columns: %q", line)
		}
	}
	if !strings.Contains(ansi.Strip(out), "[Enter] start") || !strings.Contains(ansi.Strip(out), "[l] history") {
		t.Fatalf("help lost its keys: %q", ansi.Strip(out))
	}
	if strings.Contains(out, "\x1b[1") {
		t.Fatalf("help is bold: %q", out)
	}
	if !strings.Contains(out, helpKeyStyle.Render("[Enter]")) || !strings.Contains(out, SubtextStyle.Render(" start  ")) {
		t.Fatalf("keys and labels are not styled apart: %q", out)
	}

	wrapped := renderHelp("[Enter] select  [h/l ←/→ Tab] category  [d] daily  [r] random", 30, "")
	for _, line := range strings.Split(ansi.Strip(wrapped), "\n") {
		if w := lipgloss.Width(line); w > 30 {
			t.Fatalf("help row is %d columns at width 30: %q", w, line)
		}
	}
	if !strings.Contains(ansi.Strip(wrapped), "[h/l ←/→ Tab] category") {
		t.Fatalf("a key was split from its label: %q", ansi.Strip(wrapped))
	}
}

func TestScreenHeaderTruncatesContext(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	short := m.screenHeader("level 1")
	header := m.screenHeader(strings.Repeat("streak context words ", 20))
	for _, line := range strings.Split(ansi.Strip(header), "\n") {
		if w := lipgloss.Width(line); w > 60 {
			t.Fatalf("header row is %d columns at width 60: %q", w, line)
		}
	}
	if !strings.Contains(ansi.Strip(header), "…") {
		t.Fatalf("long context was not cut: %q", ansi.Strip(header))
	}
	if lipgloss.Height(header) != lipgloss.Height(short) {
		t.Fatalf("long context wrapped: %d rows, want %d", lipgloss.Height(header), lipgloss.Height(short))
	}
}

func TestMenuLeftEdgeAndRowsAt80x24(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := m.View()
	if h := lipgloss.Height(view); h > 24 {
		t.Fatalf("menu is %d rows at 80x24", h)
	}
	rows := 0
	for _, line := range strings.Split(ansi.Strip(view), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		// The selected row keeps its border in column 0. Everything else
		// starts at column 2.
		if !strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "│ ") {
			t.Fatalf("line starts left of column 2: %q", line)
		}
		if strings.Contains(line, "PB ") {
			rows++
		}
	}
	if rows < 3 {
		t.Fatalf("menu shows %d drill rows at 80x24, want at least 3:\n%s", rows, ansi.Strip(view))
	}
}

func TestMenuRowsOmitAutoSubmit(t *testing.T) {
	it := drillItem{drill: drill.Drill{ID: "x", Title: "X", Category: "42", Difficulty: "easy", AutoSubmit: true}}
	if got := it.Description(); strings.Contains(got, "Auto-Submit") {
		t.Fatalf("menu row still shows Auto-Submit: %q", got)
	}
}

// assertColumnTwo fails when a non-blank line of the plain view starts at
// column 0 or 1.
func assertColumnTwo(t *testing.T, name, view string) {
	t.Helper()
	for _, line := range strings.Split(ansi.Strip(view), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			t.Fatalf("%s: line starts left of column 2: %q", name, line)
		}
	}
}

func TestScreenBodiesStartAtColumnTwo(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 120, Height: 40}} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.Update(size)
			m.lastDuration = 5 * time.Second
			m.earnedXP = 40
			m.levelInfo = storage.GetLevelInfo(40)
			m.resultDrill = *m.activeDrill
			m.history.CurrentStreak = 2
			m.evalOutput = "ok\tsolution\t0.01s"

			m.state = StateBriefing
			assertColumnTwo(t, "briefing", m.View())

			m.state = StateResults
			assertColumnTwo(t, "results", m.View())

			m.state = StateEvaluating
			m.evalPassed = true
			assertColumnTwo(t, "evaluation", m.View())

			m.evalPassed = false
			assertColumnTwo(t, "evaluation failed", m.View())

			m.evalPending = true
			assertColumnTwo(t, "evaluation pending", m.View())
			m.evalPending = false

			m.saveErr = fmt.Errorf("disk full")
			assertColumnTwo(t, "not saved", m.View())
			m.saveErr = nil

			m.launchFailed = true
			assertColumnTwo(t, "did not start", m.View())
		})
	}
}

func TestBriefingShowsTitleOnceInHeader(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 200})
	m.state = StateBriefing
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "ft_strlen · C · easy") {
		t.Fatalf("header does not carry the drill context: %q", view)
	}
	for _, line := range strings.Split(view, "\n") {
		if strings.TrimSpace(line) == "ft_strlen" {
			t.Fatalf("the description heading repeats the title: %q", line)
		}
	}
	if !strings.Contains(view, "Write a function that returns the length") {
		t.Fatal("description body is missing")
	}
}

// hasBackgroundSGR reports whether s sets a background colour. It reads each
// SGR sequence and skips the arguments of 38, 48 and 58 so that a foreground
// colour's index is not taken for a background.
func hasBackgroundSGR(s string) bool {
	for _, match := range regexp.MustCompile(`\x1b\[([0-9;]*)m`).FindAllStringSubmatch(s, -1) {
		params := strings.Split(match[1], ";")
		for i := 0; i < len(params); i++ {
			n, err := strconv.Atoi(params[i])
			if err != nil {
				continue
			}
			switch {
			case n == 48:
				return true
			case n == 38 || n == 58:
				if i+1 < len(params) && params[i+1] == "5" {
					i += 2
				} else if i+1 < len(params) && params[i+1] == "2" {
					i += 4
				}
			case (n >= 40 && n <= 47) || (n >= 100 && n <= 107):
				return true
			}
		}
	}
	return false
}

func TestBriefingAndResultsAreNotFilled(t *testing.T) {
	forceTrueColor(t)

	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 200})
	// Inline code in the real description keeps glamour's code background,
	// so this check uses a description without inline code.
	m.activeDrill.Description = "# ft_strlen\n\nWrite a function that returns the length of a string.\n\n## Constraints\n\n- Reproduce standard behavior."
	m.state = StateBriefing
	if view := m.View(); hasBackgroundSGR(view) {
		t.Fatalf("briefing fills a background: %q", view)
	}

	m.state = StateResults
	m.resultDrill = *m.activeDrill
	m.lastDuration = 3200 * time.Millisecond
	m.isPB = true
	m.pbDelta = 500 * time.Millisecond
	m.lastSubmits = 1
	m.earnedXP = 125
	m.levelInfo = storage.GetLevelInfo(125)
	if view := m.View(); hasBackgroundSGR(view) {
		t.Fatalf("results fill a background: %q", view)
	}
}

func TestProfileHallAndCountdownFitWindow(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 60, Height: 20}, {Width: 80, Height: 24}, {Width: 120, Height: 40}} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.Update(size)
			for _, state := range []AppState{StateStats, StateLeaderboard} {
				m.state = state
				view := m.View()
				if lipgloss.Width(view) > size.Width || lipgloss.Height(view) > size.Height {
					t.Fatalf("state %v overflows %dx%d: %dx%d", state, size.Width, size.Height, lipgloss.Width(view), lipgloss.Height(view))
				}
				assertWordmarkNearTop(t, size.Width, size.Height, ansi.Strip(view))
			}
			m.state = StateStats
			if plain := ansi.Strip(m.View()); !strings.Contains(plain, "profile · ") {
				t.Fatalf("profile lacks its header context:\n%s", plain)
			}
			m.state = StateLeaderboard
			if plain := ansi.Strip(m.View()); !strings.Contains(plain, "ft_strlen · hall of fame") {
				t.Fatalf("hall of fame lacks its header context:\n%s", plain)
			}
		})
	}
}

func TestScreensHaveNoPurpleOrFill(t *testing.T) {
	forceTrueColor(t)
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 200})
	// Inline code keeps glamour's code background, so this uses a description
	// without it. See TestBriefingAndResultsAreNotFilled.
	m.activeDrill.Description = "# ft_strlen\n\nWrite a function that returns the length of a string.\n\n## Constraints\n\n- Reproduce standard behavior."
	m.lastDuration = 3200 * time.Millisecond
	m.resultDrill = *m.activeDrill
	m.isPB = true
	m.pbDelta = 500 * time.Millisecond
	m.lastSubmits = 1
	m.earnedXP = 125
	m.levelInfo = storage.GetLevelInfo(125)
	m.history.CurrentStreak = 2
	m.evalPassed = true
	m.evalOutput = "ok\tsolution\t0.01s"

	menu := newMenuTestModel(t)
	if view := menu.View(); strings.Contains(view, "38;2;125;86;244") {
		t.Errorf("menu uses the purple accent: %q", view)
	}

	filled := map[string]AppState{
		"briefing":     StateBriefing,
		"results":      StateResults,
		"evaluation":   StateEvaluating,
		"profile":      StateStats,
		"hall of fame": StateLeaderboard,
		"countdown":    StateCountdown,
	}
	for name, state := range filled {
		m.state = state
		m.countdownCount = 3
		view := m.View()
		if strings.Contains(view, "38;2;125;86;244") {
			t.Errorf("%s uses the purple accent: %q", name, view)
		}
		if hasBackgroundSGR(view) {
			t.Errorf("%s fills a background: %q", name, view)
		}
	}
}

func TestResultsAndEvaluationUseLabelValueCopy(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 200})
	m.lastDuration = 3200 * time.Millisecond
	m.resultDrill = *m.activeDrill
	m.isPB = true
	m.pbDelta = 500 * time.Millisecond
	m.lastSubmits = 1
	m.history.CurrentStreak = 2
	m.history.BestStreak = 2
	m.state = StateResults
	plain := ansi.Strip(m.View())
	for _, want := range []string{"new personal best", "final time 3.200s", "submits 1 (first try)", "streak 2 (best 2)"} {
		if !strings.Contains(plain, want) {
			t.Errorf("results lack %q:\n%s", want, plain)
		}
	}
	for _, old := range []string{"NEW PERSONAL BEST", "Submits:", "Streak:", "final time  "} {
		if strings.Contains(plain, old) {
			t.Errorf("results still show %q", old)
		}
	}

	m.state = StateEvaluating
	m.evalPassed = true
	m.accumulatedTime = 2 * time.Second
	m.evalOutput = "ok"
	plain = ansi.Strip(m.View())
	for _, want := range []string{"Tests: passed", "Reference: unavailable", "Overall: passed", "time 2.000s", "streak 2"} {
		if !strings.Contains(plain, want) {
			t.Errorf("evaluation copy lacks %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "Time:") || strings.Contains(plain, "Streak:") {
		t.Errorf("evaluation still uses Label: value copy:\n%s", plain)
	}
}

// TestNarrowWindowsHaveNoOverlongLines checks the screens that draw the header
// in windows too narrow for the croc beside the wordmark.
func TestNarrowWindowsHaveNoOverlongLines(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 30, Height: 10}, {Width: 33, Height: 12}} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.Update(size)
			m.countdownCount = 3
			for _, state := range []AppState{StateMenu, StateCountdown, StateStats} {
				m.state = state
				view := m.View()
				for _, line := range strings.Split(view, "\n") {
					if w := lipgloss.Width(line); w > size.Width {
						t.Fatalf("state %v row is %d columns in a %d-column window: %q", state, w, size.Width, ansi.Strip(line))
					}
				}
			}
		})
	}
}

func TestNarrowHeaderDropsCrocAndKeepsColumnTwo(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 30, Height: 10})
	header := ansi.Strip(m.screenHeader("ft_strlen · finished"))
	if strings.Contains(header, ":==8") {
		t.Fatalf("narrow header still draws the croc:\n%s", header)
	}
	assertColumnTwo(t, "narrow header", header)
	if !strings.Contains(header, "codedrill") || !strings.Contains(header, "ft_strlen") {
		t.Fatalf("narrow header lost its wordmark or context:\n%s", header)
	}
	wide := newMenuTestModel(t)
	wide.Update(tea.WindowSizeMsg{Width: 34, Height: 10})
	if !strings.Contains(ansi.Strip(wide.screenHeader("x")), ":==8") {
		t.Fatal("34-column header should keep the croc")
	}
}

func TestProgressBarColoursAndRounding(t *testing.T) {
	forceTrueColor(t)
	bar := renderProgressBar(0.5, 4)
	if !strings.Contains(bar, lipgloss.NewStyle().Foreground(ColorText).Render("██")) {
		t.Fatalf("filled cells are not in the text colour: %q", bar)
	}
	if !strings.Contains(bar, lipgloss.NewStyle().Foreground(ColorBorder).Render("░░")) {
		t.Fatalf("empty cells are not in the border colour: %q", bar)
	}
	if lipgloss.NewStyle().Foreground(ColorText).Render("█") == lipgloss.NewStyle().Foreground(ColorBorder).Render("░") {
		t.Fatal("filled and empty segments have the same colour")
	}

	cases := []struct {
		progress float64
		length   int
		want     string
	}{
		{0.72, 12, "[█████████░░░] 72%"},
		{-0.1, 4, "[░░░░] 0%"},
		{1.2, 4, "[████] 100%"},
		{0.29, 4, "[█░░░] 29%"},
		{0.995, 4, "[████] 100%"},
	}
	for _, c := range cases {
		if got := ansi.Strip(renderProgressBar(c.progress, c.length)); got != c.want {
			t.Errorf("renderProgressBar(%v, %d) = %q, want %q", c.progress, c.length, got, c.want)
		}
	}
}

func TestBriefingPBLineFitsAt80x24(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.state = StateBriefing
	m.history.Stats[m.activeDrill.ID] = &storage.DrillStats{PersonalBest: 1560 * time.Millisecond}
	m.activeDrill.AutoSubmit = true
	want := "ft_strlen · C · easy · PB " + storage.FormatDuration(1560*time.Millisecond) + " (S) · auto-submit on"
	plain := ansi.Strip(m.View())
	if !strings.Contains(plain, want) {
		t.Fatalf("briefing context is cut, want %q in:\n%s", want, plain)
	}
}
