package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
)

func TestStudentClassHeadersKeepProvidedHarness(t *testing.T) {
	if _, err := exec.LookPath("c++"); err != nil {
		t.Skip("C++ compiler unavailable")
	}
	for _, name := range []string{"bigint", "vect2"} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			dir := t.TempDir()
			header, implementation := name+".hpp", name+".cpp"
			harness := fmt.Sprintf("#include <iostream>\n#include %q\nint main() { %s value; std::cout << value.get() << '\\n'; }\n", header, name)
			for file, contents := range map[string]string{
				header: "header skeleton", implementation: "implementation skeleton", "main.cpp": harness,
			} {
				if err := os.WriteFile(filepath.Join(dir, file), []byte(contents), 0644); err != nil {
					t.Fatal(err)
				}
			}
			d := drill.Drill{
				ID: name, Dir: dir, Language: "cpp", EntryFile: implementation,
				StudentFiles: []string{implementation, header},
				TestCommand:  fmt.Sprintf("c++ -Wall -Wextra -Werror %s main.cpp -o check && ./check", implementation),
			}
			for _, opts := range []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()} {
				s, err := NewSessionWithOptions(d, 0, "", Options{ReferenceOptions: &opts})
				if err != nil {
					t.Fatal(err)
				}
				s.SessionName = ""
				t.Cleanup(s.Cleanup)
				assertFileContent(t, s.Workspace, implementation, "")
				assertFileContent(t, s.Workspace, header, "")
				assertFileContent(t, s.Workspace, "main.cpp", harness)
				if passed, output := s.RunTests(); passed {
					t.Fatalf("empty class passed: %s", output)
				}
				class := fmt.Sprintf("class %s { public: int get() const; };\n", name)
				body := fmt.Sprintf("#include %q\nint %s::get() const { return 42; }\n", header, name)
				if err := os.WriteFile(filepath.Join(s.Workspace, header), []byte(class), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(s.Workspace, implementation), []byte(body), 0644); err != nil {
					t.Fatal(err)
				}
				if passed, output := s.RunTests(); !passed || output != "42\n" {
					t.Fatalf("provided harness failed with student class: %v %q", passed, output)
				}
				if output, err := exec.Command(filepath.Join(s.Workspace, "d"), "-r", "-y").CombinedOutput(); err != nil {
					t.Fatalf("reset failed: %v %s", err, output)
				}
				assertFileContent(t, s.Workspace, implementation, "")
				assertFileContent(t, s.Workspace, header, "")
				assertFileContent(t, s.Workspace, "main.cpp", harness)
			}
			assertFileContent(t, dir, header, "header skeleton")
			assertFileContent(t, dir, implementation, "implementation skeleton")
		})
	}
}
