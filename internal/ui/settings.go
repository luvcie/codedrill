package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/storage"
)

// settingsSideWidth is the window width from which the preview sits to the
// right of the theme list. Narrower windows stack it under the list.
const settingsSideWidth = 70

const settingsPreviewHeight = 8

// openSettings opens the settings screen from the menu. The Themes section
// starts folded, with the cursor on its header. The active theme is kept, so
// Esc can go back to it.
func (m *MainModel) openSettings() {
	m.settingsOrigin = CurrentTheme()
	m.settingsThemesClosed = true
	m.settingsCursor = -1
	// A config that cannot be read has no saved theme to mark. Enter will
	// report the read error when it tries to save.
	m.settingsSaved = ""
	if cfg, err := storage.LoadConfig(); err == nil {
		m.settingsSaved = ThemeOrDefault(cfg.Theme).ID
		m.shell = cfg.ShellName()
	}
	m.settingsErr = nil
	m.reset = nil
	m.state = StateSettings
}

// updateSettings handles keys on the settings screen. The cursor moves through
// the Themes header, theme rows, shell, mouse, and factory reset. Moving onto a
// theme previews it. Enter selects a setting or advances reset confirmation.
// Esc cancels a reset confirmation or closes settings.
func (m *MainModel) updateSettings(key tea.KeyMsg) tea.Cmd {
	if m.reset != nil && m.reset.running {
		return nil
	}
	last := len(Themes) + 2
	switch key.String() {
	case "up", "k":
		m.stepSettings(-1)
	case "down", "j":
		m.stepSettings(1)
	case "g", "home":
		m.moveSettings(m.settingsHome())
	case "G", "end":
		m.moveSettings(last)
	case "enter", " ":
		switch {
		case m.settingsOnReset():
			if key.String() == "enter" {
				return m.advanceFactoryReset()
			}
		case m.settingsOnThemesHeader():
			m.toggleThemes()
		case m.settingsOnMouse():
			return m.toggleMouse()
		case m.settingsOnShell():
			return m.cycleShell()
		case key.String() == "enter":
			m.saveSettings()
		}
	case "esc", "q":
		if m.reset != nil {
			m.reset = nil
		} else {
			m.closeSettings()
		}
	}
	return nil
}

func (m *MainModel) settingsOnShell() bool {
	return m.settingsCursor == len(Themes)
}

func (m *MainModel) settingsOnThemesHeader() bool {
	return m.settingsCursor < 0
}

func (m *MainModel) settingsThemesOpen() bool {
	return !m.settingsThemesClosed
}

// settingsHome is the first row Home jumps to: the Themes header when the
// section is collapsed, otherwise the first theme.
func (m *MainModel) settingsHome() int {
	if !m.settingsThemesOpen() {
		return -1
	}
	return 0
}

func (m *MainModel) settingsOrder() []int {
	order := []int{-1}
	if m.settingsThemesOpen() {
		for i := range Themes {
			order = append(order, i)
		}
	}
	return append(order, len(Themes), len(Themes)+1, len(Themes)+2)
}

func (m *MainModel) stepSettings(delta int) {
	order := m.settingsOrder()
	pos := 0
	for i, row := range order {
		if row == m.settingsCursor {
			pos = i
			break
		}
	}
	pos = min(max(pos+delta, 0), len(order)-1)
	m.moveSettings(order[pos])
}

func (m *MainModel) toggleThemes() {
	m.settingsThemesClosed = !m.settingsThemesClosed
	m.settingsCursor = -1
}

func (m *MainModel) settingsOnMouse() bool {
	return m.settingsCursor == len(Themes)+1
}

func (m *MainModel) settingsOnReset() bool {
	return m.settingsCursor == len(Themes)+2
}

