package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/solutions"
)

func TestBuiltinCppStalePrecompiledHeader(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("stale precompiled header checks need g++")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZELLIJ_SESSION_NAME", "")
	drills, err := drill.LoadBuiltins()
	if err != nil {
		t.Fatal(err)
	}
	d := drill.Find(drills, "rank05-vect2")
	if d == nil {
		t.Fatal("vect2 drill is missing")
	}
	source, err := solutions.Resolve(d.ID, d.Dir)
	if err != nil || source == nil {
		t.Fatalf("vect2 reference unavailable: %v", err)
	}
	for _, path := range []string{"r", "submission"} {
		t.Run(path, func(t *testing.T) {
			s, err := NewSession(*d, 0)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(s.Cleanup)
			unchanged := make(map[string]string)
			for name, data := range d.ExtraFiles {
				unchanged[name] = data
			}
			for _, file := range source.Files {
				data, err := file.Read()
				if err != nil {
					t.Fatal(err)
				}
				unchanged[file.Rel] = string(data)
				if err := os.WriteFile(filepath.Join(s.Workspace, file.Rel), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			header := filepath.Join(s.Workspace, "vect2.hpp")
			current := unchanged["vect2.hpp"]
			old := strings.Replace(current, "coords[1] - rhs[1]", "coords[0] - rhs[0]", 1)
			if old == current {
				t.Fatal("stale subtraction mutation has no target")
			}
			if err := os.WriteFile(header, []byte(old), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command("g++", "-x", "c++-header", "vect2.hpp", "-o", "vect2.hpp.gch")
			cmd.Dir = s.Workspace
			if output, err := cmd.CombinedOutput(); err != nil {
				t.Fatalf("precompile older header: %v\n%s", err, output)
			}
			if err := os.WriteFile(header, []byte(current), 0600); err != nil {
				t.Fatal(err)
			}
			cmd = exec.Command("g++", "-H", "-x", "c++", "-", "-o", "stale_probe")
			cmd.Dir = s.Workspace
			cmd.Stdin = strings.NewReader("#include \"vect2.hpp\"\nint main() { std::cout << vect2(-3, -2) - vect2(-2, 3); }\n")
			if output, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(output), "! vect2.hpp.gch") {
				t.Fatalf("GCC did not load the stale header: %v\n%s", err, output)
			}
			if output, err := exec.Command(filepath.Join(s.Workspace, "stale_probe")).CombinedOutput(); err != nil || string(output) != "{-1, -1}" {
				t.Fatalf("stale subtraction: %v\n%s", err, output)
			}
			cmd = exec.Command("bash", "-c", "ulimit -c 0\n"+d.TestCommand)
			cmd.Dir = s.Workspace
			if output, err := cmd.CombinedOutput(); err == nil || !strings.Contains(string(output), "test_vect2.cpp:24:") {
				t.Fatalf("stale header did not reproduce the assertion failure: %v\n%s", err, output)
			}
			t.Log("GCC loaded the old PCH: subtraction returned {-1, -1}; test_vect2.cpp:24 failed")

			passed, output := runCppHeaderCheck(t, s, path)
			if !passed || !strings.Contains(output, "stale precompiled header") {
				t.Fatalf("%s did not recover from the stale header: passed=%v\n%s", path, passed, output)
			}
			if _, err := os.Lstat(header + ".gch"); !os.IsNotExist(err) {
				t.Fatalf("active precompiled header remains: %v", err)
			}
			if _, err := os.Stat(header + ".gch.disabled"); err != nil {
				t.Fatalf("precompiled header was not preserved: %v", err)
			}
			for name, want := range unchanged {
				if data, err := os.ReadFile(filepath.Join(s.Workspace, name)); err != nil || string(data) != want {
					t.Fatalf("source or test %s changed: %v", name, err)
				}
			}
			t.Logf("%s passed with the current header and unchanged student files and assertions", path)
		})
	}
}

func TestCppPrecompiledHeaderPolicy(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZELLIJ_SESSION_NAME", "")
	for _, tc := range []struct {
		name, language, artifact string
		preset, directory        bool
	}{
		{"built-in file", "cpp", "answer.hpp.gch", true, false},
		{"built-in nested file", "cpp", "include dir/it's\na header.h.gch", true, false},
		{"built-in directory", "cpp", "answer.hpp.gch", true, true},
		{"custom C++ override", "cpp", "answer.hpp.gch", false, false},
		{"built-in C", "c", "answer.h.gch", true, false},
	} {
		for _, path := range []string{"r", "submission"} {
			t.Run(tc.name+"/"+path, func(t *testing.T) {
				s, err := NewSession(drill.Drill{ID: "rank05-vect2", Language: tc.language, IsPreset: tc.preset,
					EntryFile: "answer.cpp", TestCommand: "echo TEST_RAN"}, 0)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(s.Cleanup)
				if err := os.WriteFile(filepath.Join(s.Workspace, s.EntryFile), []byte("student work"), 0600); err != nil {
					t.Fatal(err)
				}
				artifact := tc.artifact
				if tc.directory {
					artifact = filepath.Join(artifact, "variant.gch")
				}
				if err := writeWorkspaceFile(s.Workspace, artifact, []byte("saved PCH"), 0600); err != nil {
					t.Fatal(err)
				}
				backup := tc.artifact + ".disabled"
				if err := writeWorkspaceFile(s.Workspace, backup, []byte("earlier backup"), 0600); err != nil {
					t.Fatal(err)
				}
				outside := t.TempDir()
				if err := writeWorkspaceFile(outside, "external.hpp.gch", []byte("external PCH"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(outside, filepath.Join(s.Workspace, "external")); err != nil {
					t.Fatal(err)
				}
				passed, output := runCppHeaderCheck(t, s, path)
				protected := tc.preset && tc.language == "cpp"
				if !passed || !strings.Contains(output, "TEST_RAN") || strings.Contains(output, "stale precompiled header") != protected {
					t.Fatalf("passed=%v\n%s", passed, output)
				}
				preserved := artifact
				if protected {
					if _, err := os.Lstat(filepath.Join(s.Workspace, tc.artifact)); !os.IsNotExist(err) {
						t.Fatalf("active precompiled header remains: %v", err)
					}
					preserved = backup + ".disabled"
					if tc.directory {
						preserved = filepath.Join(preserved, "variant.gch")
					}
				}
				if data, err := os.ReadFile(filepath.Join(s.Workspace, preserved)); err != nil || string(data) != "saved PCH" {
					t.Fatalf("precompiled header was not preserved: %q, %v", data, err)
				}
				if data, err := os.ReadFile(filepath.Join(s.Workspace, backup)); err != nil || string(data) != "earlier backup" {
					t.Fatalf("earlier backup changed: %q, %v", data, err)
				}
				if data, err := os.ReadFile(filepath.Join(outside, "external.hpp.gch")); err != nil || string(data) != "external PCH" {
					t.Fatalf("artifact outside the workspace changed: %q, %v", data, err)
				}
				if passed, output := runCppHeaderCheck(t, s, path); !passed || strings.Contains(output, "stale precompiled header") {
					t.Fatalf("repeat run touched disabled headers: passed=%v\n%s", passed, output)
				}
			})
		}
	}
}

func TestBuiltinCppPrecompiledHeaderFailureStopsTests(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("ZELLIJ_SESSION_NAME", "")
	for _, failingTool := range []string{"find", "mv"} {
		for _, path := range []string{"r", "submission"} {
			t.Run(failingTool+"/"+path, func(t *testing.T) {
				s, err := NewSession(drill.Drill{ID: "test", Language: "cpp", IsPreset: true,
					EntryFile: "answer.cpp", TestCommand: "echo TEST_RAN"}, 0)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(s.Cleanup)
				for _, name := range []string{"answer.cpp", "answer.hpp.gch"} {
					if err := writeWorkspaceFile(s.Workspace, name, []byte("keep"), 0600); err != nil {
						t.Fatal(err)
					}
				}
				bin := t.TempDir()
				if err := os.WriteFile(filepath.Join(bin, failingTool), []byte(bashShebang()+"echo 'simulated failure' >&2\nexit 1\n"), 0755); err != nil {
					t.Fatal(err)
				}
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
				passed, output := runCppHeaderCheck(t, s, path)
				if passed || strings.Contains(output, "TEST_RAN") || !strings.Contains(output, "Cannot run this drill:") || !strings.Contains(output, "Compilation stopped.") {
					t.Fatalf("failed safeguard allowed tests or hid its error: passed=%v\n%s", passed, output)
				}
				if data, err := os.ReadFile(filepath.Join(s.Workspace, "answer.hpp.gch")); err != nil || string(data) != "keep" {
					t.Fatalf("failed safeguard changed the artifact: %q, %v", data, err)
				}
			})
		}
	}
}

func runCppHeaderCheck(t *testing.T, s *Session, path string) (bool, string) {
	t.Helper()
	cmd := exec.Command("bash", "./r")
	cmd.Dir = s.Workspace
	if path == "r" {
		output, err := cmd.CombinedOutput()
		return err == nil, string(output)
	}
	cmd = exec.Command("bash", "./s")
	cmd.Dir = s.Workspace
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("submit helper: %v\n%s", err, output)
	}
	return s.RunTests()
}
