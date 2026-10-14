package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/storage"
)

func (m *MainModel) screenSize() (int, int) {
	w, h := m.width, m.height
	if w <= 0 {
		w = 80
	}
	if h <= 0 {
		h = 24
	}
	return w, h
}

func (m *MainModel) screenHelp() string {
	switch m.state {
	case StateBriefing:
		return "[Enter] start  [n/p] next/prev  [r] random\n[l] history  [Esc/q] back"
	case StateEvaluating:
		switch {
		case m.evalPending:
			return ""
		case m.launchFailed:
			return "[e] try again  [Enter/q] back to the briefing"
		case m.saveErr != nil:
			return "[Enter] retry save  [e] editor  [q/Esc] discard this run"
		case m.evalPassed:
			return "[Enter] finish  [e] editor  [q/Esc] quit without saving"
		}
		return "[Enter/e] editor  [q/Esc] forfeit"
	case StateResults:
		return "[Enter/q] menu  [r] retry  [n/p] next/prev\n[b] briefing  [l] history"
	case StateLeaderboard:
		return "[Enter] sprint  [n/p] next/prev  [Tab] toggle\n[Esc/q] back"
	case StateStats:
		return "[Esc/q/Enter] menu"
	}
	return ""
}

func (m *MainModel) scrollable() bool {
	switch m.state {
	case StateBriefing, StateEvaluating, StateResults, StateLeaderboard, StateStats:
		return true
	}
	return false
}

func (m *MainModel) screenLines() ([]string, string, int) {
	w, h := m.screenSize()
	content := strings.Trim(m.contentView(), "\n")
	lines := strings.Split(ansi.Wrap(content, w, ""), "\n")
	help := ansi.Wrap(m.screenHelp(), w, "")
	available := h
	if help != "" {
		available -= lipgloss.Height(help) + 1
	}
	if len(lines) > available {
		if help != "" {
			help += "\n"
		}
		help += "[↑/↓ PgUp/PgDn] scroll"
		available = h - lipgloss.Height(help) - 1
	}
	return lines, help, max(1, available)
}

func (m *MainModel) View() string {
	if !m.scrollable() {
		return m.contentView()
	}
	lines, help, available := m.screenLines()
	offset := min(m.scrollOffset, max(0, len(lines)-available))
	end := min(len(lines), offset+available)
	body := strings.Join(lines[offset:end], "\n")
	if help != "" {
		body += strings.Repeat("\n", available-(end-offset)+1) + "\n" + TimerStyle.Padding(0).Render(help)
	}
	return body
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok && m.scrollable() {
		lines, _, available := m.screenLines()
		limit := max(0, len(lines)-available)
		offset := min(m.scrollOffset, limit)
		switch key.String() {
		case "up", "k":
			offset--
		case "down", "j":
			offset++
		case "pgup":
			offset -= available
		case "pgdown":
			offset += available
		case "home":
			offset = 0
		case "end":
			offset = limit
		default:
			return m.updateScreen(msg)
		}
		m.scrollOffset = min(limit, max(0, offset))
		return m, nil
	}
	return m.updateScreen(msg)
}

func (m *MainModel) updateScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	state, active, pending := m.state, m.activeDrill, m.evalPending
	model, cmd := m.update(msg)
	if m.state != state || m.activeDrill != active || m.evalPending != pending {
		m.scrollOffset = 0
	}
	return model, cmd
}

func (m *MainModel) menuHeader() string {
	level := storage.GetLevelInfo(m.history.GetTotalXP())
	croc := LogoStyle.Bold(false).PaddingLeft(2).Render(runner.CrocArt)
	text := lipgloss.JoinVertical(lipgloss.Left,
		"",
		LogoStyle.Render("CODEDRILL"),
		SubtextStyle.Render(fmt.Sprintf("LVL %d · %d XP · streak %d", level.Level, level.TotalXP, m.history.CurrentStreak)),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, croc, "  ", text)
}

func (m *MainModel) menuHelp() string {
	w, _ := m.screenSize()
	if d := m.pendingDelete; d != nil {
		prompt := fmt.Sprintf("Delete %q? Its runs are removed too.", d.Title)
		if d.IsPreset {
			prompt = fmt.Sprintf("Hide built-in %q? Restore later with codedrill restore.", d.Title)
		}
		return DangerStyle.Render(ansi.Wrap(prompt, w, "")) + "\n" + TimerStyle.Padding(0).Render("[y] yes  [any other key] cancel")
	}
	keys := "[Enter] select  [h/l Tab] category  [d] daily  [r] random\n[s] profile  [v] history  [x] delete  [q] quit"
	if m.menuNotice != "" {
		return TimerStyle.Padding(0).Render(ansi.Wrap(m.menuNotice, w, "")) + "\n" + SubtextStyle.Render(ansi.Wrap(keys, w, ""))
	}
	return SubtextStyle.Render(ansi.Wrap(keys, w, ""))
}

func (m *MainModel) sizeList() {
	w, h := m.screenSize()
	m.list.SetSize(max(1, w-2), max(4, h-lipgloss.Height(m.menuHeader())-lipgloss.Height(m.menuHelp())-3))
}
