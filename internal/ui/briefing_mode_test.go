package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/mode"
)

func newBriefingModeModel(t *testing.T, id string) *MainModel {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m, err := NewMainModelWithTarget(id)
	if err != nil {
		t.Fatal(err)
	}
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

func TestBriefingPresetsSaveSelection(t *testing.T) {
	for _, tc := range []struct {
		key  string
		want mode.Options
		name string
	}{
		{"1", mode.Study(), "Study"},
		{"2", mode.Copy(), "Copy"},
		{"3", mode.Recall(), "Recall"},
		{"4", mode.Exam(), "Exam"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newBriefingModeModel(t, "ft_strlen")
			m.Update(runeKey(tc.key))
			if m.briefingMode != tc.want {
				t.Fatalf("preset %s selected %+v, want %+v", tc.key, m.briefingMode, tc.want)
			}
			saved, err := mode.Load("ft_strlen")
			if err != nil || saved != tc.want {
				t.Fatalf("preset %s saved %+v, error %v", tc.key, saved, err)
			}
			if view := ansi.Strip(m.View()); !strings.Contains(view, fmt.Sprintf("< %s >", tc.name)) {
				t.Fatalf("preset name missing from briefing:\n%s", view)
			}
		})
	}
}

func TestBriefingCheckboxFocusWrapsAndToggles(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.modeFocus != 3 {
		t.Fatalf("shift+tab focused option %d, want 3", m.modeFocus)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.modeFocus != 0 {
		t.Fatalf("tab did not wrap to 0, got %d", m.modeFocus)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	want := mode.Options{CheckReference: true, RequireTests: true}
	if m.briefingMode != want {
		t.Fatalf("toggling show reference gave %+v, want %+v", m.briefingMode, want)
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "< Custom >") {
		t.Fatalf("changed combination is not labelled custom:\n%s", view)
	}
	saved, err := mode.Load("ft_strlen")
	if err != nil || saved != want {
		t.Fatalf("toggle saved %+v, error %v", saved, err)
	}

	// Turning reference checking off clears showing and exact copy.
	m.modeFocus = 0
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.briefingMode.CheckReference || m.briefingMode.ShowReference || m.briefingMode.RequireExact {
		t.Fatalf("disabling reference checking kept reference options: %+v", m.briefingMode)
	}

	// Turning exact copy on implies reference checking.
	m.modeFocus = 2
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !m.briefingMode.RequireExact || !m.briefingMode.CheckReference {
		t.Fatalf("exact copy did not imply reference checking (focus %d): %+v", m.modeFocus, m.briefingMode)
	}
}

func TestBriefingInvalidGatesBlockEnter(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(runeKey("2")) // Copy has tests disabled.
	m.modeFocus = 0
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.briefingMode.RequireTests || m.briefingMode.RequireExact {
		t.Fatalf("toggling reference check changed the gates: %+v", m.briefingMode)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Enable at least one pass gate") {
		t.Fatalf("missing invalid-gate explanation:\n%s", view)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateBriefing {
		t.Fatalf("enter started a sprint with no pass gate: state %v", m.state)
	}
	// Space toggles a checkbox; it must not start the drill.
	m.modeFocus = 3
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.state != StateBriefing {
		t.Fatalf("space started the drill: state %v", m.state)
	}
}

func TestBriefingExactCopyValidationAndStudyFallback(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(runeKey("3")) // Recall, no reference attached.
	if !strings.Contains(m.briefingProblem(), "attached reference") {
		t.Fatalf("missing reference was not rejected: %+v", m.briefingMode)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Exact copy needs an attached reference") {
		t.Fatalf("missing reference is not explained:\n%s", view)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateBriefing {
		t.Fatalf("enter started a Recall run without a reference: state %v", m.state)
	}

	m.Update(runeKey("1")) // Study without a reference is tests-only.
	if m.briefingProblem() != "" {
		t.Fatalf("study without a reference was blocked: %s", m.briefingProblem())
	}
	view = ansi.Strip(m.View())
	if !strings.Contains(view, "No reference attached; tests still decide success.") {
		t.Fatalf("missing reference notice is absent:\n%s", view)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateCountdown {
		t.Fatalf("study without a reference did not start: state %v", m.state)
	}
}

func TestBriefingExactCopyChecksReferenceAndLanguage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	writeTestReference(t, "ft_strlen", testReference)
	m, err := NewMainModelWithTarget("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	m.Update(runeKey("3")) // Recall with a C reference attached.
	if m.briefingProblem() != "" {
		t.Fatalf("valid Recall was rejected: %s", m.briefingProblem())
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateCountdown || m.runMode != mode.Recall() {
		t.Fatalf("Recall did not snapshot: state %v, options %+v", m.state, m.runMode)
	}

	// An unsupported language cannot use exact copy.
	m2 := newBriefingModeModel(t, "ft_strlen")
	found := -1
	for i := range m2.drills {
		if !referenceLanguageSupported(&m2.drills[i]) {
			found = i
			break
		}
	}
	if found < 0 {
		t.Skip("no unsupported drill available")
	}
	m2.openBriefing(&m2.drills[found])
	m2.Update(runeKey("3"))
	if !strings.Contains(m2.briefingProblem(), "C and C++ drills") {
		t.Fatalf("unsupported language was not rejected: %s", m2.briefingProblem())
	}
	m2.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m2.state != StateBriefing {
		t.Fatalf("enter started exact copy for an unsupported language: state %v", m2.state)
	}
}

func TestBriefingModePersistenceAcrossNavigation(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(runeKey("2"))
	m.openBriefing(m.activeDrill)
	if m.briefingMode != mode.Copy() {
		t.Fatalf("reopened briefing did not load Copy: %+v", m.briefingMode)
	}

	m.Update(runeKey("l"))
	if m.state != StateLeaderboard {
		t.Fatalf("history did not open: state %v", m.state)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.state != StateBriefing || m.briefingMode != mode.Copy() {
		t.Fatalf("history return changed the selection: state %v, options %+v", m.state, m.briefingMode)
	}

	original := *m.activeDrill
	next := m.stepDrill(original, 1)
	if next == nil {
		t.Skip("no next drill")
	}
	if err := mode.Save(next.ID, mode.Exam()); err != nil {
		t.Fatal(err)
	}
	m.Update(runeKey("n"))
	if m.activeDrill.ID != next.ID || m.briefingMode != mode.Exam() {
		t.Fatalf("next drill did not load its own preferences: %+v", m.briefingMode)
	}
	m.Update(runeKey("p"))
	if m.activeDrill.ID != original.ID || m.briefingMode != mode.Copy() {
		t.Fatalf("previous drill did not reload its preferences: %+v", m.briefingMode)
	}
}

func TestResultsPreserveBriefingSelections(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(runeKey("4"))
	d := *m.activeDrill
	m.runMode = m.briefingMode
	m.resultDrill = d
	m.state = StateResults
	m.Update(runeKey("b"))
	if m.state != StateBriefing || m.briefingMode != mode.Exam() {
		t.Fatalf("briefing after results changed the run selections: %+v", m.briefingMode)
	}

	// Retrying keeps the run's selections even if the saved file changes.
	if err := mode.Save(d.ID, mode.Study()); err != nil {
		t.Fatal(err)
	}
	m.state = StateResults
	m.activeDrill = &d
	m.resultDrill = d
	m.Update(runeKey("r"))
	if m.state != StateCountdown || m.runMode != mode.Exam() {
		t.Fatalf("retry used stale preferences: state %v, options %+v", m.state, m.runMode)
	}
}

func TestBriefingSaveErrorIsVisibleAndEditingContinues(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	path, err := mode.PrefsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	m.Update(runeKey("2"))
	if m.briefingMode != mode.Copy() || m.briefingSaveErr == nil {
		t.Fatalf("save failure lost the selection: %+v", m.briefingMode)
	}
	if view := ansi.Strip(m.View()); !strings.Contains(view, "Preferences not saved") {
		t.Fatalf("save failure is not visible:\n%s", view)
	}
	m.Update(runeKey("4"))
	if m.briefingMode != mode.Exam() || m.briefingSaveErr == nil {
		t.Fatalf("save failure prevented further editing: %+v", m.briefingMode)
	}
}

func TestBriefingModeLayoutAtSmallSizes(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	labels := []string{"Reference check", "Show reference", "Require exact copy", "Require tests"}
	for _, size := range []tea.WindowSizeMsg{{Width: 60, Height: 20}, {Width: 80, Height: 24}} {
		m.Update(size)
		view := ansi.Strip(m.View())
		lines := strings.Split(view, "\n")
		for _, label := range labels {
			count := 0
			for _, line := range lines {
				if strings.Contains(line, label) {
					count++
					if !strings.HasPrefix(line, "  ") {
						t.Fatalf("%dx%d: checkbox is left of column 2: %q", size.Width, size.Height, line)
					}
				}
			}
			if count != 1 {
				t.Fatalf("%dx%d: label %q appears on %d rows", size.Width, size.Height, label, count)
			}
		}
		if !strings.Contains(view, "< Study >") || !strings.Contains(view, "[Space] toggle") {
			t.Fatalf("%dx%d: mode panel or help is missing:\n%s", size.Width, size.Height, view)
		}
	}
}

func TestBriefingModeTwoColumnOrderingAndAlignment(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")

	// At 80x24 the panel should use two columns.
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := ansi.Strip(m.View())
	lines := strings.Split(view, "\n")

	// Find the two checkbox rows.
	var row0, row1 int = -1, -1
	for y, line := range lines {
		if strings.Contains(line, "Reference check") && row0 < 0 {
			row0 = y
		}
		if strings.Contains(line, "Require exact copy") && row1 < 0 {
			row1 = y
		}
	}
	if row0 < 0 || row1 < 0 || row1 <= row0 {
		t.Fatalf("expected two checkbox rows, got row0=%d row1=%d:\n%s", row0, row1, view)
	}
	if row1-row0 != 1 {
		t.Fatalf("checkbox rows should be adjacent, got row0=%d row1=%d:\n%s", row0, row1, view)
	}

	// Row 0: Reference check (left), Show reference (right).
	l0 := lines[row0]
	if !strings.Contains(l0, "Reference check") || !strings.Contains(l0, "Show reference") {
		t.Fatalf("row 0 should hold Reference check + Show reference: %q", l0)
	}
	// Row 1: Require exact copy (left), Require tests (right).
	l1 := lines[row1]
	if !strings.Contains(l1, "Require exact copy") || !strings.Contains(l1, "Require tests") {
		t.Fatalf("row 1 should hold Require exact copy + Require tests: %q", l1)
	}

	// Columns must align: left-column labels start at the same x on both rows.
	left0 := strings.Index(l0, "Reference check")
	left1 := strings.Index(l1, "Require exact copy")
	if left0 != left1 {
		t.Fatalf("left column misaligned: row0 x=%d row1 x=%d:\n%s", left0, left1, view)
	}
	// Right-column labels start at the same x on both rows.
	right0 := strings.Index(l0, "Show reference")
	right1 := strings.Index(l1, "Require tests")
	if right0 != right1 {
		t.Fatalf("right column misaligned: row0 x=%d row1 x=%d:\n%s", right0, right1, view)
	}
	// Right column must be to the right of the left column with a gap.
	if right0 <= left0+len("Reference check")+1 {
		t.Fatalf("right column is not separated from left column: %q", l0)
	}

	// Mode label must be on the row immediately above the checkboxes.
	if !strings.Contains(lines[row0-1], "Mode: < Study >") {
		t.Fatalf("mode label not directly above checkboxes:\n%s", view)
	}

	// At 60x20, two columns should still fit.
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	view60 := ansi.Strip(m.View())
	if !strings.Contains(view60, "Reference check") || !strings.Contains(view60, "Show reference") {
		t.Fatalf("60x20 should still show two columns:\n%s", view60)
	}

	// At a very narrow width, fall back to vertical layout.
	m.Update(tea.WindowSizeMsg{Width: 30, Height: 20})
	view30 := ansi.Strip(m.View())
	lines30 := strings.Split(view30, "\n")
	// Each label should be on its own row.
	counts := map[string]int{}
	for _, label := range []string{"Reference check", "Show reference", "Require exact copy", "Require tests"} {
		for _, line := range lines30 {
			if strings.Contains(line, label) {
				counts[label]++
			}
		}
	}
	for label, count := range counts {
		if count != 1 {
			t.Fatalf("30-wide: label %q appears on %d rows", label, count)
		}
	}
}

func TestBriefingModeNavigationLeftRightCycling(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")

	// Initial mode is Study.
	if m.briefingMode != mode.Study() {
		t.Fatalf("expected initial Study mode, got %+v", m.briefingMode)
	}

	// Press Right: cycles to Copy, then Recall, then Exam, then wraps back to Study.
	expectedCycle := []struct {
		want mode.Options
		name string
	}{
		{mode.Copy(), "Copy"},
		{mode.Recall(), "Recall"},
		{mode.Exam(), "Exam"},
		{mode.Study(), "Study"},
	}

	for _, step := range expectedCycle {
		m.Update(tea.KeyMsg{Type: tea.KeyRight})
		if m.briefingMode != step.want {
			t.Fatalf("Right arrow gave %+v, want %s (%+v)", m.briefingMode, step.name, step.want)
		}
		saved, err := mode.Load("ft_strlen")
		if err != nil || saved != step.want {
			t.Fatalf("preset %s not persisted on disk: %+v, err %v", step.name, saved, err)
		}
		view := ansi.Strip(m.View())
		if !strings.Contains(view, fmt.Sprintf("Mode: < %s >", step.name)) {
			t.Fatalf("Mode: < %s > missing from view:\n%s", step.name, view)
		}
	}

	// Press Left: wraps backwards from Study to Exam, then Recall, then Copy, then Study.
	reverseCycle := []struct {
		want mode.Options
		name string
	}{
		{mode.Exam(), "Exam"},
		{mode.Recall(), "Recall"},
		{mode.Copy(), "Copy"},
		{mode.Study(), "Study"},
	}

	for _, step := range reverseCycle {
		m.Update(tea.KeyMsg{Type: tea.KeyLeft})
		if m.briefingMode != step.want {
			t.Fatalf("Left arrow gave %+v, want %s (%+v)", m.briefingMode, step.name, step.want)
		}
		saved, err := mode.Load("ft_strlen")
		if err != nil || saved != step.want {
			t.Fatalf("preset %s not persisted on disk: %+v, err %v", step.name, saved, err)
		}
		view := ansi.Strip(m.View())
		if !strings.Contains(view, fmt.Sprintf("Mode: < %s >", step.name)) {
			t.Fatalf("Mode: < %s > missing from view:\n%s", step.name, view)
		}
	}
}

func TestBriefingModeNavigationFromCustom(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")

	// Toggle option to make it Custom.
	m.modeFocus = 1
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if view := ansi.Strip(m.View()); !strings.Contains(view, "Mode: < Custom >") {
		t.Fatalf("expected Mode: < Custom >, got:\n%s", view)
	}

	// Press Right from Custom: starts at Study (index 0).
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.briefingMode != mode.Study() {
		t.Fatalf("Right arrow from Custom gave %+v, want Study", m.briefingMode)
	}

	// Toggle to Custom again.
	m.modeFocus = 1
	m.Update(tea.KeyMsg{Type: tea.KeySpace})

	// Press Left from Custom: starts at Exam (last index).
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.briefingMode != mode.Exam() {
		t.Fatalf("Left arrow from Custom gave %+v, want Exam", m.briefingMode)
	}
}

func TestBriefingModeLabelAboveCheckboxes(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	view := ansi.Strip(m.View())
	lines := strings.Split(view, "\n")

	modeLine := -1
	checkboxLine := -1
	for y, line := range lines {
		if strings.Contains(line, "Mode: <") && modeLine < 0 {
			modeLine = y
		}
		if strings.Contains(line, "Reference check") && checkboxLine < 0 {
			checkboxLine = y
		}
	}

	if modeLine < 0 {
		t.Fatal("Mode label missing from view")
	}
	if checkboxLine < 0 {
		t.Fatal("Reference check checkbox missing from view")
	}
	if modeLine >= checkboxLine {
		t.Fatalf("Mode label (line %d) must be ABOVE checkboxes (line %d)", modeLine, checkboxLine)
	}
}
