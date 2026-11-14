package runner

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/typingsound"
)

func TestNewSessionAndWorkspace(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sess, err := NewSession(drill.Drill{
		ID:          "test-drill",
		Title:       "Test Title",
		Category:    "Algorithms",
		Language:    "python",
		Difficulty:  "easy",
		Description: "Write a python function to solve the problem.",
		EntryFile:   "solution.py",
		StarterCode: "def solve(): pass\n",
		TestCommand: "python3 test_suite.py",
		AutoSubmit:  true,
		ExtraFiles:  map[string]string{"test_suite.py": "print('ok')\n"},
	}, 5432*time.Millisecond)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer sess.Cleanup()

	if !strings.HasPrefix(sess.SessionName, "codedrill-") {
		t.Errorf("expected SessionName to start with codedrill-, got %s", sess.SessionName)
	}

	if _, err := os.Stat(sess.Workspace); os.IsNotExist(err) {
		t.Fatalf("workspace directory does not exist: %s", sess.Workspace)
	}

	problemPath := filepath.Join(sess.Workspace, "PROBLEM.md")
	problemData, err := os.ReadFile(problemPath)
	if err != nil {
		t.Errorf("expected PROBLEM.md to exist: %v", err)
	} else if !strings.Contains(string(problemData), "Write a python function") {
		t.Errorf("expected PROBLEM.md to contain description, got: %s", string(problemData))
	}

	if _, err := os.Stat(filepath.Join(sess.Workspace, "info")); os.IsNotExist(err) {
		t.Errorf("expected info script to exist")
	}
	if _, err := os.Stat(filepath.Join(sess.Workspace, "i")); os.IsNotExist(err) {
		t.Errorf("expected i shortcut to exist")
	}

	entryPath := filepath.Join(sess.Workspace, "solution.py")
	content, err := os.ReadFile(entryPath)
	if err != nil || len(content) != 0 {
		t.Errorf("entry file missing or invalid content")
	}

	extraPath := filepath.Join(sess.Workspace, "test_suite.py")
	if _, err := os.Stat(extraPath); os.IsNotExist(err) {
		t.Errorf("extra file missing: %s", extraPath)
	}

	pace, err := os.ReadFile(filepath.Join(sess.Workspace, "pace"))
	if err != nil || !strings.Contains(string(pace), "PB_MS='5432'") {
		t.Errorf("pace script lacks the PB: %v", err)
	}

	runScript := filepath.Join(sess.Workspace, "run")
	info, err := os.Stat(runScript)
	if err != nil {
		t.Fatalf("run script missing: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Errorf("run script is not executable")
	}

	if err := os.WriteFile(entryPath, []byte("def solve(): pass\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("./run")
	cmd.Dir = sess.Workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executing ./run failed: %v, output: %s", err, string(out))
	}

	outStr := string(out)
	if !strings.Contains(outStr, "pace ") || !strings.Contains(outStr, " rank") {
		t.Errorf("expected elapsed pace and rank in output, got: %s", outStr)
	}
	if strings.Contains(outStr, "invalid option") || strings.Contains(outStr, "printf: -%") {
		t.Errorf("detected bash printf syntax error in output: %s", outStr)
	}
	if !strings.Contains(outStr, "PASS") || !strings.Contains(outStr, "s to submit") {
		t.Errorf("expected submission command in pass output, got: %s", outStr)
	}

	rScript := filepath.Join(sess.Workspace, "r")
	if _, err := os.Stat(rScript); os.IsNotExist(err) {
		t.Errorf("r shortcut missing")
	}
	sScript := filepath.Join(sess.Workspace, "s")
	if _, err := os.Stat(sScript); os.IsNotExist(err) {
		t.Errorf("s shortcut missing")
	}
	pScript := filepath.Join(sess.Workspace, "p")
	if _, err := os.Stat(pScript); os.IsNotExist(err) {
		t.Errorf("p shortcut missing")
	}

	pCmd := exec.Command("./p")
	pCmd.Dir = sess.Workspace
	pOut, err := pCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executing ./p failed: %v, output: %s", err, string(pOut))
	}
	if !strings.Contains(string(pOut), "pace ") || !strings.Contains(string(pOut), "vs PB") {
		t.Errorf("expected pace against the PB in ./p output, got: %s", string(pOut))
	}

	rCmd := exec.Command("./r")
	rCmd.Dir = sess.Workspace
	rOut, err := rCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executing ./r failed: %v, output: %s", err, string(rOut))
	}
	if !strings.Contains(string(rOut), "PASS") {
		t.Errorf("expected PASS in ./r output, got: %s", string(rOut))
	}

	diffScript := filepath.Join(sess.Workspace, "diff")
	if _, err := os.Stat(diffScript); os.IsNotExist(err) {
		t.Errorf("diff script missing")
	}
	dScript := filepath.Join(sess.Workspace, "d")
	if _, err := os.Stat(dScript); os.IsNotExist(err) {
		t.Errorf("d shortcut missing")
	}

	if err := os.WriteFile(entryPath, nil, 0644); err != nil {
		t.Fatal(err)
	}
	dCmd := exec.Command("./d")
	dCmd.Dir = sess.Workspace
	dOut, err := dCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executing ./d failed: %v, output: %s", err, string(dOut))
	}
	if !strings.Contains(string(dOut), "No changes in solution.py yet") {
		t.Errorf("expected 'No changes' in unmodified ./d output, got: %s", string(dOut))
	}

	modifiedCode := "def solve():\n    return 42\n"
	if err := os.WriteFile(entryPath, []byte(modifiedCode), 0644); err != nil {
		t.Fatalf("failed to modify entry file: %v", err)
	}

	diffCmd := exec.Command("./diff")
	diffCmd.Dir = sess.Workspace
	diffOut, err := diffCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executing ./diff failed: %v, output: %s", err, string(diffOut))
	}
	if !strings.Contains(string(diffOut), "Changes in solution.py") || !strings.Contains(string(diffOut), "return 42") {
		t.Errorf("expected diff showing modifications, got: %s", string(diffOut))
	}

	resetCmd := exec.Command("./diff", "-r", "-y")
	resetCmd.Dir = sess.Workspace
	resetOut, err := resetCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executing ./diff -r failed: %v, output: %s", err, string(resetOut))
	}
	if !strings.Contains(string(resetOut), "Reset student files to the empty baseline") {
		t.Errorf("expected restoration message, got: %s", string(resetOut))
	}

	revertedContent, err := os.ReadFile(entryPath)
	if err != nil || len(revertedContent) != 0 {
		t.Errorf("entry file was not restored to empty, got: %s", string(revertedContent))
	}
}

