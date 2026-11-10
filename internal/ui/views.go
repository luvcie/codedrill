package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/storage"
)

func (m *MainModel) contentView() string {
	switch m.state {
	case StateMenu:
		return m.menuHeader() + "\n\n" + m.list.View() + "\n" + m.menuHelp()
	case StateAddDrill:
		w, _ := m.screenSize()
		help := m.renderHelpNow(formHelp, w)
		return m.screenHeader("new drill") + "\n\n" + m.form.View() + "\n\n" + help
	case StateEditDrill:
		w, _ := m.screenSize()
		help := m.renderHelpNow(formHelp, w)
		return m.screenHeader("edit drill") + "\n\n" + m.form.View() + "\n\n" + help
	case StateStats:
		header := m.screenHeader("profile · " + storage.GetLevelInfo(m.history.GetTotalXP()).Title)
		return header + "\n" + RenderStatsDashboard(m.history, m.drills, m.width)
	case StateLeaderboard:
		header := m.screenHeader(joinContext(m.activeTitle(), "hall of fame"))
		return header + "\n" + RenderLeaderboardView(*m.activeDrill, m.history, m.leaderboardRecent, m.width)
	case StateBriefing:
		return m.viewBriefing()
	case StateCountdown:
		return m.viewCountdown()
	case StateSprinting:
		return m.screenHeader(joinContext(m.activeTitle(), "sprint in progress")) + "\n\n" + indentLines("Editor session running...")
	case StateSetupError:
		return m.viewSetupError()
	case StateEvaluating:
		return m.viewEvaluating()
	case StateResults:
		return m.viewResults()
	case StateSettings:
		return m.viewSettings()
	}
	return ""
}

func (m *MainModel) wrap(s string) string {
	return lipgloss.NewStyle().Width(max(20, m.width-2)).Render(s)
}

func (m *MainModel) activeTitle() string {
	if m.activeDrill == nil {
		return ""
	}
	return m.activeDrill.Title
}

func joinContext(parts ...string) string {
	var keep []string
	for _, p := range parts {
		if p != "" {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, " · ")
}

func indentLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = "  " + line
		}
	}
	return strings.Join(lines, "\n")
}

func (m *MainModel) viewCountdown() string {
	w, h := m.screenSize()
	context := "get ready"
	if m.countdownCount <= 0 {
		context = "go"
	}
	header := m.screenHeader(joinContext(m.activeTitle(), context))
	return header + "\n" + renderCountdownPressed(m.countdownCount, w, max(1, h-lipgloss.Height(header)), m.pressedItem())
}

func (m *MainModel) viewSetupError() string {
	w, _ := m.screenSize()
	if m.prefsErr != nil {
		header := m.screenHeader(joinContext(m.activeTitle(), "preferences unavailable"))
		body := m.wrap(indentLines("The sprint has not started because this drill's preferences could not be read.\n\n" +
			DangerStyle.Render(m.prefsErr.Error()) + "\n\nFix or remove the preferences file, then retry."))
		return header + "\n\n" + body + "\n\n" + m.renderHelpNow(setupHelp, w)
	}
	header := m.screenHeader(joinContext(m.activeTitle(), "workspace setup failed"))
	body := m.wrap(indentLines("The sprint has not started.\n\n" + DangerStyle.Render(m.setupErr.Error())))
	return header + "\n\n" + body + "\n\n" + m.renderHelpNow(setupHelp, w)
}

func (m *MainModel) viewBriefing() string {
	d := m.activeDrill
	pb := "PB --"
	if best := m.history.GetPB(d.ID); best > 0 {
		grade, _ := CalculateGrade(best, d.Difficulty)
		pb = fmt.Sprintf("PB %s (%s)", storage.FormatDuration(best), grade)
	}
	context := []string{d.Title, strings.ToUpper(d.Language), strings.ToLower(d.Difficulty), pb}
	if d.AutoSubmit {
		context = append(context, "auto-submit on")
	}

	width := boxWidth(m.width, 40)
	desc := dropTitleHeading(d.Description)
	if key := fmt.Sprintf("%s\x00%d\x00%s", d.ID, width, desc); key != m.briefingKey {
		m.briefingKey, m.briefingCache = key, RenderMarkdown(desc, width)
	}

	return m.screenHeader(joinContext(context...)) + "\n\n" + indentLines(m.briefingModePanel()) + "\n\n" + m.briefingCache + "\n\n" + RenderPaceHints(d.Difficulty)
}

