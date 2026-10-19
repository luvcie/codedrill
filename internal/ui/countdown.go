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
	return renderCountdownPressed(count, width, height, "")
}

func renderCountdownPressed(count int, width, height int, pressed string) string {
	color, art := ColorSecondary, renderBlockNumber(count)
	if count <= 0 {
		color, art = ColorSuccess, goArt
	}
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	padX := min(6, max(0, (width-lipgloss.Width(art)-2)/2))

	box := func(padY int) string {
		return lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(color).
			Padding(padY, padX).
			Align(lipgloss.Center).
			Render(lipgloss.NewStyle().Foreground(color).Bold(true).Render(art))
	}

	rows := height
	var hint string
	if count > 0 {
		hint = renderHelp(countdownHelp, width, pressed)
		rows -= lipgloss.Height(hint) + 1
		if rows < 1 {
			hint, rows = "", height
		}
	}
	content := box(1)
	if lipgloss.Height(content) > rows {
		content = box(0)
	}
	content = lipgloss.NewStyle().MaxWidth(width).MaxHeight(rows).Render(content)
	body := lipgloss.Place(width, rows, lipgloss.Center, lipgloss.Center, content)
	if hint != "" {
		body += "\n\n" + hint
	}
	return body
}
