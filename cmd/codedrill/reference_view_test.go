package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/runner"
)

const viewerSource = "int reference_secret(void) { return 42; }\n"

func viewerFixture(t *testing.T, options mode.Options) (runner.ReferenceSession, string, string) {
	t.Helper()
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	workspace := filepath.Join(t.TempDir(), "work ' $(touch injected); space")
	if err := os.Mkdir(workspace, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workspace, "life.c"), []byte("student source"), 0644); err != nil {
		t.Fatal(err)
	}
	refDir := filepath.Join(config, "codedrill", "solutions", "life")
	if err := os.MkdirAll(refDir, 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range map[string]string{"life.c": viewerSource, "life.h": "#define HEADER 1\n", "notes.txt": "not source"} {
		if err := os.WriteFile(filepath.Join(refDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	drillDir := filepath.Join(config, "codedrill", "drills", "life")
	if err := os.MkdirAll(drillDir, 0755); err != nil {
		t.Fatal(err)
	}
	definition := `{"id":"life","title":"life","category":"Custom","language":"c","difficulty":"easy","entry_file":"life.c","test_command":"true"}`
	if err := os.WriteFile(filepath.Join(drillDir, "drill.json"), []byte(definition), 0644); err != nil {
		t.Fatal(err)
	}
	snapshot := runner.ReferenceSession{Version: 1, Workspace: workspace, DrillID: "life", DrillDir: drillDir, EntryFile: "life.c", Language: "c", Options: &options}
	path := filepath.Join(t.TempDir(), "session ' $(touch injected);.json")
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	return snapshot, path, refDir
}

func TestViewerVisibilityUsesIndependentSnapshotOptions(t *testing.T) {
	for bits := 0; bits < 16; bits++ {
		options := mode.Options{CheckReference: bits&1 != 0, ShowReference: bits&2 != 0, RequireExact: bits&4 != 0, RequireTests: bits&8 != 0}
		snapshot, path, _ := viewerFixture(t, options)
		binDir := t.TempDir()
		log := filepath.Join(binDir, "arguments")
		if err := os.WriteFile(filepath.Join(binDir, "zellij"), []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \"$VIEWER_ARGS\"\n"), 0755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		t.Setenv("VIEWER_ARGS", log)
		t.Setenv("ZELLIJ", "1")
		if err := mode.Save("life", mode.Exam()); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := runCLI(t, "reference-view", "--session", path, "--open")
		if !options.ShowReference {
			if code == 0 || out != "" || !strings.Contains(errOut, "reference is hidden") {
				t.Fatalf("options %+v: exit %d, output %q %q", options, code, out, errOut)
			}
			if _, err := os.Stat(log); !os.IsNotExist(err) {
				t.Fatal("hidden reference opened a pane")
			}
			continue
		}
		if code != 0 || out != "" || errOut != "" {
			t.Fatalf("options %+v: exit %d, output %q %q", options, code, out, errOut)
		}
		data, err := os.ReadFile(log)
		if err != nil {
			t.Fatal(err)
		}
		exe, _ := os.Executable()
		want := []string{"action", "new-pane", "--floating", "--close-on-exit", "--name", "Reference", "--cwd", snapshot.Workspace, "--", exe, "reference-view", "--session", path}
		if got := strings.Split(strings.TrimSpace(string(data)), "\n"); !reflect.DeepEqual(got, want) {
			t.Fatalf("zellij arguments: %q, want %q", got, want)
		}
		entries, err := os.ReadDir(snapshot.Workspace)
		if err != nil || len(entries) != 1 || entries[0].Name() != "life.c" {
			t.Fatalf("viewer added workspace files: %v, %v", entries, err)
		}
		if data, _ := os.ReadFile(filepath.Join(snapshot.Workspace, "life.c")); string(data) != "student source" {
			t.Fatalf("viewer altered the student source: %q", data)
		}
	}
}

func TestViewerFilesAndMissingReferences(t *testing.T) {
	snapshot, path, refDir := viewerFixture(t, mode.Copy())
	files, err := loadViewerFiles(snapshot)
	if err != nil || len(files) != 2 || string(files["life.c"]) != viewerSource || string(files["life.h"]) != "#define HEADER 1\n" {
		t.Fatalf("viewer files = %q, %v", files, err)
	}
	if err := os.RemoveAll(refDir); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLI(t, "reference-view", "--session", path, "--open")
	if code == 0 || out != "" || !strings.Contains(errOut, "reference files are unavailable") || strings.Contains(errOut, viewerSource) {
		t.Fatalf("missing reference: %d %q %q", code, out, errOut)
	}
}

func TestViewerUsesCurrentSolutionPrecedence(t *testing.T) {
	snapshot, _, overlay := viewerFixture(t, mode.Copy())
	bundled := filepath.Join(snapshot.DrillDir, "solution")
	if err := os.MkdirAll(bundled, 0755); err != nil {
		t.Fatal(err)
	}
	const source = "int bundled_reference(void) { return 1; }\n"
	if err := os.WriteFile(filepath.Join(bundled, "life.c"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	files, err := loadViewerFiles(snapshot)
	if err != nil || string(files["life.c"]) != viewerSource {
		t.Fatalf("overlay was not preferred: %q, %v", files, err)
	}
	if err := os.RemoveAll(overlay); err != nil {
		t.Fatal(err)
	}
	files, err = loadViewerFiles(snapshot)
	if err != nil || len(files) != 1 || string(files["life.c"]) != source {
		t.Fatalf("bundled fallback: %q, %v", files, err)
	}
}

func TestHiddenSourcesCannotBeReenabledWithCLIFlags(t *testing.T) {
	_, path, _ := viewerFixture(t, mode.Recall())
	for _, args := range [][]string{
		{"reference-view"},
		{"reference-view", "--session", path},
		{"reference-view", "--session", path, "--open"},
		{"reference-view", "--session", path, "--mode", "copy"},
		{"match", "life", "--reference"},
		{"match", "life", "--diff"},
	} {
		code, out, errOut := runCLI(t, args...)
		if code == 0 || out != "" || strings.Contains(out+errOut, "reference_secret") {
			t.Fatalf("hidden reference escaped through %v: %d %q %q", args, code, out, errOut)
		}
	}
	for _, flag := range []string{"--json", ""} {
		workspace := filepath.Join(t.TempDir(), "work")
		if err := os.MkdirAll(workspace, 0755); err != nil {
			t.Fatal(err)
		}
		args := []string{"match", "life", "--workspace", workspace}
		if flag != "" {
			args = append(args, flag)
		}
		code, out, errOut := runCLI(t, args...)
		if code != 0 || strings.Contains(out+errOut, "reference_secret") {
			t.Fatalf("score output: %d %q %q", code, out, errOut)
		}
	}
}
