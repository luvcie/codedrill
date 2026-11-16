package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestFinishedRunSavesItsMode(t *testing.T) {
	for _, tc := range []struct {
		name string
		opts mode.Options
	}{
		{"exam", mode.Exam()},
		{"recall", mode.Recall()},
		{"copy", mode.Copy()},
		{"study", mode.Study()},
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
			if len(saved.Attempts) != 1 || saved.Attempts[0].Mode != tc.name {
				t.Fatalf("saved run has no %s mode: %s", tc.name, data)
			}
		})
	}
}

func TestExamRemainsTheMainMenuRecord(t *testing.T) {
	m := newEvaluationTestModel(t)
	for _, tc := range []struct {
		opts mode.Options
		time time.Duration
	}{
		{mode.Exam(), 20 * time.Second}, {mode.Study(), time.Second},
		{mode.Copy(), 2 * time.Second}, {mode.Recall(), 3 * time.Second},
	} {
		m.runMode = tc.opts
		m.accumulatedTime = tc.time
		m.recordPass()
		if !m.isPB || m.pbDelta != 0 {
			t.Fatalf("first %s run competed against another mode", mode.RecordLabel(tc.opts.RecordMode()))
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
	if !strings.Contains(m.View(), "Exam") {
		t.Fatal("history did not start in Exam")
	}
	m.updateLeaderboard(tea.KeyMsg{Type: tea.KeyRight})
	m.updateLeaderboard(tea.KeyMsg{Type: tea.KeyRight})
	if view := m.View(); !strings.Contains(view, "Copy") || !strings.Contains(view, "2.000s") || strings.Contains(view, "20.000s") {
		t.Fatalf("Copy leaderboard mixes modes: %s", view)
	}
}

func TestSprintAndBriefingUseSelectedModePB(t *testing.T) {
	m := newEvaluationTestModel(t)
	for category, duration := range map[storage.RecordMode]time.Duration{storage.ExamMode: 20 * time.Second, storage.CopyMode: 5 * time.Second} {
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
		t.Fatalf("sprint pace does not use Copy PB: %s %v", pace, err)
	}
	banner, err := os.ReadFile(filepath.Join(m.session.Workspace, "start-terminal.sh"))
	if err != nil || !strings.Contains(string(banner), "PB_LABEL='Copy PB'") {
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

func TestForfeitKeepsTheRunsCapturedMode(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.runMode = mode.Copy()
	if err := mode.Save(m.activeDrill.ID, mode.Exam()); err != nil {
		t.Fatal(err)
	}
	m.accumulatedTime = 10 * time.Second
	m.recordForfeit()
	if len(m.history.Attempts) != 1 || m.history.Attempts[0].Mode != storage.CopyMode || m.history.Attempts[0].Passed {
		t.Fatalf("forfeit lost the run's mode: %+v", m.history.Attempts)
	}
	if m.history.GetPB(m.activeDrill.ID, storage.CopyMode) != 0 || m.history.GetModeStats(m.activeDrill.ID, storage.ExamMode) != nil {
		t.Fatal("failed run changed a PB or used the newly saved preferences")
	}
}
