package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/luvcie/codedrill/internal/runner"
)

type freeWorkFinishedMsg struct{ err error }

func (m *MainModel) openFreeWork() tea.Cmd {
	panes := 2
	m.state = StateFreeWorkSetup
	m.menuNotice = ""
	m.form = huh.NewForm(huh.NewGroup(
		huh.NewSelect[int]().Key("panes").Title("Terminal panes").
			Options(huh.NewOption("1", 1), huh.NewOption("2", 2), huh.NewOption("3", 3), huh.NewOption("4", 4)).Value(&panes),
		huh.NewInput().Key("directory").Title("Working directory (blank = home ~)").Placeholder("~/projects/my app").
			Validate(func(value string) error { _, err := runner.FreeWorkDirectory(value); return err }),
	)).WithShowHelp(false).WithTheme(formTheme()).WithKeyMap(formKeyMap()).WithWidth(max(m.width-2, 0))
	return m.form.Init()
}

func (m *MainModel) updateFreeWork(msg tea.Msg) tea.Cmd {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	switch m.form.State {
	case huh.StateAborted:
		m.form = nil
		m.state = StateMenu
		m.sizeList()
	case huh.StateCompleted:
		session, err := runner.NewFreeWorkSession(m.form.GetInt("panes"), m.form.GetString("directory"))
		m.form = nil
		if err != nil {
			m.state = StateMenu
			m.menuNotice = "Could not start Free Work: " + err.Error()
			m.sizeList()
			return nil
		}
		m.freeWork = session
		m.state = StateFreeWorking
		m.startMusic()
		return tea.ExecProcess(session.Cmd(), func(err error) tea.Msg { return freeWorkFinishedMsg{err} })
	}
	return cmd
}

func (m *MainModel) onFreeWorkFinished(msg freeWorkFinishedMsg) {
	if m.state != StateFreeWorking {
		return
	}
	m.stopMusic()
	m.freeWork.Cleanup()
	m.freeWork = nil
	m.state = StateMenu
	m.menuNotice = "Free Work closed."
	if msg.err != nil {
		m.menuNotice = "Free Work ended: " + msg.err.Error()
	}
	if m.musicPlaybackErr != nil {
		m.menuNotice += " Music unavailable: " + m.musicPlaybackErr.Error()
	}
	m.sizeList()
}
