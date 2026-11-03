package ui

import (
	"slices"
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

func TestMenuArrowKeysSwitchCategories(t *testing.T) {
	m := newMenuTestModel(t)
	n := len(m.categories)
	if n < 2 {
		t.Fatalf("need at least 2 categories, got %v", m.categories)
	}
	// Right must act like l, Left like h, including wrap-around at both ends.
	steps := []struct {
		key  tea.KeyMsg
		want int
	}{
		{tea.KeyMsg{Type: tea.KeyRight}, 1},
		{tea.KeyMsg{Type: tea.KeyRight}, 2 % n},
		{tea.KeyMsg{Type: tea.KeyLeft}, 1},
		{tea.KeyMsg{Type: tea.KeyLeft}, 0},
		{tea.KeyMsg{Type: tea.KeyLeft}, n - 1},
		{tea.KeyMsg{Type: tea.KeyRight}, 0},
	}
	for i, s := range steps {
		m.Update(s.key)
		if m.state != StateMenu || m.categoryIndex != s.want || m.categoryFilter != m.categories[s.want] {
			t.Fatalf("step %d (%s): state %v index %d filter %q, want menu on index %d (%q)",
				i, s.key, m.state, m.categoryIndex, m.categoryFilter, s.want, m.categories[s.want])
		}
		if m.list.Index() != 0 || m.list.Paginator.Page != 0 {
			t.Fatalf("step %d (%s): list jumped to index %d page %d", i, s.key, m.list.Index(), m.list.Paginator.Page)
		}
	}

	// The list must not page on arrows, but pgdown still pages.
	if keys := m.list.KeyMap.NextPage.Keys(); slices.Contains(keys, "right") || !slices.Contains(keys, "pgdown") {
		t.Fatalf("NextPage keys = %q", keys)
	}
	if keys := m.list.KeyMap.PrevPage.Keys(); slices.Contains(keys, "left") || !slices.Contains(keys, "pgup") {
		t.Fatalf("PrevPage keys = %q", keys)
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

func TestMenuEditCustomDrill(t *testing.T) {
	m := newMenuTestModel(t)
	if err := drill.SaveDrill(drill.Drill{ID: "mine", Title: "Mine", Category: "Rust", EntryFile: "a.rs"}); err != nil {
		t.Fatal(err)
	}
	m.reloadDrills()
	selectDrill(t, m, "mine")

	m.Update(runeKey("e"))
	if m.state != StateEditDrill {
		t.Fatalf("e: got state %v, want StateEditDrill", m.state)
	}
	if m.editDrill == nil || m.editDrill.ID != "mine" || m.editDrill.Title != "Mine" {
		t.Fatalf("editDrill not set correctly: %+v", m.editDrill)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateMenu {
		t.Fatalf("Esc: got state %v, want StateMenu", m.state)
	}
	if m.editDrill != nil {
		t.Fatal("editDrill should be nil after cancel")
	}
	if !strings.Contains(m.View(), "Cancelled.") {
		t.Fatalf("missing cancel notice:\\n%s", m.View())
	}
}

func TestMenuEditPresetDrillNotAllowed(t *testing.T) {
	m := newMenuTestModel(t)
	selectDrill(t, m, "ft_strlen")

	m.Update(runeKey("e"))
	if m.state != StateMenu {
		t.Fatalf("e: got state %v, want StateMenu", m.state)
	}
	if m.editDrill != nil {
		t.Fatal("editDrill should be nil for preset drill")
	}
	if !strings.Contains(m.View(), "cannot be edited") {
		t.Fatalf("missing cannot-be-edited notice:\\n%s", m.View())
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

func TestCategoryTabOrder(t *testing.T) {
	m := newMenuTestModel(t)
	if len(m.categories) < 2 || m.categories[0] != "ALL" {
		t.Fatalf("tabs %v, want ALL first", m.categories)
	}
	if last := m.categories[len(m.categories)-1]; last != "42" {
		t.Fatalf("tabs %v, want 42 last", m.categories)
	}

	// A drill with no category shows as Custom, and an explicit Custom is the
	// same tab. Both must come after 42. Rust sorts before 42.
	if err := drill.SaveDrill(drill.Drill{ID: "bare", Title: "Bare", EntryFile: "a.txt"}); err != nil {
		t.Fatal(err)
	}
	if err := drill.SaveDrill(drill.Drill{ID: "named", Title: "Named", Category: "Custom", EntryFile: "b.txt"}); err != nil {
		t.Fatal(err)
	}
	if err := drill.SaveDrill(drill.Drill{ID: "rusty", Title: "Rusty", Category: "Rust", EntryFile: "c.rs"}); err != nil {
		t.Fatal(err)
	}
	m.reloadDrills()
	want := []string{"ALL", "Algorithms", "LeetCode", "Rust", "42", "Custom"}
	if !slices.Equal(m.categories, want) {
		t.Fatalf("tabs = %q, want %q", m.categories, want)
	}
	t.Logf("tab order: %q", m.categories)

	// The add-drill form lists the same order, then "+ New category...".
	var values []string
	for _, o := range m.categoryOptions() {
		values = append(values, o.Value)
	}
	wantValues := []string{"Algorithms", "LeetCode", "Rust", "42", "Custom", newCategoryOption}
	if !slices.Equal(values, wantValues) {
		t.Fatalf("category options = %q, want %q", values, wantValues)
	}
}
