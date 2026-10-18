package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

var (
	ColorSecondary lipgloss.Color
	ColorSuccess   lipgloss.Color
	ColorDanger    lipgloss.Color
	ColorMuted     lipgloss.Color
	ColorBorder    lipgloss.Color
	ColorBg        lipgloss.Color
	ColorText      lipgloss.Color
	ColorDescSelected lipgloss.Color
	ColorListTitle lipgloss.Color
	ColorListDesc  lipgloss.Color
	ColorCroc      lipgloss.Color

	AccentStyle  lipgloss.Style
	SuccessStyle lipgloss.Style
	DangerStyle  lipgloss.Style
	SubtextStyle lipgloss.Style
	LogoStyle    lipgloss.Style
	BoxStyle     lipgloss.Style

	activeTabStyle   lipgloss.Style
	inactiveTabStyle lipgloss.Style
	helpKeyStyle     lipgloss.Style
	noticeStyle      lipgloss.Style
	timeStyle        lipgloss.Style
	pressStyle       = lipgloss.NewStyle().Reverse(true)
)

func themeList(l *list.Model) {
	s := &l.Styles
	s.StatusBar = s.StatusBar.Foreground(ColorMuted)
	s.StatusEmpty = s.StatusEmpty.Foreground(ColorMuted)
	s.NoItems = s.NoItems.Foreground(ColorMuted)
	s.ArabicPagination = s.ArabicPagination.Foreground(ColorMuted)
	s.ActivePaginationDot = s.ActivePaginationDot.Foreground(ColorListTitle)
	s.InactivePaginationDot = s.InactivePaginationDot.Foreground(ColorBorder)
	s.DividerDot = s.DividerDot.Foreground(ColorBorder)
	s.StatusBarActiveFilter = s.StatusBarActiveFilter.Foreground(ColorListTitle)
	s.StatusBarFilterCount = s.StatusBarFilterCount.Foreground(ColorMuted)
	l.Paginator.ActiveDot = s.ActivePaginationDot.String()
	l.Paginator.InactiveDot = s.InactivePaginationDot.String()
}

func rebuildStyles() {
	AccentStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSecondary)
	SuccessStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess)
	DangerStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorDanger)
	SubtextStyle = lipgloss.NewStyle().Foreground(ColorMuted)
	LogoStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorCroc)
	BoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2)

	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorBg).Background(ColorSecondary).Padding(0, 1)
	inactiveTabStyle = SubtextStyle.Padding(0, 1)
	helpKeyStyle = lipgloss.NewStyle().Foreground(ColorText)
	noticeStyle = lipgloss.NewStyle().Foreground(ColorSecondary).PaddingLeft(2)
	timeStyle = lipgloss.NewStyle().Bold(true).Foreground(ColorText)
}

func boxWidth(screenWidth, minimum int) int {
	w := screenWidth - 6
	if w <= 0 || w > 86 {
		w = 86
	}
	return max(w, minimum)
}

func renderProgressBar(progress float64, length int) string {
	progress = min(max(progress, 0), 1)
	filled := min(max(int(math.Round(progress*float64(length))), 0), length)
	full := lipgloss.NewStyle().Foreground(ColorText).Render(strings.Repeat("█", filled))
	empty := lipgloss.NewStyle().Foreground(ColorBorder).Render(strings.Repeat("░", length-filled))
	return SubtextStyle.Render("[") + full + empty + SubtextStyle.Render(fmt.Sprintf("] %d%%", int(math.Round(progress*100))))
}
