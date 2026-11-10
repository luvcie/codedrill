package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/storage"
)

func countdownTick(id uint64, d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg { return countdownTickMsg{id: id} })
}

func (m *MainModel) triggerSprintStart() tea.Cmd {
	if m.activeDrill == nil {
		return nil
	}
	// Preferences must still be readable at sprint start. A briefing that
	// already holds this drill's selections takes precedence over disk state.
	opts, err := mode.Load(m.activeDrill.ID)
	if err != nil {
		m.setupErr = nil
		m.prefsErr = err
		m.state = StateSetupError
		return nil
	}
	m.prefsErr = nil
	if m.briefingModeID != m.activeDrill.ID {
		m.setBriefingOptions(m.activeDrill, opts)
	}
	m.runMode = m.briefingMode.Normalize()
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
	sess, err := runner.NewSessionWithOptions(*m.activeDrill, m.history.GetPB(m.activeDrill.ID), logoRGB(ColorCroc), runner.Options{
		ReferenceOptions: &m.runMode,
	})
	if err != nil {
		m.setupErr = err
		m.state = StateSetupError
		return nil
	}
	m.setupErr = nil
	m.session = sess
	m.resultReview = nil
	m.accumulatedTime = 0
	return m.startOrResumeSprint()
}

func (m *MainModel) updateSetupError(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "enter", " ":
		if m.prefsErr != nil {
			m.openBriefing(m.activeDrill)
			return nil
		}
		return m.launchSprintSession()
	case "esc", "q":
		m.setupErr = nil
		m.prefsErr = nil
		m.state = StateBriefing
	}
	return nil
}

func (m *MainModel) startOrResumeSprint() tea.Cmd {
	m.evalReview = nil
	m.referenceDiff = nil
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
	cmd := m.session.GetSprintCmd()
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return sprintFinishedMsg{err: err, zellij: cmd.Args[0] == "zellij"}
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
	m.evalRef = nil
	m.evalRefErr = nil
	m.evalReview = nil
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
	opts := m.runMode
	reviewable := msg.err == nil
	zellij := msg.zellij
	var d drill.Drill
	hasDrill := m.activeDrill != nil
	if hasDrill {
		d = *m.activeDrill
	}
	return func() tea.Msg {
		passed, output := session.RunTests()
		msg := evalFinishedMsg{
			id: id, session: session, passed: passed, output: output,
			options: opts, drill: d, reviewable: reviewable,
		}
		if opts.CheckReference && hasDrill {
			ref, err := compare.Compare(d, session.Workspace)
			msg.referenceErr = err
			if err == nil {
				msg.reference = &ref
				msg.reviewable = reviewable && (!zellij || referenceSessionEnded(session.SessionName))
			}
		}
		return msg
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
	m.evalRef = msg.reference
	m.evalRefErr = msg.referenceErr
	if msg.reviewable && msg.options.CheckReference && msg.referenceErr == nil && msg.reference != nil && len(msg.reference.Files) > 0 {
		m.evalReview = &referenceReview{options: msg.options, drill: msg.drill, comparison: msg.reference}
	}
}

// passVerdict applies the run's snapshotted gates to the separate test and
// reference results.
func (m *MainModel) passVerdict() mode.Verdict {
	return mode.Evaluate(m.runMode, m.evalPassed, m.evalRef, m.evalRefErr)
}

func (m *MainModel) endSession() {
	m.Cleanup()
	m.evalReview = nil
	m.referenceDiff = nil
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
	verdict := m.passVerdict()

	switch {
	case k == "v":
		m.openReferenceDiff()
	case k == "e":
		return m.startOrResumeSprint()
	case m.launchFailed && (finish || k == "q" || k == "esc"):
		m.endSession()
		m.state = StateBriefing
	case finish && verdict.Passed:
		return m.recordPass()
	case finish && m.saveErr != nil:
		// Retry saving the failed run.
		m.recordForfeit()
	case finish:
		return m.startOrResumeSprint()
	case (k == "q" || k == "esc") && !verdict.Passed && m.saveErr == nil:
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
	m.resultReview = m.evalReview

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
	m.evalReview = nil
	m.resultReview = nil
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
	case "v":
		m.openReferenceDiff()
	case "r":
		d := m.resultDrill
		m.activeDrill = &d
		if m.briefingModeID != d.ID {
			m.setBriefingOptions(&d, m.runMode)
		}
		return m.triggerSprintStart()
	case "n", "p":
		delta := 1
		if key.String() == "p" {
			delta = -1
		}
		if d := m.stepDrill(m.resultDrill, delta); d != nil {
			m.activeDrill = d
			if !m.loadBriefingOptions(d) {
				return nil
			}
			return m.triggerSprintStart()
		}
	case "b":
		d := m.resultDrill
		m.activeDrill = &d
		m.setBriefingOptions(&d, m.runMode)
		m.previousState = StateResults
		m.state = StateBriefing
	case "l":
		d := m.resultDrill
		m.activeDrill = &d
		m.openLeaderboard(StateResults)
	case "q", "esc", "enter":
		m.resultReview = nil
		m.state = StateMenu
	}
	return nil
}
