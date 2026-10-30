package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// bubblesDarkDefaults are the foreground colours bubbles list draws with when
// no theme sets them, on a dark background. The light values are not listed:
// the checks run on a dark terminal, and the light value of some of these
// equals a theme colour.
var bubblesDarkDefaults = []string{
	"#DDDDDD", // NormalTitle, StatusBarActiveFilter
	"#777777", // NormalDesc, DimmedTitle, StatusBar
	"#4D4D4D", // DimmedDesc
	"#979797", // ActivePaginationDot
	"#3C3C3C", // InactivePaginationDot, DividerDot, StatusBarFilterCount
	"#5C5C5C", // StatusEmpty, ArabicPagination
	"#626262", // NoItems
	"#747373", // Spinner
	"#ECFD65", // FilterPrompt
	"#EE6FF8", // FilterCursor, SelectedTitle
	"#AD58B4", // SelectedDesc, selected border
}

// menuAt80x24 is the menu at the smallest size the app is tested at, with
// the current theme applied to the list.
func menuAt80x24(t *testing.T) *MainModel {
	t.Helper()
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.applyTheme(CurrentTheme())
	return m
}

func TestLightMenuHasNoBubblesDefaultGreys(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	light, ok := ThemeByID("light")
	if !ok {
		t.Fatal("light theme is missing")
	}
	ApplyTheme(light)
	m := menuAt80x24(t)
	view := m.View()

	if !strings.Contains(view, fg(light.ListTitle)) {
		t.Fatalf("light menu lacks the list title colour %s", fg(light.ListTitle))
	}
	if !strings.Contains(view, fg(light.ListDesc)) {
		t.Fatalf("light menu lacks the list description colour %s", fg(light.ListDesc))
	}
	for _, hex := range bubblesDarkDefaults {
		if strings.Contains(view, fg(lipgloss.Color(hex))) {
			t.Errorf("light menu uses the bubbles default %s", hex)
		}
	}
}

func TestCyanMenuRowsUseBubblesDarkValues(t *testing.T) {
	forceTrueColor(t)
	useTheme(t, "cyan")

	// The cyan theme must match what bubbles draws today, so read the
	// values from bubbles rather than trusting the copy in theme.go.
	item := list.NewDefaultItemStyles()
	title, ok := item.NormalTitle.GetForeground().(lipgloss.AdaptiveColor)
	if !ok {
		t.Fatalf("bubbles NormalTitle foreground is %T, want AdaptiveColor", item.NormalTitle.GetForeground())
	}
	desc, ok := item.NormalDesc.GetForeground().(lipgloss.AdaptiveColor)
	if !ok {
		t.Fatalf("bubbles NormalDesc foreground is %T, want AdaptiveColor", item.NormalDesc.GetForeground())
	}
	if !strings.EqualFold(string(ColorListTitle), title.Dark) {
		t.Errorf("cyan ListTitle = %s, bubbles dark value is %s", ColorListTitle, title.Dark)
	}
	if !strings.EqualFold(string(ColorListDesc), desc.Dark) {
		t.Errorf("cyan ListDesc = %s, bubbles dark value is %s", ColorListDesc, desc.Dark)
	}

	view := menuAt80x24(t).View()
	if !strings.Contains(view, fg(lipgloss.Color(title.Dark))) {
		t.Errorf("cyan menu lacks the bubbles dark title %s", title.Dark)
	}
	if !strings.Contains(view, fg(lipgloss.Color(desc.Dark))) {
		t.Errorf("cyan menu lacks the bubbles dark description %s", desc.Dark)
	}
}

func TestOtherThemesListRolesFollowTextAndMuted(t *testing.T) {
	for _, th := range Themes {
		if th.ID == "cyan" {
			continue
		}
		if th.ListTitle != th.Text || th.ListDesc != th.Muted {
			t.Errorf("%s: ListTitle %s, ListDesc %s; want Text %s and Muted %s",
				th.ID, th.ListTitle, th.ListDesc, th.Text, th.Muted)
		}
	}
}
