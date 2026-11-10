package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

// mouseSizes are the windows the menu hit-tests are checked at. The short ones
// are shorter than the menu, so its top rows are cut off. The narrow ones are
// too narrow for the croc.
var mouseSizes = []tea.WindowSizeMsg{
	{Width: 60, Height: 20},
	{Width: 80, Height: 24},
	{Width: 100, Height: 30},
	{Width: 120, Height: 40},
	{Width: 80, Height: 12},
	{Width: 80, Height: 8},
	{Width: 40, Height: 16},
	{Width: 30, Height: 14},
	{Width: 33, Height: 20},
}

func leftClick(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}
}

func leftMotion(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft}
}

func leftRelease(x, y int) tea.MouseMsg {
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft}
}

func wheel(up bool, x, y int) tea.MouseMsg {
	b := tea.MouseButtonWheelDown
	if up {
		b = tea.MouseButtonWheelUp
	}
	return tea.MouseMsg{X: x, Y: y, Action: tea.MouseActionPress, Button: b}
}

// stripLines is the screen as the user sees it, one string per row. A view
// taller than the window shows only its bottom rows, as the terminal does.
func stripLines(m *MainModel) []string {
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	_, h := m.screenSize()
	return lines[max(0, len(lines)-h):]
}

// cutRows is how many rows at the top of the view the window does not show. It
// counts the rendered lines, so it does not depend on menuShift.
func cutRows(m *MainModel) int {
	_, h := m.screenSize()
	return max(0, len(strings.Split(ansi.Strip(m.View()), "\n"))-h)
}

// rowOf returns the only row that contains text, and fails the test when the
// text is missing or shows up on more than one row.
func rowOf(t *testing.T, m *MainModel, text string) int {
	t.Helper()
	return rowIn(t, stripLines(m), text)
}

// listRowOf is rowOf for the title of a drill in the list. The help block is
// not searched, because the delete prompt names the drill it asks about.
func listRowOf(t *testing.T, m *MainModel, text string) int {
	t.Helper()
	lines := stripLines(m)
	return rowIn(t, lines[:len(lines)-lipgloss.Height(m.menuHelp())], text)
}

// rowIn is rowOf over the given rows.
func rowIn(t *testing.T, lines []string, text string) int {
	t.Helper()
	found := -1
	for y, line := range lines {
		if strings.Contains(line, text) {
			if found >= 0 {
				t.Fatalf("%q is on more than one row (%d and %d)", text, found, y)
			}
			found = y
		}
	}
	if found < 0 {
		t.Fatalf("%q is not on screen", text)
	}
	return found
}

// columnOf returns the column where text starts on row y.
func columnOf(t *testing.T, m *MainModel, y int, text string) int {
	t.Helper()
	line := stripLines(m)[y]
	idx := strings.Index(line, text)
	if idx < 0 {
		t.Fatalf("%q is not on row %d: %q", text, y, line)
	}
	return lipgloss.Width(line[:idx])
}

// itemTitle is the text the row shows for list item i, cut short so it stays
// findable on a narrow window.
func itemTitle(m *MainModel, i int) string {
	it := m.list.Items()[i].(drillItem)
	if it.isAdd {
		return "+ Add a drill"
	}
	return ansi.Truncate(it.drill.Title, 12, "")
}

