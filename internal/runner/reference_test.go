package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
)

func TestReferenceSessionSnapshotAndIsolation(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	options := mode.Recall()
	options.ShowReference = true
	s, err := NewSessionWithOptions(drill.Drill{ID: "life", Language: "c", EntryFile: "life.c", StarterCode: "starter"}, 0, "", Options{
		ReferenceOptions: &options,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Cleanup)
	path := s.referencePolicy
	options.ShowReference = false
	if err := mode.Save("life", mode.Exam()); err != nil {
		t.Fatal(err)
	}
	snapshot, err := LoadReferenceSession(path)
	if err != nil || !snapshot.Options.ShowReference || snapshot.DrillID != "life" || snapshot.EntryFile != "life.c" {
		t.Fatalf("snapshot changed: %+v, %v", snapshot, err)
	}
	if rel, err := filepath.Rel(s.Workspace, path); err != nil || filepath.IsLocal(rel) {
		t.Fatalf("policy is inside the workspace: %s", path)
	}
	if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("policy permissions: %v, %v", info, err)
	}
	if data, err := os.ReadFile(filepath.Join(s.Workspace, "life.c")); err != nil || len(data) != 0 {
		t.Fatalf("workspace source changed: %q, %v", data, err)
	}
	layout, err := os.ReadFile(filepath.Join(s.Workspace, "layout.kdl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`bind "Alt v"`, `bind "Alt b"`, `"--toggle-pane"`, "floating true", "close_on_exit true", `name "Reference"`} {
		if !strings.Contains(string(layout), want) {
			t.Errorf("layout missing %q", want)
		}
	}
	if strings.Count(string(layout), "pane command=") != 2 || strings.Contains(string(layout), "clear-defaults") {
		t.Fatal("viewer changed the tiled layout or reset user keybindings")
	}
	banner, err := os.ReadFile(filepath.Join(s.Workspace, "start-terminal.sh"))
	if err != nil || !strings.Contains(string(banner), "m -p  reference beside editor (Alt+b)") || !strings.Contains(string(banner), "m -v  reference peek (Alt+v)") || !strings.Contains(string(banner), "m -d  normalized reference diff") {
		t.Fatalf("welcome text is missing viewer controls: %v", err)
	}
	s.Cleanup()
	s.SessionName = ""
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("session metadata survived cleanup")
	}
}

func TestReferencePolicyFailsClosed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.json")
	for _, content := range []string{"", "{", "null", `{}`, `{"version":1,"workspace":"/tmp/work","drill_id":"life"}`, `{"version":1,"workspace":"/tmp/work","drill_id":"life","options":{"show_reference":"yes"}}`} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := LoadReferenceSession(path); err == nil {
			t.Fatalf("invalid policy accepted: %q", content)
		}
	}
	if _, err := LoadReferenceSession(path + ".missing"); err == nil {
		t.Fatal("missing policy accepted")
	}
}

func TestReferenceHelperQuotesPaths(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "codedrill ' $(touch injected); bin")
	if err := os.WriteFile(exe, []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	policy := filepath.Join(dir, "policy ' $(touch injected);.json")
	helper := filepath.Join(dir, "m")
	if err := os.WriteFile(helper, script(matchBody, "CODEDRILL", exe, "POLICY", policy), 0755); err != nil {
		t.Fatal(err)
	}
	for flag, action := range map[string]string{"-v": "--open", "-p": "--toggle-pane"} {
		out, err := exec.Command(helper, flag).CombinedOutput()
		if err != nil {
			t.Fatalf("helper: %v, %s", err, out)
		}
		if got, want := strings.Split(strings.TrimSpace(string(out)), "\n"), []string{"reference-view", "--session", policy, action}; !reflect.DeepEqual(got, want) {
			t.Fatalf("arguments = %q, want %q", got, want)
		}
	}
	for _, flag := range []string{"-d", "--diff"} {
		out, err := exec.Command(helper, flag, "--mode", "copy", "--session", "untrusted.json").CombinedOutput()
		if err != nil {
			t.Fatalf("diff helper: %v, %s", err, out)
		}
		if got, want := strings.Split(strings.TrimSpace(string(out)), "\n"), []string{"reference-diff", "--session", policy}; !reflect.DeepEqual(got, want) {
			t.Fatalf("diff arguments = %q, want %q", got, want)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "injected")); !os.IsNotExist(err) {
		t.Fatal("helper evaluated a path as shell code")
	}
}

func TestCPPReferenceSessionPreservesOwnership(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "vect2", Language: "cpp", EntryFile: "vect2.cpp",
		StudentFiles: []string{"vect2.hpp"}, OptionalStudentFiles: []string{"extra.cpp"},
		ExtraFiles: map[string]string{"main.cpp": "PROVIDED_RESOURCE_CONTENT"},
	}
	options := mode.Recall()
	s, err := NewSessionWithOptions(d, 0, "", Options{ReferenceOptions: &options})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Cleanup)
	snapshot, err := LoadReferenceSession(s.referencePolicy)
	if err != nil {
		t.Fatal(err)
	}
	got := snapshot.Drill()
	if !reflect.DeepEqual(got.StudentFiles, d.StudentFiles) || !reflect.DeepEqual(got.OptionalStudentFiles, d.OptionalStudentFiles) {
		t.Fatalf("ownership lost: %+v", got)
	}
	if content, ok := got.ExtraFiles["main.cpp"]; !ok || content != "" {
		t.Fatalf("policy should contain resource paths only: %+v", got.ExtraFiles)
	}
	d.StudentFiles[0] = "changed.hpp"
	loaded, err := LoadReferenceSession(s.referencePolicy)
	if err != nil || loaded.StudentFiles[0] != "vect2.hpp" || loaded.Options.ShowReference {
		t.Fatalf("ownership/visibility snapshot changed: %+v, %v", loaded, err)
	}
}
