package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func RenderStatsDashboard(hist *storage.HistoryStore, drills []drill.Drill, width int) string {
	ids := make([]string, len(drills))
	for i, d := range drills {
		ids[i] = d.ID
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 2).
		Align(lipgloss.Center)
	card := func(label string, style lipgloss.Style, value string) string {
		return cardStyle.Render(SubtextStyle.Render(label) + "\n" + style.Render(value))
	}

	runsToday, passedToday := hist.RunsToday()
	today := fmt.Sprintf("%d (%d passed)", runsToday, passedToday)
	if dayStreak := hist.GetDailyStreak(); dayStreak > 1 {
		today = fmt.Sprintf("%d (%d-day streak)", runsToday, dayStreak)
	}

	cards := []string{
		card("TOTAL SPRINTS", AccentStyle, strconv.Itoa(hist.GetTotalRuns())),
		card("WIN RATE", SuccessStyle, fmt.Sprintf("%d%% (%d)", hist.GetOverallWinRate(), hist.GetPassedRuns())),
		card("WIN STREAK", AccentStyle, fmt.Sprintf("%d (best %d)", hist.CurrentStreak, hist.BestStreak)),
		card("PRACTICE TIME", AccentStyle, storage.FormatDuration(hist.GetTotalPracticeTime())),
		card("CLEARED", SuccessStyle, fmt.Sprintf("%d / %d", hist.ClearedCount(ids), len(drills))),
		card("TODAY", AccentStyle, today),
	}
	perRow := 3
	switch {
	case width >= 135:
		perRow = 6
	case width > 0 && width < 65:
		perRow = 2
	}
	var rows []string
	for i := 0; i < len(cards); i += perRow {
		var parts []string
		for j, c := range cards[i:min(i+perRow, len(cards))] {
			if j > 0 {
				parts = append(parts, "  ")
			}
			parts = append(parts, c)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, parts...))
	}
	cardsBlock := lipgloss.JoinVertical(lipgloss.Center, joinWithGaps(rows)...)

	showDate := width >= 84 || width <= 0
	cols := []lipgloss.Style{
		lipgloss.NewStyle().Width(18).Bold(true).Foreground(ColorSecondary),
		lipgloss.NewStyle().Width(8).Foreground(ColorMuted),
		lipgloss.NewStyle().Width(18).Bold(true).Foreground(ColorSuccess),
		lipgloss.NewStyle().Width(12).Foreground(ColorMuted),
		lipgloss.NewStyle().Width(16).Foreground(ColorMuted),
	}
	if !showDate {
		cols = cols[:4]
	}
	row := func(header bool, cells ...string) string {
		var out []string
		for i, col := range cols {
			if header {
				col = col.Bold(true).Foreground(ColorSecondary)
			}
			out = append(out, col.Render(cells[i]))
		}
		return strings.Join(out, " ")
	}
	dividerWidth := 56
	if showDate {
		dividerWidth = 74
	}

	lines := []string{
		row(true, "DRILL", "LANG", "BEST TIME", "SPRINTS", "LAST RUN"),
		lipgloss.NewStyle().Foreground(ColorPrimary).Render(strings.Repeat("─", dividerWidth)),
	}
	for _, d := range drills {
		st := hist.GetStats(d.ID)
		if st == nil || st.TotalRuns == 0 {
			continue
		}
		best := "--"
		if st.PersonalBest > 0 {
			grade, _ := CalculateGrade(st.PersonalBest, d.Difficulty)
			best = fmt.Sprintf("%s [%s]", storage.FormatDuration(st.PersonalBest), grade)
		}
		runs := strconv.Itoa(st.TotalRuns)
		if st.CurrentStreak > 1 {
			runs = fmt.Sprintf("%d (%d in a row)", st.TotalRuns, st.CurrentStreak)
		}
		last := "--"
		if !st.LastAttempt.IsZero() {
			last = st.LastAttempt.Format("2006-01-02 15:04")
		}
		lines = append(lines, row(false, ansi.Truncate(d.Title, 17, ".."), strings.ToUpper(d.Language), best, runs, last))
	}
	if len(lines) == 2 {
		lines = append(lines, SubtextStyle.Render("No runs yet."))
	}
	tableBox := BoxStyle.Render(strings.Join(lines, "\n"))

	lvl := storage.GetLevelInfo(hist.GetTotalXP())
	levelBadge := BadgeStyle.Render(fmt.Sprintf("LEVEL %d", lvl.Level))
	titleBadge := AccentStyle.Render("[" + lvl.Title + "]")
	progress := renderProgressBar(lvl.Progress, 14)
	xpSummary := SubtextStyle.Render(fmt.Sprintf("%d / %d XP (%d to Lv %d)", lvl.TotalXP, lvl.NextLevelXP, lvl.NextLevelXP-lvl.TotalXP, lvl.Level+1))
	levelBanner := fmt.Sprintf("%s %s   %s   %s", levelBadge, titleBadge, progress, xpSummary)
	if width > 0 && width < 75 {
		levelBanner = lipgloss.JoinVertical(lipgloss.Center, levelBadge+" "+titleBadge, progress+"   "+xpSummary)
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		TitleStyle.Render("SPEEDRUNNER PROFILE"),
		levelBanner,
		"",
		cardsBlock,
		"",
		tableBox,
	)
	if width <= 0 {
		width = 80
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, content)
}

// joinWithGaps puts an empty line between blocks.
func joinWithGaps(blocks []string) []string {
	var out []string
	for i, b := range blocks {
		if i > 0 {
			out = append(out, "")
		}
		out = append(out, b)
	}
	return out
}