// moveSettings puts the cursor on row i, clamped to the Themes header and the
// reset row. A hidden theme row maps to the header while the section is
// collapsed. A theme row applies its theme. Other rows change nothing, so the
// theme under the cursor before stays previewed. Nothing is saved.
func (m *MainModel) moveSettings(i int) {
	i = min(max(i, -1), len(Themes)+2)
	if !m.settingsThemesOpen() && i >= 0 && i < len(Themes) {
		i = -1
	}
	if i == m.settingsCursor && (i < 0 || i >= len(Themes) || CurrentTheme().ID == Themes[i].ID) {
		return
	}
	m.settingsCursor = i
	m.reset = nil
	// The error belongs to the last save attempt. Moving the cursor makes it
	// stale, so it goes.
	m.settingsErr = nil
	if i >= 0 && i < len(Themes) {
		m.applyTheme(Themes[i])
	}
}

// toggleMouse switches mouse support to the other state, saves it to
// config.json, and returns the command that changes the terminal. Nothing
// changes when the save fails: the error is shown and mouse keeps its state.
func (m *MainModel) toggleMouse() tea.Cmd {
	on := m.mouseOff
	cfg, err := storage.LoadConfig()
	if err == nil {
		cfg.Mouse = &on
		err = storage.SaveConfig(cfg)
	}
	if err != nil {
		m.settingsErr = err
		return nil
	}
	m.settingsErr = nil
	m.mouseOff = !on
	if on {
		return tea.EnableMouseCellMotion
	}
	return tea.DisableMouse
}

// cycleShell steps the Zellij pane shell between bash and zsh and saves it to
// config.json. A custom shell set with `codedrill config shell` stays active
// until Enter cycles it to zsh; the terminal pane falls back to bash with a
// message when the configured shell is not on PATH.
func (m *MainModel) cycleShell() tea.Cmd {
	next := "zsh"
	if strings.EqualFold(m.shell, "zsh") {
		next = "bash"
	}
	cfg, err := storage.LoadConfig()
	if err == nil {
		cfg.Shell = next
		err = storage.SaveConfig(cfg)
	}
	if err != nil {
		m.settingsErr = err
		return nil
	}
	m.settingsErr = nil
	m.shell = next
	return nil
}

// saveSettings writes the highlighted theme to config.json and returns to the
// menu. The file is read first and only the theme is changed, so the other
// fields survive. If the file cannot be read or written, the screen stays
// open with the preview still applied.
func (m *MainModel) saveSettings() {
	t := Themes[m.settingsCursor]
	cfg, err := storage.LoadConfig()
	if err == nil {
		cfg.Theme = t.ID
		err = storage.SaveConfig(cfg)
	}
	if err != nil {
		m.settingsErr = err
		return
	}
	m.settingsErr = nil
	m.settingsOrigin = t
	m.settingsSaved = t.ID
	m.state = StateMenu
	m.menuNotice = fmt.Sprintf("Theme set to %s.", t.Name)
	m.sizeList()
}

// closeSettings goes back to the menu with the theme that was active when the
// screen opened. Nothing is saved.
func (m *MainModel) closeSettings() {
	previewed := CurrentTheme().ID != m.settingsOrigin.ID
	if previewed {
		m.applyTheme(m.settingsOrigin)
	}
	m.settingsErr = nil
	m.state = StateMenu
	// A mouse toggle is saved as it happens, so only a theme preview is
	// cancelled. Without one there is nothing to say.
	if previewed {
		m.menuNotice = "Cancelled."
	}
	m.sizeList()
}

// settingsParts says which optional rows of the settings screen are drawn. The
// cursor row is drawn whenever the window has a row for it.
type settingsParts struct {
	preview bool // the sample of the theme roles
	top     bool // the blank row between the header and the body
	label   bool // the Themes header
	below   bool // the blank rows above the error and the help
	err     bool // the save error
	help    bool // the key help
}

