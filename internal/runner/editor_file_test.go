package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
)

func TestEditorStartupFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	t.Setenv("CODEDRILL_EDITOR", "record-editor --flag")
	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		d    drill.Drill
		want string
	}{
		{drill.Drill{ID: "matching-header", EntryFile: "main.cpp", StudentFiles: []string{"other.hpp", "main.hpp"}}, "main.hpp"},
		{drill.Drill{ID: "first-required-header", EntryFile: "main.cpp", StudentFiles: []string{"z.hpp", "a.hpp"}, ExtraFiles: map[string]string{"main.hpp": "supplied"}}, "z.hpp"},
		{drill.Drill{ID: "optional-matching-header", EntryFile: "main.cpp", StudentFiles: []string{"other.hpp"}, OptionalStudentFiles: []string{"main.hpp"}}, "other.hpp"},
		{drill.Drill{ID: "no-required-header", EntryFile: "main.cpp", OptionalStudentFiles: []string{"main.hpp"}, ExtraFiles: map[string]string{"supplied.hpp": "supplied"}}, "main.cpp"},
		{drill.Drill{ID: "nested-matching-header", EntryFile: "src/main.cpp", StudentFiles: []string{"include/other.hpp", "include/main.hpp"}}, "include/main.hpp"},
		{drill.Drill{ID: "other-header-extension", EntryFile: "main.c", StudentFiles: []string{"main.h"}}, "main.c"},
	}
	for _, preset := range []struct{ id, want string }{
		{"rank05-vect2", "vect2.hpp"},
		{"rank05-bigint", "bigint.hpp"},
		{"rank05-polyset", "searchable_array_bag.hpp"},
		{"rank05-life", "life.c"},
	} {
		d := drill.Find(drills, preset.id)
		if d == nil {
			t.Fatalf("missing drill %s", preset.id)
		}
		cases = append(cases, struct {
			d    drill.Drill
			want string
		}{*d, preset.want})
	}

	bin := t.TempDir()
	for _, name := range []string{"bash", "dirname"} {
		real, err := exec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "record-editor"), []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > \"$EDITOR_ARGS\"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	for _, tc := range cases {
		t.Run(tc.d.ID, func(t *testing.T) {
			s, err := NewSession(tc.d, 0)
			if err != nil {
				t.Fatal(err)
			}
			s.SessionName = ""
			t.Cleanup(s.Cleanup)
			log := filepath.Join(t.TempDir(), "editor-args")
			t.Setenv("EDITOR_ARGS", log)
			for _, launch := range []string{"direct", "zellij-script"} {
				cmd := s.GetSprintCmd()
				want := filepath.Join(s.Workspace, tc.want)
				if launch == "zellij-script" {
					cmd = exec.Command("bash", filepath.Join(s.Workspace, "start-editor.sh"))
					cmd.Stdin = strings.NewReader("exit\n")
					want = tc.want
				}
				if output, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("%s launch: %v\n%s", launch, err, output)
				}
				data, err := os.ReadFile(log)
				if err != nil || string(data) != "--flag\n"+want+"\n" {
					t.Errorf("%s opened %q, want --flag and %q (error: %v)", launch, data, want, err)
				}
			}
			t.Setenv("CODEDRILL_TYPING_SOUNDS", "prototype")
			cmd := s.GetSprintCmd()
			wantArgs := []string{"__typing-sound-editor", "record-editor", "--flag", filepath.Join(s.Workspace, tc.want)}
			if !slices.Equal(cmd.Args[1:], wantArgs) {
				t.Errorf("typing-sound launch = %q, want %q", cmd.Args[1:], wantArgs)
			}
			if s.EntryFile != tc.d.EntryFile || s.TestCommand != tc.d.TestCommand {
				t.Fatal("editor selection changed the entry file or test command")
			}
		})
	}
}