func TestSessionCleanup(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sess, err := NewSession(drill.Drill{ID: "test-cleanup", Language: "c", EntryFile: "main.c", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}

	ws := sess.Workspace
	if _, err := os.Stat(ws); os.IsNotExist(err) {
		t.Fatalf("workspace should exist before cleanup")
	}

	pCmd := exec.Command("./p")
	pCmd.Dir = sess.Workspace
	pOut, err := pCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("executing ./p on first run failed: %v", err)
	}
	if !strings.Contains(string(pOut), "vs S pace") {
		t.Errorf("expected S pace comparison in first run output, got: %s", string(pOut))
	}

	sess.Cleanup()

	if _, err := os.Stat(ws); !os.IsNotExist(err) {
		t.Errorf("workspace should be deleted after cleanup")
	}
}

func TestRunTestsSuccessAndTimeout(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sessSuccess, err := NewSession(drill.Drill{ID: "test-success", EntryFile: "solution.sh", TestCommand: "echo 'ALL GOOD'"}, 0)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer sessSuccess.Cleanup()
	if err := os.WriteFile(filepath.Join(sessSuccess.Workspace, sessSuccess.EntryFile), []byte("echo work"), 0644); err != nil {
		t.Fatal(err)
	}

	passed, out := sessSuccess.RunTests()
	if !passed {
		t.Errorf("expected tests to pass, got output: %s", out)
	}
	if !strings.Contains(out, "ALL GOOD") {
		t.Errorf("expected output to contain 'ALL GOOD', got: %s", out)
	}

	origTimeout := testTimeout
	testTimeout = 50 * time.Millisecond
	defer func() { testTimeout = origTimeout }()

	sessTimeout, err := NewSession(drill.Drill{ID: "test-timeout", EntryFile: "loop.sh", TestCommand: "sleep 2"}, 0)
	if err != nil {
		t.Fatalf("failed to create timeout session: %v", err)
	}
	defer sessTimeout.Cleanup()
	if err := os.WriteFile(filepath.Join(sessTimeout.Workspace, sessTimeout.EntryFile), []byte("echo work"), 0644); err != nil {
		t.Fatal(err)
	}

	tPassed, tOut := sessTimeout.RunTests()
	if tPassed {
		t.Errorf("expected tests to fail on timeout, but passed")
	}
	if !strings.Contains(tOut, "Timed out after 50ms") {
		t.Errorf("expected timeout diagnostic message, got: %s", tOut)
	}
}

