package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/typingsound"
)

// typingChoice is one selectable entry on the typing sounds tab.
type typingChoice struct {
	ID     string
	Label  string
	Credit string
	Source string
}

// The typing tab has four fixed rows before the sound-pack list. Pack rows are
// indexes 4 and up; the list scrolls as one block in short windows.
const (
	typingRowEnabled = iota
	typingRowVolume
	typingRowWords
	typingRowPreview
	typingRowPackBase
)

func typingChoices() []typingChoice {
	choices := []typingChoice{
		{ID: typingsound.SilentPackID, Label: "Silent"},
		{ID: typingsound.PrototypePackID, Label: "Generated tones (prototype)", Credit: "Built into Codedrill"},
	}
	for _, pack := range typingsound.Catalog {
		choices = append(choices, typingChoice{
			ID:     pack.ID,
			Label:  pack.Name,
			Credit: pack.Author + " (TETR.IO+)",
			Source: typingsound.TETRIODatabase,
		})
	}
	return choices
}

type typingPreviewMsg struct {
	name string
	err  error
}

// openTypingSounds shows the typing sounds tab. The cursor starts on the
// enabled row; the saved pack, volume, word setting and toggle are read here.
func (m *MainModel) openTypingSounds() {
	m.typingCursor = 0
	m.typingNotice = ""
	m.typingErr = nil
	m.typingChoices = typingChoices()
	m.typingOn = false
	m.typingWords = false
	m.typingVolume = 70
	m.typingPack = typingsound.SilentPackID
	if cfg, err := storage.LoadConfig(); err == nil {
		m.typingOn = cfg.TypingSoundsOn()
		m.typingWords = cfg.TypingWordSoundsOn()
		m.typingVolume = int(cfg.TypingVolume()*100 + 0.5)
		if pack := cfg.TypingPackID(); pack != "" {
			m.typingPack = pack
		}
	} else {
		m.typingErr = err
	}
	m.state = StateTypingSounds
}

func (m *MainModel) typingChoiceIndex() int {
	for i, choice := range m.typingChoices {
		if choice.ID == m.typingPack {
			return i
		}
	}
	return 0
}

func (m *MainModel) typingOnPackRow() bool {
	return m.typingCursor >= typingRowPackBase
}

func (m *MainModel) typingCursorChoice() typingChoice {
	if m.typingOnPackRow() {
		if i := m.typingCursor - typingRowPackBase; i >= 0 && i < len(m.typingChoices) {
			return m.typingChoices[i]
		}
	}
	return m.typingChoices[m.typingChoiceIndex()]
}

// saveTyping writes the editable values and reports a failure without leaving
// the tab.
func (m *MainModel) saveTyping() bool {
	cfg, err := storage.LoadConfig()
	if err == nil {
		if cfg.Typing == nil {
			cfg.Typing = &storage.TypingSettings{}
		}
		enabled, words, volume := m.typingOn, m.typingWords, m.typingVolume
		cfg.Typing.Enabled = &enabled
		cfg.Typing.WordCompletion = &words
		cfg.Typing.Volume = &volume
		cfg.Typing.Pack = m.typingPack
		err = storage.SaveConfig(cfg)
	}
	if err != nil {
		m.typingErr = err
		return false
	}
	m.typingErr = nil
	return true
}

func (m *MainModel) previewTyping() tea.Cmd {
	choice := m.typingCursorChoice()
	volume := m.typingVolume
	return func() tea.Msg {
		if choice.ID == typingsound.SilentPackID {
			return typingPreviewMsg{name: choice.Label}
		}
		err := typingsound.PreviewPack(choice.ID, float64(volume)/100)
		return typingPreviewMsg{name: choice.Label, err: err}
	}
}

func (m *MainModel) toggleTypingWords() {
	m.typingWords = !m.typingWords
	if m.saveTyping() {
		if m.typingWords {
			m.typingNotice = "Word completion sounds on."
		} else {
			m.typingNotice = "Word completion sounds off."
		}
	}
}

func (m *MainModel) selectTypingPack(row int) tea.Cmd {
	i := row - typingRowPackBase
	if i < 0 || i >= len(m.typingChoices) {
		return nil
	}
	m.typingCursor = row
	m.typingPack = m.typingChoices[i].ID
	if m.saveTyping() {
		if !m.typingOn && m.typingPack != typingsound.SilentPackID {
			m.typingNotice = "Pack saved. Turn sounds on to hear it while editing."
		} else {
			m.typingNotice = ""
		}
	}
	return m.previewTyping()
}

