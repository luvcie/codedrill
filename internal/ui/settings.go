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

// settingsPreviewHeight is the number of rows settingsPreview draws.
const settingsPreviewHeight = 8

// openSettings opens the settings screen from the menu. The cursor starts on
// the active theme. The active theme is kept, so Esc can go back to it.
func (m *MainModel) openSettings() {
	m.settingsOrigin = CurrentTheme()
	m.settingsCursor = 0
	for i, t := range Themes {
		if t.ID == m.settingsOrigin.ID {
			m.settingsCursor = i
		}
	}
	// A config that cannot be read has no saved theme to mark. Enter will
	// report the read error when it tries to save.
	m.settingsSaved = ""
	if cfg, err := storage.LoadConfig(); err == nil {
		m.settingsSaved = ThemeOrDefault(cfg.Theme).ID
	}
	m.settingsErr = nil
	m.state = StateSettings
}

// updateSettings handles keys on the settings screen. The cursor moves through
// the theme rows and then the mouse row. Moving onto a theme previews it. Enter
// saves the highlighted theme, or toggles mouse support on the mouse row. Esc
// or q puts back the theme that was active when the screen opened.
func (m *MainModel) updateSettings(key tea.KeyMsg) tea.Cmd {
	last := len(Themes)
	switch key.String() {
	case "up", "k":
		m.moveSettings(m.settingsCursor - 1)
	case "down", "j":
		m.moveSettings(m.settingsCursor + 1)
	case "g", "home":
		m.moveSettings(0)
	case "G", "end":
		m.moveSettings(last)
	case "enter", " ":
		if m.settingsOnMouse() {
			return m.toggleMouse()
		}
		if key.String() == "enter" {
			m.saveSettings()
		}
	case "esc", "q":
		m.closeSettings()
	}
	return nil
}

// settingsOnMouse reports whether the cursor is on the mouse row, which comes
// after the theme rows.
func (m *MainModel) settingsOnMouse() bool {
	return m.settingsCursor == len(Themes)
}

// moveSettings puts the cursor on row i, clamped to the theme rows and the
// mouse row. A theme row applies its theme. The mouse row changes nothing, so
// the theme under the cursor before stays previewed. Nothing is saved.
func (m *MainModel) moveSettings(i int) {
	i = min(max(i, 0), len(Themes))
	if i == m.settingsCursor && (i == len(Themes) || CurrentTheme().ID == Themes[i].ID) {
		return
	}
	m.settingsCursor = i
	// The error belongs to the last save attempt. Moving the cursor makes it
	// stale, so it goes.
	m.settingsErr = nil
	if i < len(Themes) {
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
	label   bool // the "theme" and "mouse" labels, and the blank above the mouse label
	below   bool // the blank rows above the error and the help
	err     bool // the save error
	help    bool // the key help
}

// settingsRows returns how many theme rows fit in a window h rows tall when the
// parts in p are drawn, and whether the mouse row fits with them. hh, hp and eh
// are the heights of the header, the help and the save error. onMouse says the
// cursor is on the mouse row, which then needs only its own row. ok is false
// when the cursor row does not fit.
//
// The mouse row is kept before the theme rows: it fits as long as one theme
// row fits with it, unless the cursor is on it.
func settingsRows(p settingsParts, h, hh, hp, eh int, side, onMouse bool) (rows int, mouse, ok bool) {
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
		return 0, false, false
	}
	themeCost, mouseCost := 0, 1
	if p.label {
		themeCost, mouseCost = 1, 3
	}
	need := mouseCost + themeCost + 1
	if onMouse {
		need = mouseCost + themeCost
	}
	mouse = region >= need
	if mouse {
		rows = region - themeCost - mouseCost
	} else {
		rows = region - themeCost
	}
	if onMouse {
		return rows, mouse, mouse
	}
	return rows, mouse, rows >= 1
}

// settingsFit is how the settings screen is laid out in one window: which
// parts are drawn, whether the preview sits beside the list, how many theme
// rows the list gets, and whether the mouse row is drawn. viewSettings draws
// from it and settingsRowAt reads it, so the two cannot disagree about where a
// row is.
type settingsFit struct {
	parts settingsParts
	side  bool // the preview sits beside the list
	rows  int  // theme rows in the list, at least 1 unless the cursor is on the mouse row
	mouse bool // the mouse row is drawn
}

// settingsError is the save error as drawn, or "" when there is none.
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
	onMouse := m.settingsOnMouse()
	// Under the list the preview needs its width and the column-2 indent.
	stacked := indentLines(settingsPreview())
	p := settingsParts{
		preview: side || lipgloss.Width(stacked) <= w,
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
	rows, mouse, ok := settingsRows(p, h, hh, hp, eh, side, onMouse)
	// The whole list, with the mouse row, comes before the preview and the
	// blank rows. Each part goes only while the list still does not fit.
	for _, drop := range []*bool{&p.preview, &p.top, &p.label, &p.below} {
		if ok && mouse && rows >= len(Themes) {
			break
		}
		if *drop {
			*drop = false
			rows, mouse, ok = settingsRows(p, h, hh, hp, eh, side, onMouse)
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
			rows, mouse, ok = settingsRows(p, h, hh, hp, eh, side, onMouse)
		}
	}
	floor := 1
	if onMouse {
		floor = 0
	}
	return settingsFit{parts: p, side: side, rows: max(floor, rows), mouse: mouse}
}

// settingsWindow returns the first theme index and the number of theme rows
// shown when rows rows of room are given for n themes. The window keeps the
// cursor in view.
func settingsWindow(cursor, rows, n int) (start, shown int) {
	shown = min(rows, n)
	start = min(max(cursor-shown/2, 0), n-shown)
	return start, shown
}

// viewSettings draws the settings screen. The theme list comes first. The
// preview sits beside it when the window is wide enough, under it when the
// window is narrower. The list is cut to a window around the cursor when the
// height is short. Every row starts at column 2.
//
// When the height is short the parts give way in a fixed order. The preview
// goes first, then the blank under the header, the label and the blank above
// the error and the help, so the whole list fits if it can. Only when the list
// still does not fit is it cut to a window around the cursor. Then the error
// goes, and last the help. The cursor row goes only when the window has no row
// left for it. The result is never taller than the window.
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

// settingsList draws the theme rows, then the mouse section when the fit has
// it. The labels come with the rows when the fit has them. At most fit.rows
// theme rows are drawn, a window that keeps the cursor in view. Each row is cut
// to w columns.
func (m *MainModel) settingsList(fit settingsFit, w int) string {
	start, shown := settingsWindow(m.settingsCursor, fit.rows, len(Themes))

	var lines []string
	if fit.parts.label {
		lines = append(lines, "  "+SubtextStyle.Render("theme"))
	}
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
	if fit.mouse {
		if fit.parts.label {
			lines = append(lines, "", "  "+SubtextStyle.Render("mouse"))
		}
		lines = append(lines, m.settingsMouseRow(w))
	}
	return strings.Join(lines, "\n")
}

// settingsMouseRow draws the mouse row, marked as the theme rows are. It is cut
// to w columns.
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
// be judged before it is saved. It is settingsPreviewHeight rows tall.
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
