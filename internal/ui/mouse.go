package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const wheelStep = 3

// mouseActive reports whether mouse input is on. The setting lives in
// mouseOff, which NewMainModel reads from config.json and the settings screen
// changes.
func (m *MainModel) mouseActive() bool {
	return !m.mouseOff
}

// mouseCmd asks the terminal to report mouse clicks and wheel events. It is
// nil when mouse input is off.
func (m *MainModel) mouseCmd() tea.Cmd {
	if !m.mouseActive() {
		return nil
	}
	return tea.EnableMouseCellMotion
}

// updateMouse sends a mouse event to the current screen. Menu presses can
// become drags: motion and release are routed to the pending press until it
// ends. Elsewhere only presses count. Screens without mouse support ignore the
// event, which also keeps it away from huh. A click on a footer item shows its
// pressed look and acts when the timer ends; until then other mouse presses
// wait, and a key press cancels the click.
func (m *MainModel) updateMouse(msg tea.MouseMsg) tea.Cmd {
	if m.helpPress != nil {
		// A clicked footer item is waiting for its action; no other mouse
		// event stacks onto it.
		return nil
	}
	if !m.mouseActive() {
		// With mouse off the terminal sends no clicks. The one exception is a
		// press on the mouse row in settings, the only way back on, which
		// toggles it as it does when mouse is on.
		if msg.Action != tea.MouseActionPress || m.state != StateSettings || msg.Button != tea.MouseButtonLeft {
			return nil
		}
		if i, ok := m.settingsRowAt(msg.X, msg.Y); !ok || i != len(Themes)+1 {
			return nil
		}
		return m.updateSettingsMouse(msg)
	}
	if m.state == StateMenu && msg.Action != tea.MouseActionPress {
		return m.updateMenuDrag(msg)
	}
	if msg.Action != tea.MouseActionPress {
		return nil
	}
	// A new press starts over: a lost release must not leave the menu stuck
	// waiting for one.
	m.cancelPress()
	if msg.Button == tea.MouseButtonLeft {
		if key, item, ok := m.helpItemAt(msg.X, msg.Y); ok {
			m.helpPress = &helpPressState{item: item, key: key}
			return footerPressTimer()
		}
	}
	switch m.state {
	case StateMenu:
		return m.updateMenuMouse(msg)
	case StateSettings:
		return m.updateSettingsMouse(msg)
	case StateBriefing:
		return m.updateBriefingMouse(msg)
	}
	if m.scrollable() {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.scrollBy(-wheelStep)
		case tea.MouseButtonWheelDown:
			m.scrollBy(wheelStep)
		}
	}
	return nil
}

// updateMenuMouse handles mouse input in the menu. The search prompt takes no
// mouse input. While the delete prompt is open, a click cancels it like any
// other key, and the wheel does nothing.
func (m *MainModel) updateMenuMouse(msg tea.MouseMsg) tea.Cmd {
	if m.search.open {
		return nil
	}
	if m.pendingDelete != nil {
		if msg.Button == tea.MouseButtonLeft {
			m.pendingDelete = nil
			m.menuNotice = ""
			m.sizeList()
		}
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.menuScroll(tea.KeyUp)
	case tea.MouseButtonWheelDown:
		return m.menuScroll(tea.KeyDown)
	case tea.MouseButtonLeft:
		return m.menuClick(msg.X, msg.Y)
	}
	return nil
}

func (m *MainModel) menuScroll(k tea.KeyType) tea.Cmd {
	m.dismissNotice()
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(tea.KeyMsg{Type: k})
	return cmd
}

// menuClick handles a left press at (x, y) in the menu. A category tab switches
// on the press. A row press selects on the first click and remembers the press;
// the release opens it on the second click, unless the press became a drag.
func (m *MainModel) menuClick(x, y int) tea.Cmd {
	if i, ok := m.menuTabAt(x, y); ok {
		if !strings.EqualFold(m.categoryFilter, m.categories[i]) {
			m.dismissNotice()
			m.setCategory(i)
		}
		return nil
	}
	return m.beginMenuPress(x, y)
}

