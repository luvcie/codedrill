package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme is a named colour palette. Each field is one role, and every screen
// draws its colours through these roles, never through a hex value.
type Theme struct {
	ID, Name string
	// Light means the terminal background is light. Briefing markdown then
	// uses a light glamour base.
	Light bool

	// Accent is the one accent: the active tab, selection, timer, focus and
	// notices.
	Accent lipgloss.Color
	// Success is PASS, PB, S rank and GO.
	Success lipgloss.Color
	// Danger is failure.
	Danger lipgloss.Color
	// Text is normal text. It also fills progress cells and help keys.
	Text lipgloss.Color
	// Muted is for dim labels and help labels.
	Muted lipgloss.Color
	// Border is for dividers, boxes and empty progress cells.
	Border lipgloss.Color
	// Logo is the croc and the wordmark.
	Logo lipgloss.Color
	// OnAccent is text drawn on an Accent fill.
	OnAccent lipgloss.Color
	// DescSelected is the description of the selected menu row.
	DescSelected lipgloss.Color
	// ListTitle is the title of an unselected menu row. It is also the active
	// page dot of the menu.
	ListTitle lipgloss.Color
	// ListDesc is the description of an unselected menu row. Dimmed rows use it
	// too.
	ListDesc lipgloss.Color
}

// Themes lists the palettes in menu order. Themes[0] is the default, the
// legacy codedrill look. The cyan palette is the redesigned look and is kept
// as the "cyan" theme.
var Themes = []Theme{
	{
		ID: "legacy", Name: "Legacy codedrill",
		Accent: "#9CC26A", Success: "#D4E157", Danger: "#D9774A", Text: "#E6EBD8",
		Muted: "#8C9A78", Border: "#3E4A32", Logo: "#7FA05A", OnAccent: "#1A2214",
		DescSelected: "#EEF2E0",
		ListTitle:    "#E6EBD8", ListDesc: "#8C9A78",
	},
	{
		ID: "cyan", Name: "Cyan",
		Accent: "#00F0FF", Success: "#00FF66", Danger: "#FF4D4D", Text: "#FFFFFF",
		Muted: "#969696", Border: "#4A4A55", Logo: "#7FA05A", OnAccent: "#1A1A24",
		DescSelected: "#EEEEEE",
		// bubbles list.NewDefaultItemStyles, Dark values.
		ListTitle: "#DDDDDD", ListDesc: "#777777",
	},
	{
		ID: "gruvbox", Name: "Gruvbox",
		Accent: "#FABD2F", Success: "#B8BB26", Danger: "#FB4934", Text: "#EBDBB2",
		Muted: "#A89984", Border: "#504945", Logo: "#7FA05A", OnAccent: "#282828",
		DescSelected: "#FBF1C7",
		ListTitle:    "#EBDBB2", ListDesc: "#A89984",
	},
	{
		ID: "tokyo-night", Name: "Tokyo Night",
		Accent: "#7AA2F7", Success: "#9ECE6A", Danger: "#F7768E", Text: "#C0CAF5",
		Muted: "#7C83A8", Border: "#3B4261", Logo: "#7FA05A", OnAccent: "#1A1B26",
		DescSelected: "#C0CAF5",
		ListTitle:    "#C0CAF5", ListDesc: "#7C83A8",
	},
	{
		ID: "catppuccin-mocha", Name: "Catppuccin Mocha",
		Accent: "#89DCEB", Success: "#A6E3A1", Danger: "#F38BA8", Text: "#CDD6F4",
		Muted: "#9399B2", Border: "#45475A", Logo: "#7FA05A", OnAccent: "#1E1E2E",
		DescSelected: "#BAC2DE",
		ListTitle:    "#CDD6F4", ListDesc: "#9399B2",
	},
	{
		ID: "nord", Name: "Nord",
		Accent: "#88C0D0", Success: "#A3BE8C", Danger: "#BF616A", Text: "#ECEFF4",
		Muted: "#7B88A1", Border: "#434C5E", Logo: "#7FA05A", OnAccent: "#2E3440",
		DescSelected: "#E5E9F0",
		ListTitle:    "#ECEFF4", ListDesc: "#7B88A1",
	},
	{
		ID: "dracula", Name: "Dracula",
		Accent: "#BD93F9", Success: "#50FA7B", Danger: "#FF5555", Text: "#F8F8F2",
		Muted: "#7584AF", Border: "#44475A", Logo: "#7FA05A", OnAccent: "#282A36",
		DescSelected: "#F8F8F2",
		ListTitle:    "#F8F8F2", ListDesc: "#7584AF",
	},
	{
		ID: "rose-pine", Name: "Rose Pine",
		Accent: "#EBBCBA", Success: "#9CCFD8", Danger: "#EB6F92", Text: "#E0DEF4",
		Muted: "#908CAA", Border: "#403D52", Logo: "#7FA05A", OnAccent: "#191724",
		DescSelected: "#E0DEF4",
		ListTitle:    "#E0DEF4", ListDesc: "#908CAA",
	},
	{
		ID: "kanagawa", Name: "Kanagawa",
		Accent: "#7E9CD8", Success: "#98BB6C", Danger: "#FF5D62", Text: "#DCD7BA",
		Muted: "#87857C", Border: "#363646", Logo: "#7FA05A", OnAccent: "#1F1F28",
		DescSelected: "#DCD7BA",
		ListTitle:    "#DCD7BA", ListDesc: "#87857C",
	},
	{
		ID: "cyberpunk", Name: "Cyberpunk Edgerunner",
		Accent: "#FCEE0A", Success: "#00FF9F", Danger: "#FF003C", Text: "#E8E8E8",
		Muted: "#8A8A9A", Border: "#3A2F4A", Logo: "#7FA05A", OnAccent: "#0D0221",
		DescSelected: "#FFFFFF",
		ListTitle:    "#E8E8E8", ListDesc: "#8A8A9A",
	},
	{
		ID: "monochrome", Name: "Monochrome",
		Accent: "#FFFFFF", Success: "#FFFFFF", Danger: "#A8A8A8", Text: "#BDBDBD",
		Muted: "#8A8A8A", Border: "#444444", Logo: "#A0A0A0", OnAccent: "#000000",
		DescSelected: "#FFFFFF",
		ListTitle:    "#BDBDBD", ListDesc: "#8A8A8A",
	},
	{
		ID: "light", Name: "Light", Light: true,
		Accent: "#007A99", Success: "#1E8A3C", Danger: "#C62828", Text: "#1A1A1A",
		Muted: "#6B6B6B", Border: "#C8C8C8", Logo: "#4F7A2E", OnAccent: "#FFFFFF",
		DescSelected: "#333333",
		ListTitle:    "#1A1A1A", ListDesc: "#6B6B6B",
	},
}

