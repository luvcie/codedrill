package ui

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/referenceview"
)

// Reference data stays in memory with the evaluation that produced it.
type referenceReview struct {
	options    mode.Options
	drill      drill.Drill
	comparison *compare.Result
}

type referenceDiffView struct {
	model referenceview.Model
	title string
}

const referenceDiffHelp = "[Esc/q] back  [Tab/Shift+Tab] file\n[↑/↓ PgUp/PgDn] scroll"

// Detaching a Zellij client leaves the sprint running. Only a confirmed end
// unlocks review; unexpected CLI failures leave the reference hidden.
func referenceSessionEnded(name string) bool {
	if name == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "zellij", "--session", name, "action", "list-panes", "--json")
	cmd.WaitDelay = 100 * time.Millisecond
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return false
	}
	if err != nil {
		if _, exited := err.(*exec.ExitError); !exited {
			return false
		}
	}
	text := strings.TrimSpace(string(output))
	return text == "There is no active session!" || strings.HasPrefix(text, fmt.Sprintf("Session '%s' not found.", name))
}

func (m *MainModel) currentReferenceReview() *referenceReview {
	switch m.state {
	case StateEvaluating:
		if !m.evalPending && !m.launchFailed {
			return m.evalReview
		}
	case StateResults:
		return m.resultReview
	}
	return nil
}

func (m *MainModel) openReferenceDiff() {
	review := m.currentReferenceReview()
	if review == nil || !review.options.CheckReference {
		return
	}
	m.helpPress = nil
	m.referenceDiff = &referenceDiffView{model: referenceview.NewDiff(*review.comparison), title: review.drill.Title}
	m.resizeReferenceDiff()
}

func (m *MainModel) resizeReferenceDiff() {
	if m.referenceDiff == nil {
		return
	}
	w, h := m.screenSize()
	header := m.screenHeader(joinContext(m.referenceDiff.title, "reference diff"))
	help := m.renderHelpNow(referenceDiffHelp, w)
	rows := max(0, h-lipgloss.Height(header)-lipgloss.Height(help)-3)
	next, _ := m.referenceDiff.model.Update(tea.WindowSizeMsg{Width: max(1, w-2), Height: rows + 2})
	m.referenceDiff.model = next.(referenceview.Model)
}

func (m *MainModel) updateReferenceDiff(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyMsg); ok {
		m.helpPress = nil
		switch key.String() {
		case "esc", "q", "ctrl+c":
			m.referenceDiff = nil
			return nil
		}
	}
	if mouse, ok := msg.(tea.MouseMsg); ok {
		if !m.mouseActive() || m.helpPress != nil {
			return nil
		}
		if mouse.Action == tea.MouseActionPress && mouse.Button == tea.MouseButtonLeft {
			if key, item, ok := m.helpItemAt(mouse.X, mouse.Y); ok {
				m.helpPress = &helpPressState{item: item, key: key}
				return footerPressTimer()
			}
		}
	}
	next, cmd := m.referenceDiff.model.Update(msg)
	m.referenceDiff.model = next.(referenceview.Model)
	return cmd
}

func (m *MainModel) viewReferenceDiff() string {
	w, _ := m.screenSize()
	view := m.referenceDiff
	header := m.screenHeader(joinContext(view.title, "reference diff"))
	file := AccentStyle.PaddingLeft(2).Render(ansi.Truncate(view.model.CurrentFile(), max(1, w-2), "…"))
	body := lipgloss.NewStyle().Foreground(ColorText).Render(view.model.ContentView())
	return header + "\n\n" + file + "\n" + indentLines(body) + "\n\n" + m.renderHelpNow(referenceDiffHelp, w)
}
