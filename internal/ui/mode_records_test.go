package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestHistoryCategorySelectorIncludesAll(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	for category, duration := range map[storage.RecordMode]time.Duration{
		storage.ExamMode: 20 * time.Second, storage.StudyMode: time.Second,
	} {
		if _, _, err := m.history.RecordAttempt(m.activeDrill.ID, category, duration, true, 1, false); err != nil {
			t.Fatal(err)
		}
	}
	for _, from := range []AppState{StateMenu, StateBriefing, StateResults} {
		m.leaderboardMode = 2
		m.runMode = mode.Copy()
		m.openLeaderboard(from)
		if view := ansi.Strip(m.contentView()); !strings.Contains(view, "All   [") || !strings.Contains(view, "20.000s") || !strings.Contains(view, "1.000s") {
			t.Fatalf("history opened from %v did not start in All:\n%s", from, view)
		}
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if view := ansi.Strip(m.contentView()); !strings.Contains(view, "All   [") || !strings.Contains(view, "recent runs") {
		t.Fatalf("Tab left the All category:\n%s", view)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if view := ansi.Strip(m.contentView()); !strings.Contains(view, "Exam   [") || strings.Contains(view, "10.000s") || strings.Contains(view, "1.000s") {
		t.Fatalf("right from All did not select Exam:\n%s", view)
	}
	m.Update(runeKey("n"))
	if view := ansi.Strip(m.contentView()); !strings.Contains(view, "All   [") {
		t.Fatalf("switching drills did not default to All:\n%s", view)
	}
}

func TestTrueAllShowsEveryCompletedRun(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	id := m.activeDrill.ID
	for seconds := 21; seconds <= 30; seconds++ {
		m.history.Attempts = append(m.history.Attempts, storage.Attempt{
			DrillID: id, Mode: storage.ExamMode, Duration: time.Duration(seconds) * time.Second, Passed: true,
		})
	}
	m.history.Attempts = append(m.history.Attempts,
		storage.Attempt{DrillID: id, Mode: storage.ExamMode, Duration: 567 * time.Millisecond},
		storage.Attempt{DrillID: id, Mode: storage.StudyMode, Duration: 40 * time.Second, Passed: true},
	)
	m.openLeaderboard(StateBriefing)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 16})
	if view := ansi.Strip(m.contentView()); strings.Contains(view, "30.000s") {
		t.Fatalf("All should still limit its tables to eight runs:\n%s", view)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assertRuns := func(first, last string) {
		t.Helper()
		view := ansi.Strip(m.contentView())
		_, rows, ok := strings.Cut(view, "──")
		if !ok || !strings.Contains(view, "True All") || !strings.Contains(view, "runs 10") ||
			!strings.Contains(rows, "40.000s") || strings.Contains(rows, "0.567s") {
			t.Fatalf("True All should show completed runs from both modes:\n%s", view)
		}
		if start, end := strings.Index(rows, first), strings.Index(rows, last); start < 0 || end <= start {
			t.Fatalf("expected %q before %q:\n%s", first, last, view)
		}
	}
	assertRuns("21.000s", "30.000s")
	m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if view := ansi.Strip(m.View()); !strings.Contains(view, "30.000s") {
		t.Fatalf("could not scroll to slower runs:\n%s", view)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assertRuns("30.000s", "21.000s")
}

func TestFinishedRunSavesItsMode(t *testing.T) {
	for _, tc := range []struct {
		name     string
		opts     mode.Options
		category storage.RecordMode
	}{
		{"exam", mode.Exam(), storage.ExamMode},
		{"hidden-copy", mode.Recall(), storage.StudyMode},
		{"study-copy", mode.Copy(), storage.StudyMode},
		{"study", mode.Study(), storage.StudyMode},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.runMode = tc.opts
			m.accumulatedTime = 20 * time.Second
			m.recordPass()
			data, err := json.Marshal(m.history)
			if err != nil {
				t.Fatal(err)
			}
			var saved struct {
				Attempts []struct{ Mode string }
			}
			if err := json.Unmarshal(data, &saved); err != nil {
				t.Fatal(err)
			}
			if len(saved.Attempts) != 1 || saved.Attempts[0].Mode != string(tc.category) {
				t.Fatalf("saved %s run has no %s category: %s", tc.name, tc.category, data)
			}
		})
	}
}