func TestResolveRunCommandFallbacks(t *testing.T) {
	cases := []struct {
		entryFile   string
		testCmd     string
		expectedSub string
	}{
		{"main.c", "", "gcc"},
		{"main.cpp", "", "g++"},
		{"solution.go", "", "go test"},
		{"solution.py", "", "python3"},
		{"main.rs", "", "rustc"},
		{"index.js", "", "node"},
		{"index.ts", "", "bun"},
		{"script.sh", "", "bash"},
		{"custom.xyz", "custom-test-runner", "custom-test-runner"},
		{"unknown.xyz", "", "No test command defined"},
	}

	for _, tc := range cases {
		s := &Session{
			EntryFile:   tc.entryFile,
			TestCommand: tc.testCmd,
		}
		cmd := s.resolveRunCommand()
		if !strings.Contains(cmd, tc.expectedSub) {
			t.Errorf("resolveRunCommand(%s, %q) = %q; want it to contain %q", tc.entryFile, tc.testCmd, cmd, tc.expectedSub)
		}
	}
}

func TestAllPresetsWorkspaceSetup(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatalf("failed to load preset drills: %v", err)
	}

	for _, d := range drills {
		t.Run(d.ID, func(t *testing.T) {
			sess, err := NewSession(d, 0)
			if err != nil {
				t.Fatalf("failed to create session for drill %s: %v", d.ID, err)
			}
			defer sess.Cleanup()

			if _, err := os.Stat(sess.Workspace); os.IsNotExist(err) {
				t.Errorf("workspace missing for drill %s", d.ID)
			}

			entryPath := filepath.Join(sess.Workspace, d.EntryFile)
			content, err := os.ReadFile(entryPath)
			if err != nil || len(content) != 0 {
				t.Errorf("entry file missing or content mismatch for drill %s", d.ID)
			}

			starterPath := filepath.Join(sess.Workspace, ".starter_code")
			starterContent, err := os.ReadFile(starterPath)
			if err != nil || len(starterContent) != 0 {
				t.Errorf(".starter_code missing or mismatch for drill %s", d.ID)
			}

			problemPath := filepath.Join(sess.Workspace, "PROBLEM.md")
			if _, err := os.Stat(problemPath); os.IsNotExist(err) {
				t.Errorf("PROBLEM.md missing for drill %s", d.ID)
			}

			for _, helper := range []string{"r", "p", "s", "i", "d", "m", "run", "pace", "submit", "info", "diff", "match"} {
				if _, err := os.Stat(filepath.Join(sess.Workspace, helper)); os.IsNotExist(err) {
					t.Errorf("expected helper %s to exist for drill %s", helper, d.ID)
				}
			}

			for filename, content := range d.ExtraFiles {
				assertFileContent(t, sess.Workspace, filename, content)
			}
			if passed, output := sess.RunTests(); passed {
				t.Errorf("empty built-in implementation passed %s: %s", d.ID, output)
			}
		})
	}
}

