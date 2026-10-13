package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/storage"
)

func countdownTick(id uint64, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return countdownTickMsg{id: id} })
}

func (m *MainModel) triggerSprintStart() tea.Cmd {
	cfg, _ := storage.LoadConfig()
	count := cfg.GetCountdown()
	if count <= 0 {
		return m.launchSprintSession()
	}
	m.state = StateCountdown
	m.countdownCount = count
	m.countdownID++
	return countdownTick(m.countdownID, time.Second)
}

func (m *MainModel) onCountdownTick(msg countdownTickMsg) tea.Cmd {
	// Ticks from a cancelled countdown are still in flight; ignore them.
	if m.state != StateCountdown || msg.id != m.countdownID {
		return nil
	}
	m.countdownCount--
	switch {
	case m.countdownCount == 0:
		storage.RingBell()
		return countdownTick(msg.id, 550*time.Millisecond)
	case m.countdownCount < 0:
		return m.launchSprintSession()
	}
	return countdownTick(msg.id, time.Second)
}

func (m *MainModel) updateCountdown(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "esc", "q":
		m.state = StateBriefing
	case "enter", " ":
		return m.launchSprintSession()
	}
	return nil
}

func (m *MainModel) launchSprintSession() tea.Cmd {
	sess, err := runner.NewSession(*m.activeDrill, m.history.GetPB(m.activeDrill.ID))
	if err != nil {
		m.setupErr = err
		m.state = StateSetupError
		return nil
	}
	m.setupErr = nil
	m.session = sess
	m.accumulatedTime = 0
	return m.startOrResumeSprint()
}

func (m *MainModel) updateSetupError(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "enter", " ":
		return m.launchSprintSession()
	case "esc", "q":
		m.setupErr = nil
		m.state = StateBriefing
	}
	return nil
}

func (m *MainModel) startOrResumeSprint() tea.Cmd {
	m.saveErr = nil
	m.launchFailed = false
	m.state = StateSprinting
	m.sprintStartTime = time.Now()
	if m.accumulatedTime > 0 {
		// The pace helpers read the start time from disk. If this fails, only
		// the in-editor pace display is off; the recorded time comes from here.
		start := time.Now().Add(-m.accumulatedTime).UnixMilli()
		_ = os.WriteFile(filepath.Join(m.session.Workspace, ".sprint_start"), []byte(strconv.FormatInt(start, 10)), 0644)
	}
	return tea.ExecProcess(m.session.GetSprintCmd(), func(err error) tea.Msg {
		return sprintFinishedMsg{err: err}
	})
}

func (m *MainModel) onSprintFinished(msg sprintFinishedMsg) tea.Cmd {
	if m.state != StateSprinting {
		return nil
	}
	runTime := time.Since(m.sprintStartTime)
	m.evalID++
	m.evalPassed = false
	m.evalOutput = ""
	m.evalPending = false
	m.state = StateEvaluating

	// An error straight away means the editor session never came up.
	if msg.err != nil && runTime < 800*time.Millisecond {
		m.launchFailed = true
		m.evalReadyTime = time.Now()
		m.evalOutput = fmt.Sprintf("The editor session failed to start:\n%v\n\nCheck that your editor (%s) runs on its own.", msg.err, runner.DetectEditor())
		return nil
	}

	m.accumulatedTime += runTime
	m.evalPending = true
	session, id := m.session, m.evalID
	return func() tea.Msg {
		passed, output := session.RunTests()
		return evalFinishedMsg{id: id, session: session, passed: passed, output: output}
	}
}

func (m *MainModel) onEvalFinished(msg evalFinishedMsg) {
	if m.state != StateEvaluating || !m.evalPending || msg.id != m.evalID || msg.session != m.session {
		return
	}
	m.evalPending = false
	m.evalReadyTime = time.Now()
	m.evalPassed = msg.passed
	m.evalOutput = msg.output
}

