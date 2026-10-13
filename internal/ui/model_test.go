package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestModelConstructors(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	m, err := NewMainModel()
	if err != nil {
		t.Fatalf("NewMainModel failed: %v", err)
	}
	if m.state != StateMenu {
		t.Errorf("expected initial state StateMenu, got %v", m.state)
	}
	if len(m.drills) == 0 {
		t.Errorf("expected loaded drills, got 0")
	}

	targetM, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatalf("NewMainModelWithTarget(ft_strlen) failed: %v", err)
	}
	if targetM.state != StateBriefing {
		t.Errorf("expected state StateBriefing, got %v", targetM.state)
	}
	if targetM.activeDrill == nil || targetM.activeDrill.ID != "ft_strlen" {
		t.Errorf("expected active drill ft_strlen, got %v", targetM.activeDrill)
	}

	_, err = NewMainModelWithTarget("non-existent-drill-xyz")
	if err == nil {
		t.Errorf("expected error for non-existent drill, got nil")
	}

	randM, err := NewMainModelWithRandomFiltered("")
	if err != nil {
		t.Fatalf("NewMainModelWithRandomFiltered with no filter failed: %v", err)
	}
	if randM.state != StateBriefing {
		t.Errorf("expected state StateBriefing, got %v", randM.state)
	}
	if randM.activeDrill == nil {
		t.Errorf("expected non-nil active drill in random model")
	}

	randFilteredM, err := NewMainModelWithRandomFiltered("c")
	if err != nil {
		t.Fatalf("NewMainModelWithRandomFiltered(c) failed: %v", err)
	}
	if !strings.EqualFold(randFilteredM.activeDrill.Language, "c") {
		t.Errorf("expected active drill language to be c, got %s", randFilteredM.activeDrill.Language)
	}

	randFilteredPy, err := NewMainModelWithRandomFiltered("python")
	if err != nil {
		t.Fatalf("NewMainModelWithRandomFiltered(python) failed: %v", err)
	}
	if !strings.EqualFold(randFilteredPy.activeDrill.Language, "python") {
		t.Errorf("expected active drill language to be python, got %s", randFilteredPy.activeDrill.Language)
	}

	randFilteredCat, err := NewMainModelWithRandomFiltered("LeetCode")
	if err != nil {
		t.Fatalf("NewMainModelWithRandomFiltered(LeetCode) failed: %v", err)
	}
	if !strings.EqualFold(randFilteredCat.activeDrill.Category, "LeetCode") {
		t.Errorf("expected active drill category to be LeetCode, got %s", randFilteredCat.activeDrill.Category)
	}

	randFilteredMed, err := NewMainModelWithRandomFiltered("medium")
	if err != nil {
		t.Fatalf("NewMainModelWithRandomFiltered(medium) failed: %v", err)
	}
	if !strings.EqualFold(randFilteredMed.activeDrill.Difficulty, "medium") {
		t.Errorf("expected active drill difficulty to be medium, got %s", randFilteredMed.activeDrill.Difficulty)
	}

	_, err = NewMainModelWithRandomFiltered("non-existent-filter-xyz")
	if err == nil {
		t.Errorf("expected error for non-existent filter, got nil")
	}

	addM, err := NewMainModelWithAddDrill()
	if err != nil {
		t.Fatalf("NewMainModelWithAddDrill failed: %v", err)
	}
	if addM.state != StateAddDrill {
		t.Errorf("expected state StateAddDrill, got %v", addM.state)
	}
	if addM.form == nil {
		t.Errorf("expected initialized form in addM")
	}
	if cmd := addM.Init(); cmd == nil {
		t.Errorf("expected non-nil tea.Cmd from Init() in StateAddDrill")
	}

	dailyM, err := NewMainModelWithDaily()
	if err != nil {
		t.Fatalf("NewMainModelWithDaily failed: %v", err)
	}
	if dailyM.state != StateBriefing {
		t.Errorf("expected state StateBriefing, got %v", dailyM.state)
	}
	if dailyM.activeDrill == nil || !dailyM.isDaily(*dailyM.activeDrill) {
		t.Errorf("expected today's daily drill, got %v", dailyM.activeDrill)
	}
}

