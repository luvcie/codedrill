package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/luvcie/codedrill/internal/storage"
)

func (m *MainModel) contentView() string {
	switch m.state {
	case StateMenu:
		return m.menuHeader() + "\n\n" + m.list.View() + "\n" + m.menuHelp()
	case StateAddDrill:
		return TitleStyle.Render("CREATE NEW DRILL") + "\n\n" + m.form.View()
	case StateStats:
		return RenderStatsDashboard(m.history, m.drills, m.width)
	case StateLeaderboard:
		return RenderLeaderboardView(*m.activeDrill, m.history, m.leaderboardRecent, m.width)
	case StateBriefing:
		return m.viewBriefing()
	case StateCountdown:
		return renderCountdown(m.countdownCount, m.width, m.height)
	case StateSprinting:
		return "\n\n\t" + TitleStyle.Render("SPRINT IN PROGRESS") + "\n\tEditor session running..."
	case StateSetupError:
		return m.wrap(fmt.Sprintf("\n%s\n\nThe sprint has not started.\n\n%s\n\n%s\n%s",
			TitleStyle.Render("WORKSPACE SETUP FAILED"),
			DangerStyle.Render(m.setupErr.Error()),
			TimerStyle.Render("[Enter] retry setup"),
			SubtextStyle.Render("[Esc/q] back to the briefing"),
		))
	case StateEvaluating:
		return m.viewEvaluating()
	case StateResults:
		return m.viewResults()
	}
	return ""
}

func (m *MainModel) wrap(s string) string {
	return lipgloss.NewStyle().Width(max(20, m.width-2)).Render(s)
}

func (m *MainModel) viewBriefing() string {
	d := m.activeDrill
	top := strings.ToUpper(d.Language) + " · PB: None yet"
	if pb := m.history.GetPB(d.ID); pb > 0 {
		grade, _ := CalculateGrade(pb, d.Difficulty)
		top = fmt.Sprintf("%s · PB: %s (%s-RANK)", strings.ToUpper(d.Language), storage.FormatDuration(pb), grade)
	}
	top += fmt.Sprintf(" · LVL %d", storage.GetLevelInfo(m.history.GetTotalXP()).Level)
	if d.AutoSubmit {
		top += " · auto-submit on"
	}

	// Rendering markdown is slow, and the view is redrawn on every key.
	width := boxWidth(m.width, 40)
	if key := fmt.Sprintf("%s\x00%d\x00%s", d.ID, width, d.Description); key != m.briefingKey {
		m.briefingKey, m.briefingCache = key, RenderMarkdown(d.Description, width)
	}

	return fmt.Sprintf("\n%s\n%s\n\n%s\n\n%s",
		TitleStyle.Render("DRILL: "+d.Title),
		SubtextStyle.Render(top),
		m.briefingCache,
		RenderPaceHints(d.Difficulty),
	)
}

func (m *MainModel) viewEvaluating() string {
	elapsed := TimerStyle.Render(storage.FormatDuration(m.accumulatedTime))
	if m.saveErr != nil {
		status := "Tests failed"
		if m.evalPassed {
			status = "Tests passed"
		}
		return m.wrap(fmt.Sprintf("\n%s\n\n%s | Time: %s\nYour workspace is still available.\n\n%s",
			TitleStyle.Render("SPRINT NOT SAVED"), status, storage.FormatDuration(m.accumulatedTime),
			DangerStyle.Render(m.saveErr.Error()),
		))
	}
	if m.launchFailed {
		return m.wrap(fmt.Sprintf("\n%s\n\n%s", TitleStyle.Render("SPRINT DID NOT START"), formatEvalOutput(m.evalOutput, m.width)))
	}
	header := TitleStyle.Render("SPRINT EVALUATION")
	if m.evalPending {
		return fmt.Sprintf("\n%s\n\n%s  Time: %s\n\n%s", header, TimerStyle.Render("Running tests..."), elapsed,
			SubtextStyle.Render("Sprint actions return when the tests finish."))
	}

	status := DangerStyle.Render("TESTS FAILED")
	notice := ""
	if m.evalPassed {
		status = SuccessStyle.Render("TESTS PASSED")
		if m.history.CurrentStreak > 0 {
			notice = AccentStyle.Render(fmt.Sprintf("Streak: %d", m.history.CurrentStreak)) + "\n\n"
		}
	} else if m.history.CurrentStreak > 0 {
		notice = DangerStyle.Render(fmt.Sprintf("Forfeiting ends your %d-win streak.", m.history.CurrentStreak)) + "\n\n"
	}
	return fmt.Sprintf("\n%s\n\n%s  Time: %s\n\n%s%s", header, status, elapsed, notice, formatEvalOutput(m.evalOutput, m.width))
}

