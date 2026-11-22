package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

type Grade string

const (
	GradeS Grade = "S"
	GradeA Grade = "A"
	GradeB Grade = "B"
	GradeC Grade = "C"
	GradeD Grade = "D"
)

var gradeTitles = map[Grade]string{
	GradeS: "GODSPEED",
	GradeA: "MASTER",
	GradeB: "SKILLED",
	GradeC: "CLEARED",
	GradeD: "CLEARED",
	"SS":   "GODSPEED",
	"U":    "GODSPEED",
	"X":    "GODSPEED",
}

func CalculateGrade(duration time.Duration, d drill.Drill) (Grade, string) {
	g := Grade(drill.CalculateRank(duration, d))
	return g, gradeTitles[Grade(strings.TrimRight(string(g), "+-"))]
}

func gradeColor(g Grade) lipgloss.Color {
	switch Grade(strings.TrimRight(string(g), "+-")) {
	case GradeS, "SS", "U", "X":
		return ColorSuccess
	case GradeA:
		return ColorSecondary
	case GradeB:
		return ColorText
	}
	return ColorMuted
}

func isSpeedGrade(g Grade) bool {
	switch Grade(strings.TrimRight(string(g), "+-")) {
	case GradeS, "SS", "U", "X":
		return true
	}
	return false
}

func NextPaceGoal(duration time.Duration, d drill.Drill) (goal string, top bool) {
	if duration <= 0 {
		return "", false
	}
	targets := drill.RankTargets(d)
	for i := len(targets) - 1; i >= 0; i-- {
		if !targets[i].Reached(duration) {
			continue
		}
		if i == len(targets)-1 {
			return "Top rank reached.", true
		}
		next := targets[i+1]
		if !next.Inclusive {
			return fmt.Sprintf("next %s-RANK under %s", next.Rank, storage.FormatDuration(next.Time)), false
		}
		return fmt.Sprintf("next %s-RANK at %s (%s to cut)", next.Rank, storage.FormatDuration(next.Time), storage.FormatDuration(duration-next.Time)), false
	}
	return "", false
}

func RenderGradeBanner(g Grade, title string) string {
	if g == "" {
		return ""
	}
	text := fmt.Sprintf("%s-RANK · %s", g, title)
	if isSpeedGrade(g) {
		text = "★ " + text
	}
	return lipgloss.NewStyle().Bold(true).Foreground(gradeColor(g)).Render(text)
}

func RenderPaceHints(d drill.Drill, width int) string {
	targets := drill.RankTargets(d)
	if targets[0].Rank == "C" {
		targets = targets[1:]
		slices.Reverse(targets)
	}
	var paces []string
	for _, target := range targets {
		letter := lipgloss.NewStyle().Bold(true).Foreground(gradeColor(Grade(target.Rank))).Render(target.Rank)
		text := " clear"
		if target.Time > 0 {
			op := "<"
			if target.Inclusive {
				op = "≤"
			}
			elapsed := fmt.Sprintf("%ds", int(target.Time.Seconds()))
			if !target.Inclusive && target.Time >= time.Minute {
				elapsed = fmt.Sprintf("%d:%02d", int(target.Time.Minutes()), int(target.Time.Seconds())%60)
			}
			text = fmt.Sprintf(" %s %s", op, elapsed)
		}
		paces = append(paces, letter+SubtextStyle.Render(text))
	}
	const gap = "   "
	heading := "  " + SubtextStyle.Render("paces")
	if len(paces) == 3 {
		inline := heading + "  " + strings.Join(paces, gap)
		if lipgloss.Width(inline) <= width {
			return inline
		}
	}
	rows := []string{heading}
	for start := 0; start < len(paces); {
		end := min(start+4, len(paces))
		row := "  " + strings.Join(paces[start:end], gap)
		for end > start+1 && lipgloss.Width(row) > width {
			end--
			row = "  " + strings.Join(paces[start:end], gap)
		}
		rows = append(rows, row)
		start = end
	}
	return strings.Join(rows, "\n")
}
