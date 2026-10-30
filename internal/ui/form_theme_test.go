package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	sgrPurple = "38;2;117;113;249"
	sgrPink   = "38;2;247;128;226"
	sgrFill   = "48;2;247;128;226"
	sgrCyan   = "38;2;0;240;255"
	sgrCyanBg = "48;2;0;240;255"
	sgrBgText = "38;2;26;26;36"
)

// openAddDrillForm opens the add-drill form from the menu at 80x24.
func openAddDrillForm(t *testing.T) *MainModel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.list.Select(len(m.list.Items()) - 1)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateAddDrill || m.form == nil {
		t.Fatalf("enter on add item: state %v, want add drill form", m.state)
	}
	return m
}

func TestFormThemeUsesPaletteOnly(t *testing.T) {
	useTheme(t, "cyan")
	forceTrueColor(t)
	m := openAddDrillForm(t)
	view := m.View()
	for _, bad := range []string{sgrPurple, sgrPink, sgrFill} {
		if strings.Contains(view, bad) {
			t.Errorf("add drill form contains %s, which is not in the palette", bad)
		}
	}
	if !strings.Contains(view, sgrCyan) {
		t.Errorf("add drill form has no cyan accent (%s)", sgrCyan)
	}
}

func TestFormStartsAtColumnTwo(t *testing.T) {
	restoreDefaultTheme(t)
	ApplyTheme(Themes[0])
	m := openAddDrillForm(t)
	for _, line := range strings.Split(ansi.Strip(m.View()), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !strings.HasPrefix(line, "  ") {
			t.Errorf("line does not start at column 2: %q", line)
		}
	}
}

func TestFormFitsScreen(t *testing.T) {
	restoreDefaultTheme(t)
	ApplyTheme(Themes[0])
	m := openAddDrillForm(t)
	view := m.View()
	if w := lipgloss.Width(view); w > 80 {
		t.Errorf("view is %d columns wide, want at most 80", w)
	}
	if h := lipgloss.Height(view); h > 24 {
		t.Errorf("view is %d lines tall, want at most 24", h)
	}
}

func TestFormTextFocusHasNoFill(t *testing.T) {
	restoreDefaultTheme(t)
	ApplyTheme(Themes[0])
	forceTrueColor(t)
	m := openAddDrillForm(t)
	// The first field is the title input, so focus is on a text field.
	if hasBackgroundSGR(m.View()) {
		t.Errorf("add drill form has a filled background while a text field is focused")
	}
}

func TestFormThemeButtonsAndHelp(t *testing.T) {
	useTheme(t, "cyan")
	forceTrueColor(t)
	th := formTheme()

	focused := th.Focused.FocusedButton.Render("Yes")
	if !strings.Contains(focused, sgrCyanBg) || !strings.Contains(focused, sgrBgText) {
		t.Errorf("focused button should be cyan fill with dark text, got %q", focused)
	}
	if hasBackgroundSGR(th.Focused.BlurredButton.Render("No")) {
		t.Errorf("blurred button should have no background")
	}
	if hasBackgroundSGR(th.Blurred.BlurredButton.Render("No")) {
		t.Errorf("blurred field's button should have no background")
	}
	if hasBackgroundSGR(th.Blurred.FocusedButton.Render("Yes")) {
		t.Errorf("blurred field's chosen button should have no background")
	}

	for name, s := range map[string]lipgloss.Style{
		"ShortKey":  th.Help.ShortKey,
		"ShortDesc": th.Help.ShortDesc,
		"FullKey":   th.Help.FullKey,
		"FullDesc":  th.Help.FullDesc,
	} {
		if s.GetBold() {
			t.Errorf("help style %s is bold", name)
		}
	}
	if !th.Focused.Title.GetBold() {
		t.Errorf("focused title should be bold")
	}
	if th.Blurred.Title.GetBold() {
		t.Errorf("blurred title should not be bold")
	}
	if got := th.Form.Base.GetMarginLeft(); got != 2 {
		t.Errorf("form margin left = %d, want 2", got)
	}
}