// settingsRows returns how many of themeRows fit in a window h rows tall when
// the parts in p are drawn, and whether the shell and mouse rows fit with them.
// hh, hp and eh are the heights of the header, the help and the save error.
// onHeader, onShell, onMouse and onReset say the cursor is on that row, which needs
// no theme row. ok is false when the cursor row does not fit.
//
// The shell and mouse rows are kept before the theme rows: they fit as long as
// one theme row fits with them, unless the cursor is on another row. When the
// window cannot hold both, the shell row goes first, so the mouse row stays
// whenever it fits.
func settingsRows(p settingsParts, h, hh, hp, eh int, side bool, themeRows int, onHeader, onShell, onMouse, onReset bool) (rows int, shell, mouse, reset, ok bool) {
	rest := hh
	if p.top {
		rest++
	}
	if p.preview && !side {
		rest += 1 + settingsPreviewHeight
	}
	if p.err {
		rest += eh
		if p.below {
			rest++
		}
	}
	if p.help {
		rest += hp
		if p.below {
			rest++
		}
	}
	region := h - rest
	// Side by side, the preview needs the height of the list and its labels.
	if p.preview && side && region < settingsPreviewHeight {
		return 0, false, false, false, false
	}
	themeCost := 0
	if p.label {
		themeCost = 1
	}
	shell, mouse, reset = true, true, true
	for {
		extras := 0
		if shell {
			extras++
		}
		if mouse {
			extras++
		}
		if reset {
			extras++
		}
		needTheme := 0
		if themeRows > 0 && !onHeader && !onShell && !onMouse && !onReset {
			needTheme = 1
		}
		if region >= themeCost+extras+needTheme {
			return min(max(0, region-themeCost-extras), themeRows), shell, mouse, reset, true
		}
		switch {
		case shell && !onShell:
			shell = false
		case mouse && !onMouse:
			mouse = false
		case reset && !onReset:
			reset = false
		default:
			return 0, shell, mouse, reset, false
		}
	}
}

// settingsFit is how the settings screen is laid out in one window: which
// parts are drawn, whether the preview sits beside the list, how many theme
// rows the list gets, and whether the mouse row is drawn. viewSettings draws
// from it and settingsRowAt reads it, so the two cannot disagree about where a
// row is.
type settingsFit struct {
	parts settingsParts
	side  bool // the preview sits beside the list
	rows  int  // theme rows in the list; zero when the section is collapsed
	shell bool // the shell row is drawn
	mouse bool // the mouse row is drawn
	reset bool // the reset row is drawn
}

func (m *MainModel) settingsError(w int) string {
	if m.settingsErr == nil {
		return ""
	}
	msg := "Could not save: " + m.settingsErr.Error()
	return DangerStyle.Render(indentLines(ansi.Wrap(msg, max(1, w-2), "")))
}

// settingsFit works out the layout of the settings screen in a w x h window.
// The parts that give way, and the rows left for the list, are decided here.
func (m *MainModel) settingsFit(w, h int) settingsFit {
	header := m.screenHeader("settings")
	help := m.renderHelpNow(m.screenHelp(), w)
	errText := m.settingsError(w)

	side := w >= settingsSideWidth
	onHeader := m.settingsOnThemesHeader()
	onShell, onMouse, onReset := m.settingsOnShell(), m.settingsOnMouse(), m.settingsOnReset()
	themeRows := 0
	if m.settingsThemesOpen() {
		themeRows = len(Themes)
	}
	// Under the list the preview needs its width and the column-2 indent.
	stacked := indentLines(settingsPreview())
	// The colour sample belongs to the Themes section, so it stays hidden
	// while that section is folded.
	p := settingsParts{
		preview: m.settingsThemesOpen() && (side || lipgloss.Width(stacked) <= w),
		top:     true,
		label:   true,
		below:   true,
		err:     errText != "",
		help:    help != "",
	}
	hh, hp, eh := lipgloss.Height(header), lipgloss.Height(help), 0
	if errText != "" {
		eh = lipgloss.Height(errText)
	}
	rows, shell, mouse, reset, ok := settingsRows(p, h, hh, hp, eh, side, themeRows, onHeader, onShell, onMouse, onReset)
	// The whole list, with the shell and mouse rows, comes before the preview
	// and the blank rows. Each part goes only while the list still does not
	// fit. Keep the Themes header while the cursor is on it.
	drops := []*bool{&p.preview, &p.top}
	if !onHeader {
		drops = append(drops, &p.label)
	}
	drops = append(drops, &p.below)
	for _, drop := range drops {
		if ok && shell && mouse && reset && rows >= themeRows {
			break
		}
		if *drop {
			*drop = false
			rows, shell, mouse, reset, ok = settingsRows(p, h, hh, hp, eh, side, themeRows, onHeader, onShell, onMouse, onReset)
		}
	}
	// The list is cut to a window only after those parts are gone. The error
	// and the help give way only when not even the cursor row fits.
	for _, drop := range []*bool{&p.err, &p.help} {
		if ok {
			break
		}
		if *drop {
			*drop = false
			rows, shell, mouse, reset, ok = settingsRows(p, h, hh, hp, eh, side, themeRows, onHeader, onShell, onMouse, onReset)
		}
	}
	floor := 1
	if onHeader || onShell || onMouse || onReset || themeRows == 0 {
		floor = 0
	}
	return settingsFit{parts: p, side: side, rows: max(floor, rows), shell: shell, mouse: mouse, reset: reset}
}