func (m *MainModel) viewResults() string {
	d := m.resultDrill
	var lines []string
	add := func(s string) {
		if s != "" {
			lines = append(lines, s)
		}
	}

	add(TitleStyle.Render("SPRINT FINISHED"))
	if m.isDailyBonus {
		add(AccentStyle.Render(fmt.Sprintf("Daily cleared (+%d XP)", storage.DailyBonusXP)))
	}
	if m.isPB {
		add(SuccessStyle.Render("NEW PERSONAL BEST"))
	}
	if streak := m.history.CurrentStreak; streak > 1 {
		if m.newBestStreak {
			add(AccentStyle.Render(fmt.Sprintf("New best streak: %d", streak)))
		} else {
			add(AccentStyle.Render(fmt.Sprintf("Streak: %d (best %d)", streak, m.history.BestStreak)))
		}
	}
	add(BadgeStyle.Render("FINAL TIME") + ": " + SuccessStyle.Render(storage.FormatDuration(m.lastDuration)))

	grade, title := CalculateGrade(m.lastDuration, d.Difficulty)
	if banner := RenderGradeBanner(grade, title); banner != "" {
		lines = append(lines, "", banner, "")
	}
	switch {
	case m.isPB && m.pbDelta > 0:
		add(SuccessStyle.Render(fmt.Sprintf("-%s faster than your previous PB", storage.FormatDuration(m.pbDelta))))
	case !m.isPB:
		if pb := m.history.GetPB(d.ID); pb > 0 && m.lastDuration > pb {
			add(DangerStyle.Render(fmt.Sprintf("+%s behind PB (%s)", storage.FormatDuration(m.lastDuration-pb), storage.FormatDuration(pb))))
		}
	}
	if goal, top := NextPaceGoal(m.lastDuration, d.Difficulty); top {
		add(SuccessStyle.Render(goal))
	} else {
		add(lipgloss.NewStyle().Foreground(ColorSecondary).Render(goal))
	}
	switch {
	case m.lastSubmits == 1:
		add(AccentStyle.Render("Submits: 1 (first try)"))
	case m.lastSubmits > 1:
		add(SubtextStyle.Render(fmt.Sprintf("Submits: %d", m.lastSubmits)))
	}
	if m.earnedXP > 0 {
		lvl := m.levelInfo
		if m.leveledUp {
			add(AccentStyle.Render(fmt.Sprintf("Level up: %d [%s]", lvl.Level, lvl.Title)))
		}
		add(BadgeStyle.Render(fmt.Sprintf("+%d XP", m.earnedXP)) + "  " + SubtextStyle.Render(fmt.Sprintf(
			"Lv %d [%s]  %s  %d/%d XP (%d to Lv %d)",
			lvl.Level, lvl.Title, renderProgressBar(lvl.Progress, 12), lvl.TotalXP, lvl.NextLevelXP, lvl.NextLevelXP-lvl.TotalXP, lvl.Level+1)))
	}

	return "\n" + BoxStyle.Padding(0, 1).Width(boxWidth(m.width, 50)).Render(strings.Join(lines, "\n"))
}

// formatEvalOutput boxes test output. Tabs become spaces because a raw tab
// moves the box's right border after lipgloss has measured the line.
func formatEvalOutput(output string, width int) string {
	output = strings.ReplaceAll(output, "\t", "    ")
	return BoxStyle.Padding(0, 1).Width(boxWidth(width, 40)).Render(strings.TrimRight(output, "\n"))
}
