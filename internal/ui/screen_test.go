package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestScreensFitAndKeepActionsVisible(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 60, Height: 20}, {Width: 80, Height: 24}, {Width: 120, Height: 40}} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.Update(size)
			m.lastDuration = 5 * time.Second
			for _, state := range []AppState{StateMenu, StateBriefing, StateEvaluating, StateResults} {
				m.state = state
				view := m.View()
				if lipgloss.Width(view) > size.Width || lipgloss.Height(view) > size.Height {
					t.Fatalf("state %v overflows %dx%d: %dx%d", state, size.Width, size.Height, lipgloss.Width(view), lipgloss.Height(view))
				}
				if !strings.Contains(view, "[Enter") {
					t.Fatalf("state %v hides primary action", state)
				}
			}
		})
	}
}

func TestLongEvaluationScrollsWithoutLosingActions(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m.state = StateEvaluating
	m.evalOutput = strings.Repeat("compiler diagnostic\n", 80) + "FINAL DIAGNOSTIC"
	if strings.Contains(m.View(), "FINAL DIAGNOSTIC") || !strings.Contains(m.View(), "scroll") {
		t.Fatal("long output is not presented as scrollable")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if !strings.Contains(m.View(), "FINAL DIAGNOSTIC") || !strings.Contains(m.View(), "[Enter/e]") {
		t.Fatal("last diagnostic or primary action is unreachable")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if !strings.Contains(m.View(), "SPRINT EVALUATION") {
		t.Fatal("home did not restore the heading")
	}
}

func TestEvaluationBorderStartsAtLeftWithStreak(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.state = StateEvaluating
	m.evalPassed = true
	m.history.CurrentStreak = 2
	m.evalOutput = "ok\tsolution\t0.01s"
	for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		if strings.Contains(line, "╭") && !strings.HasPrefix(line, "╭") {
			t.Fatalf("notice padding shifted the evaluation border: %q", line)
		}
	}
}