// settingsWindow returns the first theme index and the number of theme rows
// shown when rows rows of room are given for n themes. The window keeps the
// cursor in view.
func settingsWindow(cursor, rows, n int) (start, shown int) {
	shown = min(rows, n)
	start = min(max(cursor-shown/2, 0), n-shown)
	return start, shown
}

// viewSettings draws the settings screen. The Themes section comes first. The
// preview sits beside it when the window is wide enough, under it when the
// window is narrower. The list is cut to a window around the cursor when the
// height is short. Every row starts at column 2.
//
// When the height is short the parts give way in a fixed order. The preview
// goes first, then the blank under the header, the Themes header and the blank
// above the error and the help, so the whole list fits if it can. Only when the
// list still does not fit is it cut to a window around the cursor. Then the
// error goes, and last the help. The cursor row goes only when the window has
// no row left for it. The result is never taller than the window.
func (m *MainModel) viewSettings() string {
	w, h := m.screenSize()
	header := m.screenHeader("settings")
	help := m.renderHelpNow(m.screenHelp(), w)
	errText := m.settingsError(w)

	fit := m.settingsFit(w, h)
	p := fit.parts
	preview := settingsPreview()
	// Under the list the preview needs its width and the column-2 indent.
	stacked := indentLines(preview)

	list := m.settingsList(fit, w)
	body := list
	switch {
	case !p.preview:
	case fit.side:
		body = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(lipgloss.Width(list)).Render(list),
			"   ",
			preview,
		)
	default:
		body = list + "\n\n" + stacked
	}

	blocks := []string{header}
	if p.top {
		blocks = append(blocks, "")
	}
	blocks = append(blocks, body)
	if p.err {
		if p.below {
			blocks = append(blocks, "")
		}
		blocks = append(blocks, errText)
	}
	if p.help {
		if p.below {
			blocks = append(blocks, "")
		}
		blocks = append(blocks, help)
	}
	lines := strings.Split(strings.Join(blocks, "\n"), "\n")
	return strings.Join(lines[:min(len(lines), h)], "\n")
}

