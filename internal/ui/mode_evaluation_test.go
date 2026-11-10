package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/solutions"
)

const testReference = "int ft_strlen(char *s)\n{\n\treturn 0;\n}\n"

const otherSource = "int ft_strlen(char *s)\n{\n\treturn 1;\n}\n"

func writeTestReference(t *testing.T, drillID, source string) {
	t.Helper()
	dir, err := solutions.OverlayDir(drillID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "solution.c"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
}

func writeTestWorkspace(t *testing.T, m *MainModel, source string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(m.session.Workspace, "solution.c"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
}

func runEvaluation(t *testing.T, m *MainModel) evalFinishedMsg {
	t.Helper()
	_, cmd := m.Update(sprintFinishedMsg{})
	if cmd == nil {
		t.Fatal("sprint finish did not start evaluation")
	}
	msg, ok := cmd().(evalFinishedMsg)
	if !ok {
		t.Fatalf("sprint finish returned %T, want evalFinishedMsg", msg)
	}
	m.Update(msg)
	m.evalReadyTime = time.Now().Add(-time.Second)
	return msg
}

func pressFinish(t *testing.T, m *MainModel) {
	t.Helper()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
}

func TestStudyReferenceScoreIsInformational(t *testing.T) {
	m := newEvaluationTestModel(t)
	writeTestReference(t, "ft_strlen", testReference)
	writeTestWorkspace(t, m, otherSource)
	m.runMode = mode.Study()
	m.session.TestCommand = "printf 'ok'"

	msg := runEvaluation(t, m)
	if !msg.passed || msg.referenceErr != nil || msg.reference == nil {
		t.Fatalf("study evaluation lacks tests or reference: %+v", msg)
	}
	if msg.reference.Exact {
		t.Fatal("mismatched workspace compared as exact")
	}
	if !m.passVerdict().Passed {
		t.Fatalf("study failed tests that passed: %+v", m.passVerdict())
	}
	view := ansi.Strip(m.View())
	for _, want := range []string{"Tests: passed", "Reference: ", "Overall: passed"} {
		if !strings.Contains(view, want) {
			t.Errorf("evaluation lacks %q:\n%s", want, view)
		}
	}

	pressFinish(t, m)
	if m.state != StateResults || len(m.history.Attempts) != 1 || !m.history.Attempts[0].Passed {
		t.Fatalf("study pass was not recorded: state %v, attempts %+v", m.state, m.history.Attempts)
	}
}

func TestStudyOptionalReferenceErrorDoesNotFailTests(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.runMode = mode.Study()
	m.session.TestCommand = "printf 'ok'"

	msg := runEvaluation(t, m)
	if !msg.passed || msg.reference != nil || msg.referenceErr == nil {
		t.Fatalf("study did not preserve the missing reference: %+v", msg)
	}
	if !m.passVerdict().Passed {
		t.Fatalf("optional reference error failed a tests-only run: %+v", m.passVerdict())
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Reference: unavailable") || !strings.Contains(view, "Overall: passed") {
		t.Fatalf("optional reference error is not shown separately:\n%s", view)
	}

	pressFinish(t, m)
	if m.state != StateResults || len(m.history.Attempts) != 1 || !m.history.Attempts[0].Passed {
		t.Fatalf("tests-only pass was not recorded: state %v", m.state)
	}
}

func TestCopyPassesWhenTestsFail(t *testing.T) {
	m := newEvaluationTestModel(t)
	writeTestReference(t, "ft_strlen", testReference)
	writeTestWorkspace(t, m, testReference)
	m.runMode = mode.Copy()
	m.session.TestCommand = "printf 'failure'; exit 1"

	msg := runEvaluation(t, m)
	if msg.passed {
		t.Fatal("failed test command reported as passed")
	}
	if msg.referenceErr != nil || msg.reference == nil || !msg.reference.Exact {
		t.Fatalf("copy did not compare exactly: %+v", msg)
	}
	if !m.passVerdict().Passed {
		t.Fatalf("copy failed despite an exact reference: %+v", m.passVerdict())
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Tests: failed") || !strings.Contains(view, "100% exact") || !strings.Contains(view, "Overall: passed") {
		t.Fatalf("copy result does not separate tests from reference:\n%s", view)
	}

	pressFinish(t, m)
	if m.state != StateResults || len(m.history.Attempts) != 1 || !m.history.Attempts[0].Passed {
		t.Fatalf("copy pass was not recorded: state %v, attempts %+v", m.state, m.history.Attempts)
	}
}

func TestCopyFailsWhenTestsPass(t *testing.T) {
	m := newEvaluationTestModel(t)
	writeTestReference(t, "ft_strlen", testReference)
	writeTestWorkspace(t, m, otherSource)
	m.runMode = mode.Copy()
	m.session.TestCommand = "printf 'ok'"

	msg := runEvaluation(t, m)
	if !msg.passed || msg.referenceErr != nil || msg.reference == nil || msg.reference.Exact {
		t.Fatalf("copy did not keep a passing test and inexact reference: %+v", msg)
	}
	if m.passVerdict().Passed {
		t.Fatal("copy passed despite a mismatched reference")
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Tests: passed") || !strings.Contains(view, "Overall: failed") {
		t.Fatalf("copy failure does not separate tests from reference:\n%s", view)
	}

	pressFinish(t, m)
	if m.state != StateSprinting || len(m.history.Attempts) != 0 {
		t.Fatalf("failed copy was recorded or did not offer a retry: state %v", m.state)
	}
}

func TestBothRequiredGatesMustPass(t *testing.T) {
	opts := mode.Options{CheckReference: true, RequireExact: true, RequireTests: true}
	for _, tc := range []struct {
		name    string
		source  string
		command string
		want    bool
	}{
		{"both pass", testReference, "printf 'ok'", true},
		{"tests fail", testReference, "printf 'failure'; exit 1", false},
		{"reference differs", otherSource, "printf 'ok'", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newEvaluationTestModel(t)
			writeTestReference(t, "ft_strlen", testReference)
			writeTestWorkspace(t, m, tc.source)
			m.runMode = opts
			m.session.TestCommand = tc.command
			runEvaluation(t, m)
			if got := m.passVerdict().Passed; got != tc.want {
				t.Fatalf("verdict passed=%v, want %v (%s)", got, tc.want, m.passVerdict().Reason)
			}
		})
	}
}

func TestRequiredCopyFailsClosedWithoutReference(t *testing.T) {
	m := newEvaluationTestModel(t)
	m.runMode = mode.Copy()
	m.session.TestCommand = "printf 'ok'"

	msg := runEvaluation(t, m)
	if !msg.passed || msg.referenceErr == nil || msg.reference != nil {
		t.Fatalf("missing reference was not preserved: %+v", msg)
	}
	verdict := m.passVerdict()
	if verdict.Passed || !strings.Contains(verdict.Reason, "reference comparison failed") {
		t.Fatalf("missing required reference did not fail closed: %+v", verdict)
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Reference: unavailable") || !strings.Contains(view, "Overall: failed") {
		t.Fatalf("missing reference is not shown as a failure:\n%s", view)
	}

	pressFinish(t, m)
	if m.state != StateSprinting || len(m.history.Attempts) != 0 {
		t.Fatalf("missing reference turned into a recorded pass: state %v", m.state)
	}
}

func TestExamSkipsReferenceComparison(t *testing.T) {
	m := newEvaluationTestModel(t)
	writeTestReference(t, "ft_strlen", testReference)
	writeTestWorkspace(t, m, otherSource)
	m.runMode = mode.Exam()
	m.session.TestCommand = "printf 'ok'"

	msg := runEvaluation(t, m)
	if !msg.passed || msg.reference != nil || msg.referenceErr != nil {
		t.Fatalf("exam compared the reference: %+v", msg)
	}
	if !m.passVerdict().Passed {
		t.Fatalf("exam failed passing tests: %+v", m.passVerdict())
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Reference: not checked") || !strings.Contains(view, "Overall: passed") {
		t.Fatalf("exam does not report skipped reference:\n%s", view)
	}
}

func TestSavedModeIsSnapshottedBeforeCountdown(t *testing.T) {
	m := newEvaluationTestModel(t)
	if err := mode.Save("ft_strlen", mode.Copy()); err != nil {
		t.Fatal(err)
	}
	writeTestReference(t, "ft_strlen", testReference)
	m.openBriefing(m.activeDrill)
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateCountdown {
		t.Fatalf("expected countdown after starting a sprint, got state %v", m.state)
	}
	if m.runMode != mode.Copy() {
		t.Fatalf("saved copy preferences were not loaded: %+v", m.runMode)
	}

	// Changing preferences during the countdown must not alter this run.
	if err := mode.Save("ft_strlen", mode.Exam()); err != nil {
		t.Fatal(err)
	}
	for range 3 {
		m.Update(countdownTickMsg{id: m.countdownID})
	}
	if m.state != StateCountdown || m.runMode != mode.Copy() {
		t.Fatalf("countdown changed the snapshotted mode: state %v, options %+v", m.state, m.runMode)
	}

	// Evaluation must keep using the snapshot: Copy passes on an exact copy
	// even when the tests fail.
	writeTestReference(t, "ft_strlen", testReference)
	writeTestWorkspace(t, m, testReference)
	m.session.TestCommand = "printf 'failure'; exit 1"
	m.state = StateSprinting
	m.sprintStartTime = time.Now().Add(-time.Second)
	m.Update(sprintFinishedMsg{})
	m.Update(evalFinishedMsg{
		id:        m.evalID,
		session:   m.session,
		passed:    false,
		output:    "tests failed",
		reference: &compare.Result{Exact: true},
	})
	m.evalReadyTime = time.Now().Add(-time.Second)
	if !m.passVerdict().Passed {
		t.Fatalf("evaluation ignored the snapshotted copy mode: %+v", m.passVerdict())
	}
	pressFinish(t, m)
	if m.state != StateResults || len(m.history.Attempts) != 1 || !m.history.Attempts[0].Passed {
		t.Fatalf("snapshotted copy pass was not recorded: state %v", m.state)
	}
}

func TestStaleEvaluationDoesNotKeepReferenceResults(t *testing.T) {
	m := newEvaluationTestModel(t)
	writeTestReference(t, "ft_strlen", testReference)
	writeTestWorkspace(t, m, otherSource)
	m.runMode = mode.Study()
	m.session.TestCommand = "printf 'ok'"

	_, firstCmd := m.Update(sprintFinishedMsg{})
	first := firstCmd().(evalFinishedMsg)
	m.Update(first)
	if m.evalRef == nil {
		t.Fatal("first evaluation did not store its reference result")
	}

	m.state = StateSprinting
	m.sprintStartTime = time.Now().Add(-time.Second)
	m.Update(sprintFinishedMsg{})
	if m.evalRef != nil || m.evalRefErr != nil || m.evalOutput != "" {
		t.Fatal("new evaluation kept the previous reference result")
	}
	m.Update(first)
	if m.evalRef != nil || m.evalOutput != "" {
		t.Fatal("stale evaluation restored its reference result")
	}
}
