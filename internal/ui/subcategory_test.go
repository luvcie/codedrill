package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestAllLists42DrillsLast(t *testing.T) {
	drills := []drill.Drill{
		{ID: "42-first", Category: "42"},
		{ID: "other-first", Category: "Algorithms"},
		{ID: "exam", Category: "42 Exam 05"},
		{ID: "other-last", Category: "LeetCode"},
		{ID: "42-last", Category: "42/Exam 06"},
	}
	m := &MainModel{drills: drills, history: &storage.HistoryStore{}, categoryFilter: "ALL"}
	items := m.menuItemsFor("ALL")
	var got []string
	for _, item := range items {
		d := item.(drillItem)
		if !d.isAdd {
			got = append(got, d.drill.ID)
		}
	}
	want := []string{"other-first", "other-last", "42-first", "exam", "42-last"}
	if !slices.Equal(got, want) {
		t.Fatalf("ALL order = %v, want %v", got, want)
	}
	if next := m.stepDrill(drills[3], 1); next == nil || next.ID != "42-first" {
		t.Fatalf("next drill after other-last = %v, want 42-first", next)
	}
	if !items[len(items)-1].(drillItem).isAdd {
		t.Fatal("add row should remain last")
	}
}

func findSubcategory(t *testing.T, m *MainModel, name string) (int, drillItem) {
	t.Helper()
	for i, item := range m.list.Items() {
		di, ok := item.(drillItem)
		if ok && di.isGroup() && strings.EqualFold(di.group, name) {
			return i, di
		}
	}
	t.Fatalf("no subcategory %q in the list", name)
	return -1, drillItem{}
}

func listHasDrill(m *MainModel, id string) bool {
	for _, item := range m.list.Items() {
		if di, ok := item.(drillItem); ok && !di.isAdd && !di.isGroup() && di.drill.ID == id {
			return true
		}
	}
	return false
}

func TestSubcategoryMergesIntoTabAndCollapses(t *testing.T) {
	m := newMenuTestModel(t)
	if err := drill.SaveDrill(drill.Drill{
		ID:        "exam05",
		Title:     "Exam 05 drill",
		Category:  "42 EXAM 05",
		EntryFile: "a.c",
	}); err != nil {
		t.Fatal(err)
	}
	m.reloadDrills()

	for _, c := range m.categories {
		if strings.Contains(strings.ToUpper(c), "EXAM") {
			t.Fatalf("legacy category still has its own tab: %q", c)
		}
	}
	if !slices.Contains(m.categories, "42") {
		t.Fatalf("42 tab missing: %v", m.categories)
	}
	if d := drill.Find(m.drills, "exam05"); d == nil || d.Category != "42/Exam 05" {
		t.Fatalf("saved category not normalized: %+v", d)
	}

	switchToCategory(t, m, "42")
	idx, group := findSubcategory(t, m, "Exam 05")
	if group.collapsed {
		t.Fatal("a subcategory should start expanded")
	}
	if group.count != 1 {
		t.Fatalf("group count = %d, want 1", group.count)
	}
	if !strings.Contains(group.Title(), "EXAM 05") || !strings.HasPrefix(group.Title(), "▾") {
		t.Fatalf("expanded group title = %q", group.Title())
	}
	if !listHasDrill(m, "exam05") {
		t.Fatal("the exam drill is not under its group")
	}
	m.list.Select(idx)
	if d, ok := m.selectedDrill(); ok {
		t.Fatalf("a group header reports drill %+v", d)
	}
	m.Update(runeKey("x"))
	if m.pendingDelete != nil {
		t.Fatal("x on a group header opened the delete prompt")
	}
	m.Update(runeKey("e"))
	if m.state != StateMenu {
		t.Fatalf("e on a group header opened state %v", m.state)
	}
	found := false
	for _, hit := range m.searchHits("exam05") {
		found = found || slices.Contains(hit, true)
	}
	if !found {
		t.Fatal("search should find the drill while the group is expanded")
	}

	var options []string
	for _, o := range m.categoryOptions() {
		options = append(options, o.Value)
	}
	if !slices.Contains(options, "42/Exam 05") {
		t.Fatalf("category options lack the subcategory: %v", options)
	}

	m.list.Select(idx)
	m.activateSelected()
	idx, group = findSubcategory(t, m, "Exam 05")
	if !group.collapsed || !strings.HasPrefix(group.Title(), "▸") {
		t.Fatalf("collapsed group = %q, collapsed %v", group.Title(), group.collapsed)
	}
	if listHasDrill(m, "exam05") {
		t.Fatal("a collapsed group still lists its drills")
	}
	if slices.Contains(m.searchHits("exam05")[categoryNamed(t, m, "42")], true) {
		t.Fatal("search should not find a drill inside a collapsed group on the 42 tab")
	}

	m.list.Select(idx)
	m.activateSelected()
	_, group = findSubcategory(t, m, "Exam 05")
	if group.collapsed || !listHasDrill(m, "exam05") {
		t.Fatal("expanding the group did not bring its drill back")
	}
	if !slices.Contains(m.searchHits("exam05")[categoryNamed(t, m, "42")], true) {
		t.Fatal("search should find the drill again after expanding")
	}
}
