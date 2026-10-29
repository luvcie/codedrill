package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

const (
	statsTitleCap   = 24
	statsRunsCap    = 18
	statsTitleFloor = 8
	statsGap        = 2
	statGridGap     = 4
	statsIndent     = "  "
)

type statCell struct {
	label string
	value string
}

func RenderStatsDashboard(hist *storage.HistoryStore, drills []drill.Drill, width int) string {
	ids := make([]string, len(drills))
	for i, d := range drills {
		ids[i] = d.ID
	}

	runsToday, passedToday := hist.RunsToday()
	today := fmt.Sprintf("%d, %d passed", runsToday, passedToday)
	if dayStreak := hist.GetDailyStreak(); dayStreak > 1 {
		today = fmt.Sprintf("%d, %d-day streak", runsToday, dayStreak)
	}

	cells := []statCell{
		{"total sprints", strconv.Itoa(hist.GetTotalRuns())},
		{"win rate", fmt.Sprintf("%d%% (%d)", hist.GetOverallWinRate(), hist.GetPassedRuns())},
		{"win streak", fmt.Sprintf("%d, best %d", hist.CurrentStreak, hist.BestStreak)},
		{"practice time", storage.FormatDuration(hist.GetTotalPracticeTime())},
		{"cleared", fmt.Sprintf("%d / %d", hist.ClearedCount(ids), len(drills))},
		{"today", today},
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, renderLevelLines(hist, width)...)
	lines = append(lines, "")
	lines = append(lines, renderStatGrid(cells, width)...)
	lines = append(lines, "")
	lines = append(lines, renderDrillTable(hist, drills, width)...)

	for i, line := range lines {
		if line != "" {
			lines[i] = statsIndent + line
		}
		if width > 0 {
			lines[i] = ansi.Truncate(lines[i], width, "")
		}
	}
	return strings.Join(lines, "\n")
}

func renderLevelLines(hist *storage.HistoryStore, width int) []string {
	lvl := storage.GetLevelInfo(hist.GetTotalXP())
	level := lipgloss.NewStyle().Bold(true).Foreground(ColorText).Render(fmt.Sprintf("level %d", lvl.Level))
	title := AccentStyle.Render(lvl.Title)
	bar := renderProgressBar(lvl.Progress, 10)
	xpCounts := fmt.Sprintf("%d / %d XP", lvl.TotalXP, lvl.NextLevelXP)
	xpLeft := fmt.Sprintf("%d to level %d", lvl.NextLevelXP-lvl.TotalXP, lvl.Level+1)
	xp := SubtextStyle.Render(xpCounts + " · " + xpLeft)
	pieces := []string{level + "  " + title, bar, xp}

	if width <= 0 {
		return []string{strings.Join(pieces, "  ")}
	}
	limit := width - len(statsIndent)
	if lipgloss.Width(xp) > limit {
		pieces = []string{level + "  " + title, bar, SubtextStyle.Render(xpCounts), SubtextStyle.Render(xpLeft)}
	}

	var lines []string
	cur := pieces[0]
	for _, p := range pieces[1:] {
		if lipgloss.Width(cur+"  "+p) <= limit {
			cur += "  " + p
			continue
		}
		lines = append(lines, cur)
		cur = p
	}
	return append(lines, cur)
}

func renderStatGrid(cells []statCell, width int) []string {
	budget := 0
	if width > 0 {
		budget = width - len(statsIndent)
	}
	perRow := 3
	for perRow > 1 && budget > 0 && statGridWidth(cells, perRow) > budget {
		perRow--
	}

	labW, valW := statGridColumns(cells, perRow)
	var out []string
	for start := 0; start < len(cells); start += perRow {
		var parts []string
		for j, c := range cells[start:min(start+perRow, len(cells))] {
			cell := SubtextStyle.Render(c.label) + strings.Repeat(" ", labW[j]-lipgloss.Width(c.label)) +
				" " + lipgloss.NewStyle().Bold(true).Render(c.value)
			if j < len(labW)-1 {
				cell += strings.Repeat(" ", valW[j]-lipgloss.Width(c.value)+statGridGap)
			}
			parts = append(parts, cell)
		}
		out = append(out, strings.TrimRight(strings.Join(parts, ""), " "))
	}
	return out
}

func statGridColumns(cells []statCell, perRow int) (labW, valW []int) {
	labW = make([]int, perRow)
	valW = make([]int, perRow)
	for i, c := range cells {
		j := i % perRow
		labW[j] = max(labW[j], lipgloss.Width(c.label))
		valW[j] = max(valW[j], lipgloss.Width(c.value))
	}
	return labW, valW
}

