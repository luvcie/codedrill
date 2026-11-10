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
)

func TestExam05EmbeddedViewer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range drill.Filter(drills, "42/Exam 05") {
		code, out, errOut := runCLI(t, "solution", d.ID)
		if code != 0 || !strings.Contains(out, "Source: built-in") || !strings.Contains(out, d.EntryFile) {
			t.Fatalf("%s reference listing: %d, %s, %s", d.ID, code, out, errOut)
		}
		for _, options := range []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()} {
			workspace := t.TempDir()
			snapshot := runner.ReferenceSession{Version: 1, Workspace: workspace, DrillID: d.ID,
				EntryFile: d.EntryFile, Language: d.Language, StudentFiles: d.StudentFiles, Options: &options}
			policy := filepath.Join(t.TempDir(), "session.json")
			data, err := json.Marshal(snapshot)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(policy, data, 0600); err != nil {
				t.Fatal(err)
			}
			visible, err := visibleReferenceSession(policy)
			if !options.ShowReference {
				if err != errReferenceHidden {
					t.Fatalf("%s exposed hidden reference: %v", d.ID, err)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			files, err := loadViewerFiles(visible)
			if err != nil || len(files[d.EntryFile]) == 0 {
				t.Fatalf("%s viewer has no embedded reference: %v", d.ID, err)
			}
			entries, err := os.ReadDir(workspace)
			if err != nil || len(entries) != 0 {
				t.Fatalf("viewer extracted references into the workspace: %v, %v", entries, err)
			}
		}
	}
}
