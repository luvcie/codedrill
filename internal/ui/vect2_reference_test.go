package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/solutions"
)

func TestVect2BriefingReferenceChoice(t *testing.T) {
	m := newBriefingModeModel(t, "rank05-vect2")
	if !strings.Contains(ansi.Strip(m.View()), "Reference: < Header-only >") {
		t.Fatal("header-only is not the default briefing reference")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if !strings.Contains(m.screenKeys(), "[←/→] reference") {
		t.Fatal("focused reference has no arrow key hint")
	}
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.briefingReference != solutions.SplitFiles || m.briefingMode != mode.Study() {
		t.Fatal("reference choice changed the practice mode")
	}
	m.Update(runeKey("2"))
	m.openBriefing(m.activeDrill)
	if m.briefingReference != solutions.SplitFiles || m.briefingMode != mode.Recall() {
		t.Fatal("reference and mode were not remembered")
	}
	y := rowOf(t, m, "Reference: < Split files >")
	m.Update(leftClick(columnOf(t, m, y, "Reference:"), y))
	if m.briefingReference != solutions.HeaderOnly || m.briefingMode != mode.Recall() {
		t.Fatal("clicking the reference choice did not switch references independently")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.briefingReference != solutions.SplitFiles || m.briefingMode != mode.Recall() {
		t.Fatal("arrow key did not switch the focused reference")
	}
	reopened, err := NewMainModelWithTarget(m.activeDrill.ID)
	if err != nil || reopened.briefingReference != solutions.SplitFiles {
		t.Fatalf("reference choice did not survive a new model: %v", err)
	}
	m.openBriefing(m.stepDrill(*m.activeDrill, 1))
	if m.hasReferenceChoice() || strings.Contains(ansi.Strip(m.View()), "Reference: <") {
		t.Fatal("reference choice leaked into another drill")
	}
}

func TestVect2EvaluationUsesRunReference(t *testing.T) {
	for _, selected := range []solutions.Variant{solutions.HeaderOnly, solutions.SplitFiles} {
		m := newBriefingModeModel(t, "rank05-vect2")
		m.briefingReference = selected
		m.briefingMode = mode.Recall()
		m.triggerSprintStart()
		m.launchSprintSession()
		if m.session == nil {
			t.Fatalf("session did not start: %v", m.setupErr)
		}
		t.Cleanup(m.session.Cleanup)
		m.session.TestCommand = "true"
		source, err := solutions.ResolveVariant(m.activeDrill.ID, "", selected)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range source.Files {
			data, err := file.Read()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(m.session.Workspace, file.Rel), data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		m.briefingReference = solutions.HeaderOnly
		if selected == solutions.HeaderOnly {
			m.briefingReference = solutions.SplitFiles
		}
		if err := mode.SavePreferences(m.activeDrill.ID, mode.Preferences{Options: mode.Exam(), ReferenceVariant: m.briefingReference}); err != nil {
			t.Fatal(err)
		}
		msg := runEvaluation(t, m)
		if msg.referenceErr != nil || msg.reference == nil || !msg.reference.Exact || !m.passVerdict().Passed {
			t.Fatalf("run reference was replaced by briefing or disk preferences: %+v", msg)
		}
		m.openReferenceDiff()
		if m.referenceDiff == nil {
			t.Fatal("completed Recall comparison did not enable review")
		}
	}
}