func TestGetSprintCmd(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sess, err := NewSession(drill.Drill{ID: "test-cmd", EntryFile: "main.c", StarterCode: "code", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	defer sess.Cleanup()

	cmd := sess.GetSprintCmd()
	if cmd == nil {
		t.Fatalf("expected non-nil exec.Cmd")
	}

	if _, err := exec.LookPath("zellij"); err == nil {
		argsStr := strings.Join(cmd.Args, " ")
		if !strings.Contains(argsStr, "attach -c") {
			t.Errorf("expected zellij attach -c in args, got: %s", argsStr)
		}
		if !strings.Contains(argsStr, sess.SessionName) {
			t.Errorf("expected session name %s in args, got: %s", sess.SessionName, argsStr)
		}
		if !strings.Contains(argsStr, "options --default-layout") {
			t.Errorf("expected options --default-layout in args, got: %s", argsStr)
		}
		if !strings.Contains(argsStr, "layout.kdl") {
			t.Errorf("expected layout.kdl in args, got: %s", argsStr)
		}
	} else {
		if !strings.Contains(cmd.Path, strings.Fields(DetectEditor())[0]) {
			t.Errorf("expected editor fallback %s, got: %s", DetectEditor(), cmd.Path)
		}
	}
}

func TestTypingSoundEditorFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CODEDRILL_EDITOR", "editor --flag")
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "prototype")
	sess, err := NewSession(drill.Drill{ID: "test-sound", EntryFile: "main.c", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Cleanup()
	cmd := sess.GetSprintCmd()
	if len(cmd.Args) != 5 || cmd.Args[1] != "__typing-sound-editor" || cmd.Args[2] != "editor" || cmd.Args[3] != "--flag" || cmd.Args[4] != filepath.Join(sess.Workspace, "main.c") {
		t.Fatalf("unexpected editor command: %v", cmd.Args)
	}
}

func TestTypingSoundsFromSavedConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CODEDRILL_EDITOR", "hx")
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	on := true
	enabled := storage.Config{Typing: &storage.TypingSettings{Enabled: &on, Pack: "sfx-Starcat_JP-8Bit_Puzzler"}}
	if err := storage.SaveConfig(enabled); err != nil {
		t.Fatal(err)
	}
	sess, err := NewSession(drill.Drill{ID: "test-sound-config", EntryFile: "main.c", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Cleanup()
	if cmd := sess.GetSprintCmd(); len(cmd.Args) < 2 || cmd.Args[1] != "__typing-sound-editor" {
		t.Fatalf("saved pack did not use the wrapper: %v", cmd.Args)
	}
	if flag := typingSoundsFlag(); flag != "1" {
		t.Fatalf("typing sounds flag %q, want 1", flag)
	}

	silent := storage.Config{Typing: &storage.TypingSettings{Enabled: &on, Pack: "silent"}}
	if err := storage.SaveConfig(silent); err != nil {
		t.Fatal(err)
	}
	if flag := typingSoundsFlag(); flag != "0" {
		t.Fatalf("silent pack flag %q, want 0", flag)
	}

	off := false
	quiet := storage.Config{Typing: &storage.TypingSettings{Enabled: &off, Pack: "sfx-Starcat_JP-8Bit_Puzzler"}}
	if err := storage.SaveConfig(quiet); err != nil {
		t.Fatal(err)
	}
	if flag := typingSoundsFlag(); flag != "0" {
		t.Fatalf("disabled flag %q, want 0", flag)
	}

	prototype := storage.Config{Typing: &storage.TypingSettings{Enabled: &on, Pack: typingsound.PrototypePackID}}
	if err := storage.SaveConfig(prototype); err != nil {
		t.Fatal(err)
	}
	if flag := typingSoundsFlag(); flag != "1" {
		t.Fatalf("saved prototype flag %q, want 1", flag)
	}

	unknown := storage.Config{Typing: &storage.TypingSettings{Enabled: &on, Pack: "missing"}}
	if err := storage.SaveConfig(unknown); err != nil {
		t.Fatal(err)
	}
	if flag := typingSoundsFlag(); flag != "0" {
		t.Fatalf("unknown pack flag %q, want 0", flag)
	}
}

func TestTypingSoundsOffKeepsNormalEditorLaunch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CODEDRILL_EDITOR", "hx")
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	off := false
	cfg := storage.Config{Typing: &storage.TypingSettings{Enabled: &off, Pack: "sfx-Starcat_JP-8Bit_Puzzler"}}
	if err := storage.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	sess, err := NewSession(drill.Drill{ID: "test-sound-off", EntryFile: "main.c", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Cleanup()
	cmd := sess.GetSprintCmd()
	want := []string{"hx", filepath.Join(sess.Workspace, "main.c")}
	if strings.Join(cmd.Args, " ") != strings.Join(want, " ") {
		t.Fatalf("disabled sounds changed the editor launch: %v", cmd.Args)
	}
}

// The editor pane hands the terminal to a shell after the first editor exits.
// This drives that shell, not just the initial launch: the first editor, a
// different file, and the original file all have to use the wrapper.
func TestTypingSoundEditorShimCoversPaneLifecycle(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	t.Setenv("CODEDRILL_EDITOR", "hx")

	bin := t.TempDir()
	hx := filepath.Join(bin, "hx")
	if err := os.WriteFile(hx, []byte("#!/usr/bin/env bash\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))

	log := filepath.Join(t.TempDir(), "wrapper.log")
	codedrill := filepath.Join(bin, "codedrill-stub")
	stub := "#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" >> " + shellQuote(log) + "\n"
	if err := os.WriteFile(codedrill, []byte(stub), 0755); err != nil {
		t.Fatal(err)
	}
	oldExecutable := executablePath
	executablePath = func() string { return codedrill }
	defer func() { executablePath = oldExecutable }()

	on := true
	if err := storage.SaveConfig(storage.Config{
		Editor: "hx",
		Typing: &storage.TypingSettings{Enabled: &on, Pack: "sfx-Starcat_JP-8Bit_Puzzler"},
	}); err != nil {
		t.Fatal(err)
	}
	sess, err := NewSession(drill.Drill{ID: "shim-lifecycle", EntryFile: "vect2.cpp", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Cleanup()

	cmd := exec.Command("bash", filepath.Join(sess.Workspace, "start-editor.sh"))
	cmd.Dir = sess.Workspace
	cmd.Stdin = strings.NewReader("hx vect2.hpp\nhx vect2.cpp\nexit\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pane script: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "Editor closed.") {
		t.Fatalf("pane script did not reach its shell:\n%s", out)
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	want := []string{"vect2.cpp", "vect2.hpp", "vect2.cpp"}
	if len(lines) != len(want) {
		t.Fatalf("wrapper ran %d times, want %d:\n%s", len(lines), len(want), data)
	}
	for i, file := range want {
		if !strings.Contains(lines[i], "__typing-sound-editor") || !strings.HasSuffix(lines[i], file) {
			t.Fatalf("launch %d = %q, want the wrapper for %s", i, lines[i], file)
		}
	}
}

func TestTypingSoundEditorShimInBothPaneShells(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	t.Setenv("CODEDRILL_EDITOR", "hx")
	bin := t.TempDir()
	hx := filepath.Join(bin, "hx")
	if err := os.WriteFile(hx, []byte("#!/usr/bin/env bash\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))

	on := true
	if err := storage.SaveConfig(storage.Config{
		Typing: &storage.TypingSettings{Enabled: &on, Pack: "sfx-Starcat_JP-8Bit_Puzzler"},
	}); err != nil {
		t.Fatal(err)
	}
	newSession := func(id string) *Session {
		t.Helper()
		s, err := NewSession(drill.Drill{ID: id, EntryFile: "main.c", TestCommand: "true"}, 0)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(s.Cleanup)
		return s
	}
	sess := newSession("shim-panes")
	shim := filepath.Join(sess.Workspace, editorShimDir, "hx")
	data, err := os.ReadFile(shim)
	if err != nil {
		t.Fatalf("editor shim missing: %v", err)
	}
	if !strings.Contains(string(data), shellQuote(hx)) || !strings.Contains(string(data), "__typing-sound-editor") {
		t.Fatalf("shim does not route to the wrapper: %s", data)
	}
	info, err := os.Stat(shim)
	if err != nil || info.Mode()&0111 == 0 {
		t.Fatalf("shim is not executable: %v", err)
	}

	editorScript, err := os.ReadFile(filepath.Join(sess.Workspace, "start-editor.sh"))
	if err != nil {
		t.Fatal(err)
	}
	terminalScript, err := os.ReadFile(filepath.Join(sess.Workspace, "start-terminal.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for name, script := range map[string][]byte{"start-editor.sh": editorScript, "start-terminal.sh": terminalScript} {
		if !strings.Contains(string(script), `PATH="$PWD/.codedrill-editor:$PATH"`) {
			t.Errorf("%s does not put the editor shim on PATH:\n%s", name, script)
		}
	}
	if !strings.Contains(string(editorScript), "Editor closed.") {
		t.Fatalf("editor script no longer hands over to a shell:\n%s", editorScript)
	}
	if strings.Index(string(editorScript), "Editor closed.") > strings.Index(string(editorScript), editorShimDir) {
		t.Fatalf("editor shim PATH is set before the editor exit message:\n%s", editorScript)
	}

	off := false
	if err := storage.SaveConfig(storage.Config{
		Typing: &storage.TypingSettings{Enabled: &off, Pack: "sfx-Starcat_JP-8Bit_Puzzler"},
	}); err != nil {
		t.Fatal(err)
	}
	quiet := newSession("shim-panes-off")
	if _, err := os.Stat(filepath.Join(quiet.Workspace, editorShimDir)); !os.IsNotExist(err) {
		t.Fatalf("disabled typing sounds created an editor shim: %v", err)
	}
}

// The generated-tone diagnostic mode uses the same wrapper, so it needs the
// same later-launch shim even when the saved pack is off or silent.
func TestPrototypeTypingSoundsCreateEditorShim(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "prototype")
	t.Setenv("CODEDRILL_EDITOR", "hx")
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "hx"), []byte("#!/usr/bin/env bash\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	sess, err := NewSession(drill.Drill{ID: "shim-prototype", EntryFile: "main.c", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Cleanup()
	if _, err := os.Stat(filepath.Join(sess.Workspace, editorShimDir, "hx")); err != nil {
		t.Fatalf("prototype typing sounds did not create the editor shim: %v", err)
	}
}

// The direct (non-Zellij) path has no pane shell. Reopening the editor there
// goes through GetSprintCmd again, so every launch must wrap the editor.
func TestTypingSoundsWrapEveryDirectSprintLaunch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("PATH", t.TempDir())
	t.Setenv("CODEDRILL_EDITOR", "hx")
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	on := true
	if err := storage.SaveConfig(storage.Config{
		Typing: &storage.TypingSettings{Enabled: &on, Pack: "sfx-Starcat_JP-8Bit_Puzzler"},
	}); err != nil {
		t.Fatal(err)
	}
	sess, err := NewSession(drill.Drill{ID: "direct-repeat", EntryFile: "main.c", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Cleanup()
	entry := filepath.Join(sess.Workspace, "main.c")
	for launch := 0; launch < 3; launch++ {
		cmd := sess.GetSprintCmd()
		if len(cmd.Args) != 4 || cmd.Args[1] != "__typing-sound-editor" || cmd.Args[2] != "hx" || cmd.Args[3] != entry {
			t.Fatalf("launch %d did not use the typing-sound wrapper: %v", launch, cmd.Args)
		}
	}
}

// The shell pane runs helpers with "." first in PATH. ./diff must still find
// the real diff/git instead of calling itself.
func TestDiffWithWorkspaceFirstInPath(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	sess, err := NewSession(drill.Drill{ID: "diff-path", EntryFile: "a.py", StarterCode: "x = 1\n", TestCommand: "true"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Cleanup()
	if err := os.WriteFile(filepath.Join(sess.Workspace, "a.py"), []byte("x = 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", `export PATH=".:$PATH"; d`)
	cmd.Dir = sess.Workspace
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatal("./diff did not finish; it is probably calling itself")
	}
	if err != nil || !strings.Contains(string(out), "x = 2") {
		t.Fatalf("diff output %q, err %v", out, err)
	}
}

func TestShellQuotingInHelpers(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	title := `Cost in $USD "quoted" it's`
	sess, err := NewSession(drill.Drill{ID: "quote", Title: title, EntryFile: "my solution.sh", StarterCode: "echo hi\n"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Cleanup()
	out, err := exec.Command(filepath.Join(sess.Workspace, "info")).CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != title {
		t.Fatalf("info printed %q, err %v", out, err)
	}
	if err := os.WriteFile(filepath.Join(sess.Workspace, "my solution.sh"), []byte("echo hi\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out, err = exec.Command(filepath.Join(sess.Workspace, "run")).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "hi") {
		t.Fatalf("run with a spaced entry name: %q, %v", out, err)
	}
}

func TestDrillFileClashingWithHelperIsRejected(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, err := NewSession(drill.Drill{ID: "clash", EntryFile: "a.sh", ExtraFiles: map[string]string{"run": "echo mine"}}, 0)
	if err == nil || !strings.Contains(err.Error(), "clashes") {
		t.Fatalf("expected a clash error, got %v", err)
	}
}