// checkMenuRows compares menuItemAt with the rows on screen for every drill
// on the current page, and checks the rows around them. A click on the title
// row of an item that is not selected selects it.
func checkMenuRows(t *testing.T, m *MainModel) {
	t.Helper()
	p := m.list.Paginator
	items := m.list.VisibleItems()
	rows := len(stripLines(m))
	for slot := 0; slot < p.PerPage && p.Page*p.PerPage+slot < len(items); slot++ {
		idx := p.Page*p.PerPage + slot
		y := listRowOf(t, m, itemTitle(m, idx))
		if got, ok := m.menuItemAt(1, y); !ok || got != idx {
			t.Fatalf("title row %d of item %d: menuItemAt = %d, %v", y, idx, got, ok)
		}
		if y+1 < rows {
			if got, ok := m.menuItemAt(1, y+1); !ok || got != idx {
				t.Fatalf("description row %d of item %d: menuItemAt = %d, %v", y+1, idx, got, ok)
			}
		}
		if y+2 < rows {
			if got, ok := m.menuItemAt(1, y+2); ok {
				t.Fatalf("blank row %d after item %d hit item %d", y+2, idx, got)
			}
		}
		if m.pendingDelete == nil && idx != m.list.Index() {
			m.Update(leftClick(3, y))
			m.Update(leftRelease(3, y))
			if m.list.Index() != idx || m.state != StateMenu {
				t.Fatalf("click on row %d: index %d, state %v; want item %d selected", y, m.list.Index(), m.state, idx)
			}
		}
	}
	// The header, the blank row, the tab row, the status row and the row above
	// the status row are not drill rows. Rows cut off the top are skipped.
	cut := cutRows(m)
	for _, y := range []int{0, m.menuTabRow(), m.menuTabRow() + 1, m.menuItemTop() - 2, m.menuItemTop() - 1} {
		if y-cut < 0 {
			continue
		}
		if got, ok := m.menuItemAt(1, y-cut); ok {
			t.Fatalf("row %d (screen row %d) hit item %d", y, y-cut, got)
		}
	}
}

// checkMenuTabs compares menuTabAt with the category tabs on the tab row. A
// tab is its name with one column of padding on each side, and tabs are
// separated by one space. When the tab row is cut off the top of the window,
// no row on screen is a tab.
func checkMenuTabs(t *testing.T, m *MainModel) {
	t.Helper()
	w, h := m.screenSize()
	y := m.menuTabRow() - cutRows(m)
	if y < 0 {
		for row := range h {
			for _, x := range []int{0, w - 1} {
				if got, ok := m.menuTabAt(x, row); ok {
					t.Fatalf("tab row is off screen, but (%d, %d) is tab %d", x, row, got)
				}
			}
		}
		return
	}
	line := stripLines(m)[y]
	pos := 0
	for i, c := range m.categories {
		name := strings.ToUpper(c)
		idx := strings.Index(line[pos:], name)
		if idx < 0 {
			continue
		}
		start := pos + idx
		col := lipgloss.Width(line[:start]) - 1
		// The padding and the first and last letter. Each call renders the
		// menu, so the inside of a tab is sampled, not every column.
		for _, x := range []int{col, col + 1, col + len(name), col + len(name) + 1} {
			if got, ok := m.menuTabAt(x, y); !ok || got != i {
				t.Fatalf("tab %q column %d: menuTabAt = %d, %v", c, x, got, ok)
			}
		}
		if _, ok := m.menuTabAt(col+len(name)+2, y); ok {
			t.Fatalf("the space after tab %q is clickable", c)
		}
		pos = start + len(name)
	}
	// Nothing past the list's edge is a tab, so a tab cut off by the window
	// cannot be clicked.
	edge := m.list.Styles.TitleBar.GetPaddingLeft() + m.list.Width()
	if got, ok := m.menuTabAt(edge, y); ok {
		t.Fatalf("column %d past the window is tab %d", edge, got)
	}
}

// menuState is a state of the menu that changes what is drawn, and so where
// the rows and tabs fall.
type menuState struct {
	name  string
	setup func(t *testing.T, m *MainModel)
}

var menuStates = []menuState{
	{"plain", func(*testing.T, *MainModel) {}},
	{"notice", func(_ *testing.T, m *MainModel) {
		m.menuNotice = `Created "Test" in Custom.`
		m.sizeList()
	}},
	{"search highlight", func(t *testing.T, m *MainModel) {
		m.Update(runeKey("/"))
		m.Update(runeKey("e"))
		m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		if !m.search.hl {
			t.Fatal("the search did not leave a highlight")
		}
	}},
	{"delete prompt", func(t *testing.T, m *MainModel) {
		m.list.Select(1)
		m.Update(runeKey("x"))
		if m.pendingDelete == nil {
			t.Fatal("x did not open the delete prompt")
		}
	}},
	{"many categories", func(_ *testing.T, m *MainModel) {
		for i := range 12 {
			m.categories = append(m.categories, fmt.Sprintf("Category%02d", i))
		}
		m.refreshList()
	}},
}

