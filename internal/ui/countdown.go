package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func renderCountdown(count int, width, height int) string {
	var numArt string
	switch count {
	case 3:
		numArt = "  ██████  \n      ██  \n  ██████  \n      ██  \n  ██████  "
	case 2:
		numArt = "  ██████  \n      ██  \n  ██████  \n  ██      \n  ██████  "
	case 1:
		numArt = "    ██    \n  ████    \n    ██    \n    ██    \n  ██████  "
	default:
		numArt = "  ████   ████   ██ \n ██     ██  ██  ██ \n ██ ███ ██  ██  ██ \n ██  ██ ██  ██     \n  ████   ████   ██ "
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorSecondary).
		Padding(1, 6).
		Align(lipgloss.Center).
		Render(
			TitleStyle.Render("GET READY") + "\n\n" +
				lipgloss.NewStyle().Foreground(ColorSecondary).Bold(true).Render(numArt) + "\n\n" +
				SubtextStyle.Render(fmt.Sprintf("Sprint starting in %d...", count)),
		)

	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}