func (m *MainModel) updateTypingSounds(key tea.KeyMsg) tea.Cmd {
	w, h := m.screenSize()
	layout := m.typingLayout(w, h)
	last := typingRowPackBase + len(m.typingChoices) - 1
	switch key.String() {
	case "up", "k":
		m.typingCursor = max(0, m.typingCursor-1)
	case "down", "j":
		m.typingCursor = min(last, m.typingCursor+1)
	case "pgup":
		m.typingCursor = max(0, m.typingCursor-layout.contentRows)
	case "pgdown":
		m.typingCursor = min(last, m.typingCursor+layout.contentRows)
	case "g", "home":
		m.typingCursor = 0
	case "G", "end":
		m.typingCursor = last
	case "left", "h":
		m.stepTypingVolume(-5)
	case "right", "l":
		m.stepTypingVolume(5)
	case "enter", " ":
		switch m.typingCursor {
		case typingRowEnabled:
			m.typingOn = !m.typingOn
			if m.saveTyping() && m.typingOn && m.typingPack == typingsound.SilentPackID {
				m.typingNotice = "Sounds are on. Choose a pack below."
			}
		case typingRowWords:
			m.toggleTypingWords()
		case typingRowPreview:
			return m.previewTyping()
		default:
			if m.typingOnPackRow() {
				return m.selectTypingPack(m.typingCursor)
			}
		}
	case "p":
		return m.previewTyping()
	case "tab", "shift+tab", "t":
		m.state = StateSettings
	case "esc", "q":
		m.closeSettings()
	}
	return nil
}

func (m *MainModel) stepTypingVolume(delta int) {
	next := min(100, max(0, m.typingVolume+delta))
	if next == m.typingVolume {
		return
	}
	m.typingVolume = next
	if m.saveTyping() {
		m.typingNotice = fmt.Sprintf("Volume %d%%.", next)
	}
}

// typingLayout is the rendered position of the scrolling typing content. The
// fixed rows and every pack row form one list, so the cursor stays visible in
// short windows and the tab bar is never scrolled away.
type typingLayout struct {
	header      string
	contentTop  int
	contentRows int
	start       int
	shown       int
	total       int
	credit      string
	notice      string
	help        string
	blankAbove  bool
}

func (m *MainModel) typingLayout(w, h int) typingLayout {
	layout := typingLayout{header: m.settingsHeaderBlock(w)}
	layout.help = m.renderHelpNow(m.screenHelp(), w)
	switch {
	case m.typingErr != nil:
		layout.notice = DangerStyle.Render(ansi.Truncate("  Could not save: "+m.typingErr.Error(), w, ""))
	case m.typingNotice != "":
		layout.notice = SubtextStyle.Render(ansi.Truncate("  "+m.typingNotice, w, ""))
	}
	layout.credit = m.typingCredit(w)
	layout.total = typingRowPackBase + len(m.typingChoices)
	layout.blankAbove = h >= 16

	height := func() int {
		used := lipgloss.Height(layout.header) + lipgloss.Height(layout.credit) +
			lipgloss.Height(layout.notice) + lipgloss.Height(layout.help)
		if layout.blankAbove {
			used++
		}
		return h - used
	}
	rows := height()
	if rows < 1 && layout.blankAbove {
		layout.blankAbove = false
		rows = height()
	}
	// Give up the optional footers before the selected row.
	if rows < 1 && layout.credit != "" {
		layout.credit = ""
		rows = height()
	}
	if rows < 1 && layout.help != "" {
		layout.help = ""
		rows = height()
	}
	if rows < 1 && layout.notice != "" {
		layout.notice = ""
		rows = height()
	}
	layout.contentRows = max(1, rows)
	layout.start = typingWindow(m.typingCursor, layout.contentRows, layout.total)
	layout.shown = min(layout.contentRows, layout.total)
	layout.contentTop = lipgloss.Height(layout.header)
	if layout.blankAbove {
		layout.contentTop++
	}
	return layout
}

func typingWindow(cursor, rows, total int) int {
	if total <= rows {
		return 0
	}
	return min(max(cursor-rows/2, 0), total-rows)
}

func (m *MainModel) typingCredit(w int) string {
	choice := m.typingCursorChoice()
	if choice.Credit == "" || (!m.typingOnPackRow() && m.typingCursor != typingRowPreview) {
		return ""
	}
	credit := choice.Credit
	if choice.Source != "" {
		credit += " · " + choice.Source
	}
	return "  " + SubtextStyle.Render(ansi.Truncate(credit, w-2, ""))
}

