package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/solutions"
)

func TestCPPModeEvaluationAndPostSubmissionDiff(t *testing.T) {
	reference := "decltype(auto) f() {\nreturn x;\n}"
	cases := []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()}
	for bits := 0; bits < 16; bits++ {
		cases = append(cases, mode.Options{CheckReference: bits&1 != 0, ShowReference: bits&2 != 0, RequireExact: bits&4 != 0, RequireTests: bits&8 != 0})
	}
	for _, options := range cases {
		for _, source := range []string{reference, strings.Replace(reference, "return x", "return (x)", 1), "/* unfinished"} {
			t.Run(fmt.Sprintf("%+v/%s", options, source), func(t *testing.T) {
				m := newEvaluationTestModel(t)
				d := drill.Drill{ID: "cpp-evaluation", Title: "C++ evaluation", Language: "cpp", EntryFile: "solution.cpp", StudentFiles: []string{"solution.hpp"}}
				m.activeDrill = &d
				m.session.EntryFile = d.EntryFile
				m.session.TestCommand = "true"
				m.runMode = options
				dir, err := solutions.OverlayDir(d.ID)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(dir, 0755); err != nil {
					t.Fatal(err)
				}
				for root, body := range map[string]string{dir: reference, m.session.Workspace: source} {
					for name, content := range map[string]string{"solution.cpp": body, "solution.hpp": "int x;", "main.cpp": "provided @"} {
						if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
							t.Fatal(err)
						}
					}
				}
				m.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
				assertNoReferenceDiff(t, m)
				msg := runEvaluation(t, m)
				normalized := options.Normalize()
				wantPass := normalized.Validate() == nil && (!normalized.RequireExact || source == reference)
				if m.passVerdict().Passed != wantPass {
					t.Fatalf("verdict: %+v; comparison %+v, %v", m.passVerdict(), msg.reference, msg.referenceErr)
				}
				if !options.CheckReference {
					if msg.reference != nil || msg.referenceErr != nil {
						t.Fatal("disabled comparison ran")
					}
					assertNoReferenceDiff(t, m)
					return
				}
				if source == "/* unfinished" {
					if msg.referenceErr == nil || msg.reference != nil {
						t.Fatal("lexical errors did not fail closed")
					}
					assertNoReferenceDiff(t, m)
					return
				}
				if msg.referenceErr != nil || msg.reference == nil || len(msg.reference.Files) != 2 {
					t.Fatalf("comparison failed: %+v, %v", msg.reference, msg.referenceErr)
				}
				// Recall unlocks review only after submission, as C already does.
				m.Update(runeKey("v"))
				if m.referenceDiff == nil || !strings.Contains(m.View(), "solution.cpp") {
					t.Fatalf("post-submission diff unavailable: %s", m.View())
				}
				m.Update(runeKey("q"))
				if wantPass {
					pressFinish(t, m)
					m.Update(runeKey("v"))
					if m.state != StateResults || m.referenceDiff == nil {
						t.Fatal("results lost the submitted C++ diff")
					}
				}
			})
		}
	}
}

func TestCPPBriefingAllowsCopyAndRecall(t *testing.T) {
	for _, language := range []string{"cpp", "c++", "cc", "cxx"} {
		m := newBriefingModeModel(t, "ft_strlen")
		writeTestReference(t, m.activeDrill.ID, testReference)
		m.activeDrill.Language = language
		for _, options := range []mode.Options{mode.Copy(), mode.Recall()} {
			m.setBriefingOptions(m.activeDrill, options)
			if problem := m.briefingProblem(); problem != "" {
				t.Fatalf("%s rejected: %s", language, problem)
			}
		}
	}
}
