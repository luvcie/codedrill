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
	if m.settingsTab() != settingsTabMusic {
		t.Fatal("tab did not advance to Music")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.settingsTab() != settingsTabGeneral {
		t.Fatal("tab did not wrap to General")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.settingsTab() != settingsTabMusic {
		t.Fatal("shift+tab did not wrap to Music")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.settingsTab() != settingsTabTyping {
		t.Fatal("shift+tab did not move to Typing Sounds")
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
		"Random pack per drill  off",
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
	firstCatalogRow := typingRowPackBase + 2 // Silent and generated tones precede catalog packs.
	for range firstCatalogRow {
		m.Update(runeKey("j"))
	}
	if m.typingCursor != firstCatalogRow || m.typingCursorChoice().ID != typingsound.Catalog[0].ID {
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

func TestTypingSoundsRandomPackTogglePreservesManualPack(t *testing.T) {
	m := openTypingTestModel(t)
	if m.typingRandom {
		t.Fatal("random packs are on by default")
	}
	m.typingPack = typingsound.Catalog[0].ID
	m.typingVolume = 42
	m.typingWords = true
	if !m.saveTyping() {
		t.Fatal(m.typingErr)
	}
	m.typingCursor = typingRowRandom
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.openTypingSounds()
	cfg, err := storage.LoadConfig()
	if err != nil || !m.typingRandom || !cfg.Typing.RandomPackPerDrill || cfg.TypingSoundsOn() ||
		cfg.TypingPackID() != typingsound.Catalog[0].ID || cfg.TypingVolume() != .42 || !cfg.TypingWordSoundsOn() {
		t.Fatalf("random toggle changed other settings: %+v, %v", cfg.Typing, err)
	}
	if m.download != nil || !strings.Contains(ansi.Strip(m.View()), "Random pack per drill  on") {
		t.Fatal("random toggle started a download or was not shown")
	}
	w, h := m.screenSize()
	layout := m.typingLayout(w, h)
	m.Update(leftClick(4, layout.contentTop+typingRowRandom-layout.start))
	m.openTypingSounds()
	cfg, err = storage.LoadConfig()
	if err != nil || m.typingRandom || cfg.Typing.RandomPackPerDrill || m.typingPack != typingsound.Catalog[0].ID {
		t.Fatalf("turning random off lost the manual pack: %+v, %v", cfg.Typing, err)
	}
}

func TestTypingSoundsVolumePersists(t *testing.T) {
	m := openTypingTestModel(t)
	m.Update(runeKey("j"))
	if m.typingCursor != 1 {
		t.Fatalf("cursor %d, want the volume row", m.typingCursor)
	}
	for _, step := range []struct {
		key     tea.KeyType
		presses int
		want    int
	}{
		{tea.KeyRight, 1, 71},
		{tea.KeyRight, 1, 72},
		{tea.KeyLeft, 1, 71},
		{tea.KeyLeft, 101, 0},
		{tea.KeyLeft, 1, 0},
		{tea.KeyRight, 1, 1},
		{tea.KeyRight, 101, 100},
		{tea.KeyRight, 1, 100},
		{tea.KeyLeft, 1, 99},
	} {
		for range step.presses {
			m.Update(tea.KeyMsg{Type: step.key})
		}
		if m.typingVolume != step.want || m.typingErr != nil {
			t.Fatalf("typing volume = %d, want %d: %v", m.typingVolume, step.want, m.typingErr)
		}
		m.openTypingSounds()
		if m.typingVolume != step.want || m.typingErr != nil {
			t.Fatalf("saved typing volume = %d, want %d: %v", m.typingVolume, step.want, m.typingErr)
		}
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
