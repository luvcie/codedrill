package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestBuiltinRanksFollowSavedPersonalBest(t *testing.T) {
	drills, err := drill.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	for _, builtin := range drills {
		t.Run(builtin.ID, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			sound := false
			if err := storage.SaveConfig(storage.Config{Sound: &sound}); err != nil {
				t.Fatal(err)
			}
			m, err := NewMainModelWithTarget(builtin.ID)
			if err != nil {
				t.Fatal(err)
			}
			m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
			d := *m.activeDrill
			targets := drill.RankTargets(d)
			a, aPlus, x := targets[9].Time, targets[10].Time, targets[16].Time
			if strings.Contains(m.viewBriefing(), "X+") {
				t.Fatal("unearned hidden rank appears in briefing")
			}
			for _, tc := range []struct {
				duration time.Duration
				passed   bool
				result   string
				pb       time.Duration
				pbRank   string
			}{
				{a, true, "A-", a, "A-"},
				{time.Second, false, "", a, "A-"},
				{x, true, "U", x, "U"},
				{x - time.Millisecond, true, "X", x - time.Millisecond, "X"},
				{time.Second, true, "X+", time.Second, "X+"},
				{aPlus, true, "A", time.Second, "X+"},
			} {
				m.accumulatedTime = tc.duration
				if tc.passed {
					if cmd := m.recordPass(); cmd != nil {
						m.Update(cmd())
					}
					result := ansi.Strip(m.viewResults())
					if !strings.Contains(result, tc.result+"-RANK") {
						t.Fatalf("result at %v lacks %s: %s", tc.duration, tc.result, result)
					}
					if tc.pbRank != "X+" && strings.Contains(result, "X+") {
						t.Fatal("result reveals an unearned hidden rank")
					}
				} else {
					m.recordForfeit()
				}
				if m.saveErr != nil {
					t.Fatal(m.saveErr)
				}
				history, err := storage.LoadHistory()
				if err != nil {
					t.Fatal(err)
				}
				if got := history.GetPB(d.ID); got != tc.pb {
					t.Fatalf("saved PB = %v, want %v", got, tc.pb)
				}
				m.history = history
				var item drillItem
				for _, entry := range m.list.Items() {
					if it, ok := entry.(drillItem); ok && it.drill.ID == d.ID {
						item = it
						break
					}
				}
				if item.pb != tc.pb {
					t.Fatalf("menu PB = %v, want %v", item.pb, tc.pb)
				}
				views := map[string]struct{ view, want string }{
					"menu title": {item.Title(), "[" + tc.pbRank + "-RANK]"},
					"menu PB":    {item.Description(), "(" + tc.pbRank + ")"},
					"briefing":   {m.viewBriefing(), "PB " + storage.FormatDuration(tc.pb) + " (" + tc.pbRank + ")"},
					"stats":      {RenderStatsDashboard(history, []drill.Drill{d}, 100), storage.FormatDuration(tc.pb) + " " + tc.pbRank},
					"top":        {RenderLeaderboardView(d, history, false, 100), "[" + tc.pbRank + "]"},
					"recent":     {RenderLeaderboardView(d, history, true, 100), "[" + tc.pbRank + "]"},
				}
				for name, rendered := range views {
					view := ansi.Strip(rendered.view)
					if !strings.Contains(view, rendered.want) {
						t.Errorf("%s lacks %q: %s", name, rendered.want, view)
					}
					if tc.pbRank != "X+" && strings.Contains(view, "X+") {
						t.Errorf("%s reveals unearned X+: %s", name, view)
					}
					if name == "recent" {
						for _, line := range strings.Split(ansi.Strip(view), "\n") {
							if strings.Contains(line, "FAIL") && !strings.Contains(line, "[-]") {
								t.Errorf("failed attempt displays a rank: %s", line)
							}
						}
					}
				}
				if strings.Contains(RenderPaceHints(d), "X+") {
					t.Fatal("pace hints reveal hidden rank after earning it")
				}
			}
		})
	}
}

func TestBuiltinRankScreensFit(t *testing.T) {
	drills, err := drill.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range drills {
		t.Run(d.ID, func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.activeDrill = &d
			m.resultDrill = d
			m.lastDuration = time.Second
			for _, size := range []tea.WindowSizeMsg{{Width: 60, Height: 20}, {Width: 80, Height: 24}} {
				m.Update(size)
				for _, state := range []AppState{StateBriefing, StateResults} {
					m.state = state
					view := m.View()
					if lipgloss.Width(view) > size.Width || lipgloss.Height(view) > size.Height {
						t.Fatalf("state %v overflows %dx%d", state, size.Width, size.Height)
					}
					if !strings.Contains(view, "[Enter") {
						t.Fatalf("state %v hides primary action", state)
					}
				}
			}
		})
	}
}
