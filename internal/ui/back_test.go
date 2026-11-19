package ui

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/solutions"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestBackReturnsToBriefingWithoutEvaluationOrSaving(t *testing.T) {
	for _, opts := range []mode.Options{mode.Study(), mode.Copy(), mode.Exam()} {
		for _, variant := range []solutions.Variant{solutions.HeaderOnly, solutions.SplitFiles} {
			t.Run(fmt.Sprintf("%s/exact=%t/%s", modePresetName(opts), opts.RequireExact, variant), func(t *testing.T) {
				m := newBriefingModeModel(t, "rank05-vect2")
				bin := t.TempDir()
				if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
				m.briefingReference = variant
				m.setBriefingOptions(m.activeDrill, opts)
				m.saveBriefingMode()
				if _, _, err := m.history.RecordAttempt(m.activeDrill.ID, storage.RecallMode, time.Second, true, 1, false); err != nil {
					t.Fatal(err)
				}
				before, err := json.Marshal(m.history)
				if err != nil {
					t.Fatal(err)
				}
				m.triggerSprintStart()
				m.launchSprintSession()
				if m.session == nil {
					t.Fatalf("session did not start: %v", m.setupErr)
				}
				session := m.session
				t.Cleanup(session.Cleanup)
				t.Setenv("ZELLIJ_SESSION_NAME", session.SessionName)
				if out, err := exec.Command(filepath.Join(session.Workspace, "back")).CombinedOutput(); err != nil {
					t.Fatalf("back failed: %v: %s", err, out)
				}
				// Return to the run's selections even if the briefing changed.
				m.briefingMode = mode.Exam()
				m.briefingReference = solutions.HeaderOnly
				m.accumulatedTime = time.Minute
				_, cmd := m.Update(sprintFinishedMsg{zellij: true, err: errors.New("session closed")})
				for _, msg := range cmdMsgs(cmd) {
					if _, ok := msg.(evalFinishedMsg); ok {
						t.Fatal("back scheduled evaluation")
					}
				}
				if m.state != StateBriefing || m.session != nil || m.activeDrill.ID != "rank05-vect2" {
					t.Fatalf("back did not clean up and return to the same briefing: state %v", m.state)
				}
				if m.briefingMode != opts || m.briefingReference != variant || m.accumulatedTime != 0 {
					t.Fatalf("back lost selections or retained run time: %+v, %q, %v", m.briefingMode, m.briefingReference, m.accumulatedTime)
				}
				if session.Submits != 0 || m.evalPending || m.evalRef != nil || m.evalReview != nil || m.referenceDiff != nil {
					t.Fatal("back evaluated or retained an evaluation")
				}
				if _, err := os.Stat(session.Workspace); !os.IsNotExist(err) {
					t.Fatalf("back left its workspace: %v", err)
				}
				saved, err := storage.LoadHistory()
				if err != nil {
					t.Fatal(err)
				}
				after, err := json.Marshal(saved)
				inMemory, memoryErr := json.Marshal(m.history)
				if err != nil || memoryErr != nil || string(before) != string(after) || string(before) != string(inMemory) {
					t.Fatalf("back changed saved history: %v", err)
				}
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				if m.state != StateCountdown || m.runMode != opts || m.runReference != variant {
					t.Fatal("new run lost the returned briefing selections")
				}
			})
		}
	}
}
