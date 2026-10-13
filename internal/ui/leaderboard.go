package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func RenderLeaderboardView(d drill.Drill, hist *storage.HistoryStore, showRecent bool, width int) string {
	var attempts []storage.Attempt
	if showRecent {
		attempts = hist.GetRecentAttempts(d.ID, 8)
	} else {
		attempts = hist.GetTopAttempts(d.ID, 8)
	}

	showDate := width >= 72 || width <= 0
	padX := 3
	if !showDate {
		padX = 1
	}
	cols := []lipgloss.Style{
		lipgloss.NewStyle().Width(8).Bold(true).Foreground(ColorSecondary),
		lipgloss.NewStyle().Width(16).Bold(true).Foreground(ColorSuccess),
		lipgloss.NewStyle().Width(9).Bold(true),
		lipgloss.NewStyle().Width(10).Foreground(ColorMuted),
		lipgloss.NewStyle().Width(16).Foreground(ColorMuted),
	}
	dividerWidth := 63
	if !showDate {
		cols, dividerWidth = cols[:4], 46
	}
	row := func(styles []lipgloss.Style, cells ...string) string {
		var out []string
		for i, col := range cols {
			out = append(out, styles[i].Inherit(col).Render(cells[i]))
		}
		return strings.Join(out, " ")
	}
	header := make([]lipgloss.Style, 5)
	for i := range header {
		header[i] = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
	}

	first := "RANK"
	if showRecent {
		first = "STATUS"
	}
	lines := []string{
		row(header, first, "TIME", "GRADE", "SUBMITS", "DATE"),
		lipgloss.NewStyle().Foreground(ColorPrimary).Render(strings.Repeat("─", dividerWidth)),
	}
	if len(attempts) == 0 {
		lines = append(lines, SubtextStyle.Italic(true).Render("No runs yet."))
	}
	for i, att := range attempts {
		styles := make([]lipgloss.Style, 5)
		label := fmt.Sprintf("#%d", i+1)
		if i == 0 {
			label = "★ #1"
		}
		grade, _ := CalculateGrade(att.Duration, d.Difficulty)
		gradeText := string(grade)
		styles[2] = lipgloss.NewStyle().Foreground(gradeColor(grade))
		if showRecent {
			label = SuccessStyle.Render("PASS")
			if !att.Passed {
				label = DangerStyle.Render("FAIL")
				styles[1] = DangerStyle
				gradeText = ""
				styles[2] = SubtextStyle
			}
		}
		if gradeText == "" {
			gradeText = "-"
		}
		lines = append(lines, row(styles, label, storage.FormatDuration(att.Duration), "["+gradeText+"]",
			strconv.Itoa(att.Attempts), att.Timestamp.Format("2006-01-02 15:04")))
	}
	tableBox := BoxStyle.Padding(1, padX).Render(strings.Join(lines, "\n"))

	statBox := ""
	if stats := hist.GetStats(d.ID); stats != nil && stats.TotalRuns > 0 {
		best := storage.FormatDuration(stats.PersonalBest)
		if grade, _ := CalculateGrade(stats.PersonalBest, d.Difficulty); grade != "" {
			best += fmt.Sprintf(" (%s)", grade)
		}
		badges := []string{
			BadgeStyle.Render(fmt.Sprintf("RUNS: %d", stats.TotalRuns)),
			BadgeStyle.Render(fmt.Sprintf("PASSED: %d (%d%%)", stats.Successful, stats.Successful*100/stats.TotalRuns)),
			BadgeStyle.Render("BEST: " + best),
		}
		if stats.CurrentStreak > 0 {
			badges = append(badges, BadgeStyle.Render(fmt.Sprintf("STREAK: %d", stats.CurrentStreak)))
		}
		statBox = strings.Join(badges, "   ")
		// Split into two rows when one row would wrap in the middle of a badge.
		if width > 0 && lipgloss.Width(statBox) > width {
			statBox = lipgloss.JoinVertical(lipgloss.Center, strings.Join(badges[:2], "   "), strings.Join(badges[2:], "   "))
		}
	}

	mode := "TOP FASTEST  [Tab] recent"
	if showRecent {
		mode = "RECENT RUNS  [Tab] fastest"
	}
	content := lipgloss.JoinVertical(lipgloss.Center,
		TitleStyle.Render("HALL OF FAME: "+d.Title),
		SubtextStyle.Render(mode),
		"",
		statBox,
		"",
		tableBox,
	)
	if width <= 0 {
		width = 80
	}
	return lipgloss.PlaceHorizontal(width, lipgloss.Center, content)
}
