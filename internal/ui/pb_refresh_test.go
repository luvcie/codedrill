package ui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestFinishRefreshesMenuPB(t *testing.T) {
	for _, searched := range []bool{false, true} {
		name := "selected"
		if searched {
			name = "search"
		}
		t.Run(name, func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.categoryIndex = categoryNamed(t, m, "42")
			m.categoryFilter = "42"
			m.refreshList()
			m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			wantCategory := "42"
			if searched {
				// Search from tab 42 to palindrome, which is on LeetCode and
				// is not the first row, then finish a run on that row.
				m.state = StateMenu
				m.Update(runeKey("/"))
				typeSearch(m, "palindrome")
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				wantCategory = "LeetCode"
				if m.categoryFilter != wantCategory || selectedID(m) != "palindrome" || m.list.Index() == 0 {
					t.Fatalf("search landed on %q in %q at row %d, want palindrome on LeetCode", selectedID(m), m.categoryFilter, m.list.Index())
				}
			} else {
				m.list.Select(1)
			}
			selected := m.list.SelectedItem().(drillItem)
			m.activeDrill = &selected.drill
			width, height := m.list.Width(), m.list.Height()
			for _, duration := range []time.Duration{6 * time.Second, 3 * time.Second, 9 * time.Second} {
				m.state = StateEvaluating
				m.evalPassed = true
				m.accumulatedTime = duration
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
				if cmd != nil {
					m.Update(cmd())
				}
				if m.state != StateResults {
					t.Fatal("finish did not show results")
				}
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				want := m.history.GetPB(selected.drill.ID)
				item := m.list.SelectedItem().(drillItem)
				if item.drill.ID != selected.drill.ID || item.pb != want || !strings.Contains(item.Description(), storage.FormatDuration(want)) {
					t.Fatalf("selected menu item disagrees with history: %+v; PB %v", item, want)
				}
				if m.list.Width() != width || m.list.Height() != height || m.categoryFilter != wantCategory {
					t.Fatal("finish reset menu dimensions or category")
				}
				if searched && (m.search.pattern != "palindrome" || !m.search.hl) {
					t.Fatal("finish reset search")
				}
				reloaded, err := storage.LoadHistory()
				if err != nil || reloaded.GetPB(selected.drill.ID) != want {
					t.Fatalf("disk PB differs from menu: %v", err)
				}
			}
		})
	}
}