// TestMenuHitTestMatchesRendering checks the hit-tests against the rows on
// screen in each menu state. The sizes include windows shorter than the menu,
// where the top rows are cut off, and windows narrow enough to drop the croc.
func TestMenuHitTestMatchesRendering(t *testing.T) {
	for _, st := range menuStates {
		for _, size := range mouseSizes {
			t.Run(fmt.Sprintf("%s %dx%d", st.name, size.Width, size.Height), func(t *testing.T) {
				m := newMenuTestModel(t)
				m.Update(size)
				st.setup(t, m)
				checkMenuTabs(t, m)
				checkMenuRows(t, m)

				// Select the first item of page 2, which moves the list there.
				if m.list.Paginator.TotalPages > 1 {
					m.list.Select(m.list.Paginator.PerPage)
					if m.list.Paginator.Page != 1 {
						t.Fatalf("list is on page %d, want 1", m.list.Paginator.Page)
					}
					checkMenuRows(t, m)
				}
			})
		}
	}
}

func TestMenuClickSelectsThenOpens(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	y := rowOf(t, m, itemTitle(m, 2))
	m.Update(leftClick(3, y))
	m.Update(leftRelease(3, y))
	if m.state != StateMenu || m.list.Index() != 2 {
		t.Fatalf("first click: state %v, index %d; want menu, 2", m.state, m.list.Index())
	}

	// A click on the description row of an unselected item selects it.
	m.Update(leftClick(3, rowOf(t, m, itemTitle(m, 3))+1))
	m.Update(leftRelease(3, rowOf(t, m, itemTitle(m, 3))+1))
	if m.state != StateMenu || m.list.Index() != 3 {
		t.Fatalf("description click: state %v, index %d; want menu, 3", m.state, m.list.Index())
	}

	y = rowOf(t, m, itemTitle(m, 3))
	m.Update(leftClick(3, y))
	m.Update(leftRelease(3, y))
	if m.state != StateBriefing || m.activeDrill == nil || m.activeDrill.ID != m.list.Items()[3].(drillItem).drill.ID {
		t.Fatalf("second click: state %v; want the briefing of the selected drill", m.state)
	}
}

func TestMenuClickAddDrillOpensForm(t *testing.T) {
	m := newMenuTestModel(t)
	// A window tall enough for every row, so the add row is always on screen.
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 100})
	if m.list.Paginator.TotalPages != 1 {
		t.Fatalf("list has %d pages at 100x100, want 1", m.list.Paginator.TotalPages)
	}
	last := len(m.list.Items()) - 1
	m.list.Select(0)

	y := rowOf(t, m, "+ Add a drill")
	m.Update(leftClick(3, y))
	m.Update(leftRelease(3, y))
	if m.state != StateMenu || m.list.Index() != last {
		t.Fatalf("first click: state %v, index %d; want menu, %d", m.state, m.list.Index(), last)
	}
	m.Update(leftClick(3, y))
	m.Update(leftRelease(3, y))
	if m.state != StateAddDrill || m.form == nil {
		t.Fatalf("second click: state %v; want the add drill form", m.state)
	}
}

func TestMenuClickTabSwitchesCategory(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if len(m.categories) < 2 {
		t.Fatal("need at least two categories")
	}
	y := m.menuTabRow()
	for i, c := range m.categories {
		if i == m.categoryIndex {
			continue
		}
		m.Update(leftClick(columnOf(t, m, y, strings.ToUpper(c)), y))
		if m.categoryIndex != i || m.categoryFilter != c {
			t.Fatalf("click on %q: index %d, filter %q", c, m.categoryIndex, m.categoryFilter)
		}
		if len(m.list.Items()) != len(drill.Filter(m.drills, c))+1 {
			t.Fatalf("list does not show category %q", c)
		}
	}
}

// menuSnapshot is the part of the menu a click must not change.
type menuSnapshot struct {
	state  AppState
	cat    string
	index  int
	delete bool
	notice string
	search bool
}

func snapshotMenu(m *MainModel) menuSnapshot {
	return menuSnapshot{
		state:  m.state,
		cat:    m.categoryFilter,
		index:  m.list.Index(),
		delete: m.pendingDelete != nil,
		notice: m.menuNotice,
		search: m.search.open,
	}
}

