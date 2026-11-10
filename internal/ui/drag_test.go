package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func layoutBlocksFor(t *testing.T, category string) []storage.LayoutBlock {
	t.Helper()
	layout, err := storage.LoadLayout()
	if err != nil {
		t.Fatal(err)
	}
	for key, blocks := range layout.Categories {
		if strings.EqualFold(key, category) {
			return blocks
		}
	}
	t.Fatalf("no saved layout for %q", category)
	return nil
}

func dragRow(t *testing.T, m *MainModel, from, to int) {
	t.Helper()
	m.Update(leftClick(3, from))
	if m.press == nil {
		t.Fatal("the press did not start")
	}
	m.Update(leftMotion(3, to))
	if !m.press.dragging {
		t.Fatalf("moving from row %d to %d did not start a drag", from, to)
	}
	m.Update(leftRelease(3, to))
	if m.press != nil {
		t.Fatal("release left the drag active")
	}
}

func TestDragAutoScrollsToNextPage(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	switchToCategory(t, m, "42")
	if m.list.Paginator.TotalPages < 2 {
		t.Skip("the list has one page at this size")
	}
	per := m.list.Paginator.PerPage
	source := m.list.Items()[0].(drillItem)
	sourceRow := listRowOf(t, m, itemTitle(m, 0))

	m.Update(leftClick(3, sourceRow))
	_, cmd := m.Update(leftMotion(3, 29))
	if m.press == nil || !m.press.dragging {
		t.Fatal("the edge motion did not start a drag")
	}
	if cmd == nil {
		t.Fatal("holding at the bottom did not schedule an auto-scroll")
	}
	if m.list.Paginator.Page != 0 {
		t.Fatalf("the timer moved the page before it fired: page %d", m.list.Paginator.Page)
	}

	m.Update(dragScrollMsg{press: m.press})
	if m.list.Paginator.Page != 1 {
		t.Fatalf("after the auto-scroll: page %d, want 1", m.list.Paginator.Page)
	}

	targetIndex := per + 1
	if targetIndex >= len(m.list.Items()) {
		targetIndex = per
	}
	target := m.list.Items()[targetIndex].(drillItem)
	if target.isAdd || target.isGroup() {
		t.Fatalf("target row is not a drill: %+v", target)
	}
	targetRow := listRowOf(t, m, itemTitle(m, targetIndex))
	m.Update(leftMotion(3, targetRow))
	if m.press == nil || m.press.target != targetIndex {
		t.Fatalf("drop target = %v, want %d", m.press, targetIndex)
	}
	m.Update(leftRelease(3, targetRow))

	pos := make(map[string]int)
	for i, item := range m.list.Items() {
		it := item.(drillItem)
		if !it.isAdd && !it.isGroup() {
			pos[it.drill.ID] = i
		}
	}
	if pos[source.drill.ID] != pos[target.drill.ID]+1 {
		t.Fatalf("order after scrolling = %v, want %q after %q", pos, source.drill.ID, target.drill.ID)
	}

	blocks := layoutBlocksFor(t, "42")
	seenSource, seenTarget := false, false
	for _, block := range blocks {
		switch block.Drill {
		case source.drill.ID:
			seenSource = true
			if !seenTarget {
				t.Fatalf("saved order has %q before %q", source.drill.ID, target.drill.ID)
			}
		case target.drill.ID:
			seenTarget = true
		}
	}
	if !seenSource || !seenTarget {
		t.Fatalf("saved layout is missing the moved drills: %+v", blocks)
	}
}

