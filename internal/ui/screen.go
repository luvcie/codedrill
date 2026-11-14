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
	help := m.screenKeys()
	if m.currentReferenceReview() != nil {
		help += "\n[v] Reference diff"
	}
	return help
}

func (m *MainModel) screenKeys() string {
	switch m.state {
	case StateBriefing:
		return "[Tab/Shift+Tab] option  [Space] toggle  [←/→] mode\n[Enter] start  [n/p] next/prev  [r] random\n[l] history  [Esc/q] back"
	case StateEvaluating:
		switch {
		case m.evalPending:
			return ""
		case m.launchFailed:
			return "[e] try again  [Enter/q] back to the briefing"
		case m.saveErr != nil:
			return "[Enter] retry save  [e] editor  [q/Esc] discard this run"
		case m.passVerdict().Passed:
			return "[Enter] finish  [e] editor  [q/Esc] quit without saving"
		}
		return "[Enter/e] editor  [q/Esc] forfeit"
	case StateResults:
		return "[Enter/q] menu  [r] retry  [n/p] next/prev\n[b] briefing  [l] history"
	case StateLeaderboard:
		return "[Enter] sprint  [n/p] next/prev  [Tab] toggle\n[Esc/q] back"
	case StateStats:
		return "[Esc/q/Enter] menu"
	case StateSettings:
		return "[Tab] tabs  [↑/↓] move  [Enter] select  [Esc/q] back"
	case StateTypingSounds:
		return "[Tab] tabs  [↑/↓] move  [Enter] select  [←/→] volume\n[p] preview  [Esc/q] back"
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
	help := m.renderHelpNow(m.screenHelp(), w)
	available := h
	if help != "" {
		available -= lipgloss.Height(help) + 1
	}
	if len(lines) > available {
		if help != "" {
			help += "\n"
		}
		help += m.renderHelpNow("[↑/↓ PgUp/PgDn] scroll", w)
		available = h - lipgloss.Height(help) - 1
	}
	return lines, help, max(1, available)
}

func (m *MainModel) View() string {
	if m.referenceDiff != nil {
		return m.viewReferenceDiff()
	}
	if !m.scrollable() {
		return m.contentView()
	}
	lines, help, available := m.screenLines()
	offset := min(m.scrollOffset, max(0, len(lines)-available))
	end := min(len(lines), offset+available)
	body := strings.Join(lines[offset:end], "\n")
	if help != "" {
		body += strings.Repeat("\n", available-(end-offset)+1) + "\n" + help
	}
	return body
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.referenceDiff != nil {
		switch msg.(type) {
		case tea.KeyMsg, tea.MouseMsg:
			return m, m.updateReferenceDiff(msg)
		}
	}
	if key, ok := msg.(tea.KeyMsg); ok && m.scrollable() {
		lines, _, available := m.screenLines()
		switch key.String() {
		case "up", "k":
			m.scrollBy(-1)
		case "down", "j":
			m.scrollBy(1)
		case "pgup":
			m.scrollBy(-available)
		case "pgdown":
			m.scrollBy(available)
		case "home":
			m.scrollOffset = 0
		case "end":
			m.scrollOffset = max(0, len(lines)-available)
		default:
			return m.updateScreen(msg)
		}
		return m, nil
	}
	return m.updateScreen(msg)
}

func (m *MainModel) scrollBy(delta int) {
	lines, _, available := m.screenLines()
	limit := max(0, len(lines)-available)
	m.scrollOffset = min(limit, max(0, min(m.scrollOffset, limit)+delta))
}

func (m *MainModel) updateScreen(msg tea.Msg) (tea.Model, tea.Cmd) {
	state, active, pending := m.state, m.activeDrill, m.evalPending
	model, cmd := m.update(msg)
	if m.state != state || m.activeDrill != active || m.evalPending != pending {
		m.scrollOffset = 0
		m.helpPress = nil
	}
	return model, cmd
}

func (m *MainModel) screenHeader(context string) string {
	w, _ := m.screenSize()
	const wordmark = "codedrill"
	croc := LogoStyle.Bold(false).PaddingLeft(2).Render(runner.CrocArt)
	if w < lipgloss.Width(croc)+2+ansi.StringWidth(wordmark) {
		room := max(1, w-2)
		return lipgloss.JoinVertical(lipgloss.Left,
			"",
			LogoStyle.PaddingLeft(2).Render(ansi.Truncate(wordmark, room, "…")),
			SubtextStyle.PaddingLeft(2).Render(ansi.Truncate(context, room, "…")),
		)
	}
	room := max(1, w-lipgloss.Width(croc)-2)
	text := lipgloss.JoinVertical(lipgloss.Left,
		"",
		LogoStyle.Render(wordmark),
		SubtextStyle.Render(ansi.Truncate(context, room, "…")),
	)
	return lipgloss.JoinHorizontal(lipgloss.Top, croc, "  ", text)
}

func (m *MainModel) menuHeader() string {
	level := storage.GetLevelInfo(m.history.GetTotalXP())
	return m.screenHeader(fmt.Sprintf("level %d · %d XP · streak %d", level.Level, level.TotalXP, m.history.CurrentStreak))
}

func renderHelp(s string, width int, pressed string) string {
	if s == "" {
		return ""
	}
	var rows []string
	for _, line := range strings.Split(s, "\n") {
		for _, row := range wrapHelpLine(line, max(1, width-2)) {
			rows = append(rows, "  "+styleHelpRow(row, pressed))
		}
	}
	return strings.Join(rows, "\n")
}

func wrapHelpLine(line string, width int) []string {
	var rows []string
	row := ""
	for _, item := range strings.Split(line, "  ") {
		switch {
		case row == "":
			row = item
		case ansi.StringWidth(row)+2+ansi.StringWidth(item) <= width:
			row += "  " + item
		default:
			rows = append(rows, row)
			row = item
		}
	}
	rows = append(rows, row)

	var out []string
	for _, r := range rows {
		out = append(out, strings.Split(ansi.Wrap(r, width, ""), "\n")...)
	}
	return out
}

func styleHelpRow(row, pressed string) string {
	var b strings.Builder
	lo, hi := -1, -1
	if pressed != "" {
		if i := strings.Index(row, pressed); i >= 0 {
			lo, hi = i, i+len(pressed)
		}
	}
	pos := 0
	write := func(text, styled string) {
		if lo >= 0 && pos < hi && lo < pos+len(text) {
			styled = pressStyle.Render(styled)
		}
		b.WriteString(styled)
		pos += len(text)
	}
	dim := func(text string) {
		if text != "" {
			write(text, SubtextStyle.Render(text))
		}
	}
	for {
		open := strings.IndexByte(row[pos:], '[')
		if open < 0 {
			break
		}
		open += pos
		end := strings.IndexByte(row[open:], ']')
		if end < 0 {
			break
		}
		end += open
		dim(row[pos:open])
		write(row[open:end+1], helpKeyStyle.Render(row[open:end+1]))
		pos = end + 1
	}
	dim(row[pos:])
	return b.String()
}

func (m *MainModel) menuHelp() string {
	w, _ := m.screenSize()
	iw := max(1, w-2)
	if d := m.pendingDelete; d != nil {
		prompt := fmt.Sprintf("Delete %q? Its runs are removed too.", d.Title)
		if d.IsPreset {
			prompt = fmt.Sprintf("Hide built-in %q? Restore later with codedrill restore.", d.Title)
		}
		return DangerStyle.PaddingLeft(2).Render(ansi.Wrap(prompt, iw, "")) + "\n" + m.renderHelpNow(deleteHelp, w)
	}
	if m.search.open {
		return noticeStyle.Render(ansi.Wrap("/"+m.search.text+"_", iw, ""))
	}
	keys := m.menuKeys()
	if m.menuNotice != "" {
		return noticeStyle.Render(ansi.Wrap(m.menuNotice, iw, "")) + "\n" + m.renderHelpNow(keys, w)
	}
	return m.renderHelpNow(keys, w)
}

func (m *MainModel) menuKeys() string {
	keys := "[Enter] select  [h/l ←/→ Tab] category  [d] daily  [r] random\n[Shift+↑/↓] reorder  [/] search  [s] profile  [v] history  [o] settings  [x] delete  [e] edit  [q] quit"
	if m.search.pattern != "" {
		hint := "[n/N] next/prev"
		if m.search.hl {
			hint += "  [Esc] clear highlight"
		}
		keys += "\n" + hint
	}
	return keys
}

func (m *MainModel) sizeList() {
	w, h := m.screenSize()
	m.list.SetSize(max(1, w-2), max(4, h-lipgloss.Height(m.menuHeader())-lipgloss.Height(m.menuHelp())-3))
}
