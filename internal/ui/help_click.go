package ui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

const (
	formHelp      = "[Enter] next  [Shift+Tab] back  [Esc] cancel"
	setupHelp     = "[Enter] retry setup\n[Esc/q] back to the briefing"
	countdownHelp = "[Space/Enter] skip  [Esc] cancel"
	deleteHelp    = "[y] yes  [any other key] cancel"
)

// helpPressState is a footer item that was clicked and is showing its pressed
// look while it waits for footerPressMsg to run key.
type helpPressState struct {
	item string
	key  tea.KeyMsg
}

// footerPressDelay is how long a clicked footer item shows its pressed look
// before its action fires. The delay is a variable so tests can shrink it.
var footerPressDelay = 150 * time.Millisecond

// footerPressMsg ends the pressed state and runs the clicked item's action.
type footerPressMsg struct{}

func footerPressTimer() tea.Cmd {
	return tea.Tick(footerPressDelay, func(time.Time) tea.Msg {
		return footerPressMsg{}
	})
}

// pressedItem is the footer item waiting for its action, or "" when there is
// none.
func (m *MainModel) pressedItem() string {
	if m.helpPress == nil {
		return ""
	}
	return m.helpPress.item
}

// renderHelpNow draws footer text the way the screen shows it, with the item
// that is waiting for its action in inverse video. Footer render sites call
// this instead of renderHelp, so the pressed look follows the click without
// passing the current item around.
func (m *MainModel) renderHelpNow(s string, width int) string {
	return renderHelp(s, width, m.pressedItem())
}

func (m *MainModel) helpText() string {
	switch m.state {
	case StateMenu:
		if m.search.open {
			return ""
		}
		if m.pendingDelete != nil {
			return deleteHelp
		}
		return m.menuKeys()
	case StateAddDrill:
		return formHelp
	case StateSetupError:
		return setupHelp
	case StateCountdown:
		return countdownHelp
	default:
		return m.screenHelp()
	}
}

// helpItemAt finds the footer item under cell (x, y). It returns the key the
// item stands for and its text, which rendering can find again to draw the
// pressed look. Body text that resembles a shortcut is not a target.
func (m *MainModel) helpItemAt(x, y int) (tea.KeyMsg, string, bool) {
	w, h := m.screenSize()
	if x < 0 || x >= w || y < 0 || y >= h {
		return tea.KeyMsg{}, "", false
	}
	text := m.helpText()
	if text == "" {
		return tea.KeyMsg{}, "", false
	}
	view := strings.Split(ansi.Strip(m.View()), "\n")
	footer := strings.Split(ansi.Strip(renderHelp(text, w, "")), "\n")
	// Some screens pad the footer to their full width.
	start := -1
	for i := len(view) - len(footer); i >= 0; i-- {
		match := true
		for j, row := range footer {
			if strings.TrimRight(view[i+j], " ") != strings.TrimRight(row, " ") {
				match = false
				break
			}
		}
		if match {
			start = i
			break
		}
	}
	row := y + max(0, len(view)-h) - start
	if start < 0 || row < 0 || row >= len(footer) {
		return tea.KeyMsg{}, "", false
	}
	col := 2
	for _, item := range strings.Split(strings.TrimPrefix(footer[row], "  "), "  ") {
		width := ansi.StringWidth(item)
		if x >= col && x < col+width {
			key, ok := helpItemKey(item, x-col)
			return key, item, ok
		}
		col += width + 2
	}
	return tea.KeyMsg{}, "", false
}

func (m *MainModel) helpKeyAt(x, y int) (tea.KeyMsg, bool) {
	key, _, ok := m.helpItemAt(x, y)
	return key, ok
}

func helpItemKey(item string, x int) (tea.KeyMsg, bool) {
	end := strings.IndexByte(item, ']')
	if !strings.HasPrefix(item, "[") || end < 0 {
		return tea.KeyMsg{}, false
	}
	key := item[1:end]
	switch key {
	case "↑/↓ j/k", "↑/↓ PgUp/PgDn":
		return tea.KeyMsg{}, false
	case "h/l ←/→ Tab":
		key = "Tab"
	case "any other key":
		key = "Esc"
	case "n/p", "n/N":
		parts := strings.Split(key, "/")
		key = parts[0]
		if x >= 3 && x <= 4 {
			key = parts[1]
		}
	default:
		// A slash alone is the search key, not a separator.
		if key != "/" {
			key = strings.Split(key, "/")[0]
		}
	}
	types := map[string]tea.KeyType{
		"Enter": tea.KeyEnter, "Esc": tea.KeyEsc, "Tab": tea.KeyTab,
		"Shift+Tab": tea.KeyShiftTab, "Space": tea.KeySpace,
	}
	if t, ok := types[key]; ok {
		return tea.KeyMsg{Type: t}, true
	}
	if len([]rune(key)) == 1 {
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}, true
	}
	return tea.KeyMsg{}, false
}
