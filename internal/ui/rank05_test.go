package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/drill"
)

func TestRank5RandomCategoryFilters(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, filter := range []string{"42/rank 5", "rank 5", "42/RANK 5"} {
		m, err := NewMainModelWithRandomFiltered(filter)
		if err != nil {
			t.Fatal(err)
		}
		if m.activeDrill == nil || m.activeDrill.Category != "42/rank 5" || !strings.HasPrefix(m.activeDrill.ID, "rank05-") {
			t.Fatalf("random filter %q chose the wrong drill: %+v", filter, m.activeDrill)
		}
	}
}

func TestRank05MenuAndBriefing(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	switchToCategory(t, m, "42")
	index, group := findSubcategory(t, m, "rank 5")
	if group.count != 5 || group.collapsed {
		t.Fatalf("fresh rank 5 group: %+v", group)
	}
	for i, item := range m.list.Items() {
		it := item.(drillItem)
		m.list.Select(i)
		row := listRowOf(t, m, it.Title())
		if got, ok := m.menuItemAt(1, row); !ok || got != i {
			t.Fatalf("fresh menu row %q hit item %d, %v; want %d", it.Title(), got, ok, i)
		}
	}
	m.list.Select(index)
	m.activateSelected()
	if listHasDrill(m, "rank05-vect2") {
		t.Fatal("collapsed group exposes its children")
	}
	m.activateSelected()
	if !listHasDrill(m, "rank05-vect2") {
		t.Fatal("expanded group has no vector drill")
	}
	for _, d := range drill.Filter(m.drills, "42/rank 5") {
		briefing, err := NewMainModelWithTarget(d.ID)
		if err != nil {
			t.Fatal(err)
		}
		for _, preset := range []string{"1", "2", "3", "4"} {
			briefing.Update(runeKey(preset))
			if issue := briefing.briefingProblem(); issue != "" {
				t.Fatalf("%s mode %s is unavailable: %s", d.ID, preset, issue)
			}
		}
	}
}