// dismissNotice clears the menu notice, as a key press does. The help line
// under the list changes height with it, so the list is resized.
func (m *MainModel) dismissNotice() {
	if m.menuNotice != "" {
		m.menuNotice = ""
		m.sizeList()
	}
}

// updateSettingsMouse handles the wheel and clicks on the settings screen. The
// wheel moves the cursor a row, and the preview follows it. A click on the
// mouse row toggles mouse support, as Enter does there. A click on the shell
// row selects it, and a second click cycles it. A click on another theme row
// moves the cursor there. A click on the cursor theme row saves the theme, as
// Enter does.
func (m *MainModel) updateSettingsMouse(msg tea.MouseMsg) tea.Cmd {
	if m.reset != nil && m.reset.running {
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.stepSettings(-1)
	case tea.MouseButtonWheelDown:
		m.stepSettings(1)
	case tea.MouseButtonLeft:
		i, ok := m.settingsRowAt(msg.X, msg.Y)
		switch {
		case !ok:
		case i == -1:
			if i == m.settingsCursor {
				m.toggleThemes()
			} else {
				m.moveSettings(i)
			}
		case i == len(Themes)+2:
			m.moveSettings(i)
		case i == len(Themes)+1:
			m.moveSettings(i)
			return m.toggleMouse()
		case i == len(Themes):
			if i == m.settingsCursor {
				return m.cycleShell()
			}
			m.moveSettings(i)
		case i == m.settingsCursor:
			m.saveSettings()
		default:
			m.moveSettings(i)
		}
	}
	return nil
}

// menuTabRow is the row of the category tabs. The menu header comes first, then
// a blank row, then the list, and the list's title bar is its first row.
func (m *MainModel) menuTabRow() int {
	return lipgloss.Height(m.menuHeader()) + 1
}

// menuItemTop is the row of the first visible drill row. The title bar and the
// status bar of the list come before the rows, and each one is two rows: the
// text and the blank row under it.
func (m *MainModel) menuItemTop() int {
	return m.menuTabRow() +
		lipgloss.Height(m.list.Styles.TitleBar.Render("")) +
		lipgloss.Height(m.list.Styles.StatusBar.Render(""))
}

// menuShift is how many rows of the menu render are cut off the top of the
// window. When the menu is taller than the window, Bubble Tea's renderer keeps
// the bottom rows, so a row on screen is shift rows above its row in the
// render. contentView is the menu View returns, so the count cannot drift from
// what is drawn.
func (m *MainModel) menuShift() int {
	if m.height <= 0 {
		return 0
	}
	return max(0, lipgloss.Height(m.contentView())-m.height)
}

// menuItemAt returns the list index of the drill row at (x, y) in the menu. The
// title and description rows of a drill count. The blank row between two rows
// and everything else do not.
func (m *MainModel) menuItemAt(x, y int) (int, bool) {
	d := newDrillDelegate("")
	step := d.Height() + d.Spacing()
	r := y + m.menuShift() - m.menuItemTop()
	if x < 0 || x >= m.list.Width() || r < 0 {
		return 0, false
	}
	slot := r / step
	if r%step >= d.Height() {
		return 0, false
	}
	p := m.list.Paginator
	if slot >= p.ItemsOnPage(len(m.list.VisibleItems())) {
		return 0, false
	}
	return p.Page*p.PerPage + slot, true
}

// menuTabAt returns the index in m.categories of the tab at (x, y). It covers
// the tabs as the title bar draws them, so a tab cut off by the window is not
// clickable. A filter tab added by "codedrill random" has no index in
// m.categories and is not clickable.
func (m *MainModel) menuTabAt(x, y int) (int, bool) {
	if y+m.menuShift() != m.menuTabRow() {
		return 0, false
	}
	left := m.list.Styles.TitleBar.GetPaddingLeft()
	end := left + m.list.Width()
	col := left
	for i, tab := range m.categoryTabs() {
		w := lipgloss.Width(tab)
		if x >= col && x < col+w && x < end {
			return i, i < len(m.categories)
		}
		// Tabs are joined with one space.
		col += w + 1
	}
	return 0, false
}

