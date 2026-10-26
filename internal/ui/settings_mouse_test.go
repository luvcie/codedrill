package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/storage"
)

// send delivers msg to the model and returns the command it produced.
func send(m *MainModel, msg tea.Msg) tea.Cmd {
	_, cmd := m.Update(msg)
	return cmd
}

// mouseCmdKind names the mouse command a settings action returned, from the
// message it produces: "on", "off", or "" when there is no command.
func mouseCmdKind(cmd tea.Cmd) string {
	if cmd == nil {
		return ""
	}
	switch fmt.Sprintf("%T", cmd()) {
	case "tea.enableMouseCellMotionMsg":
		return "on"
	case "tea.disableMouseMsg":
		return "off"
	}
	return "other"
}

// mouseRowY returns the row where the mouse row is drawn.
func mouseRowY(t *testing.T, m *MainModel) (int, int) {
	t.Helper()
	for y, line := range stripLines(m) {
		if idx := strings.Index(line, "Mouse  "); idx >= 0 {
			return y, ansi.StringWidth(line[:idx])
		}
	}
	t.Fatal("the mouse row is not on screen")
	return 0, 0
}

func mouseConfigOn(t *testing.T) bool {
	t.Helper()
	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	return cfg.MouseOn()
}

func TestStartupReadsMouseOffFromConfig(t *testing.T) {
	restoreDefaultTheme(t)
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	off := false
	if err := storage.SaveConfig(storage.Config{Mouse: &off}); err != nil {
		t.Fatal(err)
	}
	m, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if !m.mouseOff {
		t.Fatal("config with mouse false did not switch mouse off")
	}
	if n := mouseEnableCount(m.Init()); n != 0 {
		t.Fatalf("Init enables mouse %d times with mouse off", n)
	}
	before := snapshotMenu(m)
	m.Update(leftClick(3, rowOf(t, m, itemTitle(m, 2))))
	m.Update(wheel(false, 3, 10))
	if got := snapshotMenu(m); got != before {
		t.Fatalf("mouse events changed the menu while off: %+v -> %+v", before, got)
	}
}

func TestStartupMouseOnByDefault(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	if m.mouseOff {
		t.Fatal("fresh config starts with mouse off")
	}
	if n := mouseEnableCount(m.Init()); n != 1 {
		t.Fatalf("Init enables mouse %d times, want 1", n)
	}
}

func TestSettingsCursorReachesMouseRow(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.moveSettings(0)
	for range len(Themes) {
		m.Update(runeKey("j"))
	}
	if !m.settingsOnMouse() || m.settingsCursor != len(Themes) {
		t.Fatalf("cursor %d after %d downs, want the mouse row %d", m.settingsCursor, len(Themes), len(Themes))
	}
	if CurrentTheme().ID != Themes[len(Themes)-1].ID {
		t.Fatalf("moving onto the mouse row changed the theme to %q", CurrentTheme().ID)
	}
	m.Update(runeKey("j"))
	if m.settingsCursor != len(Themes) {
		t.Fatalf("down past the mouse row moved the cursor to %d", m.settingsCursor)
	}
}

func TestSettingsEnterTogglesMouseOff(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(runeKey("G"))
	cmd := send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateSettings {
		t.Fatalf("state %v after toggling mouse, want to stay in settings", m.state)
	}
	if got := mouseCmdKind(cmd); got != "off" {
		t.Fatalf("toggle off returned %q command, want disable", got)
	}
	if !m.mouseOff || mouseConfigOn(t) {
		t.Fatalf("mouse off %v, config on %v; want both off", m.mouseOff, mouseConfigOn(t))
	}
	if !strings.Contains(ansi.Strip(m.viewSettings()), "Mouse  off") {
		t.Fatalf("row does not read off:\n%s", ansi.Strip(m.viewSettings()))
	}
	if m.settingsErr != nil {
		t.Fatalf("settings error after a good save: %v", m.settingsErr)
	}
}

func TestSettingsSpaceTogglesMouseOnMouseRow(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(runeKey("G"))
	if got := mouseCmdKind(send(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})); got != "off" {
		t.Fatalf("space on the mouse row returned %q, want disable", got)
	}
	// On a theme row space does nothing.
	m.moveSettings(0)
	if got := mouseCmdKind(send(m, tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}})); got != "" {
		t.Fatalf("space on a theme row returned %q, want nothing", got)
	}
	if m.state != StateSettings || !m.mouseOff {
		t.Fatalf("space on a theme row changed state %v or mouse %v", m.state, m.mouseOff)
	}
}

func TestSettingsClickTogglesMouseBackOn(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(runeKey("G"))
	send(m, tea.KeyMsg{Type: tea.KeyEnter})

	// Mouse is off, so the click is the one the settings row accepts.
	y, x := mouseRowY(t, m)
	cmd := send(m, leftClick(x, y))
	if got := mouseCmdKind(cmd); got != "on" {
		t.Fatalf("click toggled to %q command, want enable", got)
	}
	if m.mouseOff || !mouseConfigOn(t) {
		t.Fatalf("mouse off %v, config on %v; want both on", m.mouseOff, mouseConfigOn(t))
	}
	if m.state != StateSettings {
		t.Fatal("click on the mouse row left settings")
	}
	if !strings.Contains(ansi.Strip(m.viewSettings()), "Mouse  on") {
		t.Fatalf("row does not read on:\n%s", ansi.Strip(m.viewSettings()))
	}
}

