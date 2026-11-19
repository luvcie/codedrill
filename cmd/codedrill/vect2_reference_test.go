package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/solutions"
)

func TestVect2ReferenceSelectionAndModes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	d := drill.Find(drills, "rank05-vect2")
	if d == nil {
		t.Fatal("vect2 drill is missing")
	}
	workspace := t.TempDir()
	for _, student := range []solutions.Variant{solutions.HeaderOnly, solutions.SplitFiles} {
		source, err := solutions.ResolveVariant(d.ID, d.Dir, student)
		if err != nil {
			t.Fatal(err)
		}
		for _, file := range source.Files {
			data, err := file.Read()
			if err != nil {
				t.Fatal(err)
			}
			writeDiffFile(t, workspace, file.Rel, string(data))
		}
		for _, selected := range []solutions.Variant{solutions.HeaderOnly, solutions.SplitFiles} {
			for _, gates := range []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()} {
				session := runner.ReferenceSession{Version: 1, Workspace: workspace, DrillID: d.ID,
					EntryFile: d.EntryFile, Language: d.Language, StudentFiles: d.StudentFiles,
					Options: &gates, ReferenceVariant: selected}
				policy := filepath.Join(t.TempDir(), "session.json")
				data, err := json.Marshal(session)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(policy, data, 0600); err != nil {
					t.Fatal(err)
				}
				// Saved preferences may change after the sprint starts.
				other := solutions.HeaderOnly
				if selected == solutions.HeaderOnly {
					other = solutions.SplitFiles
				}
				if err := mode.SavePreferences(d.ID, mode.Preferences{Options: mode.Exam(), ReferenceVariant: other}); err != nil {
					t.Fatal(err)
				}
				visible, err := visibleReferenceSession(policy)
				if !gates.ShowReference {
					if err != errReferenceHidden {
						t.Fatalf("hidden reference was visible: %v", err)
					}
					if _, err := sprintDiff(policy); err != errReferenceHidden {
						t.Fatalf("hidden sprint diff was available: %v", err)
					}
				} else {
					if err != nil {
						t.Fatal(err)
					}
					files, err := loadViewerFiles(visible)
					if err != nil || (string(files["vect2.cpp"]) == "#include \"vect2.hpp\"\n") != (selected == solutions.HeaderOnly) {
						t.Fatalf("viewer used the wrong reference: %v", err)
					}
					result, err := sprintDiff(policy)
					if err != nil || result.Exact != (student == selected) {
						t.Fatalf("sprint diff used the wrong target: exact %v, %v", result.Exact, err)
					}
				}
				code, out, errOut := runCLI(t, "match", d.ID, "--session", policy, "--json")
				if !gates.CheckReference {
					if code == 0 || out != "" || !strings.Contains(errOut, "reference checking is disabled") {
						t.Fatalf("Exam match ignored run mode: %d, %q, %q", code, out, errOut)
					}
				} else {
					var report matchReport
					if code != 0 || json.Unmarshal([]byte(out), &report) != nil || report.Exact != (student == selected) {
						t.Fatalf("sprint match used the wrong target: %d, %q, %q", code, out, errOut)
					}
				}
			}
			if err := mode.SavePreferences(d.ID, mode.Preferences{Options: mode.Study(), ReferenceVariant: selected}); err != nil {
				t.Fatal(err)
			}
			code, out, errOut := runCLI(t, "match", d.ID, "--workspace", workspace, "--json")
			var report matchReport
			if code != 0 || json.Unmarshal([]byte(out), &report) != nil || report.Exact != (student == selected) {
				t.Fatalf("match ignored the saved choice: %d, %q, %q", code, out, errOut)
			}
		}
	}
}
