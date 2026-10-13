package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSecondary = lipgloss.Color("#00F0FF")
	ColorSuccess   = lipgloss.Color("#00FF66")
	ColorDanger    = lipgloss.Color("#FF0055")
	ColorMuted     = lipgloss.Color("#969696")
	ColorBg        = lipgloss.Color("#1A1A24")
	ColorText      = lipgloss.Color("#FFFFFF")
	// ColorCroc is only for the codedrill logo. Green elsewhere means success.
	ColorCroc = lipgloss.Color("#7FA05A")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText).
			Background(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	BadgeStyle = lipgloss.NewStyle().
			Foreground(ColorBg).
			Background(ColorSecondary).
			Bold(true).
			Padding(0, 1)

	TimerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Padding(0, 1)

	AccentStyle  = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
	SuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess)
	DangerStyle  = lipgloss.NewStyle().Bold(true).Foreground(ColorDanger)
	SubtextStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	LogoStyle    = lipgloss.NewStyle().Bold(true).Foreground(ColorCroc)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)
)

// boxWidth fits a content box to the terminal: 6 columns narrower than the
// screen, at most 86 wide and never below minimum.
func boxWidth(screenWidth, minimum int) int {
	w := screenWidth - 6
	if w <= 0 || w > 86 {
		w = 86
	}
	return max(w, minimum)
}

func renderProgressBar(progress float64, length int) string {
	filled := min(max(int(progress*float64(length)), 0), length)
	return fmt.Sprintf("[%s%s] %d%%", strings.Repeat("█", filled), strings.Repeat("░", length-filled), int(progress*100))
}
