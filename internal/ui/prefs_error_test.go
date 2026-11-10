package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/mode"
)

func newPrefsErrorTestModel(t *testing.T) *MainModel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func writePrefsFile(t *testing.T, data string) string {
	t.Helper()
	path, err := mode.PrefsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestMissingPreferencesStartStudy(t *testing.T) {
	m := newPrefsErrorTestModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateCountdown || cmd == nil || m.runMode != mode.Study() || m.prefsErr != nil {
		t.Fatalf("missing preferences did not start Study: state %v, options %+v, error %v", m.state, m.runMode, m.prefsErr)
	}
}

func TestMissingDrillEntryStartsStudy(t *testing.T) {
	m := newPrefsErrorTestModel(t)
	if err := mode.Save("another_drill", mode.Copy()); err != nil {
		t.Fatal(err)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateCountdown || cmd == nil || m.runMode != mode.Study() || m.prefsErr != nil {
		t.Fatalf("missing drill entry did not start Study: state %v, options %+v", m.state, m.runMode)
	}
}

func TestCorruptPreferencesBlockSprintUntilFixed(t *testing.T) {
	m := newPrefsErrorTestModel(t)
	path := writePrefsFile(t, "{")
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.state != StateSetupError || m.prefsErr == nil {
		t.Fatalf("corrupt preferences did not block the sprint: state %v, command %v", m.state, cmd)
	}
	if m.session != nil || m.countdownCount != 0 || len(m.history.Attempts) != 0 {
		t.Fatal("corrupt preferences started a countdown, workspace, or attempt")
	}
	view := m.View()
	if !strings.Contains(view, "preferences unavailable") || !strings.Contains(view, "could not be read") {
		t.Fatalf("corrupt preferences are not explained:\n%s", view)
	}

	if err := os.WriteFile(path, []byte(`{"ft_strlen":{"check_reference":false,"show_reference":false,"require_exact":false,"require_tests":true}}`), 0644); err != nil {
		t.Fatal(err)
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.state != StateBriefing || m.briefingMode != mode.Exam() || m.prefsErr != nil {
		t.Fatalf("retry did not reopen fixed preferences: state %v, options %+v", m.state, m.briefingMode)
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || m.state != StateCountdown || m.runMode != mode.Exam() || m.prefsErr != nil {
		t.Fatalf("fixed preferences did not start the sprint: state %v, options %+v", m.state, m.runMode)
	}
}

func TestUnreadablePreferencesBlockSprint(t *testing.T) {
	m := newPrefsErrorTestModel(t)
	path, err := mode.PrefsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	// A directory cannot be read as a preference file, even when the test
	// process has permission to bypass file mode checks.
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil || m.state != StateSetupError || m.prefsErr == nil {
		t.Fatalf("unreadable preferences did not block the sprint: state %v, command %v", m.state, cmd)
	}
	if m.session != nil || m.countdownCount != 0 || len(m.history.Attempts) != 0 {
		t.Fatal("unreadable preferences started a countdown, workspace, or attempt")
	}
	view := m.View()
	if !strings.Contains(view, "preferences unavailable") || !strings.Contains(view, "could not be read") {
		t.Fatalf("unreadable preferences are not explained:\n%s", view)
	}
}
