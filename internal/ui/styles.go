package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Palettes: Cyberpunk / TETR.IO / Speedrun aesthetic
	ColorPrimary   = lipgloss.Color("#7D56F4") // Electric Purple
	ColorSecondary = lipgloss.Color("#00F0FF") // Cyan / Neon Blue
	ColorSuccess   = lipgloss.Color("#00FF66") // Neon Green (PB / Pass)
	ColorDanger    = lipgloss.Color("#FF0055") // Hot Pink / Red (Fail / Slower)
	ColorMuted     = lipgloss.Color("#626262")
	ColorBg        = lipgloss.Color("#1A1A24")

	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 2).
			MarginBottom(1)

	BadgeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#1A1A24")).
			Background(ColorSecondary).
			Bold(true).
			Padding(0, 1)

	TimerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Padding(0, 1)

	PBStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorSuccess)

	DeltaPlusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorDanger)

	DeltaMinusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSuccess)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	SubtextStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)
)