// viewTypingSounds draws the typing tab and keeps the selected row on screen.
func (m *MainModel) viewTypingSounds() string {
	w, h := m.screenSize()
	layout := m.typingLayout(w, h)

	lines := strings.Split(layout.header, "\n")
	if layout.blankAbove {
		lines = append(lines, "")
	}
	for row := layout.start; row < layout.start+layout.shown; row++ {
		lines = append(lines, m.typingRow(w, row))
	}
	if layout.credit != "" {
		lines = append(lines, layout.credit)
	}
	if layout.notice != "" {
		lines = append(lines, layout.notice)
	}
	if layout.help != "" {
		lines = append(lines, layout.help)
	}
	lines = lines[:min(len(lines), h)]
	return strings.Join(lines, "\n")
}

func (m *MainModel) typingRow(w, row int) string {
	switch row {
	case typingRowEnabled:
		value := "off"
		if m.typingOn {
			value = "on"
		}
		return cursorRow(m.typingCursor == row, w, "Typing sounds  "+value)
	case typingRowVolume:
		return cursorRow(m.typingCursor == row, w, fmt.Sprintf("Volume  %d%%", m.typingVolume))
	case typingRowWords:
		value := "off"
		if m.typingWords {
			value = "on"
		}
		return cursorRow(m.typingCursor == row, w, "Word completion sounds  "+value)
	case typingRowPreview:
		return cursorRow(m.typingCursor == row, w, "Preview selected pack")
	default:
		i := row - typingRowPackBase
		if i < 0 || i >= len(m.typingChoices) {
			return ""
		}
		text := m.typingChoices[i].Label
		if m.typingChoices[i].ID == m.typingPack {
			text += " (current)"
		}
		return cursorRow(m.typingCursor == row, w, text)
	}
}

// typingRowAt returns the logical row at cell (x, y), including the fixed rows.
func (m *MainModel) typingRowAt(x, y int) (int, bool) {
	w, h := m.screenSize()
	if x < 0 || x >= w || y < 0 || y >= h {
		return 0, false
	}
	layout := m.typingLayout(w, h)
	if y < layout.contentTop || y >= layout.contentTop+layout.shown {
		return 0, false
	}
	row := layout.start + y - layout.contentTop
	if row < 0 || row >= layout.total {
		return 0, false
	}
	return row, true
}

// updateTypingSoundsMouse switches tabs, scrolls with the wheel, and selects
// rows. A click on a pack row selects and previews it, as Enter does.
func (m *MainModel) updateTypingSoundsMouse(msg tea.MouseMsg) tea.Cmd {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		m.typingCursor = max(0, m.typingCursor-1)
	case tea.MouseButtonWheelDown:
		last := typingRowPackBase + len(m.typingChoices) - 1
		m.typingCursor = min(last, m.typingCursor+1)
	case tea.MouseButtonLeft:
		if tab, ok := m.settingsTabAt(msg.X, msg.Y); ok {
			if tab == settingsTabGeneral {
				m.state = StateSettings
			}
			return nil
		}
		row, ok := m.typingRowAt(msg.X, msg.Y)
		if !ok {
			return nil
		}
		switch row {
		case typingRowEnabled:
			m.typingCursor = row
			m.typingOn = !m.typingOn
			if m.saveTyping() && m.typingOn && m.typingPack == typingsound.SilentPackID {
				m.typingNotice = "Sounds are on. Choose a pack below."
			}
		case typingRowWords:
			m.typingCursor = row
			m.toggleTypingWords()
		case typingRowPreview:
			m.typingCursor = row
			return m.previewTyping()
		default:
			if row >= typingRowPackBase {
				return m.selectTypingPack(row)
			}
			m.typingCursor = row
		}
	}
	return nil
}

// cursorRow draws one settings row, with the same bar the settings screen uses.
func cursorRow(on bool, w int, text string) string {
	var line string
	if on {
		bar := lipgloss.NewStyle().Foreground(ColorSecondary).Render("│ ")
		line = "  " + bar + lipgloss.NewStyle().Foreground(ColorSecondary).Render(text)
	} else {
		line = "    " + lipgloss.NewStyle().Foreground(ColorText).Render(text)
	}
	return ansi.Truncate(line, w, "")
}