func TestMenuClickOnBlankRowsDoesNothing(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.menuNotice = "kept"
	m.sizeList()
	before := snapshotMenu(m)

	y := m.menuTabRow()
	helpRow := len(stripLines(m)) - 1
	for _, p := range [][2]int{{0, 0}, {1, 0}, {40, 0}, {1, y + 1}, {1, m.menuItemTop() - 1}, {0, helpRow}, {1, y - 1}} {
		m.Update(leftClick(p[0], p[1]))
		if got := snapshotMenu(m); got != before {
			t.Fatalf("click at %v changed the menu: %+v -> %+v", p, before, got)
		}
	}
}

func TestMenuWheelMovesSelection(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(wheel(false, 3, 10))
	if m.list.Index() != 1 {
		t.Fatalf("wheel down: index %d, want 1", m.list.Index())
	}
	m.Update(wheel(true, 3, 10))
	m.Update(wheel(true, 3, 10))
	if m.list.Index() != 0 {
		t.Fatalf("wheel up at the top: index %d, want 0", m.list.Index())
	}
}

func TestMenuWheelCrossesPages(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	per := m.list.Paginator.PerPage
	if len(m.list.VisibleItems()) <= per {
		t.Skip("the list has one page at this size")
	}
	m.list.Select(per - 1)
	m.Update(wheel(false, 3, 10))
	if m.list.Index() != per || m.list.Paginator.Page != 1 {
		t.Fatalf("wheel down over the page edge: index %d, page %d", m.list.Index(), m.list.Paginator.Page)
	}
	m.Update(wheel(true, 3, 10))
	if m.list.Index() != per-1 || m.list.Paginator.Page != 0 {
		t.Fatalf("wheel up over the page edge: index %d, page %d", m.list.Index(), m.list.Paginator.Page)
	}
}

func TestMenuDeletePromptClickCancels(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.list.Select(1)
	drills := len(m.drills)
	items := len(m.list.Items())
	m.Update(runeKey("x"))
	if m.pendingDelete == nil {
		t.Fatal("x did not open the delete prompt")
	}

	m.Update(wheel(false, 3, 10))
	if m.pendingDelete == nil {
		t.Fatal("the wheel cancelled the delete prompt")
	}

	m.Update(leftClick(3, rowOf(t, m, itemTitle(m, 0))))
	if m.pendingDelete != nil {
		t.Fatal("a click did not cancel the delete prompt")
	}
	if len(m.drills) != drills || len(m.list.Items()) != items {
		t.Fatal("a click deleted a drill")
	}
}

func TestMenuSearchPromptIgnoresMouse(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(runeKey("/"))
	if !m.search.open {
		t.Fatal("/ did not open the search prompt")
	}
	before := snapshotMenu(m)
	m.Update(leftClick(3, rowOf(t, m, itemTitle(m, 2))))
	m.Update(leftClick(columnOf(t, m, m.menuTabRow(), "42"), m.menuTabRow()))
	m.Update(wheel(false, 3, 10))
	if got := snapshotMenu(m); got != before {
		t.Fatalf("mouse changed the search prompt: %+v -> %+v", before, got)
	}
}

func TestMouseMotionAndReleaseChangeNothing(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	before := snapshotMenu(m)
	y := rowOf(t, m, itemTitle(m, 2))
	for _, action := range []tea.MouseAction{tea.MouseActionMotion, tea.MouseActionRelease} {
		m.Update(tea.MouseMsg{X: 3, Y: y, Action: action, Button: tea.MouseButtonLeft})
		m.Update(tea.MouseMsg{X: 3, Y: y, Action: action, Button: tea.MouseButtonWheelDown})
		if got := snapshotMenu(m); got != before {
			t.Fatalf("%v changed the menu: %+v -> %+v", action, before, got)
		}
	}

	m.settingsCursor = 0
	m.state = StateSettings
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	cursor := m.settingsCursor
	for y := range 24 {
		m.Update(tea.MouseMsg{X: 4, Y: y, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft})
		m.Update(tea.MouseMsg{X: 4, Y: y, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft})
	}
	if m.settingsCursor != cursor || m.state != StateSettings {
		t.Fatal("motion or release on settings changed the screen")
	}
	restoreDefaultTheme(t)
}

