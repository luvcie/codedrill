package ui

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestStudentWorkSurvivesResumeButNotRetry(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	zero, sound := 0, false
	if err := storage.SaveConfig(storage.Config{Countdown: &zero, Sound: &sound}); err != nil {
		t.Fatal(err)
	}
	m.runMode = mode.Exam()
	m.launchSprintSession()
	if m.session == nil {
		t.Fatal(m.setupErr)
	}
	t.Cleanup(m.Cleanup)
	first := m.session
	path := filepath.Join(first.Workspace, first.EntryFile)
	if err := os.WriteFile(path, []byte("student work"), 0644); err != nil {
		t.Fatal(err)
	}
	m.state = StateEvaluating
	m.updateEvaluating(runeKey("e"))
	if data, err := os.ReadFile(path); err != nil || string(data) != "student work" || m.session != first {
		t.Fatalf("resume replaced student work: %q %v", data, err)
	}
	m.resultDrill = *m.activeDrill
	m.endSession()
	if _, err := os.Stat(first.Workspace); !os.IsNotExist(err) {
		t.Fatal("finished workspace survived cleanup")
	}
	m.state = StateResults
	m.updateResults(runeKey("r"))
	if m.session == nil || m.session.Workspace == first.Workspace {
		t.Fatalf("retry did not create a new workspace: %v", m.setupErr)
	}
	data, err := os.ReadFile(filepath.Join(m.session.Workspace, m.session.EntryFile))
	if err != nil || len(data) != 0 {
		t.Fatalf("retry did not start empty: %q %v", data, err)
	}
}

func TestEditingDrillPreservesOwnershipAndResources(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	d := drill.Drill{ID: "edit-owned", Title: "Owned", Category: "Custom", Language: "cpp", EntryFile: "main.cpp",
		StarterCode: "legacy source", StudentFiles: []string{"part.cpp"}, OptionalStudentFiles: []string{"extra.hpp"}, ExtraFiles: map[string]string{"test.cpp": "provided test"}}
	m.editDrill = &d
	m.initEditForm()
	m.saveEditDrill()
	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	got := drill.Find(drills, d.ID)
	if got == nil || !reflect.DeepEqual(got.StudentFiles, d.StudentFiles) || !reflect.DeepEqual(got.OptionalStudentFiles, d.OptionalStudentFiles) || !reflect.DeepEqual(got.ExtraFiles, d.ExtraFiles) || got.StarterCode != d.StarterCode {
		t.Fatalf("editing lost metadata or resources: %+v", got)
	}
}
