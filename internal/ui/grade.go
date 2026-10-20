package ui

import (
	"fmt"
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
)

var gradeTitles = map[Grade]string{
	GradeS: "GODSPEED",
	GradeA: "MASTER",
	GradeB: "SKILLED",
	GradeC: "CLEARED",
}

func CalculateGrade(d time.Duration, difficulty string) (Grade, string) {
	if d <= 0 {
		return "", ""
	}
	s, a, b := drill.PaceTargets(difficulty)
	g := GradeC
	switch {
	case d <= s:
		g = GradeS
	case d <= a:
		g = GradeA
	case d <= b:
		g = GradeB
	}
	return g, gradeTitles[g]
}

func gradeColor(g Grade) lipgloss.Color {
	switch g {
	case GradeS:
		return ColorSuccess
	case GradeA:
		return ColorSecondary
	case GradeB:
		return ColorText
	}
	return ColorMuted
}

func NextPaceGoal(d time.Duration, difficulty string) (goal string, top bool) {
	if d <= 0 {
		return "", false
	}
	s, a, b := drill.PaceTargets(difficulty)
	var target time.Duration
	var rank Grade
	switch {
	case d <= s:
		return "Top rank reached.", true
	case d <= a:
		target, rank = s, GradeS
	case d <= b:
		target, rank = a, GradeA
	default:
		target, rank = b, GradeB
	}
	return fmt.Sprintf("next %s-RANK at %s (%s to cut)", rank, storage.FormatDuration(target), storage.FormatDuration(d-target)), false
}

func RenderGradeBanner(g Grade, title string) string {
	if g == "" {
		return ""
	}
	text := fmt.Sprintf("%s-RANK · %s", g, title)
	if g == GradeS {
		text = "★ " + text
	}
	return lipgloss.NewStyle().Bold(true).Foreground(gradeColor(g)).Render(text)
}

func RenderPaceHints(difficulty string) string {
	s, a, b := drill.PaceTargets(difficulty)
	pace := func(g Grade, d time.Duration) string {
		letter := lipgloss.NewStyle().Bold(true).Foreground(gradeColor(g)).Render(string(g))
		return letter + SubtextStyle.Render(fmt.Sprintf(" ≤ %ds", int(d.Seconds())))
	}
	return "  " + SubtextStyle.Render("paces") + "  " +
		pace(GradeS, s) + "   " + pace(GradeA, a) + "   " + pace(GradeB, b)
}
