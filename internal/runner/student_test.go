package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/solutions"
)

func studentFixture(t *testing.T) drill.Drill {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	for name, content := range map[string]string{
		"main.cpp": "ENTRY_SKELETON", "src/impl.cpp": "IMPLEMENTATION_SKELETON",
		"optional.hpp": "OPTIONAL_SKELETON", "given.cpp": "PROVIDED_IMPLEMENTATION",
		"ft_list.h": "MANDATORY_HEADER", "data/input.txt": "FIXTURE",
		"solution/main.cpp": "REFERENCE_SECRET",
	} {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return drill.Drill{ID: "owned", Language: "cpp", EntryFile: "main.cpp", Dir: dir,
		StarterCode: "LEGACY_SKELETON", StudentFiles: []string{"src/impl.cpp", "api.hpp"},
		OptionalStudentFiles: []string{"optional.hpp"},
		ExtraFiles:           map[string]string{"api.hpp": "INLINE_SKELETON", "main.cpp": "INLINE_ENTRY", "test.py": "PROVIDED_TEST"}}
}

func TestStudentOwnershipAcrossModes(t *testing.T) {
	d := studentFixture(t)
	for _, opts := range []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()} {
		s, err := NewSessionWithOptions(d, 0, "", Options{ReferenceOptions: &opts})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(s.Cleanup)
		for _, name := range []string{"main.cpp", "src/impl.cpp", "api.hpp", ".starter_code"} {
			assertFileContent(t, s.Workspace, name, "")
		}
		if _, err := os.Stat(filepath.Join(s.Workspace, "optional.hpp")); !os.IsNotExist(err) {
			t.Fatal("optional student file was created")
		}
		for name, want := range map[string]string{"given.cpp": "PROVIDED_IMPLEMENTATION", "ft_list.h": "MANDATORY_HEADER", "data/input.txt": "FIXTURE", "test.py": "PROVIDED_TEST"} {
			assertFileContent(t, s.Workspace, name, want)
		}
		err = filepath.WalkDir(s.Workspace, func(path string, entry os.DirEntry, err error) error {
			if err != nil || entry.IsDir() {
				return err
			}
			data, err := os.ReadFile(path)
			if strings.Contains(string(data), "SKELETON") || strings.Contains(string(data), "REFERENCE_SECRET") {
				t.Errorf("student/reference source leaked into %s", path)
			}
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	assertFileContent(t, d.Dir, "main.cpp", "ENTRY_SKELETON")
	assertFileContent(t, d.Dir, "src/impl.cpp", "IMPLEMENTATION_SKELETON")
	assertFileContent(t, d.Dir, "optional.hpp", "OPTIONAL_SKELETON")
}

func assertFileContent(t *testing.T, root, name, want string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, name))
	if err != nil || string(data) != want {
		t.Fatalf("%s: got %q, want %q, error %v", name, data, want, err)
	}
}

func TestCopiedEntryDoesNotSurviveEmptyStarterCode(t *testing.T) {
	d := studentFixture(t)
	d.StarterCode = ""
	s, err := NewSession(d, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	assertFileContent(t, s.Workspace, d.EntryFile, "")
	assertFileContent(t, d.Dir, d.EntryFile, "ENTRY_SKELETON")
}

func TestResetStudentFilesAndPreserveResources(t *testing.T) {
	d := studentFixture(t)
	d.StudentFiles = append(d.StudentFiles, "name ' $(touch injected);.cpp")
	s, err := NewSession(d, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	required, optional, _ := d.StudentPaths()
	for _, name := range append(required, optional...) {
		if err := os.WriteFile(filepath.Join(s.Workspace, name), []byte("student work"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	reset := func(input string, args ...string) error {
		t.Helper()
		cmd := exec.Command(filepath.Join(s.Workspace, "d"), args...)
		cmd.Stdin = strings.NewReader(input)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("reset: %s", out)
		}
		return err
	}
	if err := reset("n\n", "-r"); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, s.Workspace, d.EntryFile, "student work")
	if err := reset("", "-r", "-y"); err != nil {
		t.Fatal(err)
	}
	for _, name := range required {
		assertFileContent(t, s.Workspace, name, "")
	}
	for _, name := range optional {
		if _, err := os.Stat(filepath.Join(s.Workspace, name)); !os.IsNotExist(err) {
			t.Fatalf("optional file survived reset: %s", name)
		}
	}
	assertFileContent(t, s.Workspace, "given.cpp", "PROVIDED_IMPLEMENTATION")
	assertFileContent(t, s.Workspace, "ft_list.h", "MANDATORY_HEADER")
	assertFileContent(t, s.Workspace, "data/input.txt", "FIXTURE")
	if _, err := os.Stat(filepath.Join(s.Workspace, "injected")); !os.IsNotExist(err) {
		t.Fatal("reset evaluated a filename as shell code")
	}
	entry := filepath.Join(s.Workspace, d.EntryFile)
	if err := os.Remove(entry); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("given.cpp", entry); err != nil {
		t.Fatal(err)
	}
	if err := reset("", "-r", "-y"); err == nil {
		t.Fatal("reset followed a student symlink into a provided resource")
	}
	assertFileContent(t, s.Workspace, "given.cpp", "PROVIDED_IMPLEMENTATION")
	if err := os.Remove(entry); err != nil {
		t.Fatal(err)
	}
	if err := os.Link(filepath.Join(s.Workspace, "given.cpp"), entry); err != nil {
		t.Fatal(err)
	}
	if err := reset("", "-r", "-y"); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, s.Workspace, d.EntryFile, "")
	assertFileContent(t, s.Workspace, "given.cpp", "PROVIDED_IMPLEMENTATION")
}

func TestLifeStartsEmptyWithReferenceCompatibilityHeader(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := exec.LookPath("cc"); err != nil {
		t.Skip("C compiler is unavailable")
	}
	const header = "#include <stdio.h>\n"
	d := drill.Drill{ID: "life", Language: "c", EntryFile: "life.c", StarterCode: "#include \"life.h\"\nint main(void){return 1;}",
		ExtraFiles: map[string]string{"life.h": header}, TestCommand: "cc -Wall -Wextra -Werror life.c -o life && ./life"}
	s, err := NewSession(d, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	assertFileContent(t, s.Workspace, "life.c", "")
	assertFileContent(t, s.Workspace, "life.h", header)
	if passed, output := s.RunTests(); passed {
		t.Fatalf("empty standalone Life program passed: %s", output)
	}
	if err := os.WriteFile(filepath.Join(s.Workspace, "life.c"), []byte("#include \"life.h\"\nint main(void){puts(\"alive\");return 0;}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if passed, output := s.RunTests(); !passed || !strings.Contains(output, "alive") {
		t.Fatalf("compatibility header no longer supports reference-style source: %s", output)
	}
}

func TestEmptyFallbackNeverPasses(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, ext := range []string{"c", "cpp", "py", "sh", "go", "js", "rs", "txt"} {
		t.Run(ext, func(t *testing.T) {
			s, err := NewSession(drill.Drill{ID: "empty", EntryFile: "empty file." + ext}, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Cleanup()
			for _, content := range []string{"", " \n\t"} {
				if err := os.WriteFile(filepath.Join(s.Workspace, s.EntryFile), []byte(content), 0644); err != nil {
					t.Fatal(err)
				}
				if passed, out := s.RunTests(); passed || !strings.Contains(out, "Write your implementation") {
					t.Fatalf("empty fallback evaluation: %v %q", passed, out)
				}
				out, err := exec.Command(filepath.Join(s.Workspace, "r")).CombinedOutput()
				if err == nil || !strings.Contains(string(out), "Write your implementation") || !strings.Contains(string(out), "FAIL") || strings.Contains(string(out), "PASS") {
					t.Fatalf("empty fallback helper: %v %q", err, out)
				}
			}
		})
	}
}

func TestCopyStartsEmptyAndStillRequiresReferenceMatch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "copy-empty", EntryFile: "main.c", Language: "c", StarterCode: "int answer;", TestCommand: "true"}
	ref, err := solutions.OverlayDir(d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ref, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ref, "main.c"), []byte("int answer;"), 0644); err != nil {
		t.Fatal(err)
	}
	opts := mode.Copy()
	s, err := NewSessionWithOptions(d, 0, "", Options{ReferenceOptions: &opts})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	result, err := compare.Compare(d, s.Workspace)
	if err != nil || result.Exact || mode.Evaluate(opts, true, &result, err).Passed {
		t.Fatalf("empty student code passed Copy: %+v %v", result, err)
	}
	if err := os.WriteFile(filepath.Join(s.Workspace, "main.c"), []byte("int answer;"), 0644); err != nil {
		t.Fatal(err)
	}
	result, err = compare.Compare(d, s.Workspace)
	if err != nil || !mode.Evaluate(opts, false, &result, err).Passed {
		t.Fatalf("Copy pass gates changed: %+v %v", result, err)
	}
}

func TestEmptyStudentFilesFailEvenWhenExplicitTestsWouldPass(t *testing.T) {
	d := studentFixture(t)
	d.TestCommand = "printf 'TEST_RAN'"
	s, err := NewSession(d, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	if passed, output := s.RunTests(); passed || !strings.Contains(output, "Write your implementation") || strings.Contains(output, "TEST_RAN") {
		t.Fatalf("provided code counted as student work: %v %q", passed, output)
	}
	// Header-only and multi-file implementations need not put code in the entry.
	if err := os.WriteFile(filepath.Join(s.Workspace, "api.hpp"), []byte("int implemented;"), 0644); err != nil {
		t.Fatal(err)
	}
	if passed, output := s.RunTests(); !passed || output != "TEST_RAN" {
		t.Fatalf("non-entry implementation was ignored: %v %q", passed, output)
	}
}

func TestStudentFilesCannotReplaceInfrastructure(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, name := range []string{"run", ".starter_code", "solution/secret.c", "PROBLEM.md", "../outside", "src/../../outside"} {
		_, err := NewSession(drill.Drill{ID: "unsafe", EntryFile: "main.c", OptionalStudentFiles: []string{name}}, 0)
		if err == nil {
			t.Fatalf("unsafe student path accepted: %q", name)
		}
	}
}