func TestMouseIgnoredOnOtherScreens(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.state = StateAddDrill
	m.initForm()
	form := m.form

	m.Update(leftClick(3, 5))
	m.Update(wheel(true, 3, 5))
	if m.state != StateAddDrill || m.form != form || form.State != huh.StateNormal {
		t.Fatalf("mouse changed the add drill form: state %v, form state %v", m.state, form.State)
	}

	for _, state := range []AppState{StateCountdown, StateSprinting, StateSetupError} {
		m.setupErr = errors.New("workspace unavailable")
		m.state = state
		m.Update(leftClick(3, 5))
		m.Update(wheel(false, 3, 5))
		if m.state != state {
			t.Fatalf("mouse left state %v for %v", state, m.state)
		}
	}
}

func TestMouseOffIgnoresEverything(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.mouseOff = true
	before := snapshotMenu(m)
	m.Update(leftClick(3, rowOf(t, m, itemTitle(m, 2))))
	m.Update(wheel(false, 3, 10))
	if got := snapshotMenu(m); got != before {
		t.Fatalf("mouse changed the menu while off: %+v -> %+v", before, got)
	}
	if m.Init() != nil {
		t.Fatal("Init asks for mouse input while it is off")
	}
}

func TestScrollableScreenWheelScrolls(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m.state = StateEvaluating
	m.evalOutput = strings.Repeat("compiler diagnostic\n", 80) + "FINAL DIAGNOSTIC"
	lines, _, available := m.screenLines()
	limit := max(0, len(lines)-available)
	if limit < 2*wheelStep {
		t.Fatalf("content does not overflow enough to test: limit %d", limit)
	}

	m.Update(wheel(false, 3, 5))
	if m.scrollOffset != wheelStep {
		t.Fatalf("wheel down: offset %d, want %d", m.scrollOffset, wheelStep)
	}
	m.Update(wheel(false, 3, 5))
	if m.scrollOffset != 2*wheelStep {
		t.Fatalf("second wheel down: offset %d, want %d", m.scrollOffset, 2*wheelStep)
	}
	m.Update(wheel(true, 3, 5))
	if m.scrollOffset != wheelStep {
		t.Fatalf("wheel up: offset %d, want %d", m.scrollOffset, wheelStep)
	}
	for range limit {
		m.Update(wheel(false, 3, 5))
	}
	if m.scrollOffset != limit {
		t.Fatalf("wheel past the end: offset %d, want the limit %d", m.scrollOffset, limit)
	}
	for range limit + 5 {
		m.Update(wheel(true, 3, 5))
	}
	if m.scrollOffset != 0 {
		t.Fatalf("wheel past the top: offset %d, want 0", m.scrollOffset)
	}
}

func TestScrollableScreenWheelWithoutOverflow(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.state = StateResults
	m.Update(wheel(false, 3, 5))
	m.Update(wheel(false, 3, 5))
	if m.scrollOffset != 0 {
		t.Fatalf("offset %d on a screen that fits, want 0", m.scrollOffset)
	}
}

// mouseEnableCount counts the enable-mouse messages a command produces. It runs
// the command and looks through batches. Bubble Tea does not export the message
// type, so the match is on its type name.
func mouseEnableCount(cmd tea.Cmd) int {
	if cmd == nil {
		return 0
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		n := 0
		for _, c := range batch {
			n += mouseEnableCount(c)
		}
		return n
	}
	if fmt.Sprintf("%T", msg) == "tea.enableMouseCellMotionMsg" {
		return 1
	}
	return 0
}

