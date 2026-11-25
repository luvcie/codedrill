package ui

import (
	"fmt"
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
const hallRunLimit = 8

const (
	allHistoryCategory     storage.RecordMode = "all"
	trueAllHistoryCategory storage.RecordMode = "true-all"
)

func (m *MainModel) leaderboardCategories() []storage.RecordMode {
	categories := append([]storage.RecordMode{allHistoryCategory}, m.history.RecordModes(m.activeDrill.ID)...)
	return append(categories, trueAllHistoryCategory)
}

func RenderLeaderboardView(d drill.Drill, hist *storage.HistoryStore, showRecent bool, category storage.RecordMode, width int) string {
	if width <= 0 {
		width = 80
	}
	all := category == allHistoryCategory || category == trueAllHistoryCategory
	categories := []storage.RecordMode{category}
	label := mode.RecordLabel(category)
	limit := hallRunLimit
	fastest := fmt.Sprintf("fastest %d runs  [Tab] recent", limit)
	recent := "recent runs  [Tab] fastest"
	if all {
		categories = hist.RecordModes(d.ID)
		label = "All"
	}
	if category == trueAllHistoryCategory {
		label = "True All"
		limit = 0
		fastest = "all runs by time  [Tab] recent"
		recent = "all recent runs  [Tab] fastest"
	}
	viewMode := fastest
	if showRecent {
		viewMode = recent
	}
	categoryLine := label + "   [←/→] category"
	lines := []string{"", "  " + ansi.Truncate(categoryLine, max(0, width-2), ""), renderHelp(viewMode, width, ""), ""}

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
			attempts = hist.GetRecentPasses(d.ID, category, limit)
		} else {
			attempts = hist.GetTopAttempts(d.ID, category, limit)
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
	lines = append(lines, "", RenderPaceHints(d, width))
	return strings.Join(lines, "\n")
}

func hallStats(hist *storage.HistoryStore, d drill.Drill, category storage.RecordMode, width int) []string {
	stats := hist.GetModeStats(d.ID, category)
	if stats == nil || stats.Successful == 0 {
		return nil
	}
	items := []string{
		labelValue("runs", strconv.Itoa(stats.Successful), hallValueStyle),
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
	head := []string{"rank", "time", "grade", "date"}
	flex := 0
	if showRecent {
		head[0] = ""
		flex = 1
	}
	rows := [][]string{}
	for i, att := range attempts {
		grade, _ := CalculateGrade(att.Duration, d)
		gradeText := string(grade)
		rank := ""
		if !showRecent {
			rank = fmt.Sprintf("#%d", i+1)
			if i == 0 {
				rank = AccentStyle.Render("★ #1")
			}
		}
		timeCell := storage.FormatDuration(att.Duration)
		gradeStyle := lipgloss.NewStyle().Bold(true).Foreground(gradeColor(grade))
		if att.NoLint {
			timeCell += " " + AccentStyle.Render("no lint")
		}
		if gradeText == "" {
			gradeText = "-"
		}
		rows = append(rows, []string{
			rank,
			timeCell,
			gradeStyle.Render("[" + gradeText + "]"),
			SubtextStyle.Render(att.Timestamp.Format("2006-01-02 15:04")),
		})
	}

	content := make([]int, len(head))
	for i, h := range head {
		content[i] = lipgloss.Width(h)
		head[i] = SubtextStyle.Render(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			content[i] = max(content[i], lipgloss.Width(cell))
		}
	}
	if width < 72 {
		content[3] = 0
	}

	widths := fitTable(content, flex, content[flex], hallGap, width-2, []int{2})

	lines := []string{
		tableRow(head, widths, hallGap),
		lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", tableRowWidth(widths, hallGap))),
	}
	for _, row := range rows {
		lines = append(lines, tableRow(row, widths, hallGap))
	}
	return lines
}