func TestKeyboardDragReordersDrillAndPersists(t *testing.T) {
	m := newMenuTestModel(t)
	switchToCategory(t, m, "42")
	items := m.list.Items()
	if len(items) < 3 {
		t.Fatalf("need two drills, got %d rows", len(items))
	}
	first, second := items[0].(drillItem), items[1].(drillItem)
	if first.isAdd || first.isGroup() || second.isAdd || second.isGroup() {
		t.Fatalf("expected loose drills, got %+v and %+v", first, second)
	}
	m.list.Select(0)

	m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	items = m.list.Items()
	if items[0].(drillItem).drill.ID != second.drill.ID || items[1].(drillItem).drill.ID != first.drill.ID {
		t.Fatalf("shift+down order = %q, %q; want %q, %q",
			items[0].(drillItem).drill.ID, items[1].(drillItem).drill.ID, second.drill.ID, first.drill.ID)
	}
	blocks := layoutBlocksFor(t, "42")
	if len(blocks) < 2 || blocks[0].Drill != second.drill.ID || blocks[1].Drill != first.drill.ID {
		t.Fatalf("saved blocks after shift+down = %+v", blocks)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	items = m.list.Items()
	if items[0].(drillItem).drill.ID != first.drill.ID || items[1].(drillItem).drill.ID != second.drill.ID {
		t.Fatalf("shift+up order = %q, %q; want %q, %q",
			items[0].(drillItem).drill.ID, items[1].(drillItem).drill.ID, first.drill.ID, second.drill.ID)
	}
}

func TestKeyboardDragReordersSubcategoryAndChild(t *testing.T) {
	m := newMenuTestModel(t)
	for _, d := range []drill.Drill{
		{ID: "alpha", Title: "Alpha", Category: "42/Exam 05", EntryFile: "a.c"},
		{ID: "beta", Title: "Beta", Category: "42/Exam 05", EntryFile: "b.c"},
		{ID: "gamma", Title: "Gamma", Category: "42/Exam 06", EntryFile: "c.c"},
	} {
		if err := drill.SaveDrill(d); err != nil {
			t.Fatal(err)
		}
	}
	m.reloadDrills()
	switchToCategory(t, m, "42")

	idx, _ := findSubcategory(t, m, "Exam 05")
	m.list.Select(idx)
	m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	first, _ := findSubcategory(t, m, "Exam 06")
	second, _ := findSubcategory(t, m, "Exam 05")
	if first >= second {
		t.Fatalf("group rows at %d and %d, want Exam 06 above Exam 05", first, second)
	}

	selectDrill(t, m, "alpha")
	m.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	idx, _ = findSubcategory(t, m, "Exam 05")
	items := m.list.Items()
	if idx+2 >= len(items) {
		t.Fatalf("group at %d has no room for its children", idx)
	}
	if items[idx+1].(drillItem).drill.ID != "beta" || items[idx+2].(drillItem).drill.ID != "alpha" {
		t.Fatalf("group children = %q, %q; want beta, alpha",
			items[idx+1].(drillItem).drill.ID, items[idx+2].(drillItem).drill.ID)
	}

	for _, block := range layoutBlocksFor(t, "42") {
		if !strings.EqualFold(block.Group, "Exam 05") {
			continue
		}
		if len(block.Drills) != 2 || block.Drills[0] != "beta" || block.Drills[1] != "alpha" {
			t.Fatalf("saved Exam 05 drills = %v", block.Drills)
		}
		return
	}
	t.Fatal("saved layout has no Exam 05 group")
}

func TestDragDrillReordersAndPersists(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	switchToCategory(t, m, "42")

	from := rowOf(t, m, "ft_strlen")
	to := rowOf(t, m, "ft_swap")
	m.Update(leftClick(3, from))
	m.Update(leftMotion(3, to))
	if m.press.target != 1 {
		t.Fatalf("drop target = %d, want 1", m.press.target)
	}
	if !strings.Contains(m.View(), "drop here") {
		t.Fatal("the drop marker is not drawn")
	}
	m.Update(leftRelease(3, to))
	if m.state != StateMenu || m.press != nil {
		t.Fatalf("after drop: state %v, press %v", m.state, m.press != nil)
	}

	items := m.list.Items()
	if got := items[0].(drillItem).drill.ID; got != "ft_swap" {
		t.Fatalf("first drill = %q, want ft_swap", got)
	}
	if got := items[1].(drillItem).drill.ID; got != "ft_strlen" {
		t.Fatalf("second drill = %q, want ft_strlen", got)
	}
	blocks := layoutBlocksFor(t, "42")
	if len(blocks) < 2 || blocks[0].Drill != "ft_swap" || blocks[1].Drill != "ft_strlen" {
		t.Fatalf("saved blocks = %+v", blocks)
	}

	restarted, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	restarted.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	switchToCategory(t, restarted, "42")
	if got := restarted.list.Items()[0].(drillItem).drill.ID; got != "ft_swap" {
		t.Fatalf("after restart first drill = %q, want ft_swap", got)
	}
}

func TestDragSubcategoryReordersAndPersists(t *testing.T) {
	m := newMenuTestModel(t)
	for _, d := range []drill.Drill{
		{ID: "exam05a", Title: "Exam 05 drill", Category: "42/Exam 05", EntryFile: "a.c"},
		{ID: "exam06a", Title: "Exam 06 drill", Category: "42/Exam 06", EntryFile: "b.c"},
	} {
		if err := drill.SaveDrill(d); err != nil {
			t.Fatal(err)
		}
	}
	m.reloadDrills()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	switchToCategory(t, m, "42")

	from := rowOf(t, m, "EXAM 06")
	to := rowOf(t, m, "EXAM 05")
	dragRow(t, m, from, to)

	first, ok := m.list.Items()[0].(drillItem)
	if !ok || !first.isGroup() || !strings.EqualFold(first.group, "Exam 06") {
		t.Fatalf("first row = %+v, want the Exam 06 group", first)
	}
	blocks := layoutBlocksFor(t, "42")
	if len(blocks) < 2 || !strings.EqualFold(blocks[0].Group, "Exam 06") || !strings.EqualFold(blocks[1].Group, "Exam 05") {
		t.Fatalf("saved blocks = %+v", blocks)
	}

	restarted, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	restarted.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	switchToCategory(t, restarted, "42")
	first, ok = restarted.list.Items()[0].(drillItem)
	if !ok || !first.isGroup() || !strings.EqualFold(first.group, "Exam 06") {
		t.Fatalf("after restart first row = %+v, want the Exam 06 group", first)
	}
}

func TestDragDrillWithinSubcategoryAndPersists(t *testing.T) {
	m := newMenuTestModel(t)
	for _, d := range []drill.Drill{
		{ID: "alpha", Title: "Alpha", Category: "42/Exam 05", EntryFile: "a.c"},
		{ID: "beta", Title: "Beta", Category: "42/Exam 05", EntryFile: "b.c"},
	} {
		if err := drill.SaveDrill(d); err != nil {
			t.Fatal(err)
		}
	}
	m.reloadDrills()
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	switchToCategory(t, m, "42")

	dragRow(t, m, rowOf(t, m, "Alpha"), rowOf(t, m, "Beta"))
	items := m.list.Items()
	if items[1].(drillItem).drill.ID != "beta" || items[2].(drillItem).drill.ID != "alpha" {
		t.Fatalf("group order = %q, %q; want beta, alpha", items[1].(drillItem).drill.ID, items[2].(drillItem).drill.ID)
	}
	blocks := layoutBlocksFor(t, "42")
	if len(blocks) == 0 || len(blocks[0].Drills) != 2 || blocks[0].Drills[0] != "beta" || blocks[0].Drills[1] != "alpha" {
		t.Fatalf("saved group = %+v", blocks)
	}

	restarted, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	restarted.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	switchToCategory(t, restarted, "42")
	items = restarted.list.Items()
	if items[1].(drillItem).drill.ID != "beta" || items[2].(drillItem).drill.ID != "alpha" {
		t.Fatalf("after restart group order = %q, %q; want beta, alpha", items[1].(drillItem).drill.ID, items[2].(drillItem).drill.ID)
	}
}
