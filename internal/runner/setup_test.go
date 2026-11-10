package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

func TestWorkspaceSetupFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	source := t.TempDir()
	if err := os.Mkdir(filepath.Join(source, "data"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "data", "input.txt"), []byte("input"), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := NewSession(drill.Drill{ID: "qa", EntryFile: "src/solution.py", StarterCode: "pass", TestCommand: "true",
		Dir: source, ExtraFiles: map[string]string{"tests/check.py": "pass"}}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	for name, want := range map[string]string{"src/solution.py": "", "tests/check.py": "pass", "data/input.txt": "input", ".starter_code": ""} {
		data, err := os.ReadFile(filepath.Join(s.Workspace, name))
		if err != nil || string(data) != want {
			t.Fatalf("%s: content %q, error %v", name, data, err)
		}
	}
}

func TestWorkspaceSetupFailureCleansUp(t *testing.T) {
	for _, name := range []string{"source", "entry", "extra", "helper", "escape", "absolute", "symlink"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("TMPDIR", root)
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			entry, source := "solution.py", ""
			extra := map[string]string{}
			switch name {
			case "source":
				source = filepath.Join(t.TempDir(), "missing")
			case "entry":
				entry = "blocked/solution.py"
				extra["blocked"] = "file"
			case "extra":
				extra["tests"] = "file"
				extra["tests/check.py"] = "pass"
			case "helper":
				extra["run/blocked"] = "file"
			case "escape":
				extra["../outside"] = "file"
			case "absolute":
				entry = filepath.Join(root, "outside")
			case "symlink":
				source = t.TempDir()
				if err := os.Symlink(root, filepath.Join(source, "link")); err != nil {
					t.Fatal(err)
				}
			}
			s, err := NewSession(drill.Drill{ID: "qa", EntryFile: entry, StarterCode: "pass", TestCommand: "true", Dir: source, ExtraFiles: extra}, 0)
			if err == nil || s != nil {
				t.Fatal("setup failure returned a usable session")
			}
			files, err := os.ReadDir(root)
			if err != nil || len(files) != 0 {
				t.Fatalf("setup left files behind: %v, %v", files, err)
			}
		})
	}
}

func TestWorkspaceShellFollowsConfig(t *testing.T) {
	for _, tc := range []struct{ name, config, want string }{
		{"default", "", "bash"},
		{"zsh", "zsh", "zsh"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			if tc.config != "" {
				if err := storage.SaveConfig(storage.Config{Shell: tc.config}); err != nil {
					t.Fatal(err)
				}
			}
			s, err := NewSession(drill.Drill{ID: "qa", EntryFile: "solution.py", StarterCode: "pass", TestCommand: "true"}, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Cleanup()
			for _, name := range []string{"start-editor.sh", "start-terminal.sh"} {
				data, err := os.ReadFile(filepath.Join(s.Workspace, name))
				if err != nil {
					t.Fatalf("%s: %v", name, err)
				}
				script := string(data)
				if !strings.Contains(script, "SHELL_CMD='"+tc.want+"'") {
					t.Errorf("%s: shell is not %q:\n%s", name, tc.want, script)
				}
				if !strings.Contains(script, `exec "$SHELL_CMD"`) {
					t.Errorf("%s: does not hand the terminal to the shell:\n%s", name, script)
				}
				if !strings.Contains(script, "not found in PATH") {
					t.Errorf("%s: has no fallback for a missing shell:\n%s", name, script)
				}
			}
		})
	}
}

func TestWorkspaceHelperWriteFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := &Session{Workspace: filepath.Join(t.TempDir(), "missing"), drill: drill.Drill{EntryFile: "main.c"}}
	if err := s.writeHelpers(); err == nil || !strings.Contains(err.Error(), ".sprint_start") {
		t.Fatalf("helper failure not returned: %v", err)
	}
}

func TestWorkspaceExcludesBundledSolution(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	source := t.TempDir()
	for name, content := range map[string]string{
		"drill.json":           `{"title":"qa"}`,
		"main.c":               "int main(void) { return 0; }\n",
		"solution/answer.c":    "SECRET_REFERENCE\n",
		"solution/deep/spec.h": "SECRET_REFERENCE\n",
		"src/solution/keep.c":  "keep\n",
	} {
		path := filepath.Join(source, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := NewSession(drill.Drill{ID: "qa", EntryFile: "main.c", TestCommand: "true", Dir: source}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()

	if _, err := os.Stat(filepath.Join(s.Workspace, "solution")); !os.IsNotExist(err) {
		t.Fatalf("bundled solution directory reached the workspace: %v", err)
	}
	if _, err := os.Stat(filepath.Join(s.Workspace, "src", "solution", "keep.c")); err != nil {
		t.Fatalf("nested solution directory should still be copied: %v", err)
	}
	err = filepath.WalkDir(s.Workspace, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.Contains(string(data), "SECRET_REFERENCE") {
			t.Errorf("reference content leaked into %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