func (m *MainModel) endSession() {
	m.Cleanup()
	m.saveErr = nil
	m.launchFailed = false
}

func (m *MainModel) updateEvaluating(key tea.KeyMsg) tea.Cmd {
	if m.evalPending {
		return nil
	}
	k := key.String()
	finish := k == "enter" || k == "s" || k == " "
	// Ignore a held-down Enter from the editor landing on this screen.
	if finish && time.Since(m.evalReadyTime) < 300*time.Millisecond {
		return nil
	}

	switch {
	case k == "e":
		return m.startOrResumeSprint()
	case m.launchFailed && (finish || k == "q" || k == "esc"):
		m.endSession()
		m.state = StateBriefing
	case finish && m.evalPassed:
		return m.recordPass()
	case finish && m.saveErr != nil:
		// Retry saving the failed run.
		m.recordForfeit()
	case finish:
		return m.startOrResumeSprint()
	case (k == "q" || k == "esc") && !m.evalPassed && m.saveErr == nil:
		m.recordForfeit()
	case k == "q" || k == "esc":
		// Leaving a pass, or a run that could not be saved, records nothing.
		m.endSession()
		m.state = StateMenu
	}
	return nil
}

func (m *MainModel) submits() int {
	if m.session != nil && m.session.Submits > 0 {
		return m.session.Submits
	}
	return 1
}

func (m *MainModel) recordPass() tea.Cmd {
	d := *m.activeDrill
	submits := m.submits()
	firstDailyClear := m.isDaily(d) && !m.history.IsDailyDrillPassedToday(d.ID, time.Now())
	oldLevel := storage.GetLevelInfo(m.history.GetTotalXP()).Level
	oldBestStreak := m.history.BestStreak

	isPB, delta, err := m.history.RecordAttempt(d.ID, m.accumulatedTime, true, submits, firstDailyClear)
	if err != nil {
		m.saveErr = err
		return nil
	}

	m.resultDrill = d
	m.lastDuration = m.accumulatedTime
	m.lastSubmits = submits
	m.isPB, m.pbDelta = isPB, delta
	m.isDailyBonus = firstDailyClear
	m.newBestStreak = m.history.BestStreak > oldBestStreak && m.history.CurrentStreak > 1
	m.earnedXP = storage.AttemptXP(m.lastDuration, true, submits, firstDailyClear)
	m.levelInfo = storage.GetLevelInfo(m.history.GetTotalXP())
	m.leveledUp = m.levelInfo.Level > oldLevel

	if grade, _ := CalculateGrade(m.lastDuration, d.Difficulty); isPB || m.leveledUp || firstDailyClear || grade == GradeS {
		storage.RingChime()
	}
	m.endSession()
	m.state = StateResults
	return m.refreshPB(d)
}

// recordForfeit saves a failed run and returns to the menu. On a save error
// the evaluation screen stays up so the user can retry or discard.
func (m *MainModel) recordForfeit() {
	_, _, err := m.history.RecordAttempt(m.activeDrill.ID, m.accumulatedTime, false, m.submits(), false)
	if err != nil {
		m.saveErr = err
		return
	}
	m.endSession()
	m.state = StateMenu
}

func (m *MainModel) updateResults(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "r":
		d := m.resultDrill
		m.activeDrill = &d
		return m.triggerSprintStart()
	case "n", "p":
		delta := 1
		if key.String() == "p" {
			delta = -1
		}
		if d := m.stepDrill(m.resultDrill, delta); d != nil {
			m.activeDrill = d
			return m.triggerSprintStart()
		}
	case "b":
		d := m.resultDrill
		m.activeDrill = &d
		m.previousState = StateResults
		m.state = StateBriefing
	case "l":
		d := m.resultDrill
		m.activeDrill = &d
		m.openLeaderboard(StateResults)
	case "q", "esc", "enter":
		m.state = StateMenu
	}
	return nil
}