func TestSettingsWheelMovesOntoMouseRow(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.moveSettings(0)
	for range len(Themes) {
		m.Update(wheel(false, 3, 5))
	}
	if !m.settingsOnMouse() {
		t.Fatalf("wheel down stopped at %d, want the mouse row", m.settingsCursor)
	}
	m.Update(wheel(true, 3, 5))
	if m.settingsCursor != len(Themes)-1 || CurrentTheme().ID != Themes[len(Themes)-1].ID {
		t.Fatalf("wheel up from the mouse row: cursor %d theme %q", m.settingsCursor, CurrentTheme().ID)
	}
}

func TestSettingsEscAfterMouseToggleSaysNothing(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(runeKey("G"))
	send(m, tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateMenu {
		t.Fatalf("esc left state %v, want the menu", m.state)
	}
	if m.menuNotice == "Cancelled." {
		t.Fatal("esc after only a mouse toggle said Cancelled.")
	}
	if !m.mouseOff || mouseConfigOn(t) {
		t.Fatal("esc undid the saved mouse setting")
	}
}

func TestSettingsEscAfterThemePreviewStillCancels(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(runeKey("j"))
	if CurrentTheme().ID != Themes[1].ID {
		t.Fatalf("preview did not apply: %q", CurrentTheme().ID)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if CurrentTheme().ID != Themes[0].ID {
		t.Fatalf("esc left the previewed theme %q", CurrentTheme().ID)
	}
	if m.menuNotice != "Cancelled." {
		t.Fatalf("notice %q, want Cancelled.", m.menuNotice)
	}
	if m.mouseOff || !mouseConfigOn(t) {
		t.Fatal("esc after a theme preview changed the mouse setting")
	}
}

func TestSettingsMouseSaveFailureRevertsAndShowsError(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(runeKey("G"))
	path := settingsFile(t)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatalf("failed toggle returned a command: %s", mouseCmdKind(cmd))
	}
	if m.mouseOff {
		t.Fatal("failed save left mouse off at runtime")
	}
	if m.settingsErr == nil || !strings.Contains(ansi.Strip(m.viewSettings()), "Could not save: ") {
		t.Fatalf("failed toggle did not show the error:\n%s", ansi.Strip(m.viewSettings()))
	}
	if data, _ := os.ReadFile(path); string(data) != "{not json" {
		t.Fatalf("failed save changed config.json to %q", data)
	}
	if m.state != StateSettings {
		t.Fatalf("failed toggle left settings: %v", m.state)
	}
}

// TestSettingsMouseRowHitTestMatchesRendering checks the mouse row and the
// theme rows at each size, with the cursor on the mouse row and on a theme.
func TestSettingsMouseRowHitTestMatchesRendering(t *testing.T) {
	restoreDefaultTheme(t)
	sizes := []tea.WindowSizeMsg{{Width: 60, Height: 20}, {Width: 80, Height: 24}, {Width: 40, Height: 10}, {Width: 30, Height: 6}}
	for _, size := range sizes {
		for _, onMouse := range []bool{false, true} {
			t.Run(fmt.Sprintf("%dx%d mouse=%v", size.Width, size.Height, onMouse), func(t *testing.T) {
				ApplyTheme(Themes[0])
				m := newMenuTestModel(t)
				m.Update(size)
				m.Update(runeKey("o"))
				if onMouse {
					m.Update(runeKey("G"))
				}
				rows := settingsThemeRows(m)
				if onMouse {
					y, x := mouseRowY(t, m)
					if got, ok := m.settingsRowAt(x, y); !ok || got != len(Themes) {
						t.Fatalf("mouse row %d col %d: settingsRowAt = %d, %v", y, x, got, ok)
					}
				}
				for y := range size.Height {
					row, drawn := rows[y]
					if !drawn {
						for x := range size.Width {
							if i, ok := m.settingsRowAt(x, y); ok {
								t.Fatalf("row %d has no row but column %d hits %d", y, x, i)
							}
						}
						continue
					}
					if got, ok := m.settingsRowAt(row[1], y); !ok || got != row[0] {
						t.Fatalf("row %d drawn as %d: settingsRowAt = %d, %v", y, row[0], got, ok)
					}
				}
			})
		}
	}
}

// TestSettingsMouseRowStaysVisibleAtSmallSizes checks that the mouse row is on
// screen when the cursor is on it, and that the view fits the window.
func TestSettingsMouseRowStaysVisibleAtSmallSizes(t *testing.T) {
	restoreDefaultTheme(t)
	sizes := []struct{ w, h int }{{60, 20}, {80, 24}, {40, 10}, {30, 6}}
	for _, size := range sizes {
		for _, withErr := range []bool{false, true} {
			ApplyTheme(Themes[0])
			m := newMenuTestModel(t)
			m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
			m.Update(runeKey("o"))
			m.Update(runeKey("G"))
			if withErr {
				m.settingsErr = errors.New("a long reason that wraps across more than one row of the window")
			}
			lines := strings.Split(m.View(), "\n")
			if len(lines) > size.h {
				t.Fatalf("%dx%d err %v: %d lines in a window of %d", size.w, size.h, withErr, len(lines), size.h)
			}
			if !strings.Contains(ansi.Strip(m.View()), "│ Mouse  on") {
				t.Errorf("%dx%d err %v: cursor on the mouse row but it is not drawn:\n%s", size.w, size.h, withErr, ansi.Strip(m.View()))
			}
		}
	}
}
