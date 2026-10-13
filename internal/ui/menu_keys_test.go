package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/drill"
)

func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func newMenuTestModel(t *testing.T) *MainModel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return m
}

func selectDrill(t *testing.T, m *MainModel, id string) {
	t.Helper()
	for i, it := range m.list.Items() {
		if di := it.(drillItem); !di.isAdd && di.drill.ID == id {
			m.list.Select(i)
			return
		}
	}
	t.Fatalf("drill %s not in list", id)
}

func TestMenuHLSwitchCategories(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("l"))
	if m.state != StateMenu || m.categoryFilter != m.categories[1] {
		t.Fatalf("l: state %v filter %q, want menu on %q", m.state, m.categoryFilter, m.categories[1])
	}
	m.Update(runeKey("h"))
	m.Update(runeKey("h"))
	if last := m.categories[len(m.categories)-1]; m.categoryFilter != last {
		t.Fatalf("h should wrap to %q, got %q", last, m.categoryFilter)
	}
}

func TestMenuVOpensHistory(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("v"))
	if m.state != StateLeaderboard {
		t.Fatalf("v: got state %v", m.state)
	}
}

func Test42CategoryName(t *testing.T) {
	m := newMenuTestModel(t)
	for _, c := range m.categories {
		if strings.EqualFold(c, "42 Exam") {
			t.Fatal("old 42 Exam category still present")
		}
	}
	if !strings.Contains(m.renderCategoryHeader(), "42") {
		t.Fatalf("header lacks 42: %s", m.renderCategoryHeader())
	}
}

func TestMenuDeleteCustomDrill(t *testing.T) {
	m := newMenuTestModel(t)
	if err := drill.SaveDrill(drill.Drill{ID: "mine", Title: "Mine", Category: "Rust", EntryFile: "a.rs"}); err != nil {
		t.Fatal(err)
	}
	m.reloadDrills()
	selectDrill(t, m, "mine")

	m.Update(runeKey("x"))
	if m.pendingDelete == nil || !strings.Contains(m.View(), `Delete "Mine"?`) {
		t.Fatalf("x should ask for confirmation, view:\n%s", m.View())
	}
	m.Update(runeKey("n"))
	if m.pendingDelete != nil {
		t.Fatal("n should cancel")
	}
	selectDrill(t, m, "mine")

	m.Update(runeKey("x"))
	m.Update(runeKey("y"))
	for _, d := range m.drills {
		if d.ID == "mine" {
			t.Fatal("drill still loaded after confirming delete")
		}
	}
	for _, c := range m.categories {
		if c == "Rust" {
			t.Fatal("empty category Rust still listed")
		}
	}
	if !strings.Contains(m.View(), `Deleted "Mine"`) {
		t.Fatalf("missing delete notice:\n%s", m.View())
	}
}

func TestMenuHidePresetDrill(t *testing.T) {
	m := newMenuTestModel(t)
	selectDrill(t, m, "ft_strlen")
	m.Update(runeKey("x"))
	if !strings.Contains(m.View(), "Hide built-in") {
		t.Fatalf("preset prompt should say hide:\n%s", m.View())
	}
	m.Update(runeKey("y"))
	for _, d := range m.drills {
		if d.ID == "ft_strlen" {
			t.Fatal("hidden preset still loaded")
		}
	}
	if !strings.Contains(m.View(), "codedrill restore ft_strlen") {
		t.Fatalf("notice should explain restore:\n%s", m.View())
	}
}

func TestCategoryOptionsIncludeUserCategories(t *testing.T) {
	m := newMenuTestModel(t)
	m.categories = append(m.categories, "Rust")
	var values []string
	for _, o := range m.categoryOptions() {
		values = append(values, o.Value)
	}
	joined := strings.Join(values, ",")
	if !strings.Contains(joined, "42") || !strings.Contains(joined, "Rust") || values[len(values)-1] != newCategoryOption {
		t.Fatalf("options = %q", values)
	}
	if validateCategoryName("  ") == nil || validateCategoryName("all") == nil || validateCategoryName("Rust") != nil {
		t.Fatal("category name validation is wrong")
	}
}
