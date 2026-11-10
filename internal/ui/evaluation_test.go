package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/storage"
)

func newEvaluationTestModel(t *testing.T) *MainModel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sound := false
	if err := storage.SaveConfig(storage.Config{Sound: &sound}); err != nil {
		t.Fatal(err)
	}
	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	m.session = &runner.Session{
		Workspace:   t.TempDir(),
		EntryFile:   "solution.c",
		TestCommand: "printf 'current evaluation'",
	}
	if err := os.WriteFile(filepath.Join(m.session.Workspace, m.session.EntryFile), []byte("int student_work;"), 0644); err != nil {
		t.Fatal(err)
	}
	m.state = StateSprinting
	m.sprintStartTime = time.Now().Add(-time.Second)
	return m
}

func TestEvaluationBlocksActionsUntilResultsArrive(t *testing.T) {
	keys := []struct {
		name string
		key  tea.KeyMsg
	}{
		{"enter", tea.KeyMsg{Type: tea.KeyEnter}},
		{"space", tea.KeyMsg{Type: tea.KeySpace}},
		{"submit", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}},
		{"edit", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")}},
		{"forfeit", tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}},
		{"escape", tea.KeyMsg{Type: tea.KeyEsc}},
	}
	for _, tt := range keys {
		t.Run(tt.name, func(t *testing.T) {
			m := newEvaluationTestModel(t)
			m.evalPassed = true
			m.evalOutput = "previous passing result"
			session := m.session
			_, evaluate := m.Update(sprintFinishedMsg{})
			m.evalReadyTime = time.Now().Add(-time.Second)
			_, cmd := m.Update(tt.key)
			if m.state != StateEvaluating || m.session != session || cmd != nil {
				t.Fatal("pending input changed state, replaced the session, or scheduled an action")
			}
			if len(m.history.Attempts) != 0 {
				t.Fatal("recorded an attempt before evaluation completed")
			}
			if _, err := os.Stat(session.Workspace); err != nil {
				t.Fatalf("pending input removed the workspace: %v", err)
			}
			view := m.View()
			if !strings.Contains(view, "Running tests") || strings.Contains(view, "previous passing result") || strings.Contains(view, "TESTS PASSED") {
				t.Fatalf("pending view displays an old result: %s", view)
			}
			m.Update(evaluate())
			if !m.evalPassed || m.evalOutput != "current evaluation" {
				t.Fatal("did not accept the current evaluation result")
			}
			m.evalReadyTime = time.Now().Add(-time.Second)
			m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
			if m.state != StateResults || len(m.history.Attempts) != 1 {
				t.Fatal("could not finish after evaluation completed")
			}
		})
	}
}

func TestEvaluationCommandUsesOriginalSession(t *testing.T) {
	m := newEvaluationTestModel(t)
	session := m.session
	_, evaluate := m.Update(sprintFinishedMsg{})
	m.session = nil
	msg := evaluate().(evalFinishedMsg)
	if !msg.passed || msg.output != "current evaluation" || session.Submits != 1 {
		t.Fatalf("evaluator did not use its original session: %+v", msg)
	}
}

func TestEvaluationIgnoresOlderAndDuplicateResults(t *testing.T) {
	m := newEvaluationTestModel(t)
	_, first := m.Update(sprintFinishedMsg{})
	oldResult := first()
	m.session.TestCommand = "printf 'current failure'; exit 1"
	m.state = StateSprinting
	m.sprintStartTime = time.Now()
	_, second := m.Update(sprintFinishedMsg{})
	m.Update(oldResult)
	if m.evalPassed || m.evalOutput != "" {
		t.Fatal("older evaluation replaced pending results")
	}
	currentResult := second().(evalFinishedMsg)
	m.Update(currentResult)
	if m.evalPassed || m.evalOutput != "current failure" {
		t.Fatal("current evaluation result was not applied")
	}
	m.Update(oldResult)
	if m.evalPassed || m.evalOutput != "current failure" {
		t.Fatal("older result replaced completed results")
	}
	currentResult.passed = true
	currentResult.output = "duplicate result"
	m.Update(currentResult)
	if m.evalPassed || m.evalOutput != "current failure" {
		t.Fatal("duplicate result replaced completed results")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	if m.state != StateSprinting {
		t.Fatal("could not resume after failed evaluation completed")
	}
}

func TestEvaluationLaunchFailureClearsPreviousPass(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.evalPassed = true
	m.evalOutput = "previous passing result"
	m.sprintStartTime = time.Now()
	_, cmd := m.Update(sprintFinishedMsg{err: errors.New("workspace unavailable")})
	if m.evalPending || m.evalPassed || !strings.Contains(m.evalOutput, "workspace unavailable") {
		t.Fatal("launch failure retained a previous result or queued an evaluator")
	}
	// The only command is the one that turns mouse back on. It must not start
	// an evaluation.
	if n := mouseEnableCount(cmd); n != 1 {
		t.Fatalf("launch failure enables mouse %d times, want 1", n)
	}
	for _, msg := range cmdMsgs(cmd) {
		if _, ok := msg.(evalFinishedMsg); ok {
			t.Fatal("launch failure queued an evaluator")
		}
	}
}

func TestEvaluationLaunchFailureWithMouseOffQueuesNothing(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.mouseOff = true
	m.evalPassed = true
	m.sprintStartTime = time.Now()
	_, cmd := m.Update(sprintFinishedMsg{err: errors.New("workspace unavailable")})
	if cmd != nil {
		t.Fatalf("launch failure with mouse off returned a command: %v", cmd())
	}
	if m.evalPending || m.evalPassed {
		t.Fatal("launch failure retained a previous result or queued an evaluator")
	}
}

// When the editor never starts there is no run, so quitting must not record a
// failed attempt or reset the streak.
func TestLaunchFailureRecordsNothing(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.history.CurrentStreak = 4
	m.sprintStartTime = time.Now()
	m.Update(sprintFinishedMsg{err: errors.New("zellij missing")})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if len(m.history.Attempts) != 0 || m.history.CurrentStreak != 4 {
		t.Fatalf("launch failure recorded a run: %+v, streak %d", m.history.Attempts, m.history.CurrentStreak)
	}
	if m.state != StateBriefing || m.session != nil {
		t.Fatalf("expected the briefing with no session, got state %v", m.state)
	}
}
