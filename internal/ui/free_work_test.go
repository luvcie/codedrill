package ui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestFreeWorkSetupAndCancel(t *testing.T) {
	m := newMenuTestModel(t)
	if !strings.Contains(ansi.Strip(m.View()), "[w] Free Work") {
		t.Fatal("Free Work missing from main menu")
	}
	press(t, m, runeKey("w"))
	if m.state != StateFreeWorkSetup || m.form.GetFocusedField().GetValue() != 2 {
		t.Fatal("Free Work did not open with two panes")
	}
	if !strings.Contains(ansi.Strip(m.View()), "blank = home ~") {
		t.Fatal("home default is not visible")
	}
	press(t, m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateMenu || m.form != nil || m.freeWork != nil {
		t.Fatal("cancel did not return to menu")
	}
}

func TestFreeWorkReturnsToMenuWithoutRecording(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{{"closed", nil}, {"launch failed", errors.New("launch failed")}} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenuTestModel(t)
			t.Cleanup(m.Cleanup)
			home := t.TempDir()
			t.Setenv("HOME", home)
			bin := t.TempDir()
			if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			if err := storage.SaveConfig(storage.Config{Shell: "bash", Music: &storage.MusicSettings{Enabled: true}}); err != nil {
				t.Fatal(err)
			}
			before, _ := json.Marshal(m.history)
			m.accumulatedTime = time.Second
			press(t, m, runeKey("w"))
			press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
			press(t, m, tea.KeyMsg{Type: tea.KeyEnter})
			if m.state != StateFreeWorking || m.freeWork == nil || m.session != nil || m.musicPlayer == nil {
				t.Fatalf("Free Work did not launch independently: state %v", m.state)
			}
			tempDir := filepath.Dir(m.freeWork.MusicSocket())
			m.Update(freeWorkFinishedMsg{tc.err})
			after, _ := json.Marshal(m.history)
			if m.state != StateMenu || m.freeWork != nil || m.musicPlayer != nil || m.accumulatedTime != time.Second || string(before) != string(after) {
				t.Fatal("Free Work affected drill state or failed to clean up")
			}
			if _, err := os.Stat(tempDir); !os.IsNotExist(err) {
				t.Fatalf("temporary session files survived: %v", err)
			}
		})
	}
}