// cmdMsgs runs cmd and returns every message it produces, looking through
// batches.
func cmdMsgs(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, cmdMsgs(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func TestInitEnablesMouse(t *testing.T) {
	m := newMenuTestModel(t)
	if n := mouseEnableCount(m.Init()); n != 1 {
		t.Fatalf("Init enables mouse %d times, want 1", n)
	}
}

// TestSprintFinishedEnablesMouse covers the paths out of the sprint. The test
// run after a finished sprint is checked on the result that comes back.
func TestSprintFinishedEnablesMouse(t *testing.T) {
	restoreDefaultTheme(t)
	t.Run("launch failure", func(t *testing.T) {
		m := newEvaluationTestModel(t)
		m.sprintStartTime = time.Now()
		_, cmd := m.Update(sprintFinishedMsg{err: errors.New("no editor")})
		if m.state != StateEvaluating || !m.launchFailed {
			t.Fatal("launch failure did not reach the failed screen")
		}
		if n := mouseEnableCount(cmd); n != 1 {
			t.Fatalf("launch failure enables mouse %d times, want 1", n)
		}
	})
	t.Run("finished", func(t *testing.T) {
		m := newEvaluationTestModel(t)
		_, evaluate := m.Update(sprintFinishedMsg{})
		// The test run is not wrapped, so it does not enable mouse itself.
		if _, ok := evaluate().(evalFinishedMsg); !ok {
			t.Fatal("the test run is not the command returned after the sprint")
		}
		_, cmd := m.Update(evalFinishedMsg{id: m.evalID, session: m.session, passed: true})
		if n := mouseEnableCount(cmd); n != 1 {
			t.Fatalf("finished test run enables mouse %d times, want 1", n)
		}
	})
	t.Run("off", func(t *testing.T) {
		m := newEvaluationTestModel(t)
		m.mouseOff = true
		_, cmd := m.Update(sprintFinishedMsg{err: errors.New("no editor")})
		if n := mouseEnableCount(cmd); n != 0 {
			t.Fatalf("mouse off still enables mouse %d times", n)
		}
	})
}

// settingsThemeRows maps each settings row on screen to its index and column,
// from the rendered text. The Themes header maps to -1, the shell row to
// len(Themes) and the mouse row to len(Themes)+1.
func settingsThemeRows(m *MainModel) map[int][2]int {
	rows := map[int][2]int{}
	for y, line := range stripLines(m) {
		if idx := strings.Index(line, "Themes"); idx >= 0 {
			rows[y] = [2]int{-1, lipgloss.Width(line[:idx])}
		}
		for i, th := range Themes {
			if idx := strings.Index(line, th.Name); idx >= 0 {
				rows[y] = [2]int{i, lipgloss.Width(line[:idx])}
			}
		}
		if idx := strings.Index(line, "Shell  "); idx >= 0 {
			rows[y] = [2]int{len(Themes), lipgloss.Width(line[:idx])}
		}
		if idx := strings.Index(line, "Mouse  "); idx >= 0 {
			rows[y] = [2]int{len(Themes) + 1, lipgloss.Width(line[:idx])}
		}
		if idx := strings.Index(line, "Factory Reset"); idx >= 0 {
			rows[y] = [2]int{len(Themes) + 2, lipgloss.Width(line[:idx])}
		}
	}
	return rows
}

func TestSettingsHitTestMatchesRendering(t *testing.T) {
	restoreDefaultTheme(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 60, Height: 20}, {Width: 80, Height: 24}, {Width: 40, Height: 10}} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m := newMenuTestModel(t)
			m.Update(size)
			m.Update(runeKey("o"))
			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if m.state != StateSettings {
				t.Fatal("o did not open settings")
			}
			rows := settingsThemeRows(m)
			if len(rows) == 0 {
				t.Fatal("no theme rows on screen")
			}
			for y := range size.Height {
				row, drawn := rows[y]
				if !drawn {
					for x := range size.Width {
						if i, ok := m.settingsRowAt(x, y); ok {
							t.Fatalf("row %d has no theme but column %d hits theme %d", y, x, i)
						}
					}
					continue
				}
				if got, ok := m.settingsRowAt(row[1], y); !ok || got != row[0] {
					t.Fatalf("row %d drawn with theme %d: settingsRowAt = %d, %v", y, row[0], got, ok)
				}
				// The preview side of a wide window is not part of the list.
				if got, ok := m.settingsRowAt(size.Width-1, y); ok {
					t.Fatalf("row %d: the right edge hits theme %d", y, got)
				}
			}
		})
	}
}