func TestModelTransitions(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	m, err := NewMainModel()
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(*MainModel)
	if m.width != 100 || m.height != 40 {
		t.Errorf("window size not updated: %d x %d", m.width, m.height)
	}

	m.width = 80
	menu80 := m.View()
	if !strings.Contains(menu80, "CODEDRILL") || !strings.Contains(menu80, "LVL") {
		t.Errorf("expected CODEDRILL and LVL in 80-col menu view, got: %s", menu80)
	}

	m.width = 120
	menu120 := m.View()
	if !strings.Contains(menu120, "CODEDRILL") || !strings.Contains(menu120, "LVL") {
		t.Errorf("expected CODEDRILL and LVL in 120-col menu view, got: %s", menu120)
	}
	m.width = 100

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(*MainModel)
	if m.state != StateStats {
		t.Errorf("expected StateStats after pressing 's', got %v", m.state)
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(*MainModel)
	if m.state != StateMenu {
		t.Errorf("expected StateMenu after pressing Esc in Stats, got %v", m.state)
	}

	initialCat := m.categoryFilter
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(*MainModel)
	if m.categoryFilter == initialCat && len(m.categories) > 1 {
		t.Errorf("expected category to change after Tab")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = updated.(*MainModel)
	if m.categoryFilter != initialCat {
		t.Errorf("expected category to return to %s after Shift+Tab, got %s", initialCat, m.categoryFilter)
	}

	briefingM, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatalf("failed to create briefing model: %v", err)
	}
	view := briefingM.View()
	if !strings.Contains(view, "ft_strlen") {
		t.Errorf("expected drill title in briefing view, got: %s", view)
	}
	if !strings.Contains(view, "LVL") {
		t.Errorf("expected LVL badge in briefing view, got: %s", view)
	}

	briefingM.width = 60
	view60 := briefingM.View()
	if !strings.Contains(view60, "ft_strlen") {
		t.Errorf("expected drill title in 60-col briefing view")
	}

	briefingM.width = 120
	view120 := briefingM.View()
	if !strings.Contains(view120, "ft_strlen") {
		t.Errorf("expected drill title in 120-col briefing view")
	}
}

func TestResultsScreenNavigation(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	m.state = StateResults
	m.resultDrill = *m.activeDrill
	m.lastDuration = 2500 * time.Millisecond
	m.lastSubmits = 1
	m.isPB = true

	view := m.View()
	if !strings.Contains(view, "SPRINT FINISHED") {
		t.Errorf("expected SPRINT FINISHED in results view, got: %s", view)
	}
	if !strings.Contains(view, "(first try)") {
		t.Errorf("expected the first-try note in results view, got: %s", view)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	rm := updated.(*MainModel)
	if rm.state != StateCountdown || rm.countdownCount != 3 {
		t.Errorf("expected StateCountdown with count 3 on 'r', got state %v count %d", rm.state, rm.countdownCount)
	}

	m.state = StateResults
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	nm := updated.(*MainModel)
	if nm.state != StateCountdown || nm.countdownCount != 3 {
		t.Errorf("expected StateCountdown with count 3 on 'n', got state %v count %d", nm.state, nm.countdownCount)
	}

	m.state = StateResults
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	lm := updated.(*MainModel)
	if lm.state != StateLeaderboard {
		t.Errorf("expected StateLeaderboard on 'l', got state %v", lm.state)
	}

	m.state = StateResults
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	bm := updated.(*MainModel)
	if bm.state != StateBriefing || bm.previousState != StateResults {
		t.Errorf("expected StateBriefing with previousState StateResults on 'b', got state %v prev %v", bm.state, bm.previousState)
	}
	escUpdated, _ := bm.Update(tea.KeyMsg{Type: tea.KeyEscape})
	escm := escUpdated.(*MainModel)
	if escm.state != StateResults {
		t.Errorf("expected StateResults when pressing esc from briefing, got state %v", escm.state)
	}

	m.state = StateResults
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	qm := updated.(*MainModel)
	if qm.state != StateMenu {
		t.Errorf("expected StateMenu on enter, got state %v", qm.state)
	}
}

func TestBriefingKeyNavigation(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	initialID := m.activeDrill.ID
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	nextM := updated.(*MainModel)
	if nextM.state != StateBriefing {
		t.Errorf("expected state StateBriefing on 'n', got %v", nextM.state)
	}
	if nextM.activeDrill == nil {
		t.Errorf("expected non-nil active drill on 'n'")
	}
	if len(nextM.drills) > 1 && nextM.activeDrill.ID == initialID {
		t.Errorf("expected active drill to change on 'n'")
	}

	updated, _ = nextM.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
	lm := updated.(*MainModel)
	if lm.state != StateLeaderboard {
		t.Errorf("expected StateLeaderboard on 'l', got %v", lm.state)
	}

	lm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if lm.state != StateBriefing {
		t.Fatalf("expected the briefing after Esc from its leaderboard, got %v", lm.state)
	}
	lm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if lm.state != StateMenu {
		t.Errorf("expected StateMenu on Esc, got %v", lm.state)
	}
}

func TestCountdownKeyNavigation(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	m.state = StateCountdown
	m.countdownCount = 3

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	cancelled := updated.(*MainModel)
	if cancelled.state != StateBriefing {
		t.Errorf("expected StateBriefing on Esc in countdown, got %v", cancelled.state)
	}

	m.state = StateCountdown
	m.countdownCount = 2
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	skipped := updated.(*MainModel)
	if skipped.state != StateSprinting {
		t.Errorf("expected StateSprinting after skipping countdown with space, got %v", skipped.state)
	}
	if cmd == nil {
		t.Errorf("expected non-nil tea.Cmd launching sprint")
	}
	if skipped.session != nil {
		skipped.session.Cleanup()
	}

	zero := 0
	if err := storage.SaveConfig(storage.Config{Countdown: &zero}); err != nil {
		t.Fatal(err)
	}
	instantM, _ := NewMainModelWithTarget("ft_strlen")
	updated, _ = instantM.Update(tea.KeyMsg{Type: tea.KeyEnter})
	started := updated.(*MainModel)
	if started.state != StateSprinting {
		t.Errorf("expected immediate StateSprinting when countdown is 0, got %v", started.state)
	}
	if started.session != nil {
		started.session.Cleanup()
	}
}

func TestLeaderboardKeyNavigation(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	m.state = StateLeaderboard
	m.previousState = StateBriefing
	m.leaderboardRecent = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	lm := updated.(*MainModel)
	if !lm.leaderboardRecent {
		t.Errorf("expected leaderboardRecent to be true after tab")
	}

	updated, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	lm = updated.(*MainModel)
	if lm.leaderboardRecent {
		t.Errorf("expected leaderboardRecent to be false after 't'")
	}

	lm.leaderboardRecent = true
	updated, _ = lm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	returned := updated.(*MainModel)
	if returned.state != StateBriefing {
		t.Errorf("expected StateBriefing on Esc, got %v", returned.state)
	}
	if returned.leaderboardRecent {
		t.Errorf("expected leaderboardRecent to be reset to false on Esc")
	}

	lm.state = StateLeaderboard
	lm.previousState = StateResults
	updated, _ = lm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if updated.(*MainModel).state != StateResults {
		t.Errorf("expected StateResults on Esc from results-opened leaderboard")
	}

	lm.state = StateLeaderboard
	curDrillID := lm.activeDrill.ID
	updated, _ = lm.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	nextM := updated.(*MainModel)
	if nextM.state != StateLeaderboard {
		t.Errorf("expected StateLeaderboard on 'n' from leaderboard, got state %v", nextM.state)
	}
	if nextM.activeDrill.ID == curDrillID && len(nextM.drills) > 1 {
		t.Errorf("expected active drill to change on 'n'")
	}

	updated, _ = nextM.Update(tea.KeyMsg{Type: tea.KeyEnter})
	enteredM := updated.(*MainModel)
	if enteredM.state != StateCountdown || enteredM.countdownCount != 3 {
		t.Errorf("expected StateCountdown with count 3 on Enter from leaderboard, got state %v count %d", enteredM.state, enteredM.countdownCount)
	}
}

func TestRenderScreensVisual(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 35})
	m = updated.(*MainModel)

	briefingView := m.View()
	if !strings.Contains(briefingView, "DRILL: ft_strlen") || !strings.Contains(briefingView, "S-RANK") {
		t.Errorf("briefing view missing expected markers: %s", briefingView)
	}

	m.state = StateResults
	m.resultDrill = *m.activeDrill
	m.lastDuration = 3200 * time.Millisecond
	m.isPB = true
	m.pbDelta = 500 * time.Millisecond
	m.lastSubmits = 1
	m.earnedXP = 125
	m.levelInfo = storage.GetLevelInfo(125)
	resultsView := m.View()
	if !strings.Contains(resultsView, "SPRINT FINISHED") || !strings.Contains(resultsView, "+125 XP") {
		t.Errorf("results view missing expected markers: %s", resultsView)
	}

	m.state = StateMenu
	menuView := m.View()
	if !strings.Contains(menuView, "CODEDRILL") || !strings.Contains(menuView, "LVL") {
		t.Errorf("menu view missing expected markers: %s", menuView)
	}
}

