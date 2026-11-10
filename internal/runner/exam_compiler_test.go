package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
)

func TestExamCompilerCheck(t *testing.T) {
	for _, tc := range []struct {
		name, compiler, language, version, macros string
		pass, warn, cpp                           bool
	}{
		{"matching GCC", "gcc", "c", "16.2.1", "#define __GNUC__ 16", true, false, false},
		{"different GCC", "gcc", "c", "16.2.0", "#define __GNUC__ 16", true, true, false},
		{"C++ unknown school version", "g++", "cpp", "16.2.1", "#define __GNUC__ 16", true, false, true},
		{"different C++", "g++", "cpp", "15.1.0", "#define __GNUC__ 15", true, true, true},
		{"clang disguised as GCC", "gcc", "c", "16.2.1", "#define __GNUC__ 4\\n#define __clang__ 1", false, false, false},
		{"missing compiler", "gcc", "c", "", "", false, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bin := t.TempDir()
			if tc.version != "" {
				script := "#!/bin/sh\ncase \"$1\" in\n--version) echo '" + tc.compiler + " (GCC) " + tc.version + "';;\n-dumpfullversion) echo '" + tc.version + "';;\n-dM) printf '" + tc.macros + "\\n';;\n*) exit 1;;\nesac\n"
				if err := os.WriteFile(filepath.Join(bin, tc.compiler), []byte(script), 0755); err != nil {
					t.Fatal(err)
				}
			} else {
				// Shadow a compiler that might exist on the test host.
				if err := os.WriteFile(filepath.Join(bin, tc.compiler), []byte("#!/bin/sh\nexit 127\n"), 0755); err != nil {
					t.Fatal(err)
				}
			}
			t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			s, err := NewSession(drill.Drill{ID: "test", Category: "42/Exam 05", IsPreset: true, Language: tc.language, EntryFile: "answer.c", TestCommand: "echo TEST_RAN"}, 0)
			if err != nil {
				t.Fatal(err)
			}
			defer s.Cleanup()
			if err := os.WriteFile(filepath.Join(s.Workspace, "answer.c"), []byte("work"), 0600); err != nil {
				t.Fatal(err)
			}
			passed, output := s.RunTests()
			if passed != tc.pass || strings.Contains(output, "TEST_RAN") != tc.pass || strings.Contains(output, "differs from the school C compiler") != tc.warn || strings.Contains(output, "School g++ version and C++ standard are unconfirmed") != tc.cpp {
				t.Fatalf("passed=%v output=%q", passed, output)
			}
			if tc.pass && !strings.Contains(output, "Compiler: "+tc.compiler+" (GCC) "+tc.version) {
				t.Fatalf("missing compiler version: %s", output)
			}
			if !tc.pass && !strings.Contains(output, "must be GCC") {
				t.Fatalf("missing GCC error: %s", output)
			}
			if tc.pass {
				command := exec.Command("bash", filepath.Join(s.Workspace, "run"))
				command.Dir = s.Workspace
				output, err := command.CombinedOutput()
				if err != nil || !strings.Contains(string(output), "Compiler: "+tc.compiler+" (GCC) "+tc.version) {
					t.Fatalf("sprint helper: %s %v", output, err)
				}
			}
		})
	}
}

func TestExamCompilerRejectsClang(t *testing.T) {
	clang, err := exec.LookPath("clang")
	if err != nil {
		t.Skip("clang is not installed")
	}
	bin := t.TempDir()
	if err := os.Symlink(clang, filepath.Join(bin, "gcc")); err != nil {
		t.Skipf("cannot create compiler symlink: %v", err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s, err := NewSession(drill.Drill{ID: "test", Category: "42/Exam 05", IsPreset: true, Language: "c", EntryFile: "answer.c", TestCommand: "echo TEST_RAN"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	if err := os.WriteFile(filepath.Join(s.Workspace, "answer.c"), []byte("work"), 0600); err != nil {
		t.Fatal(err)
	}
	passed, output := s.RunTests()
	if passed || strings.Contains(output, "TEST_RAN") || !strings.Contains(output, "must be GCC") {
		t.Fatalf("Clang was not rejected: passed=%v output=%q", passed, output)
	}
}

func TestCustomExamDrillKeepsItsCompilerCommand(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s, err := NewSession(drill.Drill{ID: "custom", Category: "42/Exam 05", Language: "c", EntryFile: "answer.c", TestCommand: "echo CUSTOM"}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	if err := os.WriteFile(filepath.Join(s.Workspace, "answer.c"), []byte("work"), 0600); err != nil {
		t.Fatal(err)
	}
	passed, output := s.RunTests()
	if !passed || output != "CUSTOM\n" {
		t.Fatalf("custom drill changed: passed=%v output=%q", passed, output)
	}
}
