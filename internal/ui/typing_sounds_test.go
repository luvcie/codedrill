package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/typingsound"
)

func openTypingTestModel(t *testing.T) *MainModel {
	t.Helper()
	m := newMenuTestModel(t)
	m.Update(runeKey("o"))
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.state != StateTypingSounds {
		t.Fatalf("state %v after tab, want StateTypingSounds", m.state)
	}
	return m
}

// settingsTabClick returns a cell that lands on the named tab.
func settingsTabClick(m *MainModel, tab int) (int, int) {
	labels := m.settingsTabLabels()
	bar := ansi.Strip(m.settingsTabBar())
	at := strings.Index(bar, labels[tab])
	return 2 + at + ansi.StringWidth(labels[tab])/2, m.settingsTabRow()
}

func TestSettingsShowsBothTabs(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))
	plain := ansi.Strip(m.viewSettings())
	for _, want := range []string{"[ General ]", "[ Typing Sounds ]"} {
		if !strings.Contains(plain, want) {
			t.Errorf("general tab view is missing %q:\n%s", want, plain)
		}
	}
	if m.settingsTab() != settingsTabGeneral {
		t.Fatalf("settings opened on tab %d, want General", m.settingsTab())
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	plain = ansi.Strip(m.viewTypingSounds())
	for _, want := range []string{"[ General ]", "[ Typing Sounds ]", "Typing sounds  off"} {
		if !strings.Contains(plain, want) {
			t.Errorf("typing tab view is missing %q:\n%s", want, plain)
		}
	}
	if m.settingsTab() != settingsTabTyping {
		t.Fatalf("tab key did not switch to Typing Sounds")
	}

	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.settingsTab() != settingsTabGeneral {
		t.Fatal("tab key did not wrap back to General")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.settingsTab() != settingsTabTyping {
		t.Fatal("shift+tab did not wrap to Typing Sounds")
	}
}

func TestSettingsTabMouseSwitching(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.Update(runeKey("o"))

	x, y := settingsTabClick(m, settingsTabTyping)
	m.Update(leftClick(x, y))
	if m.state != StateTypingSounds {
		t.Fatalf("click on Typing Sounds left state %v", m.state)
	}
	x, y = settingsTabClick(m, settingsTabGeneral)
	m.Update(leftClick(x, y))
	if m.state != StateSettings {
		t.Fatalf("click on General left state %v", m.state)
	}
}

func TestTypingSoundsScreenContents(t *testing.T) {
	m := openTypingTestModel(t)
	if m.typingCursor != 0 {
		t.Fatalf("cursor %d, want the enabled row", m.typingCursor)
	}
	plain := ansi.Strip(m.viewTypingSounds())
	for _, want := range []string{
		"Typing sounds  off", "Volume  70%", "Word completion sounds  off",
		"Preview selected pack", "Silent", "GameCube Controller SFX", "Generated tones",
	} {
		if !strings.Contains(plain, want) {
			t.Errorf("view does not contain %q:\n%s", want, plain)
		}
	}
}

func TestTypingSoundsTogglePersists(t *testing.T) {
	m := openTypingTestModel(t)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.typingOn {
		t.Fatal("enter on the enabled row did not turn sounds on")
	}
	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Typing == nil || cfg.Typing.Enabled == nil || !*cfg.Typing.Enabled || !cfg.TypingSoundsOn() {
		t.Fatalf("saved config is not enabled: %+v", cfg.Typing)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.typingOn {
		t.Fatal("second enter did not turn sounds off")
	}
	cfg, _ = storage.LoadConfig()
	if cfg.TypingSoundsOn() {
		t.Fatal("saved config is still enabled")
	}
}

func TestTypingSoundsPackSelectionPersists(t *testing.T) {
	m := openTypingTestModel(t)
	// Rows are Enabled, Volume, Words, Preview, Silent, Generated tones, then
	// the catalog, so the first catalog pack is row 6.
	for range 6 {
		m.Update(runeKey("j"))
	}
	if m.typingCursor != 6 || m.typingCursorChoice().ID != typingsound.Catalog[0].ID {
		t.Fatalf("cursor %d choice %q, want the first catalog pack", m.typingCursor, m.typingCursorChoice().ID)
	}
	// The preview command may fail without an audio server; only the saved
	// selection matters here.
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TypingPackID() != typingsound.Catalog[0].ID {
		t.Fatalf("saved pack %q, want %q", cfg.TypingPackID(), typingsound.Catalog[0].ID)
	}
	if !strings.Contains(ansi.Strip(m.viewTypingSounds()), "(current)") {
		t.Fatal("selected pack has no current marker")
	}
	if !strings.Contains(ansi.Strip(m.viewTypingSounds()), typingsound.TETRIODatabase) {
		t.Fatal("selected community pack has no source credit")
	}
}

func TestTypingSoundsVolumePersists(t *testing.T) {
	m := openTypingTestModel(t)
	m.Update(runeKey("j"))
	if m.typingCursor != 1 {
		t.Fatalf("cursor %d, want the volume row", m.typingCursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.typingVolume != 75 {
		t.Fatalf("volume %d after right, want 75", m.typingVolume)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got := int(cfg.TypingVolume()*100 + 0.5); got != 80 {
		t.Fatalf("saved volume %d, want 80", got)
	}
	for range 30 {
		m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	}
	if m.typingVolume != 0 {
		t.Fatalf("volume %d after many left presses, want 0", m.typingVolume)
	}
	cfg, _ = storage.LoadConfig()
	if cfg.TypingVolume() != 0 {
		t.Fatalf("saved volume %v, want 0", cfg.TypingVolume())
	}
}

func TestTypingSoundsWordTogglePersists(t *testing.T) {
	m := openTypingTestModel(t)
	m.Update(runeKey("j"))
	m.Update(runeKey("j"))
	if m.typingCursor != typingRowWords {
		t.Fatalf("cursor %d, want the word completion row", m.typingCursor)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.typingWords {
		t.Fatal("enter on the word row did not turn word sounds on")
	}
	cfg, err := storage.LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.TypingWordSoundsOn() {
		t.Fatalf("saved config has word sounds off: %+v", cfg.Typing)
	}
	if !strings.Contains(ansi.Strip(m.viewTypingSounds()), "Word completion sounds  on") {
		t.Fatal("view does not show word sounds on")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	cfg, _ = storage.LoadConfig()
	if cfg.TypingWordSoundsOn() {
		t.Fatal("second enter did not turn word sounds off")
	}
}

func TestTypingSoundsEscReturnsToMenu(t *testing.T) {
	m := openTypingTestModel(t)
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateMenu {
		t.Fatalf("state %v after esc, want StateMenu", m.state)
	}
}

func TestTypingSoundsLongListScrollsWithKeyboard(t *testing.T) {
	m := openTypingTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	m.Update(runeKey("G"))
	view := ansi.Strip(m.viewTypingSounds())
	last := m.typingChoices[len(m.typingChoices)-1].Label
	if !strings.Contains(view, last) {
		t.Fatalf("last pack %q is not on screen after G:\n%s", last, view)
	}
	if !strings.Contains(view, "│ "+last) {
		t.Fatalf("last pack is not the cursor row:\n%s", view)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.typingCursor >= typingRowPackBase+len(m.typingChoices)-1 {
		t.Fatalf("pgup did not move the cursor: %d", m.typingCursor)
	}
	view = ansi.Strip(m.viewTypingSounds())
	current := m.typingChoices[m.typingCursor-typingRowPackBase].Label
	if !strings.Contains(view, "│ "+current) {
		t.Fatalf("cursor row %q scrolled out of view:\n%s", current, view)
	}
}

func TestTypingSoundsMouseScrollsAndSelects(t *testing.T) {
	m := openTypingTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 12})
	m.Update(wheel(false, 10, 10))
	if m.typingCursor == 0 {
		t.Fatal("wheel down did not move the cursor")
	}
	if m.settingsTab() != settingsTabTyping {
		t.Fatal("wheel left the typing tab")
	}
	// Click the cursor row to select its pack.
	row := m.typingCursor
	x, y := 4, m.typingLayout(60, 12).contentTop+(row-m.typingLayout(60, 12).start)
	m.Update(leftClick(x, y))
	if m.typingPack != m.typingCursorChoice().ID {
		t.Fatalf("click selected pack %q, want %q", m.typingPack, m.typingCursorChoice().ID)
	}
}

func TestTypingSoundsViewFitsWindows(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {60, 20}, {40, 12}, {30, 6}} {
		m := openTypingTestModel(t)
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m.Update(runeKey("G"))
		view := m.View()
		lines := strings.Split(view, "\n")
		if len(lines) > size[1] {
			t.Errorf("%v: %d lines, want at most %d", size, len(lines), size[1])
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Errorf("%v: line wider than the window: %q", size, ansi.Strip(line))
			}
		}
		if !strings.Contains(ansi.Strip(view), "│") {
			t.Errorf("%v: the cursor row is not drawn:\n%s", size, ansi.Strip(view))
		}
		if !strings.Contains(ansi.Strip(view), "[General]") && !strings.Contains(ansi.Strip(view), "[ General ]") {
			t.Errorf("%v: the tabs are not drawn:\n%s", size, ansi.Strip(view))
		}
	}
}
