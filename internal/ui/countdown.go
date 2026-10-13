package ui

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var digitArt = map[int][5]string{
	0: {
		"  ██████  ",
		"  ██  ██  ",
		"  ██  ██  ",
		"  ██  ██  ",
		"  ██████  ",
	},
	1: {
		"    ██    ",
		"  ████    ",
		"    ██    ",
		"    ██    ",
		"  ██████  ",
	},
	2: {
		"  ██████  ",
		"      ██  ",
		"  ██████  ",
		"  ██      ",
		"  ██████  ",
	},
	3: {
		"  ██████  ",
		"      ██  ",
		"  ██████  ",
		"      ██  ",
		"  ██████  ",
	},
	4: {
		"  ██  ██  ",
		"  ██  ██  ",
		"  ██████  ",
		"      ██  ",
		"      ██  ",
	},
	5: {
		"  ██████  ",
		"  ██      ",
		"  ██████  ",
		"      ██  ",
		"  ██████  ",
	},
	6: {
		"  ██████  ",
		"  ██      ",
		"  ██████  ",
		"  ██  ██  ",
		"  ██████  ",
	},
	7: {
		"  ██████  ",
		"      ██  ",
		"      ██  ",
		"      ██  ",
		"      ██  ",
	},
	8: {
		"  ██████  ",
		"  ██  ██  ",
		"  ██████  ",
		"  ██  ██  ",
		"  ██████  ",
	},
	9: {
		"  ██████  ",
		"  ██  ██  ",
		"  ██████  ",
		"      ██  ",
		"  ██████  ",
	},
}

func renderBlockNumber(n int) string {
	digits := strconv.Itoa(max(n, 0))
	var rows [5]string
	for i, c := range digits {
		art := digitArt[int(c-'0')]
		for r := range rows {
			if i > 0 {
				rows[r] += "  "
			}
			rows[r] += art[r]
		}
	}
	return strings.Join(rows[:], "\n")
}

const goArt = "  ████   ████ \n ██     ██  ██\n ██ ███ ██  ██\n ██  ██ ██  ██\n  ████   ████ "

func renderCountdown(count int, width, height int) string {
	color, header, art := ColorSecondary, "GET READY", renderBlockNumber(count)
	if count <= 0 {
		color, header, art = ColorSuccess, "GO", goArt
	}

	content := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(color).
		Padding(1, 6).
		Align(lipgloss.Center).
		Render(TitleStyle.Foreground(color).Render(header) + "\n\n" + lipgloss.NewStyle().Foreground(color).Bold(true).Render(art))
	if count > 0 {
		content += "\n\n" + SubtextStyle.Render("[Space/Enter] skip  [Esc] cancel")
	}

	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