func TestExamRemainsTheMainMenuRecord(t *testing.T) {
	m := newEvaluationTestModel(t)
	for _, tc := range []struct {
		opts  mode.Options
		time  time.Duration
		isPB  bool
		delta time.Duration
	}{
		{mode.Exam(), 20 * time.Second, true, 0}, {mode.Study(), time.Second, true, 0},
		{mode.Copy(), 2 * time.Second, false, -time.Second}, {mode.Recall(), 3 * time.Second, false, -2 * time.Second},
	} {
		m.runMode = tc.opts
		m.accumulatedTime = tc.time
		m.recordPass()
		if m.isPB != tc.isPB || m.pbDelta != tc.delta {
			t.Fatalf("%s record: PB=%v delta=%v, want %v %v", mode.RecordLabel(tc.opts.RecordMode()), m.isPB, m.pbDelta, tc.isPB, tc.delta)
		}
		for _, item := range m.list.Items() {
			if row, ok := item.(drillItem); ok && row.drill.ID == m.activeDrill.ID {
				if row.pb != 20*time.Second || !strings.Contains(row.Description(), "Exam PB") {
					t.Fatalf("practice changed the main Exam record: %+v", row)
				}
			}
		}
	}
	if got := m.history.GetPB(m.activeDrill.ID, storage.ExamMode); got != 20*time.Second {
		t.Fatalf("Exam PB = %v after faster practice runs", got)
	}
	m.openLeaderboard(StateMenu)
	if !strings.Contains(m.View(), "All   [") {
		t.Fatal("history did not start in All")
	}
	m.updateLeaderboard(tea.KeyMsg{Type: tea.KeyRight})
	m.updateLeaderboard(tea.KeyMsg{Type: tea.KeyRight})
	if view := m.View(); !strings.Contains(view, "Study") || !strings.Contains(view, "1.000s") || !strings.Contains(view, "2.000s") || strings.Contains(view, "20.000s") {
		t.Fatalf("Study leaderboard did not group both variants: %s", view)
	}
}

func TestSprintAndBriefingUseSelectedModePB(t *testing.T) {
	m := newEvaluationTestModel(t)
	for category, duration := range map[storage.RecordMode]time.Duration{storage.ExamMode: 20 * time.Second, storage.StudyMode: 5 * time.Second} {
		if _, _, err := m.history.RecordAttempt(m.activeDrill.ID, category, duration, true, 1, false); err != nil {
			t.Fatal(err)
		}
	}
	m.briefingMode = mode.Copy()
	if view := m.viewBriefing(); !strings.Contains(view, "PB 5.000s") || !strings.Contains(view, "Copy") || strings.Contains(view, "20.000s") {
		t.Fatalf("briefing compares against another mode: %s", view)
	}
	m.runMode = mode.Copy()
	m.launchSprintSession()
	if m.session == nil {
		t.Fatalf("session did not start: %v", m.setupErr)
	}
	t.Cleanup(m.session.Cleanup)
	pace, err := os.ReadFile(filepath.Join(m.session.Workspace, "pace"))
	if err != nil || !strings.Contains(string(pace), "PB_MS='5000'") {
		t.Fatalf("sprint pace does not use the shared Study PB: %s %v", pace, err)
	}
	banner, err := os.ReadFile(filepath.Join(m.session.Workspace, "start-terminal.sh"))
	if err != nil || !strings.Contains(string(banner), "PB_LABEL='Study PB'") {
		t.Fatalf("sprint does not identify the PB category: %s %v", banner, err)
	}
}

func TestCustomConfigurationsHaveSeparatePBs(t *testing.T) {
	m := newEvaluationTestModel(t)
	visible, hidden := mode.Copy(), mode.Recall()
	visible.RequireTests, hidden.RequireTests = true, true
	for i, opts := range []mode.Options{visible, hidden} {
		m.runMode = opts
		m.accumulatedTime = time.Duration(i+1) * time.Second
		m.recordPass()
		if !m.isPB || m.pbDelta != 0 {
			t.Fatal("custom configurations shared their first PB")
		}
	}
	if m.history.GetPB(m.activeDrill.ID, visible.RecordMode()) != time.Second || m.history.GetPB(m.activeDrill.ID, hidden.RecordMode()) != 2*time.Second {
		t.Fatal("reference visibility did not separate custom PBs")
	}
	if m.history.GetPB(m.activeDrill.ID, storage.ExamMode) != 0 {
		t.Fatal("custom PB appeared in Exam")
	}
}

func TestDiscardUnfinishedRunLeavesHistoryAndStreak(t *testing.T) {
	m := newEvaluationTestModel(t)
	id := m.activeDrill.ID
	if _, _, err := m.history.RecordAttempt(id, storage.ExamMode, time.Second, true, 1, false); err != nil {
		t.Fatal(err)
	}
	m.runMode = mode.Exam()
	m.state = StateEvaluating
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateMenu || m.session != nil || len(m.history.Attempts) != 1 {
		t.Fatalf("discard did not clean up the session or changed history: state %v, attempts %+v", m.state, m.history.Attempts)
	}
	history, err := storage.LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Attempts) != 1 || history.CurrentStreak != 1 || history.GetPB(id, storage.ExamMode) != time.Second {
		t.Fatalf("discard changed saved progress: %+v", history)
	}
}
