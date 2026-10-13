package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
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
	for name, want := range map[string]string{"src/solution.py": "pass", "tests/check.py": "pass", "data/input.txt": "input", ".starter_code": "pass"} {
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

func TestWorkspaceHelperWriteFailure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s := &Session{Workspace: filepath.Join(t.TempDir(), "missing")}
	if err := s.writeHelpers(nil); err == nil || !strings.Contains(err.Error(), ".sprint_start") {
		t.Fatalf("helper failure not returned: %v", err)
	}
}
