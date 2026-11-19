package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestRank5CategoryFilters(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	history, err := storage.LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	pb := 90 * time.Second
	if _, _, err := history.RecordAttempt("rank05-vect2", storage.ExamMode, pb, true, 1, false); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"list", "42/rank 5"}, {"list", "rank 5"}, {"search", "rank 5"}} {
		code, out, errOut := runCLI(t, args...)
		if code != 0 || !strings.Contains(out, "42/rank 5") || strings.Contains(out, "ft_strlen") {
			t.Fatalf("%v did not use the renamed category: %d, %s, %s", args, code, out, errOut)
		}
		for _, id := range []string{"rank05-vect2", "rank05-bigint", "rank05-bsq", "rank05-life", "rank05-polyset"} {
			if !strings.Contains(out, id) {
				t.Fatalf("%v lost the existing drill ID %s:\n%s", args, id, out)
			}
		}
		if !strings.Contains(out, storage.FormatDuration(pb)) {
			t.Fatalf("%v lost the existing personal best:\n%s", args, out)
		}
	}
	for _, command := range []string{"info", "top", "recent"} {
		code, out, errOut := runCLI(t, command, "rank05-vect2")
		if code != 0 || !strings.Contains(out, storage.FormatDuration(pb)) {
			t.Fatalf("%s lost the drill's saved history: %d, %s, %s", command, code, out, errOut)
		}
	}
}

func TestRank05EmbeddedViewer(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range drill.Filter(drills, "42/rank 5") {
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