// settingsRowAt returns the row index drawn at (x, y) on the settings screen.
// The Themes header gives -1. A theme row gives its index in Themes. The shell
// row gives len(Themes), the mouse row len(Themes)+1, and reset len(Themes)+2.
// The blank rows, the
// preview and the error and help rows do not count.
func (m *MainModel) settingsRowAt(x, y int) (int, bool) {
	w, h := m.screenSize()
	if y < 0 || y >= h {
		return 0, false
	}
	fit := m.settingsFit(w, h)
	top := lipgloss.Height(m.screenHeader("settings"))
	if fit.parts.top {
		top++
	}
	if fit.parts.label {
		top++
	}
	list := m.settingsList(fit, w)
	if x < 0 || x >= lipgloss.Width(list) {
		return 0, false
	}
	if fit.parts.label && y == top-1 {
		return -1, true
	}
	shown := 0
	if m.settingsThemesOpen() {
		start, rows := settingsWindow(m.settingsCursor, fit.rows, len(Themes))
		shown = rows
		if k := y - top; k >= 0 && k < shown {
			return start + k, true
		}
	}
	row := top + shown
	if fit.shell {
		if y == row {
			return len(Themes), true
		}
		row++
	}
	if fit.mouse {
		if y == row {
			return len(Themes) + 1, true
		}
		row++
	}
	if fit.reset && y == row {
		return len(Themes) + 2, true
	}
	return 0, false
}

// updateBriefingMouse handles mouse events on the briefing screen. Clicking a
// checkbox toggles it and updates keyboard focus. Scrolling moves the view.
// Accidental sprint starts are impossible from checkbox clicks.
func (m *MainModel) updateBriefingMouse(msg tea.MouseMsg) tea.Cmd {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.scrollBy(-wheelStep)
	case tea.MouseButtonWheelDown:
		m.scrollBy(wheelStep)
	case tea.MouseButtonLeft:
		if i, ok := m.briefingCheckboxAt(msg.X, msg.Y); ok {
			m.modeFocus = i
			m.toggleBriefingOption()
		}
	}
	return nil
}

var briefingOptionLabels = [briefingOptionCount]string{
	"Reference check",
	"Show reference",
	"Require exact copy",
	"Require tests",
}

// briefingCheckboxAt returns the checkbox index (0..3) under cell (x, y) on the
// briefing screen, and whether cell (x, y) hit a checkbox. The four checkboxes
// are arranged two per row (options 0 and 1 on row 0, options 2 and 3 on row 1)
// when the terminal is wide enough; otherwise they fall back to a vertical
// layout. x must land on the checkbox box itself ([x] or [ ]), not on the label
// or the gap between columns.
func (m *MainModel) briefingCheckboxAt(x, y int) (int, bool) {
	if m.state != StateBriefing || m.activeDrill == nil {
		return 0, false
	}
	w, h := m.screenSize()
	if x < 0 || x >= w || y < 0 || y >= h {
		return 0, false
	}
	lines, _, available := m.screenLines()
	offset := min(m.scrollOffset, max(0, len(lines)-available))
	end := min(len(lines), offset+available)
	if y >= end-offset {
		return 0, false
	}
	lineIdx := offset + y
	if lineIdx < 0 || lineIdx >= len(lines) {
		return 0, false
	}
	stripped := ansi.Strip(lines[lineIdx])

	// Find a checkbox box ([x] or [ ]) immediately followed by a known label.
	// x must land on the box itself, so clicks in the gap between columns or on
	// a label do not toggle anything.
	for i, label := range briefingOptionLabels {
		for _, box := range []string{"[x]", "[ ]"} {
			pattern := box + " " + label
			for pos := 0; pos+len(pattern) <= len(stripped); pos++ {
				if stripped[pos:pos+len(pattern)] != pattern {
					continue
				}
				if x >= pos && x < pos+len(box) {
					return i, true
				}
			}
		}
	}
	return 0, false
}
