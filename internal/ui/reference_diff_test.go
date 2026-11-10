package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/solutions"
	"github.com/luvcie/codedrill/internal/storage"
)

const reviewReference = "int REVIEW_REFERENCE_SECRET(void)\n{\nreturn 42;\n}\n"
const reviewStudent = "int review_student(void)\n{\nreturn 7;\n}\n"

func newReferenceReviewModel(t *testing.T, options mode.Options, source string) *MainModel {
	t.Helper()
	m := newEvaluationTestModel(t)
	m.runMode = options
	writeTestReference(t, m.activeDrill.ID, reviewReference)
	writeTestWorkspace(t, m, source)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return m
}

func assertNoReferenceDiff(t *testing.T, m *MainModel) {
	t.Helper()
	if strings.Contains(m.screenHelp(), "Reference diff") {
		t.Fatal("unavailable diff has a footer action")
	}
	m.openReferenceDiff()
	if m.referenceDiff != nil || strings.Contains(m.View(), "REVIEW_REFERENCE_SECRET") {
		t.Fatal("reference content is available outside a completed comparison")
	}
}

func TestPostSubmissionDiffModeOptions(t *testing.T) {
	cases := []struct {
		name    string
		options mode.Options
	}{
		{"Study", mode.Study()}, {"Copy", mode.Copy()},
		{"Recall", mode.Recall()}, {"Exam", mode.Exam()},
	}
	for bits := 0; bits < 16; bits++ {
		cases = append(cases, struct {
			name    string
			options mode.Options
		}{fmt.Sprintf("custom-%04b", bits), mode.Options{
			CheckReference: bits&1 != 0, ShowReference: bits&2 != 0,
			RequireExact: bits&4 != 0, RequireTests: bits&8 != 0,
		}})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newReferenceReviewModel(t, tc.options, reviewStudent)
			assertNoReferenceDiff(t, m)
			m.Update(runeKey("v"))
			if m.state != StateSprinting || m.referenceDiff != nil {
				t.Fatal("v opened post-submission content during the sprint")
			}
			msg := runEvaluation(t, m)
			if !tc.options.CheckReference {
				if msg.reference != nil || msg.referenceErr != nil {
					t.Fatal("reference checking was enabled for a run that disabled it")
				}
				assertNoReferenceDiff(t, m)
				return
			}
			if msg.reference == nil || msg.referenceErr != nil || !strings.Contains(ansi.Strip(m.View()), "[v] Reference diff") {
				t.Fatalf("completed comparison has no diff: %+v\n%s", msg, m.View())
			}
			verdict, score := m.passVerdict(), m.evalRef.Percent()
			m.Update(runeKey("v"))
			if m.referenceDiff == nil || !strings.Contains(m.View(), "REVIEW_REFERENCE_SECRET") {
				t.Fatalf("completed diff did not open: %s", m.View())
			}
			for _, want := range []string{"solution.c", "- 3: 42", "+ 3: 7"} {
				if !strings.Contains(ansi.Strip(m.View()), want) {
					t.Fatalf("diff lacks %q: %s", want, m.View())
				}
			}
			m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			if m.state != StateEvaluating || m.referenceDiff != nil || m.passVerdict() != verdict || m.evalRef.Percent() != score || len(m.history.Attempts) != 0 {
				t.Fatal("viewing changed evaluation, scores, or history")
			}
		})
	}
}