func (m *MainModel) viewEvaluating() string {
	title := m.activeTitle()
	elapsed := labelValue("time", storage.FormatDuration(m.accumulatedTime), timeStyle)
	if m.saveErr != nil {
		body := DangerStyle.Render("Not saved") + "\n" + "Tests: " + passedValue(m.evalPassed) + "  " + elapsed +
			"\nYour workspace is still available.\n\n" + DangerStyle.Render(m.saveErr.Error())
		return m.screenHeader(joinContext(title, "not saved")) + "\n\n" + m.wrap(indentLines(body))
	}
	if m.launchFailed {
		return m.screenHeader(joinContext(title, "did not start")) + "\n\n" + formatEvalOutput(m.evalOutput, m.width)
	}
	header := m.screenHeader(joinContext(title, "evaluation"))
	if m.evalPending {
		return header + "\n\n" + indentLines(AccentStyle.Render("Running tests...")+"  "+elapsed+
			"\n\n"+SubtextStyle.Render("Sprint actions return when evaluation finishes."))
	}

	verdict := m.passVerdict()
	body := evaluationStatus(verdict, m.evalPassed, m.runMode, m.evalRef, m.evalRefErr) + "  " + elapsed
	if verdict.Passed {
		if m.history.CurrentStreak > 0 {
			body += "\n\n" + labelValue("streak", strconv.Itoa(m.history.CurrentStreak), lipgloss.NewStyle())
		}
	} else if m.history.CurrentStreak > 0 {
		body += "\n\n" + DangerStyle.Render(fmt.Sprintf("Forfeiting ends your %d-win streak.", m.history.CurrentStreak))
	}
	return header + "\n\n" + indentLines(body) + "\n\n" + formatEvalOutput(m.evalOutput, m.width)
}

func evaluationStatus(verdict mode.Verdict, testsPassed bool, opts mode.Options, ref *compare.Result, refErr error) string {
	lines := []string{"Tests: " + passedValue(testsPassed)}
	lines = append(lines, referenceStatus(opts, ref, refErr))
	overall := "Overall: " + passedValue(verdict.Passed)
	if !verdict.Passed && verdict.Reason != "" {
		overall += " (" + verdict.Reason + ")"
	}
	return strings.Join(append(lines, overall), "\n")
}

func referenceStatus(opts mode.Options, ref *compare.Result, refErr error) string {
	if !opts.CheckReference {
		return "Reference: " + SubtextStyle.Render("not checked")
	}
	if refErr != nil {
		return "Reference: " + DangerStyle.Render("unavailable") + " (" + refErr.Error() + ")"
	}
	if ref == nil {
		return "Reference: " + DangerStyle.Render("unavailable")
	}
	percent := strconv.FormatFloat(ref.Percent(), 'f', -1, 64) + "%"
	if ref.Exact {
		return "Reference: " + SuccessStyle.Render(percent+" exact")
	}
	return "Reference: " + percent
}

func passedValue(passed bool) string {
	if passed {
		return SuccessStyle.Render("passed")
	}
	return DangerStyle.Render("failed")
}

func labelValue(label, value string, valueStyle lipgloss.Style) string {
	return SubtextStyle.Render(label) + " " + valueStyle.Render(value)
}

func (m *MainModel) viewResults() string {
	d := m.resultDrill
	var lines []string
	add := func(s string) {
		if s != "" {
			lines = append(lines, s)
		}
	}

	if m.isDailyBonus {
		add(AccentStyle.Render(fmt.Sprintf("daily cleared (+%d XP)", storage.DailyBonusXP)))
	}
	if m.isPB {
		add(SuccessStyle.Render("new personal best"))
	}
	if streak := m.history.CurrentStreak; streak > 1 {
		if m.newBestStreak {
			add(AccentStyle.Render(fmt.Sprintf("new best streak %d", streak)))
		} else {
			add(labelValue("streak", fmt.Sprintf("%d (best %d)", streak, m.history.BestStreak), lipgloss.NewStyle()))
		}
	}
	add(labelValue("final time", storage.FormatDuration(m.lastDuration), timeStyle))

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
		add(labelValue("submits", "1 (first try)", lipgloss.NewStyle()))
	case m.lastSubmits > 1:
		add(labelValue("submits", strconv.Itoa(m.lastSubmits), lipgloss.NewStyle()))
	}
	if m.earnedXP > 0 {
		lvl := m.levelInfo
		if m.leveledUp {
			add(AccentStyle.Render(fmt.Sprintf("level up %d [%s]", lvl.Level, lvl.Title)))
		}
		add(AccentStyle.Render(fmt.Sprintf("+%d XP", m.earnedXP)))
		add(SubtextStyle.Render(fmt.Sprintf("level %d [%s]  ", lvl.Level, lvl.Title)) + renderProgressBar(lvl.Progress, 10) +
			SubtextStyle.Render(fmt.Sprintf("  %d/%d XP (%d to level %d)", lvl.TotalXP, lvl.NextLevelXP, lvl.NextLevelXP-lvl.TotalXP, lvl.Level+1)))
	}

	return m.screenHeader(joinContext(d.Title, "finished")) + "\n\n" + indentLines(strings.Join(lines, "\n"))
}

func formatEvalOutput(output string, width int) string {
	output = strings.ReplaceAll(output, "\t", "    ")
	return BoxStyle.Padding(0, 1).MarginLeft(2).Width(boxWidth(width, 40)).Render(strings.TrimRight(output, "\n"))
}