func TestSettingsClickPreviewsThenSaves(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	rows := settingsThemeRows(m)

	target := 2
	if target == m.settingsCursor {
		target = 3
	}
	y := -1
	for row, r := range rows {
		if r[0] == target {
			y = row
		}
	}
	if y < 0 {
		t.Fatalf("theme %d is not on screen", target)
	}
	x := rows[y][1]

	m.Update(leftClick(x, y))
	if m.settingsCursor != target || CurrentTheme().ID != Themes[target].ID {
		t.Fatalf("first click: cursor %d, theme %s; want the preview of %s", m.settingsCursor, CurrentTheme().ID, Themes[target].ID)
	}
	if m.state != StateSettings {
		t.Fatal("first click left settings")
	}
	if cfg, _ := storage.LoadConfig(); cfg.Theme == Themes[target].ID {
		t.Fatal("first click saved the theme")
	}

	m.Update(leftClick(x, y))
	if m.state != StateMenu {
		t.Fatalf("second click: state %v, want the menu", m.state)
	}
	cfg, err := storage.LoadConfig()
	if err != nil || cfg.Theme != Themes[target].ID {
		t.Fatalf("second click: saved theme %q, err %v; want %q", cfg.Theme, err, Themes[target].ID)
	}
	if want := fmt.Sprintf("Theme set to %s.", Themes[target].Name); m.menuNotice != want {
		t.Fatalf("notice %q, want %q", m.menuNotice, want)
	}
}

func TestSettingsWheelMovesCursor(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.moveSettings(0)

	m.Update(wheel(false, 3, 5))
	if m.settingsCursor != 1 || CurrentTheme().ID != Themes[1].ID {
		t.Fatalf("wheel down: cursor %d, theme %s", m.settingsCursor, CurrentTheme().ID)
	}
	m.Update(wheel(true, 3, 5))
	m.Update(wheel(true, 3, 5))
	if !m.settingsOnThemesHeader() || CurrentTheme().ID != Themes[0].ID {
		t.Fatalf("wheel up: cursor %d, theme %s; want the Themes header", m.settingsCursor, CurrentTheme().ID)
	}
	if m.state != StateSettings {
		t.Fatal("wheel left settings")
	}
}

func TestSettingsClickOutsideListDoesNothing(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cursor := m.settingsCursor
	rows := settingsThemeRows(m)
	// A click on a blank row or the preview side of a wide window does nothing.
	// The rows span the list's width, so a click on one is covered elsewhere.
	for y := range 24 {
		if _, isRow := rows[y]; isRow {
			continue
		}
		m.Update(leftClick(0, y))
		m.Update(leftClick(79, y))
	}
	if m.settingsCursor != cursor || m.state != StateSettings {
		t.Fatal("a click outside the list changed settings")
	}
}

// TestMenuResizedAfterFormCancel checks that the list is sized for the notice
// the menu shows after a cancelled form. The menu view must be as tall as it is
// after the next key press, which clears the notice and resizes the list.
func TestMenuResizedAfterFormCancel(t *testing.T) {
	cancels := map[string]func(t *testing.T, m *MainModel){
		"esc": func(_ *testing.T, m *MainModel) {
			m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		},
		"confirm no": func(t *testing.T, m *MainModel) {
			press(t, m, runeKey("Abandoned"))
			for range 8 {
				press(t, m, tea.KeyMsg{Type: tea.KeyTab})
			}
			press(t, m, runeKey("n"))
		},
	}
	for name, cancel := range cancels {
		t.Run(name, func(t *testing.T) {
			m := newMenuTestModel(t)
			m.Update(tea.WindowSizeMsg{Width: 80, Height: 20})
			m.list.Select(len(m.list.Items()) - 1)
			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if m.state != StateAddDrill {
				t.Fatalf("enter on add item: state %v, want the add drill form", m.state)
			}
			cancel(t, m)
			if m.state != StateMenu || m.menuNotice != "Cancelled." {
				t.Fatalf("state %v, notice %q; want the menu with a cancelled notice", m.state, m.menuNotice)
			}
			withNotice := lipgloss.Height(m.View())

			m.Update(runeKey("j"))
			if m.menuNotice != "" {
				t.Fatalf("a key press left the notice %q", m.menuNotice)
			}
			if afterKey := lipgloss.Height(m.View()); withNotice != afterKey {
				t.Fatalf("menu is %d rows after the cancel and %d after a key press", withNotice, afterKey)
			}
		})
	}
}
