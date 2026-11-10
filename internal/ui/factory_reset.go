package ui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

var factoryResetSteps = []string{
	"Factory Reset",
	"Are you sure?",
	"Really sure?",
	"Last chance. Reset everything?",
}

type factoryResetFlow struct {
	step    int
	running bool
}

type factoryResetFinishedMsg struct {
	backup string
	err    error
}

func (m *MainModel) advanceFactoryReset() tea.Cmd {
	if m.reset == nil {
		m.reset = &factoryResetFlow{}
	}
	if m.reset.step < len(factoryResetSteps)-1 {
		m.reset.step++
		return nil
	}
	m.reset.running = true
	return func() tea.Msg {
		builtins, err := drill.LoadBuiltins()
		if err == nil {
			err = checkResetBuiltins(builtins)
		}
		if err != nil {
			return factoryResetFinishedMsg{err: err}
		}
		backup, err := storage.FactoryReset()
		return factoryResetFinishedMsg{backup: backup, err: err}
	}
}

func checkResetBuiltins(builtins []drill.Drill) error {
	for _, name := range []string{"bigint", "bsq", "life", "polyset", "vect2"} {
		if d := drill.Find(builtins, "exam05-"+name); d == nil || !d.IsPreset {
			return fmt.Errorf("nothing reset: this executable is missing the built-in Exam 05 %s drill", name)
		}
	}
	return nil
}

func (m *MainModel) finishFactoryReset(msg factoryResetFinishedMsg) tea.Cmd {
	if m.state != StateSettings || m.reset == nil || !m.reset.running {
		return nil
	}
	if msg.err != nil {
		m.reset = nil
		m.settingsErr = fmt.Errorf("factory reset: %w", msg.err)
		if msg.backup != "" {
			m.settingsErr = fmt.Errorf("%w; backup: %s", m.settingsErr, msg.backup)
		}
		return nil
	}
	fresh, err := NewMainModel()
	if err != nil {
		m.reset = nil
		m.settingsErr = fmt.Errorf("data reset; could not reload defaults: %w; backup: %s", err, msg.backup)
		return nil
	}
	w, h := m.screenSize()
	*m = *fresh
	m.width, m.height = w, h
	m.applyTheme(Themes[0])
	m.menuNotice = "Factory Reset complete. Defaults restored."
	if msg.backup != "" {
		m.menuNotice += " Backup: " + msg.backup
	}
	m.sizeList()
	return m.mouseCmd()
}
