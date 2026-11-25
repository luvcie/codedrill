package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestSaveFailureRecovery(t *testing.T) {
	for _, action := range []string{"retry", "discard", "edit"} {
		t.Run(action, func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.state = StateEvaluating
			m.evalPassed = true
			m.accumulatedTime = 5 * time.Second
			m.session.Submits = 2
			m.evalOutput = "original test output"
			m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
			path, err := storage.GetHistoryPath()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0755); err != nil {
				t.Fatal(err)
			}
			session := m.session
			pressFinish(t, m)
			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if m.state != StateEvaluating || m.session != session || len(m.history.Attempts) != 0 || m.history.GetTotalXP() != 0 {
				t.Fatal("failed save advanced, removed workspace, or awarded progress")
			}
			if _, err := os.Stat(session.Workspace); err != nil {
				t.Fatal("failed save removed workspace")
			}
			if !strings.Contains(m.View(), "not saved") || !strings.Contains(m.View(), "retry") || m.evalOutput != "original test output" {
				t.Fatal("missing retry feedback or lost evaluation output")
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			switch action {
			case "retry":
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				h, err := storage.LoadHistory()
				if err != nil || m.state != StateResults || len(h.Attempts) != 1 {
					t.Fatalf("retry failed: state %v, error %v", m.state, err)
				}
				if h.Attempts[0].Duration != 5*time.Second || h.Attempts[0].Attempts != 2 || !h.Attempts[0].Passed {
					t.Fatal("retry changed the result")
				}
			case "discard":
				m.Update(tea.KeyMsg{Type: tea.KeyEsc})
				if m.state != StateMenu || m.session != nil || len(m.history.Attempts) != 0 {
					t.Fatal("discard recorded or retained the attempt")
				}
			case "edit":
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
				if m.state != StateSprinting || m.session != session || cmd == nil {
					t.Fatal("cannot resume editor after failed save")
				}
			}
		})
	}
}