func TestPostSubmissionDiffSnapshotAndCleanup(t *testing.T) {
	m := newReferenceReviewModel(t, mode.Recall(), reviewReference)
	session := m.session
	_, evaluate := m.Update(sprintFinishedMsg{})
	assertNoReferenceDiff(t, m)
	// Neither mutable preferences nor a later UI selection can change this run.
	if err := mode.Save(m.activeDrill.ID, mode.Exam()); err != nil {
		t.Fatal(err)
	}
	m.briefingMode = mode.Exam()
	m.Update(evaluate())
	m.evalReadyTime = time.Now().Add(-time.Second)
	if m.evalReview == nil || m.evalReview.options != mode.Recall() {
		t.Fatal("evaluation did not capture the run's options")
	}
	m.Update(runeKey("v"))
	if !strings.Contains(m.View(), "Normalized sources match.") {
		t.Fatalf("exact comparison has no match message: %s", m.View())
	}
	m.Update(runeKey("q"))
	duration, submits := m.accumulatedTime, m.submits()
	pressFinish(t, m)
	if m.state != StateResults || m.session != nil || m.resultReview == nil {
		t.Fatal("recordPass did not preserve the completed diff")
	}
	if _, err := os.Stat(session.Workspace); !os.IsNotExist(err) {
		t.Fatalf("workspace survived recording: %v", err)
	}
	if len(m.history.Attempts) != 1 || !m.history.Attempts[0].Passed || m.history.Attempts[0].Duration != duration || m.history.Attempts[0].Attempts != submits {
		t.Fatalf("recordPass changed recording behavior: %+v", m.history.Attempts)
	}
	dir, err := solutions.OverlayDir(m.resultDrill.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	m.runMode = mode.Exam()
	m.Update(runeKey("v"))
	if m.referenceDiff == nil || !strings.Contains(m.View(), "Normalized sources match.") {
		t.Fatal("saved results reread the workspace, reference, or current mode")
	}
	m.Update(runeKey("q"))
	if m.state != StateResults || len(m.history.Attempts) != 1 {
		t.Fatal("closing results diff changed navigation or recorded twice")
	}
	path, err := storage.GetHistoryPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || strings.Contains(string(data), "REVIEW_REFERENCE_SECRET") || strings.Contains(string(data), "comparison") {
		t.Fatalf("reference data leaked into history: %s, %v", data, err)
	}
}

func TestPostSubmissionDiffMultipleFilesAndMissingCounterparts(t *testing.T) {
	m := newReferenceReviewModel(t, mode.Study(), reviewStudent)
	d := *m.activeDrill
	d.EntryFile = "solution.c"
	m.activeDrill = &d
	dir, err := solutions.OverlayDir(d.ID)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"a.h": "#define MISSING_HEADER 1\n", "z.c": "int exact;\n", "empty.c": ""} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(m.session.Workspace, "z.c"), []byte("int exact;\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runEvaluation(t, m)
	var paths []string
	for _, file := range m.evalRef.Files {
		paths = append(paths, file.Rel)
	}
	if !reflect.DeepEqual(paths, []string{"solution.c", "a.h", "empty.c", "z.c"}) {
		t.Fatalf("unstable file order: %v", paths)
	}
	m.Update(runeKey("v"))
	for i, want := range []string{"solution.c", "a.h", "empty.c", "z.c"} {
		if i != 0 {
			m.Update(tea.KeyMsg{Type: tea.KeyTab})
		}
		if m.referenceDiff.model.CurrentFile() != want {
			t.Fatalf("selected %s, want %s", m.referenceDiff.model.CurrentFile(), want)
		}
		if (want == "a.h" || want == "empty.c") && !strings.Contains(m.View(), "Required workspace file is missing.") {
			t.Fatalf("missing file was not identified: %s", m.View())
		}
		if want == "z.c" && !strings.Contains(m.View(), "Normalized sources match.") {
			t.Fatal("matching file has no match message")
		}
	}
	m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.referenceDiff.model.CurrentFile() != "empty.c" {
		t.Fatal("Shift+Tab did not navigate backwards")
	}
	m.Update(runeKey("q"))
	pressFinish(t, m)
	m.Update(runeKey("v"))
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.state != StateResults || !strings.Contains(m.View(), "MISSING_HEADER") {
		t.Fatal("multi-file diff was lost after cleanup")
	}
}

func TestPostSubmissionDiffComparisonErrors(t *testing.T) {
	for _, kind := range []string{"missing reference", "student syntax", "reference syntax", "unsupported language", "workspace missing"} {
		t.Run(kind, func(t *testing.T) {
			m := newReferenceReviewModel(t, mode.Study(), reviewStudent)
			switch kind {
			case "missing reference":
				dir, err := solutions.OverlayDir(m.activeDrill.ID)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.RemoveAll(dir); err != nil {
					t.Fatal(err)
				}
			case "student syntax":
				writeTestWorkspace(t, m, "/* unterminated")
			case "reference syntax":
				writeTestReference(t, m.activeDrill.ID, "/* unterminated")
			case "unsupported language":
				d := *m.activeDrill
				d.Language = "python"
				m.activeDrill = &d
			case "workspace missing":
				if err := os.RemoveAll(m.session.Workspace); err != nil {
					t.Fatal(err)
				}
			}
			msg := runEvaluation(t, m)
			if msg.referenceErr == nil || msg.reference != nil || !strings.Contains(ansi.Strip(m.View()), "Reference: unavailable") {
				t.Fatalf("comparison error was not preserved: %+v", msg)
			}
			m.Update(runeKey("v"))
			assertNoReferenceDiff(t, m)
		})
	}
}

func TestPostSubmissionDiffSizesNavigationAndReturn(t *testing.T) {
	for _, size := range [][2]int{{60, 20}, {80, 24}} {
		m := newReferenceReviewModel(t, mode.Recall(), reviewStudent)
		writeTestReference(t, m.activeDrill.ID, strings.Repeat("int missing_token;\n", 100))
		runEvaluation(t, m)
		m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m.scrollOffset = 1
		before, session := m.View(), m.session
		for _, close := range []tea.KeyMsg{runeKey("q"), {Type: tea.KeyEsc}, {Type: tea.KeyCtrlC}} {
			m.Update(runeKey("v"))
			m.Update(tea.KeyMsg{Type: tea.KeyEnd})
			if !strings.Contains(m.View(), "- 100: missing_token") {
				t.Fatalf("%v: diff cannot scroll to final token: %s", size, m.View())
			}
			for _, key := range []tea.KeyMsg{runeKey("e"), runeKey("r"), runeKey("s"), runeKey("b"), runeKey("n"), runeKey("p"), {Type: tea.KeyPgUp}, {Type: tea.KeyDown}} {
				m.Update(key)
			}
			m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			lines := strings.Split(m.View(), "\n")
			if len(lines) > size[1] {
				t.Fatalf("%v: diff has %d rows", size, len(lines))
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > size[0] {
					t.Fatalf("%v: overflowing row %q", size, line)
				}
			}
			_, cmd := m.Update(close)
			if cmd != nil || m.referenceDiff != nil || m.state != StateEvaluating || m.session != session || m.View() != before || len(m.history.Attempts) != 0 {
				t.Fatal("closing diff did not restore the evaluation without side effects")
			}
		}
		clickHelp(t, m, "[v] Reference diff", 0)
		if m.referenceDiff == nil {
			t.Fatal("diff footer action is not clickable")
		}
		clickHelp(t, m, "[Esc/q] back", 0)
		if m.referenceDiff != nil || m.View() != before {
			t.Fatal("diff back action is not clickable or lost evaluation scroll")
		}
	}
}

func TestPostSubmissionDiffRejectsStaleEvaluations(t *testing.T) {
	m := newReferenceReviewModel(t, mode.Study(), reviewStudent)
	_, firstCmd := m.Update(sprintFinishedMsg{})
	first := firstCmd().(evalFinishedMsg)
	oldSession := m.session
	m.session = &runner.Session{Workspace: t.TempDir(), EntryFile: "solution.c", TestCommand: "true"}
	writeTestWorkspace(t, m, reviewReference)
	m.state = StateSprinting
	m.sprintStartTime = time.Now().Add(-time.Second)
	_, secondCmd := m.Update(sprintFinishedMsg{})
	second := secondCmd().(evalFinishedMsg)
	m.Update(first)
	if m.evalReview != nil {
		t.Fatal("stale evaluation granted reference access")
	}
	wrongSession := first
	wrongSession.id = second.id
	m.Update(wrongSession)
	assertNoReferenceDiff(t, m)
	m.Update(second)
	m.evalReadyTime = time.Now().Add(-time.Second)
	m.Update(runeKey("v"))
	view := m.View()
	for _, stale := range []evalFinishedMsg{first, wrongSession, second} {
		stale.reference = first.reference
		m.Update(stale)
		if m.View() != view {
			t.Fatal("stale or duplicate message replaced the open diff")
		}
	}
	m.Update(runeKey("q"))
	pressFinish(t, m)
	m.Update(first)
	m.Update(runeKey("v"))
	if !strings.Contains(m.View(), "Normalized sources match.") {
		t.Fatal("old evaluation replaced saved results")
	}
	oldSession.Cleanup()
}

func TestPostSubmissionDiffRetryAndResultNavigation(t *testing.T) {
	m := newReferenceReviewModel(t, mode.Study(), reviewStudent)
	runEvaluation(t, m)
	pressFinish(t, m)
	oldReview := m.resultReview
	for _, key := range []string{"b", "l"} {
		m.Update(runeKey(key))
		m.Update(runeKey("n"))
		m.Update(tea.KeyMsg{Type: tea.KeyEsc})
		m.Update(runeKey("v"))
		if m.referenceDiff == nil || m.resultReview != oldReview || !strings.Contains(m.View(), "REVIEW_REFERENCE_SECRET") {
			t.Fatal("browsing another drill changed the saved reference diff")
		}
		m.Update(runeKey("q"))
	}
	// Cancel a later countdown whose snapshot disables reference checking.
	m.Update(runeKey("b"))
	m.applyBriefingPreset(mode.Exam())
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != StateCountdown || m.runMode.CheckReference {
		t.Fatal("next attempt did not snapshot Exam")
	}
	assertNoReferenceDiff(t, m)
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m.Update(runeKey("v"))
	if m.referenceDiff == nil || m.resultReview != oldReview {
		t.Fatal("cancelled countdown changed earlier results' visibility")
	}
	m.Update(runeKey("q"))
	// Retry launches a different workspace; old evaluation data is inaccessible.
	m.Update(runeKey("r"))
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.state != StateSprinting || m.session == nil {
		t.Fatal("retry did not launch a fresh session")
	}
	t.Cleanup(m.Cleanup)
	assertNoReferenceDiff(t, m)
	if err := os.WriteFile(filepath.Join(m.session.Workspace, m.session.EntryFile), []byte(reviewStudent), 0644); err != nil {
		t.Fatal(err)
	}
	m.session.TestCommand = "true"
	runEvaluation(t, m)
	assertNoReferenceDiff(t, m)
	pressFinish(t, m)
	assertNoReferenceDiff(t, m)
	if len(m.history.Attempts) != 2 || m.state != StateResults {
		t.Fatal("retry changed pass recording")
	}
	m.Update(runeKey("n"))
	if m.state != StateCountdown {
		t.Fatal("next drill no longer starts its countdown")
	}
	assertNoReferenceDiff(t, m)
}

func TestPostSubmissionDiffUnavailableAfterForfeitOrSprintError(t *testing.T) {
	for _, kind := range []string{"forfeit", "launch error", "later sprint error", "resume"} {
		t.Run(kind, func(t *testing.T) {
			m := newReferenceReviewModel(t, mode.Recall(), reviewStudent)
			if kind == "launch error" || kind == "later sprint error" {
				if kind == "launch error" {
					m.sprintStartTime = time.Now()
				}
				_, cmd := m.Update(sprintFinishedMsg{err: errors.New("session interrupted")})
				if kind == "later sprint error" {
					m.Update(cmd())
				}
			} else {
				runEvaluation(t, m)
				if kind == "forfeit" {
					m.Update(runeKey("q"))
					if len(m.history.Attempts) != 1 || m.history.Attempts[0].Passed {
						t.Fatal("forfeit recording changed")
					}
				} else {
					m.Update(runeKey("e"))
				}
			}
			assertNoReferenceDiff(t, m)
		})
	}
}

func TestPostSubmissionDiffSaveErrorRecovery(t *testing.T) {
	for _, forfeit := range []bool{false, true} {
		t.Run(fmt.Sprintf("forfeit=%v", forfeit), func(t *testing.T) {
			m := newReferenceReviewModel(t, mode.Study(), reviewStudent)
			if forfeit {
				m.session.TestCommand = "exit 1"
			}
			runEvaluation(t, m)
			path, err := storage.GetHistoryPath()
			if err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(path, 0755); err != nil {
				t.Fatal(err)
			}
			if forfeit {
				m.Update(runeKey("q"))
			} else {
				pressFinish(t, m)
			}
			if m.saveErr == nil || m.state != StateEvaluating || len(m.history.Attempts) != 0 {
				t.Fatal("failed save changed evaluation or history")
			}
			if forfeit {
				assertNoReferenceDiff(t, m)
			} else {
				clickHelp(t, m, "[v] Reference diff", 0)
				if m.referenceDiff == nil {
					t.Fatal("failed pass save lost its reference diff")
				}
				m.Update(runeKey("q"))
				if m.saveErr == nil {
					t.Fatal("closing diff cleared the save error")
				}
			}
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			if len(m.history.Attempts) != 1 {
				t.Fatal("retry save did not record exactly once")
			}
			if forfeit {
				assertNoReferenceDiff(t, m)
			} else if m.resultReview == nil || m.state != StateResults {
				t.Fatal("retry save lost the completed diff")
			}
		})
	}
}

func TestPostSubmissionDiffRequiresEndedZellijSession(t *testing.T) {
	name := "phase8b ' $(touch injected); session"
	for _, tc := range []struct {
		name, reply string
		exitCode    string
		allowed     bool
	}{
		{"detached live session", `[{"id":1}]`, "0", false},
		{"ended session", fmt.Sprintf("Session '%s' not found. The following sessions are active:", name), "0", true},
		{"no active sessions", "There is no active session!", "1", true},
		{"other missing session", "Session 'other' not found.", "0", false},
		{"unknown error", "cannot inspect sessions", "1", false},
		{"invalid output", "", "0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newReferenceReviewModel(t, mode.Study(), reviewStudent)
			m.session.SessionName = name
			bin := t.TempDir()
			log := filepath.Join(bin, "args")
			stub := "#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$REVIEW_ARGS\"\nprintf '%s\\n' \"$REVIEW_REPLY\" >&2\nexit \"$REVIEW_EXIT\"\n"
			if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte(stub), 0755); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("REVIEW_ARGS", log)
			t.Setenv("REVIEW_REPLY", tc.reply)
			t.Setenv("REVIEW_EXIT", tc.exitCode)
			_, evaluate := m.Update(sprintFinishedMsg{zellij: true})
			m.Update(evaluate())
			m.evalReadyTime = time.Now().Add(-time.Second)
			if !m.passVerdict().Passed || m.evalRef == nil {
				t.Fatal("session completion check changed scores or the pass gate")
			}
			if tc.allowed {
				m.Update(runeKey("v"))
				if m.referenceDiff == nil {
					t.Fatal("confirmed submission did not unlock its diff")
				}
				m.Update(runeKey("q"))
			} else {
				assertNoReferenceDiff(t, m)
			}
			data, err := os.ReadFile(log)
			if err != nil || string(data) != "--session\n"+name+"\naction\nlist-panes\n--json\n" {
				t.Fatalf("session probe arguments: %q, %v", data, err)
			}
			m.session.SessionName = ""
			pressFinish(t, m)
			if !tc.allowed {
				assertNoReferenceDiff(t, m)
			}
		})
	}
}

func TestReferenceSessionEndedFailsClosed(t *testing.T) {
	if referenceSessionEnded("") {
		t.Fatal("empty session name unlocked review")
	}
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	if referenceSessionEnded("phase8b") {
		t.Fatal("missing Zellij command unlocked review")
	}
	if err := os.WriteFile(filepath.Join(bin, "zellij"), []byte("#!/bin/sh\nexec /bin/sleep 5\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if referenceSessionEnded("phase8b") {
		t.Fatal("timed-out session probe unlocked review")
	}
}