// current is the palette last passed to ApplyTheme.
var current Theme

func init() {
	ApplyTheme(Themes[0])
}

// themeAliases maps old theme IDs to the ID that replaced them. "default" was
// the cyan palette before the legacy palette became the default, so a config
// that still says "default" keeps the cyan look.
var themeAliases = map[string]string{"default": "cyan"}

// ThemeByID finds a theme by its ID. The match ignores case. The old ID
// "default" resolves to the cyan theme.
func ThemeByID(id string) (Theme, bool) {
	if alias, ok := themeAliases[strings.ToLower(id)]; ok {
		id = alias
	}
	for _, t := range Themes {
		if strings.EqualFold(t.ID, id) {
			return t, true
		}
	}
	return Theme{}, false
}

// ThemeOrDefault is ThemeByID with the default theme, Themes[0], for an empty
// or unknown ID.
func ThemeOrDefault(id string) Theme {
	if t, ok := ThemeByID(id); ok {
		return t
	}
	return Themes[0]
}

// ApplyTheme makes t the palette for everything drawn after the call. It only
// assigns package variables, so it is safe to call from the Bubble Tea
// goroutine between frames. Caches of rendered output are not cleared here:
// MainModel.applyTheme does that.
func ApplyTheme(t Theme) {
	current = t
	ColorSecondary = t.Accent
	ColorSuccess = t.Success
	ColorDanger = t.Danger
	ColorText = t.Text
	ColorMuted = t.Muted
	ColorBorder = t.Border
	ColorCroc = t.Logo
	ColorBg = t.OnAccent
	ColorDescSelected = t.DescSelected
	ColorListTitle = t.ListTitle
	ColorListDesc = t.ListDesc
	rebuildStyles()
}

// CurrentTheme returns the palette last passed to ApplyTheme.
func CurrentTheme() Theme {
	return current
}

// logoRGB returns the logo colour as "r;g;b", the form the terminal banner
// script takes inside an SGR sequence.
func logoRGB(c lipgloss.Color) string {
	v, err := strconv.ParseUint(strings.TrimPrefix(string(c), "#"), 16, 32)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%d;%d;%d", v>>16&0xFF, v>>8&0xFF, v&0xFF)
}

// applyTheme switches the palette and rebuilds what was built from it: the
// briefing markdown, the list rows and the add-drill form. The list keeps its
// category, cursor and search state.
//
// huh does not restyle fields that already have a theme, so an open add-drill
// form is rebuilt. Its typed text is lost and its category choice is kept.
// The settings screen is only reached from the menu, so this does not happen
// today.
func (m *MainModel) applyTheme(t Theme) {
	ApplyTheme(t)
	m.briefingKey, m.briefingCache = "", ""
	cursor := m.list.Index()
	m.refreshList()
	m.list.Select(min(cursor, len(m.list.Items())-1))
	if m.form != nil {
		category := m.formCategory
		m.initForm()
		m.formCategory = category
		// Init focuses the first field itself. The command it returns only
		// starts the cursor blink and refreshes dynamic titles, and this form
		// has none, so the command is dropped.
		m.form.Init()
	}
}
