package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func clickHelp(t *testing.T, m *MainModel, text string, dx int) tea.Cmd {
	t.Helper()
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	shift := max(0, len(lines)-m.height)
	for y := len(lines) - 1; y >= shift; y-- {
		if i := strings.Index(lines[y], text); i >= 0 {
			x := ansi.StringWidth(lines[y][:i]) + dx
			old := footerPressDelay
			footerPressDelay = 0
			t.Cleanup(func() { footerPressDelay = old })
			_, cmd := m.Update(leftClick(x, y-shift))
			return pressNow(t, m, cmd)
		}
	}
	t.Fatalf("help item %q not visible", text)
	return nil
}

// clickHelpPending clicks a footer item and leaves its action waiting for the
// timer, so tests can look at the pressed screen.
func clickHelpPending(t *testing.T, m *MainModel, text string, dx int) {
	t.Helper()
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	shift := max(0, len(lines)-m.height)
	for y := len(lines) - 1; y >= shift; y-- {
		if i := strings.Index(lines[y], text); i >= 0 {
			x := ansi.StringWidth(lines[y][:i]) + dx
			m.Update(leftClick(x, y-shift))
			return
		}
	}
	t.Fatalf("help item %q not visible", text)
}

// pressNow fires the timer of a pressed footer item on the spot and returns
// the command of the action it runs.
func pressNow(t *testing.T, m *MainModel, cmd tea.Cmd) tea.Cmd {
	t.Helper()
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if _, ok := msg.(footerPressMsg); !ok {
		t.Fatalf("footer press returned %T, want footerPressMsg", msg)
	}
	_, cmd = m.Update(msg)
	return cmd
}

func TestMenuFooterClicks(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {40, 20}, {80, 12}} {
		for _, tc := range []struct {
			text  string
			state AppState
		}{
			{"[s] profile", StateStats}, {"[v] history", StateLeaderboard},
			{"[o] settings", StateSettings}, {"[d] daily", StateBriefing},
			{"[r] random", StateBriefing}, {"[Enter] select", StateBriefing},
		} {
			m := newMenuTestModel(t)
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			clickHelp(t, m, tc.text, 0)
			if m.state != tc.state {
				t.Fatalf("%v %s: got state %v, want %v", size, tc.text, m.state, tc.state)
			}
		}
	}
}

func TestHelpItemKeys(t *testing.T) {
	for _, tc := range []struct {
		item string
		x    int
		want string
	}{
		{"[/] search", 0, "/"}, {"[n/p] next/prev", 1, "n"},
		{"[n/p] next/prev", 3, "p"}, {"[n/p] next/prev", 8, "n"},
		{"[n/N] next/prev", 3, "N"}, {"[h/l ←/→ Tab] category", 2, "tab"},
		{"[Esc/q/Enter] menu", 0, "esc"}, {"[Shift+Tab] back", 0, "shift+tab"},
		{"[any other key] cancel", 0, "esc"},
	} {
		key, ok := helpItemKey(tc.item, tc.x)
		if !ok || key.String() != tc.want {
			t.Fatalf("%s at %d: %q, %v; want %q", tc.item, tc.x, key.String(), ok, tc.want)
		}
	}
	for _, item := range []string{"[↑/↓ j/k] move", "[↑/↓ PgUp/PgDn] scroll", "[unknown] nothing"} {
		if _, ok := helpItemKey(item, 0); ok {
			t.Fatalf("unexpected action for %s", item)
		}
	}
}

func TestFooterBackAcrossScreens(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {40, 20}} {
		for _, state := range []AppState{StateBriefing, StateStats, StateLeaderboard, StateResults, StateSettings, StateAddDrill, StateSetupError} {
			m := newMenuTestModel(t)
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			m.activeDrill = &m.drills[0]
			m.state = state
			m.setupErr = errors.New("unavailable")
			text := "[Esc/q] back"
			switch state {
			case StateStats:
				text = "[Esc/q/Enter] menu"
			case StateResults:
				text = "[Enter/q] menu"
			case StateSettings:
				m.openSettings()
			case StateAddDrill:
				m.initForm()
				text = "[Esc] cancel"
			}
			clickHelp(t, m, text, 0)
			want := StateMenu
			if state == StateSetupError {
				want = StateBriefing
			}
			if m.state != want {
				t.Fatalf("%v state %v: got %v want %v", size, state, m.state, want)
			}
		}
	}
}

