package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/solutions"
	"github.com/luvcie/codedrill/internal/storage"
)

func resetTestModel(t *testing.T) (*MainModel, string) {
	t.Helper()
	m := settingsTestModel(t)
	dir, err := storage.ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := drill.SaveDrill(drill.Drill{ID: "private-reset-test", Title: "Private", EntryFile: "answer.c"}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := m.history.RecordAttempt("private-reset-test", time.Second, true, 1, false); err != nil {
		t.Fatal(err)
	}
	if err := storage.SaveConfig(storage.Config{Theme: "nord", Shell: "zsh"}); err != nil {
		t.Fatal(err)
	}
	if err := mode.Save("exam05-vect2", mode.Copy()); err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(runeKey("o"))
	if !strings.Contains(ansi.Strip(m.View()), "Factory Reset") {
		t.Fatal("Settings has no Factory Reset action")
	}
	m.Update(runeKey("G"))
	if !m.settingsOnReset() {
		t.Fatal("Settings did not select Factory Reset")
	}
	return m, dir
}

func TestFactoryResetCancelEveryStep(t *testing.T) {
	for step := 1; step <= 3; step++ {
		for _, cancel := range []string{"esc", "up"} {
			t.Run(factoryResetSteps[step]+cancel, func(t *testing.T) {
				m, dir := resetTestModel(t)
				before, err := os.ReadFile(filepath.Join(dir, "history.json"))
				if err != nil {
					t.Fatal(err)
				}
				for i := 1; i <= step; i++ {
					if cmd := send(m, tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
						t.Fatal("reset started before fourth Enter")
					}
					if m.state != StateSettings || !strings.Contains(ansi.Strip(m.View()), factoryResetSteps[i]) {
						t.Fatalf("confirmation %d missing in settings: %s", i, ansi.Strip(m.View()))
					}
				}
				if cancel == "esc" {
					m.Update(tea.KeyMsg{Type: tea.KeyEsc})
				} else {
					m.Update(tea.KeyMsg{Type: tea.KeyUp})
					m.Update(tea.KeyMsg{Type: tea.KeyDown})
				}
				if m.state != StateSettings || m.reset != nil || !strings.Contains(ansi.Strip(m.View()), "Factory Reset") {
					t.Fatal("cancellation did not restore the action")
				}
				after, err := os.ReadFile(filepath.Join(dir, "history.json"))
				if err != nil || string(before) != string(after) {
					t.Fatalf("cancellation changed progress: %v", err)
				}
				backups, err := filepath.Glob(filepath.Join(filepath.Dir(dir), "codedrill-backup-*"))
				if err != nil || len(backups) != 0 {
					t.Fatal("cancellation started a reset backup")
				}
			})
		}
	}
}

func TestFactoryResetSuccessRestoresFreshInterface(t *testing.T) {
	m, dir := resetTestModel(t)
	for range 3 {
		if cmd := send(m, tea.KeyMsg{Type: tea.KeyEnter}); cmd != nil {
			t.Fatal("reset started too early")
		}
	}
	cmd := send(m, tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !m.reset.running {
		t.Fatal("fourth Enter did not start reset")
	}
	result := cmd().(factoryResetFinishedMsg)
	if result.err != nil || result.backup == "" {
		t.Fatalf("reset failed: %+v", result)
	}
	m.Update(result)
	if m.state != StateMenu || m.reset != nil || m.categoryFilter != "ALL" || m.mouseOff || m.shell != "bash" || CurrentTheme().ID != "legacy" {
		t.Fatalf("default interface not restored: state=%v filter=%s shell=%s theme=%s", m.state, m.categoryFilter, m.shell, CurrentTheme().ID)
	}
	if !strings.Contains(m.menuNotice, "Factory Reset complete") || !strings.Contains(m.menuNotice, result.backup) {
		t.Fatalf("missing success message or backup path: %s", m.menuNotice)
	}
	if len(m.history.Attempts) != 0 || len(m.history.Stats) != 0 {
		t.Fatal("progress survived reset in memory")
	}
	if drill.Find(m.drills, "private-reset-test") != nil {
		t.Fatal("custom drill survived reset")
	}
	if err := checkResetBuiltins(m.drills); err != nil {
		t.Fatal("hidden presets were not restored:", err)
	}
	for _, d := range drill.Filter(m.drills, "42/Exam 05") {
		source, err := solutions.Resolve(d.ID, d.Dir)
		if err != nil || source == nil || source.Local {
			t.Fatalf("bundled reference unavailable for %s: %v", d.ID, err)
		}
	}
	if options, err := mode.Load("exam05-vect2"); err != nil || options != mode.Study() {
		t.Fatalf("saved practice preferences survived: %+v, %v", options, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); !os.IsNotExist(err) {
		t.Fatal("old settings survived reset")
	}
	if _, err := os.Stat(filepath.Join(result.backup, "data", "drills", "private-reset-test", "drill.json")); err != nil {
		t.Fatal("custom drill is not restorable:", err)
	}
}

func TestFactoryResetRequiresBundledExam05(t *testing.T) {
	builtins, err := drill.LoadBuiltins()
	if err != nil || checkResetBuiltins(builtins) != nil {
		t.Fatal("the built-in reset prerequisite is not satisfied")
	}
	var missing []drill.Drill
	for _, d := range builtins {
		if d.ID != "exam05-polyset" {
			missing = append(missing, d)
		}
	}
	if err := checkResetBuiltins(missing); err == nil {
		t.Fatal("reset accepted an executable missing polyset")
	}
}

func TestFactoryResetFailure(t *testing.T) {
	m, dir := resetTestModel(t)
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	var cmd tea.Cmd
	for range 4 {
		cmd = send(m, tea.KeyMsg{Type: tea.KeyEnter})
	}
	if cmd == nil {
		t.Fatal("no reset command")
	}
	m.Update(cmd())
	if m.state != StateSettings || m.reset != nil || m.settingsErr == nil {
		t.Fatal("reset error was not surfaced in settings")
	}
	if _, err := os.Stat(filepath.Join(dir, "history.json")); err != nil {
		t.Fatal("failed reset changed user data")
	}
}

func TestFactoryResetMouseNavigationCancels(t *testing.T) {
	m, dir := resetTestModel(t)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for y, row := range settingsThemeRows(m) {
		if row[0] == len(Themes) {
			m.Update(leftClick(row[1], y))
			if m.reset != nil || !m.settingsOnShell() {
				t.Fatal("clicking another setting did not cancel confirmation")
			}
			if _, err := os.Stat(filepath.Join(dir, "history.json")); err != nil {
				t.Fatal("navigation changed user data")
			}
			return
		}
	}
	t.Fatal("shell row was not visible")
}
