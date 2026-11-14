package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/storage"
)

// settingsFile is where newMenuTestModel's config directory keeps config.json.
func settingsFile(t *testing.T) string {
	t.Helper()
	path, err := storage.GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func settingsTestModel(t *testing.T) *MainModel {
	t.Helper()
	restoreDefaultTheme(t)
	return newMenuTestModel(t)
}

func TestSettingsOpensFoldedOnCurrentTheme(t *testing.T) {
	m := settingsTestModel(t)
	m.applyTheme(Themes[3])
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.Update(runeKey("o"))
	if m.state != StateSettings {
		t.Fatalf("state %v after o, want StateSettings", m.state)
	}
	if !m.settingsThemesClosed || !m.settingsOnThemesHeader() {
		t.Errorf("open: cursor %d closed %v, want the folded Themes header", m.settingsCursor, m.settingsThemesClosed)
	}
	plain := ansi.Strip(m.viewSettings())
	if !strings.Contains(plain, "▸ Themes") {
		t.Error("opening does not draw the folded Themes header")
	}
	if strings.Contains(plain, Themes[3].Name) {
		t.Error("opening draws theme rows though the section starts folded")
	}
	if strings.Contains(plain, "PASS") || strings.Contains(plain, "[GO] ft_strlen") {
		t.Error("opening draws the theme preview though the section starts folded")
	}
	if CurrentTheme().ID != Themes[3].ID {
		t.Errorf("opening changed the theme to %q", CurrentTheme().ID)
	}
}

func TestSettingsMarksSavedTheme(t *testing.T) {
	m := settingsTestModel(t)
	if err := storage.SaveConfig(storage.Config{Theme: "nord"}); err != nil {
		t.Fatal(err)
	}
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settingsSaved != "nord" {
		t.Fatalf("saved theme %q, want nord", m.settingsSaved)
	}
	if !strings.Contains(ansi.Strip(m.viewSettings()), "Nord (current)") {
		t.Errorf("saved theme is not marked current:\n%s", ansi.Strip(m.viewSettings()))
	}
}

func TestSettingsActiveTabUsesThemeAccent(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	if view := m.viewSettings(); !strings.Contains(view, bg(CurrentTheme().Accent)) {
		t.Error("the General tab is not highlighted with the theme accent")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if view := m.viewTypingSounds(); !strings.Contains(view, bg(CurrentTheme().Accent)) {
		t.Error("the Typing Sounds tab is not highlighted with the theme accent")
	}
}

func TestSettingsListsLegacyThenCyanWithLegacyCurrent(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.settingsThemesOpen() {
		t.Fatal("enter on the folded Themes header did not unfold it")
	}

	legacy, cyan := -1, -1
	lines := strings.Split(ansi.Strip(m.viewSettings()), "\n")
	for i, line := range lines {
		switch {
		case strings.Contains(line, "Legacy codedrill"):
			legacy = i
		case strings.Contains(line, "Cyan"):
			cyan = i
		}
	}
	if legacy < 0 || cyan < 0 {
		t.Fatalf("legacy row %d, cyan row %d; want both drawn:\n%s", legacy, cyan, ansi.Strip(m.viewSettings()))
	}
	if cyan != legacy+1 {
		t.Errorf("cyan row %d, want directly after legacy row %d", cyan, legacy)
	}
	if !strings.Contains(lines[legacy], "(current)") {
		t.Errorf("fresh config does not mark legacy as current: %q", lines[legacy])
	}
	if strings.Contains(lines[cyan], "(current)") {
		t.Errorf("cyan is marked current on a fresh config: %q", lines[cyan])
	}
}

func TestSettingsMovePreviewsWithoutSaving(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("j"))
	m.Update(runeKey("j"))
	if got := CurrentTheme().ID; got != Themes[1].ID {
		t.Fatalf("after j the active theme is %q, want %q", got, Themes[1].ID)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if got := CurrentTheme().ID; got != Themes[2].ID {
		t.Fatalf("after down the active theme is %q, want %q", got, Themes[2].ID)
	}
	if _, err := os.Stat(settingsFile(t)); !os.IsNotExist(err) {
		t.Fatalf("moving the cursor wrote config.json (stat err %v)", err)
	}
}

func TestSettingsHomeEndJumpToFirstAndLast(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("G"))
	if !m.settingsOnReset() || CurrentTheme().ID != Themes[0].ID {
		t.Fatalf("G: cursor %d, want the reset row", m.settingsCursor)
	}
	m.Update(runeKey("k"))
	if !m.settingsOnMouse() || CurrentTheme().ID != Themes[0].ID {
		t.Fatalf("G: cursor %d theme %q, want the mouse row with the theme kept", m.settingsCursor, CurrentTheme().ID)
	}
	m.Update(runeKey("k"))
	if !m.settingsOnShell() || CurrentTheme().ID != Themes[0].ID {
		t.Fatalf("k from the mouse row: cursor %d theme %q, want the shell row with the theme kept", m.settingsCursor, CurrentTheme().ID)
	}
	m.Update(runeKey("k"))
	if m.settingsCursor != len(Themes)-1 || CurrentTheme().ID != Themes[len(Themes)-1].ID {
		t.Fatalf("k from the shell row: cursor %d theme %q, want the last theme", m.settingsCursor, CurrentTheme().ID)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.settingsCursor != 0 || CurrentTheme().ID != Themes[0].ID {
		t.Fatalf("home: cursor %d theme %q, want the first theme", m.settingsCursor, CurrentTheme().ID)
	}
	m.Update(runeKey("k"))
	if !m.settingsOnThemesHeader() {
		t.Fatalf("up from the first theme: cursor %d, want the Themes header", m.settingsCursor)
	}
	m.Update(runeKey("k"))
	if !m.settingsOnThemesHeader() {
		t.Fatalf("up on the Themes header moved the cursor to %d", m.settingsCursor)
	}
}

func TestSettingsThemesSectionCollapses(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))

	if !m.settingsThemesClosed || !m.settingsOnThemesHeader() {
		t.Fatalf("open: cursor %d closed %v, want the folded Themes header", m.settingsCursor, m.settingsThemesClosed)
	}
	plain := ansi.Strip(m.viewSettings())
	if !strings.Contains(plain, "▸ Themes") {
		t.Fatal("folded section lost its header")
	}
	if strings.Contains(plain, Themes[1].Name) {
		t.Fatalf("folded section draws %q:\n%s", Themes[1].Name, plain)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.settingsThemesClosed {
		t.Fatal("enter did not unfold the section")
	}
	plain = ansi.Strip(m.viewSettings())
	if !strings.Contains(plain, "▾ Themes") {
		t.Fatal("unfolded section has no open header")
	}
	if !strings.Contains(plain, Themes[1].Name) {
		t.Fatalf("unfolded section does not draw %q:\n%s", Themes[1].Name, plain)
	}
	m.Update(runeKey("j"))
	if m.settingsCursor != 0 {
		t.Fatalf("down from the open header: cursor %d, want the first theme", m.settingsCursor)
	}

	m.Update(runeKey("k"))
	if !m.settingsOnThemesHeader() {
		t.Fatalf("up from the first theme: cursor %d, want the Themes header", m.settingsCursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.settingsThemesClosed {
		t.Fatal("enter on the Themes header did not fold the section")
	}
	m.Update(runeKey("j"))
	if !m.settingsOnShell() {
		t.Fatalf("down from the folded header: cursor %d, want the shell row", m.settingsCursor)
	}
	m.Update(runeKey("k"))
	if !m.settingsOnThemesHeader() {
		t.Fatalf("up from the shell row: cursor %d, want the Themes header", m.settingsCursor)
	}
}

func TestSettingsPreviewHiddenWhileThemesFolded(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))

	plain := ansi.Strip(m.viewSettings())
	if strings.Contains(plain, "PASS") || strings.Contains(plain, "[GO] ft_strlen") {
		t.Fatalf("folded themes draw the preview:\n%s", plain)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	plain = ansi.Strip(m.viewSettings())
	if !strings.Contains(plain, "PASS") || !strings.Contains(plain, "[GO] ft_strlen") {
		t.Fatalf("unfolded themes do not draw the preview:\n%s", plain)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	plain = ansi.Strip(m.viewSettings())
	if strings.Contains(plain, "PASS") || strings.Contains(plain, "[GO] ft_strlen") {
		t.Fatalf("refolded themes draw the preview:\n%s", plain)
	}
}

func TestSettingsCollapsedThemesFitShortWindows(t *testing.T) {
	restoreDefaultTheme(t)
	for _, size := range []struct{ w, h int }{{60, 10}, {40, 8}, {30, 6}} {
		m := newMenuTestModel(t)
		m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
		m.Update(runeKey("o"))
		if !m.settingsThemesClosed || !m.settingsOnThemesHeader() {
			t.Fatalf("%dx%d: cursor %d closed %v, want the Themes header folded", size.w, size.h, m.settingsCursor, m.settingsThemesClosed)
		}
		checkSettingsWindow(t, m, size.w, size.h)
	}
}

func TestSettingsShellRowCyclesAndSaves(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.moveSettings(len(Themes))
	if !m.settingsOnShell() || m.shell != "bash" {
		t.Fatalf("cursor %d shell %q, want the shell row with bash", m.settingsCursor, m.shell)
	}
	if !strings.Contains(ansi.Strip(m.viewSettings()), "Shell  bash") {
		t.Fatalf("row does not read bash:\n%s", ansi.Strip(m.viewSettings()))
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateSettings || m.shell != "zsh" {
		t.Fatalf("after enter: state %v shell %q, want settings and zsh", m.state, m.shell)
	}
	if cfg, err := storage.LoadConfig(); err != nil || cfg.Shell != "zsh" {
		t.Fatalf("saved shell %q, err %v; want zsh", cfg.Shell, err)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.shell != "bash" {
		t.Fatalf("second enter: shell %q, want bash", m.shell)
	}
	if cfg, err := storage.LoadConfig(); err != nil || cfg.Shell != "bash" {
		t.Fatalf("saved shell %q, err %v; want bash", cfg.Shell, err)
	}
}

func TestSettingsEnterSavesTheme(t *testing.T) {
	m := settingsTestModel(t)
	sound := false
	countdown := 5
	if err := storage.SaveConfig(storage.Config{Editor: "vim", Sound: &sound, Countdown: &countdown}); err != nil {
		t.Fatal(err)
	}
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("j"))
	m.Update(runeKey("j"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != StateMenu {
		t.Fatalf("state %v after enter, want StateMenu", m.state)
	}
	want := "Theme set to " + Themes[1].Name + "."
	if m.menuNotice != want {
		t.Errorf("notice %q, want %q", m.menuNotice, want)
	}
	if !strings.Contains(m.View(), want) {
		t.Errorf("menu does not show the notice %q", want)
	}
	if got := CurrentTheme().ID; got != Themes[1].ID {
		t.Errorf("theme %q after save, want it kept applied as %q", got, Themes[1].ID)
	}

	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Theme != Themes[1].ID {
		t.Errorf("saved theme %q, want %q", cfg.Theme, Themes[1].ID)
	}
	if cfg.Editor != "vim" || cfg.IsSoundEnabled() || cfg.GetCountdown() != 5 {
		t.Errorf("saving the theme changed other fields: %+v", cfg)
	}
}

func TestSettingsEscAndQRevertWithoutSaving(t *testing.T) {
	for _, k := range []tea.KeyMsg{{Type: tea.KeyEsc}, runeKey("q")} {
		m := settingsTestModel(t)
		m.applyTheme(Themes[4])
		m.Update(runeKey("o"))
		m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		m.moveSettings(4)
		m.Update(runeKey("j"))
		m.Update(runeKey("j"))
		if CurrentTheme().ID != Themes[6].ID {
			t.Fatalf("preview did not apply: %q", CurrentTheme().ID)
		}
		m.Update(k)

		if m.state != StateMenu {
			t.Errorf("%q: state %v, want StateMenu", k.String(), m.state)
		}
		if got := CurrentTheme().ID; got != Themes[4].ID {
			t.Errorf("%q: theme %q, want the one active on open %q", k.String(), got, Themes[4].ID)
		}
		if m.menuNotice != "Cancelled." {
			t.Errorf("%q: notice %q, want Cancelled.", k.String(), m.menuNotice)
		}
		if _, err := os.Stat(settingsFile(t)); !os.IsNotExist(err) {
			t.Errorf("%q: cancelling wrote config.json (stat err %v)", k.String(), err)
		}
	}
}

func TestSettingsSearchPromptTakesO(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(runeKey("/"))
	m.Update(runeKey("o"))
	if m.state != StateMenu {
		t.Fatalf("o opened settings while the search prompt was open")
	}
	if !m.search.open || m.search.text != "o" {
		t.Fatalf("search prompt open %v text %q, want open with o typed", m.search.open, m.search.text)
	}
}

func TestSettingsDeletePromptCancelsOnO(t *testing.T) {
	m := settingsTestModel(t)
	selectDrill(t, m, "ft_strlen")
	m.Update(runeKey("x"))
	if m.pendingDelete == nil {
		t.Fatal("x did not open the delete prompt")
	}
	m.Update(runeKey("o"))
	if m.state != StateMenu {
		t.Fatalf("o opened settings while the delete prompt was open")
	}
	if m.pendingDelete != nil {
		t.Errorf("o should cancel the delete prompt like any other key")
	}
}

func TestSettingsKeepsMenuStateAfterReturn(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	category := m.categoryFilter
	m.list.Select(1)
	index := m.list.Index()

	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("j"))
	m.Update(runeKey("j"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.categoryFilter != category {
		t.Errorf("category %q after settings, want %q", m.categoryFilter, category)
	}
	if m.list.Index() != index {
		t.Errorf("list cursor %d after settings, want %d", m.list.Index(), index)
	}
}

func TestSettingsSaveFailureStaysOpen(t *testing.T) {
	m := settingsTestModel(t)
	// A config that is not JSON cannot be read, so it must not be written over.
	path := settingsFile(t)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0644); err != nil {
		t.Fatal(err)
	}
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("j"))
	m.Update(runeKey("j"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.state != StateSettings {
		t.Fatalf("state %v after a failed save, want StateSettings", m.state)
	}
	if m.settingsErr == nil || !strings.Contains(ansi.Strip(m.viewSettings()), "Could not save: ") {
		t.Fatalf("failed save did not show the error:\n%s", ansi.Strip(m.viewSettings()))
	}
	if got := CurrentTheme().ID; got != Themes[1].ID {
		t.Errorf("preview %q after failed save, want it kept applied", got)
	}
	if data, _ := os.ReadFile(path); string(data) != "{not json" {
		t.Errorf("failed save changed config.json to %q", data)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateMenu || CurrentTheme().ID != Themes[0].ID {
		t.Errorf("esc after failed save: state %v theme %q, want menu and the original theme", m.state, CurrentTheme().ID)
	}
}

func TestSettingsSaveFailureKeepsEnterWorking(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("j"))
	m.settingsErr = errors.New("disk full")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateMenu {
		t.Fatalf("enter with a writable config left state %v", m.state)
	}
	if m.settingsErr != nil {
		t.Errorf("a successful save kept the old error %v", m.settingsErr)
	}
}

func TestSettingsViewFitsWindows(t *testing.T) {
	restoreDefaultTheme(t)
	for _, size := range []struct{ w, h int }{{60, 20}, {80, 24}, {120, 40}} {
		m := newMenuTestModel(t)
		m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
		m.Update(runeKey("o"))
		m.Update(runeKey("G"))
		m.settingsErr = errors.New("a long reason that wraps across more than one row of the window")
		checkSettingsLayout(t, m, size.w, size.h)

		m.settingsErr = nil
		checkSettingsLayout(t, m, size.w, size.h)
	}
}

func checkSettingsLayout(t *testing.T, m *MainModel, w, h int) {
	t.Helper()
	view := m.View()
	lines := strings.Split(view, "\n")
	if len(lines) > h {
		t.Errorf("%dx%d: %d lines, window has %d", w, h, len(lines), h)
	}
	for _, line := range lines {
		plain := ansi.Strip(line)
		if ansi.StringWidth(line) > w {
			t.Errorf("%dx%d: line wider than the window: %q", w, h, plain)
		}
		if strings.TrimSpace(plain) == "" {
			continue
		}
		if indent := len(plain) - len(strings.TrimLeft(plain, " ")); indent < 2 {
			t.Errorf("%dx%d: line starts at column %d, want 2 or more: %q", w, h, indent, plain)
		}
	}
	// The label gives way before the list is cut, so it may be missing only
	// when every theme is drawn.
	plain := ansi.Strip(view)
	if strings.Contains(plain, "Themes") {
		return
	}
	for _, th := range Themes {
		if !strings.Contains(plain, th.Name) {
			t.Errorf("%dx%d: theme label missing and theme %q is cut:\n%s", w, h, th.Name, plain)
			return
		}
	}
}

func TestSettingsLongListStillShowsCursor(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("G"))
	m.Update(runeKey("k"))
	plain := ansi.Strip(m.View())
	if !strings.Contains(plain, "│ Mouse  on") {
		t.Errorf("60x20: cursor on the mouse row, but the row is not on screen:\n%s", plain)
	}
	m.Update(runeKey("k"))
	plain = ansi.Strip(m.View())
	if !strings.Contains(plain, "│ Shell  bash") {
		t.Errorf("60x20: cursor on the shell row, but it is not on screen:\n%s", plain)
	}
	m.Update(runeKey("k"))
	plain = ansi.Strip(m.View())
	last := Themes[len(Themes)-1].Name
	if !strings.Contains(plain, "│ "+last) {
		t.Errorf("60x20: cursor row %q is not on screen:\n%s", last, plain)
	}
}

func TestSettingsShowsEveryThemeWhenItFits(t *testing.T) {
	restoreDefaultTheme(t)
	// The Factory Reset action adds one footer row at the narrow size.
	for _, size := range []struct{ w, h int }{{60, 21}, {80, 24}} {
		for _, withErr := range []bool{false, true} {
			m := newMenuTestModel(t)
			m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
			m.Update(runeKey("o"))
			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			m.moveSettings(0)
			if withErr {
				m.settingsErr = errors.New("a long reason that wraps across more than one row of the window")
			}
			plain := ansi.Strip(m.viewSettings())
			for _, th := range Themes {
				if !strings.Contains(plain, th.Name) {
					t.Errorf("%dx%d err %v: theme %q not drawn:\n%s", size.w, size.h, withErr, th.Name, plain)
				}
			}
		}
	}
}

func TestSettingsSideBySidePreviewAt80x24(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	plain := ansi.Strip(m.viewSettings())
	for _, th := range Themes {
		if !strings.Contains(plain, th.Name) {
			t.Errorf("80x24: theme %q not drawn:\n%s", th.Name, plain)
		}
	}
	if !strings.Contains(plain, "[GO] ft_strlen") || !strings.Contains(plain, "PASS") {
		t.Errorf("80x24: side-by-side preview missing:\n%s", plain)
	}
}

func TestSettingsPreviewUsesThemeAccentAndSuccess(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	for _, th := range []Theme{Themes[3], Themes[10]} {
		m := newMenuTestModel(t)
		m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m.applyTheme(th)
		m.Update(runeKey("o"))
		m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		view := m.viewSettings()
		if !strings.Contains(ansi.Strip(view), "PASS") {
			t.Errorf("%s: settings view has no preview", th.ID)
		}
		if !strings.Contains(view, fg(th.Accent)) {
			t.Errorf("%s: settings view has no Accent colour %s", th.ID, fg(th.Accent))
		}
		if !strings.Contains(view, fg(th.Success)) {
			t.Errorf("%s: settings view has no Success colour %s", th.ID, fg(th.Success))
		}
	}
}

func TestSettingsPreviewPlacement(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if !strings.Contains(ansi.Strip(m.viewSettings()), "PASS") {
		t.Errorf("80x24: preview missing beside the list")
	}
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 40})
	if !strings.Contains(ansi.Strip(m.viewSettings()), "PASS") {
		t.Errorf("60x40: preview missing below the list")
	}
	// The full list matters more than the preview, so at 60x20 the stacked
	// preview goes and every theme is drawn.
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	plain := ansi.Strip(m.viewSettings())
	if strings.Contains(plain, "PASS") {
		t.Errorf("60x20: stacked preview drawn though the full list needs its rows:\n%s", plain)
	}
	if last := Themes[len(Themes)-1].Name; !strings.Contains(plain, last) {
		t.Errorf("60x20: full list not drawn, last theme %q missing:\n%s", last, plain)
	}

	m.Update(tea.WindowSizeMsg{Width: 20, Height: 40})
	if strings.Contains(ansi.Strip(m.viewSettings()), "PASS") {
		t.Errorf("20x40: preview drawn though it is wider than the window")
	}
}

func TestSettingsViewFitsShortWindows(t *testing.T) {
	restoreDefaultTheme(t)
	sizes := []struct{ w, h int }{{60, 20}, {60, 10}, {40, 8}, {30, 6}, {20, 4}, {40, 3}, {20, 40}}
	for _, size := range sizes {
		for _, withErr := range []bool{false, true} {
			for _, moves := range []int{0, len(Themes) / 2, len(Themes) - 1, len(Themes), len(Themes) + 1} {
				// The cursor opens on the Themes header, so unfold it before moving.
				ApplyTheme(Themes[0])
				m := newMenuTestModel(t)
				m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
				m.Update(runeKey("o"))
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				for range moves {
					m.Update(runeKey("j"))
				}
				if withErr {
					m.settingsErr = errors.New("a long reason that wraps across more than one row of the window")
				}
				checkSettingsWindow(t, m, size.w, size.h)
			}
		}
	}
}

// checkSettingsWindow checks that the settings view fits a w by h window, that
// every row starts at column 2 or more, and that the cursor theme is on screen
// whenever the height holds the header and one row.
func checkSettingsWindow(t *testing.T, m *MainModel, w, h int) {
	t.Helper()
	view := m.View()
	plain := ansi.Strip(view)
	if lines := strings.Split(view, "\n"); len(lines) > h {
		t.Errorf("%dx%d cursor %d: %d lines, window has %d:\n%s", w, h, m.settingsCursor, len(lines), h, plain)
	}
	for _, line := range strings.Split(view, "\n") {
		plainLine := ansi.Strip(line)
		if ansi.StringWidth(line) > w {
			t.Errorf("%dx%d cursor %d: line wider than the window: %q", w, h, m.settingsCursor, plainLine)
		}
		if strings.TrimSpace(plainLine) == "" {
			continue
		}
		if indent := len(plainLine) - len(strings.TrimLeft(plainLine, " ")); indent < 2 {
			t.Errorf("%dx%d cursor %d: line starts at column %d, want 2 or more: %q", w, h, m.settingsCursor, indent, plainLine)
		}
	}
	if h >= lipgloss.Height(m.screenHeader("settings"))+1 {
		var name string
		switch {
		case m.settingsOnReset():
			name = "Factory Reset"
		case m.settingsOnMouse():
			name = "Mouse  " + map[bool]string{true: "on", false: "off"}[!m.mouseOff]
		case m.settingsOnShell():
			name = "Shell  " + m.shell
		case m.settingsOnThemesHeader():
			name = "▾ Themes"
			if !m.settingsThemesOpen() {
				name = "▸ Themes"
			}
		default:
			name = ansi.Truncate(Themes[m.settingsCursor].Name, w-4, "")
		}
		if !strings.Contains(plain, "│ "+name) {
			t.Errorf("%dx%d: cursor row %q not on screen:\n%s", w, h, name, plain)
		}
	}
}

func TestSettingsSaveErrorClearsOnCursorMove(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("j"))
	m.Update(runeKey("j"))
	for _, k := range []tea.KeyMsg{runeKey("k"), runeKey("j"), runeKey("g"), runeKey("G"), {Type: tea.KeyHome}, {Type: tea.KeyEnd}} {
		m.settingsErr = errors.New("disk full")
		before := m.settingsCursor
		m.Update(k)
		if m.settingsCursor == before && k.String() != "g" && k.String() != "G" {
			t.Fatalf("%q did not move the cursor from %d", k.String(), before)
		}
		if m.settingsErr != nil {
			t.Errorf("%q: save error kept after the cursor moved", k.String())
		}
		if strings.Contains(ansi.Strip(m.viewSettings()), "Could not save") {
			t.Errorf("%q: save error still drawn after the cursor moved", k.String())
		}
	}
}

func TestSettingsSaveErrorStaysWhenCursorDoesNotMove(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(runeKey("o"))
	m.moveSettings(-1)
	m.settingsErr = errors.New("disk full")
	m.Update(runeKey("k"))
	if m.settingsErr == nil {
		t.Errorf("up on the Themes header cleared the save error, but the cursor did not move")
	}
}

func TestMenuHelpListsSettings(t *testing.T) {
	m := settingsTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	help := ansi.Strip(m.menuHelp())
	if !strings.Contains(help, "[o] settings") {
		t.Errorf("menu help has no [o] settings:\n%s", help)
	}
	for _, line := range strings.Split(help, "\n") {
		if ansi.StringWidth(line) > 80 {
			t.Errorf("menu help line wider than 80: %q", line)
		}
	}
}