func TestFooterClickPromptsAndDisabledMouse(t *testing.T) {
	m := newMenuTestModel(t)
	clickHelp(t, m, "[/] search", 0)
	if !m.search.open {
		t.Fatal("search did not open")
	}
	if _, ok := m.helpKeyAt(3, m.height-2); ok {
		t.Fatal("search has an active footer")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m.Update(runeKey("x"))
	clickHelp(t, m, "[any other key] cancel", 0)
	if m.pendingDelete != nil {
		t.Fatal("delete not cancelled")
	}
	m.Update(runeKey("x"))
	clickHelp(t, m, "[y] yes", 0)
	if m.pendingDelete != nil || !strings.Contains(m.menuNotice, "restore") {
		t.Fatal("hide action not applied")
	}
	m.mouseOff = true
	clickHelp(t, m, "[s] profile", 0)
	if m.state != StateMenu {
		t.Fatal("disabled mouse invoked footer")
	}
}

func TestFooterCategoryAndQuit(t *testing.T) {
	m := newMenuTestModel(t)
	clickHelp(t, m, "[h/l ←/→ Tab] category", 0)
	if m.categoryIndex != 1 {
		t.Fatal("category shortcut did not advance")
	}
	cmd := clickHelp(t, m, "[q] quit", 0)
	if cmd == nil {
		t.Fatal("quit returned no command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("quit shortcut did not quit")
	}
}

func TestFooterAlternativesAndGaps(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	id := m.activeDrill.ID
	clickHelp(t, m, "[n/p] next/prev", 1)
	if m.activeDrill.ID == id {
		t.Fatal("next did not change drill")
	}
	clickHelp(t, m, "[n/p] next/prev", 3)
	if m.activeDrill.ID != id {
		t.Fatal("previous did not return to drill")
	}
	// The two spaces separating items are not buttons.
	lines := strings.Split(ansi.Strip(m.View()), "\n")
	for y, line := range lines {
		if i := strings.Index(line, "[Enter] start"); i >= 0 {
			x := ansi.StringWidth(line[:i]) + len("[Enter] start")
			if _, ok := m.helpKeyAt(x, y); ok {
				t.Fatal("gap is clickable")
			}
		}
	}
}

func TestCountdownFooterCancel(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.activeDrill = &m.drills[0]
	m.state = StateCountdown
	m.countdownCount = 3
	clickHelp(t, m, "[Esc] cancel", 0)
	if m.state != StateBriefing {
		t.Fatal("countdown footer did not cancel")
	}
}

func TestLeaderboardFooterToggle(t *testing.T) {
	m := newMenuTestModel(t)
	m.activeDrill = &m.drills[0]
	m.state = StateLeaderboard
	before := m.leaderboardRecent
	clickHelp(t, m, "[Tab] toggle", 0)
	if m.leaderboardRecent == before {
		t.Fatal("history footer did not toggle")
	}
}

func TestFooterPressFlash(t *testing.T) {
	forceTrueColor(t)
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if strings.Contains(m.View(), "\x1b[7m") {
		t.Fatal("the footer is pressed before any click")
	}
	clickHelpPending(t, m, "[s] profile", 0)
	if m.helpPress == nil || m.helpPress.item != "[s] profile" {
		t.Fatalf("pressed state is %v", m.helpPress)
	}
	if !strings.Contains(m.View(), "\x1b[7m") {
		t.Fatal("the clicked item is not drawn pressed")
	}
	if m.state != StateMenu {
		t.Fatalf("state changed to %v before the timer ended", m.state)
	}
	m.Update(footerPressMsg{})
	if m.state != StateStats {
		t.Fatalf("after the timer: state %v, want profile", m.state)
	}
	if m.helpPress != nil {
		t.Fatal("the pressed state survived its action")
	}
}

func TestFooterPressSuppression(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	clickHelpPending(t, m, "[s] profile", 0)
	// A second click waits; it neither fires nor replaces the first.
	clickHelpPending(t, m, "[o] settings", 0)
	if m.helpPress == nil || m.helpPress.item != "[s] profile" {
		t.Fatalf("the second click replaced the pressed item: %v", m.helpPress)
	}
	if m.state != StateMenu {
		t.Fatalf("the second click acted: state %v", m.state)
	}
	// The wheel waits with everything else.
	if _, cmd := m.Update(wheel(false, 3, 10)); cmd != nil || m.list.Index() != 0 {
		t.Fatal("the wheel was not ignored while an item is pressed")
	}
	// A key press cancels the item and acts as itself.
	m.Update(runeKey("o"))
	if m.state != StateSettings {
		t.Fatalf("the key press was lost: state %v", m.state)
	}
	if m.helpPress != nil {
		t.Fatal("typing did not cancel the pressed item")
	}
}

func TestFooterPressDroppedOnStateChange(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.helpPress = &helpPressState{item: "[Enter/e] editor", key: tea.KeyMsg{Type: tea.KeyEnter}}
	// A background message moves the sprint to evaluation while a footer
	// item on another screen is waiting; the item is dropped so its key
	// cannot misfire on the new screen.
	m.state = StateSprinting
	m.sprintStartTime = time.Now()
	m.Update(sprintFinishedMsg{})
	if m.state != StateEvaluating {
		t.Fatalf("state %v, want evaluating", m.state)
	}
	if m.helpPress != nil {
		t.Fatal("the state change kept the pressed item")
	}
	if _, cmd := m.Update(footerPressMsg{}); cmd != nil {
		t.Fatalf("the dropped item still acts: %v", cmd)
	}
}
