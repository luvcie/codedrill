package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/runner"
)

type paneTestState struct {
	Panes   []referencePane
	Focus   string
	Actions [][]string
	NextID  uint32
	Quits   int
}

// The fake CLI runs in a separate process, exercising exec argument boundaries
// and retaining pane state across successive controller invocations.
func TestReferencePaneProcess(t *testing.T) {
	path := os.Getenv("CODEDRILL_TEST_PANES")
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	var state paneTestState
	if err := json.Unmarshal(data, &state); err != nil {
		panic(err)
	}
	args := os.Args
	for len(args) > 0 && args[0] != "--" {
		args = args[1:]
	}
	args = args[1:]
	state.Actions = append(state.Actions, args)
	value := func(flag string) string {
		for i := range args {
			if args[i] == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
		return ""
	}
	failed := args[1] == os.Getenv("CODEDRILL_TEST_PANE_FAILURE")
	if !failed {
		switch args[1] {
		case "list-panes":
			data, _ := json.Marshal(state.Panes)
			_, _ = os.Stdout.Write(data)
		case "new-pane":
			state.Quits = 0
			for i := range state.Panes {
				if state.Panes[i].paneID() == value("--pane-id") {
					state.Panes[i].Suppressed = true
				}
			}
			command := args
			for command[0] != "--" {
				command = command[1:]
			}
			state.Panes = append(state.Panes, referencePane{ID: state.NextID, Command: strings.Join(command[1:], " ")})
			state.NextID++
		case "send-keys":
			if args[len(args)-1] != "q" {
				panic("expected viewer quit key")
			}
			if state.Focus != value("--pane-id") {
				panic("focus the viewer before restoring an in-place pane")
			}
			state.Quits++
			if os.Getenv("CODEDRILL_TEST_VIEWER_STARTING") != "" && state.Quits == 1 {
				break
			}
			for i, p := range state.Panes {
				if p.paneID() == value("--pane-id") {
					state.Panes = append(state.Panes[:i], state.Panes[i+1:]...)
					break
				}
			}
			for i := range state.Panes {
				state.Panes[i].Suppressed = false
			}
		case "focus-pane-id":
			state.Focus = args[2]
		}
	}
	data, _ = json.Marshal(state)
	if err := os.WriteFile(path, data, 0600); err != nil {
		panic(err)
	}
	if failed {
		_, _ = os.Stderr.WriteString(viewerSource)
		os.Exit(1)
	}
	os.Exit(0)
}

func installPaneCLI(t *testing.T, session runner.ReferenceSession) func() paneTestState {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	state := paneTestState{Panes: []referencePane{
		{ID: 7, Command: "bash " + filepath.Join(session.Workspace, "start-editor.sh")},
		{ID: 12, Command: "bash " + filepath.Join(session.Workspace, "start-terminal.sh")},
	}, Focus: "terminal_12", NextID: 20}
	data, _ := json.Marshal(state)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	exe, _ := os.Executable()
	t.Setenv("CODEDRILL_TEST_PANES", path)
	t.Setenv("CODEDRILL_TEST_BINARY", exe)
	t.Setenv("ZELLIJ", "1")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	if err := os.WriteFile(filepath.Join(dir, "zellij"), []byte("#!/usr/bin/env bash\nexec \"$CODEDRILL_TEST_BINARY\" -test.run='^TestReferencePaneProcess$' -- \"$@\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	return func() paneTestState {
		t.Helper()
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var state paneTestState
		if err := json.Unmarshal(data, &state); err != nil {
			t.Fatal(err)
		}
		return state
	}
}

func TestReferencePaneToggleAndReturn(t *testing.T) {
	snapshot, policy, refs := viewerFixture(t, mode.Copy())
	state := installPaneCLI(t, snapshot)
	before, _ := os.ReadFile(policy)
	if err := mode.Save("life", mode.Exam()); err != nil {
		t.Fatal(err)
	}
	for cycle := 0; cycle < 3; cycle++ {
		code, out, errOut := runCLI(t, "reference-view", "--session", policy, "--toggle-pane")
		if code != 0 || out != "" || errOut != "" {
			t.Fatalf("open: %d %q %q", code, out, errOut)
		}
		s := state()
		if len(s.Panes) != 3 || !s.Panes[1].Suppressed || s.Focus != "terminal_7" {
			t.Fatalf("reference must replace terminal and focus editor: %+v", s)
		}
		exe, _ := os.Executable()
		want := []string{"action", "new-pane", "--in-place", "--pane-id", "terminal_12", "--no-focus", "--close-on-exit", "--name", "Reference beside editor", "--cwd", snapshot.Workspace, "--", exe, "reference-view", "--session", policy, "--side-pane"}
		if got := s.Actions[len(s.Actions)-2]; !reflect.DeepEqual(got, want) {
			t.Fatalf("new-pane arguments = %q, want %q", got, want)
		}
		if cycle == 2 {
			// Closing must still restore the shell if reference files disappear.
			if err := os.RemoveAll(refs); err != nil {
				t.Fatal(err)
			}
		}
		code, out, errOut = runCLI(t, "reference-view", "--session", policy, "--toggle-pane")
		if code != 0 || out != "" || errOut != "" {
			t.Fatalf("close: %d %q %q", code, out, errOut)
		}
		s = state()
		if len(s.Panes) != 2 || s.Panes[1].Suppressed || s.Focus != "terminal_12" {
			t.Fatalf("original terminal not restored: %+v", s)
		}
	}
	after, _ := os.ReadFile(policy)
	if string(before) != string(after) {
		t.Fatal("toggle changed the sprint snapshot")
	}
	entries, _ := os.ReadDir(snapshot.Workspace)
	if len(entries) != 1 || entries[0].Name() != "life.c" {
		t.Fatalf("viewer wrote into workspace: %v", entries)
	}
	if data, _ := os.ReadFile(filepath.Join(snapshot.Workspace, "life.c")); string(data) != "student source" {
		t.Fatal("viewer changed student source")
	}
}

func TestReferencePaneVisibility(t *testing.T) {
	for bits := 0; bits < 16; bits++ {
		opts := mode.Options{CheckReference: bits&1 != 0, ShowReference: bits&2 != 0, RequireExact: bits&4 != 0, RequireTests: bits&8 != 0}
		snapshot, policy, _ := viewerFixture(t, opts)
		state := installPaneCLI(t, snapshot)
		code, out, errOut := runCLI(t, "reference-view", "--session", policy, "--toggle-pane")
		if opts.ShowReference {
			if code != 0 || len(state().Panes) != 3 {
				t.Fatalf("custom visible options %+v: %d %q", opts, code, errOut)
			}
		} else if code == 0 || out != "" || len(state().Actions) != 0 || !strings.Contains(errOut, "reference is hidden") {
			t.Fatalf("hidden options %+v: %d %q %q", opts, code, out, errOut)
		}
	}
}

func TestConcurrentReferencePaneToggles(t *testing.T) {
	snapshot, policy, _ := viewerFixture(t, mode.Study())
	state := installPaneCLI(t, snapshot)
	var wg sync.WaitGroup
	errors := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Go(func() { errors <- (&cli{}).referenceView([]string{"--session", policy, "--toggle-pane"}) })
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	s := state()
	if len(s.Panes) != 2 || s.Panes[1].Suppressed {
		t.Fatalf("concurrent toggles left duplicate panes: %+v", s)
	}
	var transitions []string
	for _, args := range s.Actions {
		if args[1] == "new-pane" || args[1] == "send-keys" {
			transitions = append(transitions, args[1])
		}
	}
	if want := strings.Repeat("new-pane send-keys ", 4); strings.Join(transitions, " ")+" " != want {
		t.Fatalf("toggles overlapped: %v", transitions)
	}
}

func TestReferencePaneCloseDuringStartup(t *testing.T) {
	snapshot, policy, _ := viewerFixture(t, mode.Copy())
	state := installPaneCLI(t, snapshot)
	if err := (&cli{}).referenceView([]string{"--session", policy, "--toggle-pane"}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEDRILL_TEST_VIEWER_STARTING", "1")
	if err := (&cli{}).referenceView([]string{"--session", policy, "--toggle-pane"}); err != nil {
		t.Fatal(err)
	}
	if s := state(); s.Quits != 2 || len(s.Panes) != 2 || s.Focus != "terminal_12" {
		t.Fatalf("startup toggle did not restore original terminal: %+v", s)
	}
}

func TestReferencePaneRejectsDuplicatePanes(t *testing.T) {
	snapshot, policy, _ := viewerFixture(t, mode.Copy())
	state := installPaneCLI(t, snapshot)
	if err := (&cli{}).referenceView([]string{"--session", policy, "--toggle-pane"}); err != nil {
		t.Fatal(err)
	}
	s := state()
	duplicate := s.Panes[2]
	duplicate.ID = 99
	s.Panes = append(s.Panes, duplicate)
	s.Actions = nil
	data, _ := json.Marshal(s)
	if err := os.WriteFile(os.Getenv("CODEDRILL_TEST_PANES"), data, 0600); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLI(t, "reference-view", "--session", policy, "--toggle-pane")
	if code == 0 || out != "" || !strings.Contains(errOut, "ambiguous") || len(state().Actions) != 1 {
		t.Fatalf("duplicate discovery changed panes: %d %q %q %+v", code, out, errOut, state())
	}
}

func TestReferencePaneInvalidPolicyAndFailures(t *testing.T) {
	snapshot, policy, _ := viewerFixture(t, mode.Recall())
	state := installPaneCLI(t, snapshot)
	for _, args := range [][]string{
		{"--session", policy, "--toggle-pane"},
		{"--session", policy, "--side-pane"},
		{"--session", policy, "--toggle-pane", "--mode", "copy"},
		{"--session", policy + ".missing", "--toggle-pane"},
	} {
		code, out, errOut := runCLI(t, append([]string{"reference-view"}, args...)...)
		if code == 0 || out != "" || strings.Contains(errOut, "reference_secret") || len(state().Actions) != 0 {
			t.Fatalf("unauthorized pane request: %d %q %q", code, out, errOut)
		}
	}
	for _, invalid := range []string{"{", `{"version":1,"workspace":"/tmp/work","drill_id":"life","options":null}`, `{"version":1,"workspace":"/tmp/work","drill_id":"life","options":{"show_reference":"true"}}`} {
		if err := os.WriteFile(policy, []byte(invalid), 0600); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := runCLI(t, "reference-view", "--session", policy, "--toggle-pane")
		if code == 0 || out != "" || !strings.Contains(errOut, "policy is missing or invalid") || len(state().Actions) != 0 {
			t.Fatalf("invalid policy: %d %q %q", code, out, errOut)
		}
	}
	snapshot.Options.ShowReference = true
	data, _ := json.Marshal(snapshot)
	if err := os.WriteFile(policy, data, 0600); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"list-panes", "new-pane"} {
		t.Setenv("CODEDRILL_TEST_PANE_FAILURE", action)
		code, out, errOut := runCLI(t, "reference-view", "--session", policy, "--toggle-pane")
		if code == 0 || out != "" || strings.Contains(errOut, "reference_secret") || len(state().Panes) != 2 {
			t.Fatalf("failed %s: %d %q %q", action, code, out, errOut)
		}
	}
}
