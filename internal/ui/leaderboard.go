package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/storage"
)

var hallValueStyle = lipgloss.NewStyle().Bold(true)

const hallGap = 3

// All is a history view, never a saved practice mode.
const allHistoryCategory storage.RecordMode = "all"

func (m *MainModel) leaderboardCategories() []storage.RecordMode {
	return append([]storage.RecordMode{allHistoryCategory}, m.history.RecordModes(m.activeDrill.ID)...)
}

func RenderLeaderboardView(d drill.Drill, hist *storage.HistoryStore, showRecent bool, category storage.RecordMode, width int) string {
	if width <= 0 {
		width = 80
	}
	all := category == allHistoryCategory
	categories := []storage.RecordMode{category}
	label := mode.RecordLabel(category)
	if all {
		categories = hist.RecordModes(d.ID)
		label = "All"
	}

	viewMode := "fastest runs   [Tab] recent"
	if showRecent {
		viewMode = "recent runs   [Tab] fastest"
	}
	categoryLine := label + "   [←/→] category"
	lines := []string{"", "  " + ansi.Truncate(categoryLine, max(0, width-2), ""), "  " + styleHelpRow(viewMode, ""), ""}

	for i, category := range categories {
		if i > 0 {
			lines = append(lines, "")
		}
		if all {
			lines = append(lines, "  "+hallValueStyle.Render(ansi.Truncate(mode.RecordLabel(category), max(0, width-2), "")))
		}
		if stat := hallStats(hist, d, category, width); len(stat) > 0 {
			lines = append(lines, stat...)
			lines = append(lines, "")
		}

		var attempts []storage.Attempt
		if showRecent {
			attempts = hist.GetRecentAttempts(d.ID, category, 8)
		} else {
			attempts = hist.GetTopAttempts(d.ID, category, 8)
		}
		if len(attempts) > 0 || !all {
			for _, line := range hallTable(d, attempts, showRecent, width) {
				lines = append(lines, "  "+line)
			}
		}
		if len(attempts) == 0 {
			lines = append(lines, "  "+SubtextStyle.Render("No runs yet."))
		}
	}
	return strings.Join(lines, "\n")
}

func hallStats(hist *storage.HistoryStore, d drill.Drill, category storage.RecordMode, width int) []string {
	stats := hist.GetModeStats(d.ID, category)
	if stats == nil || stats.TotalRuns == 0 {
		return nil
	}
	items := []string{
		labelValue("runs", strconv.Itoa(stats.TotalRuns), hallValueStyle),
		labelValue("passed", fmt.Sprintf("%d (%d%%)", stats.Successful, stats.Successful*100/stats.TotalRuns), hallValueStyle),
	}
	best := labelValue("best", storage.FormatDuration(stats.PersonalBest), hallValueStyle)
	if grade, _ := CalculateGrade(stats.PersonalBest, d); grade != "" {
		best += " " + lipgloss.NewStyle().Bold(true).Foreground(gradeColor(grade)).Render("["+string(grade)+"]")
	}
	items = append(items, best)
	if stats.CurrentStreak > 0 {
		items = append(items, labelValue("streak", strconv.Itoa(stats.CurrentStreak), hallValueStyle))
	}

	const sep = "   "
	one := strings.Join(items, sep)
	if ansi.StringWidth(one)+2 <= width {
		return []string{"  " + one}
	}
	return []string{
		"  " + strings.Join(items[:2], sep),
		"  " + strings.Join(items[2:], sep),
	}
}

func hallTable(d drill.Drill, attempts []storage.Attempt, showRecent bool, width int) []string {
	head := []string{"rank", "time", "grade", "submits", "date"}
	if showRecent {
		head[0] = "status"
	}
	rows := [][]string{}
	for i, att := range attempts {
		grade, _ := CalculateGrade(att.Duration, d)
		gradeText := string(grade)
		rank := fmt.Sprintf("#%d", i+1)
		if i == 0 {
			rank = AccentStyle.Render("★ #1")
		}
		timeCell := storage.FormatDuration(att.Duration)
		gradeStyle := lipgloss.NewStyle().Bold(true).Foreground(gradeColor(grade))
		if showRecent {
			rank = SuccessStyle.Render("PASS")
			if !att.Passed {
				rank = DangerStyle.Render("FAIL")
				timeCell = DangerStyle.Render(timeCell)
				gradeText = ""
				gradeStyle = SubtextStyle
			}
		}
		if gradeText == "" {
			gradeText = "-"
		}
		rows = append(rows, []string{
			rank,
			timeCell,
			gradeStyle.Render("[" + gradeText + "]"),
			strconv.Itoa(att.Attempts),
			SubtextStyle.Render(att.Timestamp.Format("2006-01-02 15:04")),
		})
	}

	content := make([]int, len(head))
	for i, h := range head {
		content[i] = lipgloss.Width(SubtextStyle.Render(h))
	}
	for _, row := range rows {
		for i, cell := range row {
			content[i] = max(content[i], lipgloss.Width(cell))
		}
	}
	if width < 72 {
		content[4] = 0
	}

	avail := math.MaxInt
	if width > 0 {
		avail = width - len("  ")
	}
	widths := fitTable(content, 0, content[0], hallGap, avail, []int{3, 2})

	dim := make([]string, len(head))
	for i, h := range head {
		dim[i] = SubtextStyle.Render(h)
	}
	lines := []string{
		tableRow(dim, widths, hallGap),
		lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", tableRowWidth(widths, hallGap))),
	}
	for _, row := range rows {
		lines = append(lines, tableRow(row, widths, hallGap))
	}
	return lines
}