func TestModelCleanup(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatalf("failed to create model: %v", err)
	}

	m.Cleanup()
	if m.session != nil {
		t.Errorf("expected session to remain nil")
	}

	m.state = StateCountdown
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	started := updated.(*MainModel)
	if started.session == nil {
		t.Fatalf("expected active session created")
	}
	ws := started.session.Workspace
	if _, err := os.Stat(ws); os.IsNotExist(err) {
		t.Fatalf("expected workspace directory to exist: %s", ws)
	}

	started.Cleanup()
	if started.session != nil {
		t.Errorf("expected session to be nil after Cleanup()")
	}
	if _, err := os.Stat(ws); !os.IsNotExist(err) {
		t.Errorf("expected workspace directory to be deleted after Cleanup(): %s", ws)
	}
}

func TestFormatEvalOutput(t *testing.T) {
	if strings.Contains(formatEvalOutput("ok\tpackage\t0.01s", 80), "\t") {
		t.Fatal("raw tabs can move terminal borders after layout")
	}
	short := "ALL TESTS PASSED\n"
	res := formatEvalOutput(short, 80)
	if !strings.Contains(res, "ALL TESTS PASSED") {
		t.Errorf("expected output to contain text")
	}

	longText := ""
	for i := 1; i <= 30; i++ {
		longText += fmt.Sprintf("Error line %d: syntax failure\n", i)
	}
	output := formatEvalOutput(longText, 80)
	for i := 1; i <= 30; i++ {
		if !strings.Contains(output, fmt.Sprintf("Error line %d:", i)) {
			t.Errorf("missing diagnostic %d from scrollable output", i)
		}
	}
}

// Picking a drill from the menu must reset where the briefing's Esc goes, or
// it can reopen an old results screen.
func TestBriefingFromMenuReturnsToMenu(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMainModel()
	if err != nil {
		t.Fatal(err)
	}
	m.previousState = StateResults
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateBriefing {
		t.Fatalf("enter on a drill: state %v", m.state)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateMenu {
		t.Fatalf("Esc from a menu briefing went to %v", m.state)
	}
}

// Browsing other drills from the results screen must not change the drill the
// results describe.
func TestResultsKeepTheirDrill(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m.state = StateResults
	m.resultDrill = *m.activeDrill
	m.lastDuration = 3 * time.Second
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	if m.activeDrill.ID == "ft_strlen" {
		t.Fatal("n in the briefing did not move to another drill")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateResults || m.resultDrill.ID != "ft_strlen" {
		t.Fatalf("results now describe %s (state %v)", m.resultDrill.ID, m.state)
	}
}

func TestCancelledCountdownTicksAreIgnored(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	stale := countdownTickMsg{id: m.countdownID}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(stale)
	if m.countdownCount != 3 {
		t.Fatalf("a tick from the cancelled countdown advanced the new one to %d", m.countdownCount)
	}
}