// settingsList draws the Themes header, theme rows, and remaining settings
// when the fit has them. At most fit.rows theme rows are drawn,
// a window that keeps the cursor in view. Each row is cut to w columns.
func (m *MainModel) settingsList(fit settingsFit, w int) string {
	var lines []string
	if fit.parts.label {
		lines = append(lines, m.settingsThemesRow(w))
	}
	if m.settingsThemesOpen() {
		start, shown := settingsWindow(m.settingsCursor, fit.rows, len(Themes))
		for i := start; i < start+shown; i++ {
			t := Themes[i]
			var line string
			if i == m.settingsCursor {
				bar := lipgloss.NewStyle().Foreground(ColorSecondary).Render("│ ")
				line = "  " + bar + lipgloss.NewStyle().Foreground(ColorSecondary).Render(t.Name)
			} else {
				line = "    " + lipgloss.NewStyle().Foreground(ColorText).Render(t.Name)
			}
			if t.ID == m.settingsSaved {
				line += " " + SubtextStyle.Render("(current)")
			}
			lines = append(lines, ansi.Truncate(line, w, ""))
		}
	}
	if fit.shell {
		lines = append(lines, m.settingsShellRow(w))
	}
	if fit.mouse {
		lines = append(lines, m.settingsMouseRow(w))
	}
	if fit.reset {
		text := factoryResetSteps[0]
		if m.reset != nil {
			text = factoryResetSteps[m.reset.step]
			if m.reset.running {
				text = "Backing up and resetting..."
			}
		}
		if m.settingsOnReset() {
			lines = append(lines, ansi.Truncate("  "+lipgloss.NewStyle().Foreground(ColorSecondary).Render("│ "+text), w, ""))
		} else {
			lines = append(lines, ansi.Truncate("    "+lipgloss.NewStyle().Foreground(ColorText).Render(text), w, ""))
		}
	}
	return strings.Join(lines, "\n")
}

func (m *MainModel) settingsThemesRow(w int) string {
	arrow := "▾"
	if !m.settingsThemesOpen() {
		arrow = "▸"
	}
	text := arrow + " Themes"
	var line string
	if m.settingsOnThemesHeader() {
		bar := lipgloss.NewStyle().Foreground(ColorSecondary).Render("│ ")
		line = "  " + bar + lipgloss.NewStyle().Foreground(ColorSecondary).Render(text)
	} else {
		line = "    " + SubtextStyle.Render(text)
	}
	return ansi.Truncate(line, w, "")
}

func (m *MainModel) settingsShellRow(w int) string {
	text := "Shell  " + m.shell
	var line string
	if m.settingsOnShell() {
		bar := lipgloss.NewStyle().Foreground(ColorSecondary).Render("│ ")
		line = "  " + bar + lipgloss.NewStyle().Foreground(ColorSecondary).Render(text)
	} else {
		line = "    " + lipgloss.NewStyle().Foreground(ColorText).Render(text)
	}
	return ansi.Truncate(line, w, "")
}

func (m *MainModel) settingsMouseRow(w int) string {
	value := "on"
	if m.mouseOff {
		value = "off"
	}
	text := "Mouse  " + value
	var line string
	if m.settingsOnMouse() {
		bar := lipgloss.NewStyle().Foreground(ColorSecondary).Render("│ ")
		line = "  " + bar + lipgloss.NewStyle().Foreground(ColorSecondary).Render(text)
	} else {
		line = "    " + lipgloss.NewStyle().Foreground(ColorText).Render(text)
	}
	return ansi.Truncate(line, w, "")
}

// settingsPreview draws a sample of every role the screens use, so a theme can
// be judged before it is saved. It is settingsPreviewHeight rows tall and is
// drawn only while the Themes section is open.
func settingsPreview() string {
	tabs := activeTabStyle.Render("ALL") + " " + inactiveTabStyle.Render("42")
	selected := lipgloss.NewStyle().Foreground(ColorSecondary).Render("│ [GO] ft_strlen")
	normal := "  " + lipgloss.NewStyle().Foreground(ColorText).Render("[C] ft_split")
	status := SuccessStyle.Render("PASS") + " " + DangerStyle.Render("FAIL")

	var ranks []string
	for _, g := range []Grade{GradeS, GradeA, GradeB, GradeC} {
		ranks = append(ranks, lipgloss.NewStyle().Bold(true).Foreground(gradeColor(g)).Render(string(g)))
	}
	divider := lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", 28))

	return strings.Join([]string{
		tabs,
		selected,
		normal,
		status,
		strings.Join(ranks, " "),
		renderProgressBar(0.6, 10),
		labelValue("streak", "3", lipgloss.NewStyle()),
		divider,
	}, "\n")
}
