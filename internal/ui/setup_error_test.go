package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestWorkspaceSetupErrorRecovery(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	missing := filepath.Join(root, "missing")
	t.Setenv("TMPDIR", missing)
	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	m.state = StateCountdown
	m.countdownCount = 0
	_, cmd := m.Update(countdownTickMsg{})
	if cmd != nil || m.state != StateSetupError || m.session != nil || len(m.history.Attempts) != 0 {
		t.Fatal("failed setup launched a workspace or did not leave countdown")
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 120, Height: 40}, {Width: 80, Height: 24}, {Width: 60, Height: 20}} {
		m.Update(size)
		if view := m.View(); !strings.Contains(view, "workspace setup failed") || !strings.Contains(view, "retry setup") {
			t.Fatal("missing setup feedback")
		}
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateSetupError {
		t.Fatal("retry ignored the remaining fault")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateBriefing || m.setupErr != nil {
		t.Fatal("escape did not return to briefing")
	}
	m.launchSprintSession()
	if err := os.Mkdir(missing, 0755); err != nil {
		t.Fatal(err)
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || m.state != StateSprinting || m.session == nil || m.setupErr != nil {
		t.Fatal("could not retry after restoring the temp directory")
	}
	defer m.session.Cleanup()
}