func statGridWidth(cells []statCell, perRow int) int {
	labW, valW := statGridColumns(cells, perRow)
	total := 0
	for j := range labW {
		total += labW[j] + 1 + valW[j]
	}
	return total + (perRow-1)*statGridGap
}

func renderDrillTable(hist *storage.HistoryStore, drills []drill.Drill, width int) []string {
	var entries [][]string
	for _, d := range drills {
		st := hist.GetStats(d.ID)
		if st == nil || st.TotalRuns == 0 {
			continue
		}
		best := SubtextStyle.Render("--")
		if st.PersonalBest > 0 {
			grade, _ := CalculateGrade(st.PersonalBest, d.Difficulty)
			gradeText := lipgloss.NewStyle().Foreground(gradeColor(grade)).Render(string(grade))
			best = lipgloss.NewStyle().Bold(true).Render(storage.FormatDuration(st.PersonalBest)) + " " + gradeText
		}
		runs := strconv.Itoa(st.TotalRuns)
		if st.CurrentStreak > 1 {
			runs = fmt.Sprintf("%d, %d in a row", st.TotalRuns, st.CurrentStreak)
		}
		last := "--"
		if !st.LastAttempt.IsZero() {
			last = st.LastAttempt.Format("2006-01-02 15:04")
		}
		entries = append(entries, []string{
			ansi.Truncate(d.Title, statsTitleCap, "…"),
			SubtextStyle.Render(strings.ToUpper(d.Language)),
			best,
			SubtextStyle.Render(ansi.Truncate(runs, statsRunsCap, "…")),
			SubtextStyle.Render(last),
		})
	}

	headers := []string{"drill", "lang", "best", "sprints", "last run"}
	content := make([]int, len(headers))
	for i, h := range headers {
		content[i] = lipgloss.Width(h)
	}
	for _, cells := range entries {
		for i, cell := range cells {
			content[i] = max(content[i], lipgloss.Width(cell))
		}
	}
	if width > 0 && width < 84 {
		content[4] = 0
	}

	avail := math.MaxInt
	if width > 0 {
		avail = width - len(statsIndent)
	}
	widths := fitTable(content, 0, statsTitleFloor, statsGap, avail, []int{3, 1})
	tableWidth := tableRowWidth(widths, statsGap)

	headerCells := make([]string, len(headers))
	for i, h := range headers {
		headerCells[i] = SubtextStyle.Render(h)
	}
	lines := []string{
		tableRow(headerCells, widths, statsGap),
		lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("─", tableWidth)),
	}
	if len(entries) == 0 {
		return append(lines, SubtextStyle.Render("No runs yet."))
	}
	for _, cells := range entries {
		lines = append(lines, tableRow(cells, widths, statsGap))
	}
	return lines
}

func fitTable(content []int, flex, floor, gap, avail int, drop []int) []int {
	kept := make([]bool, len(content))
	for i, w := range content {
		kept[i] = w > 0
	}
	room := func() int {
		count, fixed := 0, 0
		for i, w := range content {
			if !kept[i] {
				continue
			}
			count++
			if i != flex {
				fixed += w
			}
		}
		return avail - fixed - max(0, count-1)*gap
	}
	for room() < min(content[flex], floor) {
		dropped := false
		for _, i := range drop {
			if kept[i] {
				kept[i] = false
				dropped = true
				break
			}
		}
		if !dropped {
			break
		}
	}

	widths := make([]int, len(content))
	for i, w := range content {
		if kept[i] {
			widths[i] = w
		}
	}
	if kept[flex] {
		widths[flex] = max(1, min(content[flex], room()))
	}
	return widths
}

func tableRowWidth(widths []int, gap int) int {
	total, shown := 0, 0
	for _, w := range widths {
		if w > 0 {
			total += w
			shown++
		}
	}
	return total + max(0, shown-1)*gap
}

func tableRow(cells []string, widths []int, gap int) string {
	last := -1
	for i, w := range widths {
		if w > 0 {
			last = i
		}
	}
	var parts []string
	for i, cell := range cells {
		if widths[i] == 0 {
			continue
		}
		cell = ansi.Truncate(cell, widths[i], "…")
		if i != last {
			cell += strings.Repeat(" ", widths[i]-lipgloss.Width(cell))
		}
		parts = append(parts, cell)
	}
	return strings.Join(parts, strings.Repeat(" ", gap))
}
