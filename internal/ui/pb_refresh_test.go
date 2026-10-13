package ui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestFinishRefreshesMenuPB(t *testing.T) {
	for _, filtered := range []bool{false, true} {
		name := "selected"
		if filtered {
			name = "search"
		}
		t.Run(name, func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.categoryFilter = "42"
			m.refreshList()
			m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			if filtered {
				m.list.SetFilterText("ft_s")
			}
			m.list.Select(1)
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
				if m.list.Width() != width || m.list.Height() != height || m.categoryFilter != "42" {
					t.Fatal("finish reset menu dimensions or category")
				}
				if filtered && (m.list.FilterValue() != "ft_s" || m.list.FilterState() != list.FilterApplied) {
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
