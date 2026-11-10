package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/referenceview"
	"github.com/luvcie/codedrill/internal/runner"
)

func diffFixture(t *testing.T, options mode.Options) (runner.ReferenceSession, string, string) {
	t.Helper()
	session, policy, referenceDir := viewerFixture(t, options)
	for _, name := range []string{"notes.txt", "life.h"} {
		if err := os.Remove(filepath.Join(referenceDir, name)); err != nil {
			t.Fatal(err)
		}
	}
	return session, policy, referenceDir
}

func writeDiffFile(t *testing.T, root, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestSprintDiffVisibilityAndSnapshot(t *testing.T) {
	cases := []struct {
		name    string
		options mode.Options
	}{
		{"Study", mode.Study()}, {"Copy", mode.Copy()},
		{"Recall", mode.Recall()}, {"Exam", mode.Exam()},
	}
	for bits := 0; bits < 16; bits++ {
		cases = append(cases, struct {
			name    string
			options mode.Options
		}{fmt.Sprintf("checkboxes-%04b", bits), mode.Options{
			CheckReference: bits&1 != 0, ShowReference: bits&2 != 0,
			RequireExact: bits&4 != 0, RequireTests: bits&8 != 0,
		}})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			session, policy, referenceDir := diffFixture(t, tc.options)
			writeDiffFile(t, session.Workspace, "life.c", viewerSource)
			// Changing preferences after launch must not alter sprint visibility.
			preferences := mode.Copy()
			if tc.options.ShowReference {
				preferences = mode.Exam()
			} else {
				// Authorization must precede reference resolution or tokenization.
				writeDiffFile(t, referenceDir, "life.c", "SECRET_REFERENCE /* unterminated")
			}
			if err := mode.Save("life", preferences); err != nil {
				t.Fatal(err)
			}
			code, out, errOut := runCLI(t, "reference-diff", "--session", policy)
			if tc.options.ShowReference {
				if code != 0 || out != "Normalized sources match.\n" || errOut != "" {
					t.Fatalf("visible diff: %d %q %q", code, out, errOut)
				}
			} else if code == 0 || out != "" || !strings.Contains(errOut, "reference is hidden") {
				t.Fatalf("hidden diff: %d %q %q", code, out, errOut)
			}
			if strings.Contains(out+errOut, "reference_secret") || strings.Contains(out+errOut, "SECRET_REFERENCE") {
				t.Fatalf("reference leaked: %q %q", out, errOut)
			}
		})
	}
}

func TestSprintDiffComparisonAndReadOnlyWorkspace(t *testing.T) {
	for _, tc := range []struct {
		name, source, want string
		exact              bool
	}{
		{"exact", viewerSource, "Normalized sources match.", true},
		{"partial", "int reference_secret(void) {\n\nreturn 7; }", "+ 3: 7", false},
		{"comments", "/* before */ int reference_secret ( void )\n{ return /* middle */ 42 ; } // after", "Normalized sources match.", true},
		{"return parentheses", "int reference_secret(void) { return (((42))); }", "Normalized sources match.", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			session, policy, _ := diffFixture(t, mode.Study())
			writeDiffFile(t, session.Workspace, "life.c", tc.source)
			before, err := os.ReadDir(session.Workspace)
			if err != nil {
				t.Fatal(err)
			}
			result, err := sprintDiff(policy)
			if err != nil || result.Exact != tc.exact {
				t.Fatalf("comparison: %+v, %v", result, err)
			}
			if view := referenceview.NewDiff(result).View(); !strings.Contains(view, tc.want) {
				t.Fatalf("diff lacks %q: %s", tc.want, view)
			}
			after, err := os.ReadDir(session.Workspace)
			if err != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("workspace entries changed: %v, %v", after, err)
			}
			data, err := os.ReadFile(filepath.Join(session.Workspace, "life.c"))
			if err != nil || string(data) != tc.source {
				t.Fatalf("student file changed: %q, %v", data, err)
			}
		})
	}
}

func TestSprintDiffMultipleFilesAndMissingFiles(t *testing.T) {
	session, policy, referenceDir := diffFixture(t, mode.Copy())
	writeDiffFile(t, session.Workspace, "life.c", viewerSource)
	for _, name := range []string{"z.c", "a.h", "empty.c"} {
		content := "int helper(void);"
		if name == "empty.c" {
			content = ""
		}
		writeDiffFile(t, referenceDir, name, content)
	}
	writeDiffFile(t, session.Workspace, "z.c", "int changed(void);")
	result, err := sprintDiff(policy)
	if err != nil || result.Exact || len(result.Files) != 4 {
		t.Fatalf("multi-file diff: %+v, %v", result, err)
	}
	var paths []string
	for _, file := range result.Files {
		paths = append(paths, file.Rel)
	}
	if want := []string{"life.c", "a.h", "empty.c", "z.c"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("file order: %v", paths)
	}
	if !result.Files[1].Missing || !result.Files[2].Missing || result.Files[3].Missing || result.Files[3].Result.Exact {
		t.Fatalf("missing and changed files: %+v", result.Files)
	}
	if diff := result.Files[1].Result.DiffText(); !strings.Contains(diff, "- 1: int helper ( void ) ;") {
		t.Fatalf("missing file tokens: %s", diff)
	}
}

func TestSprintDiffErrors(t *testing.T) {
	for _, kind := range []string{"unsupported language", "missing reference", "student syntax", "reference syntax", "file read", "workspace missing"} {
		t.Run(kind, func(t *testing.T) {
			session, policy, referenceDir := diffFixture(t, mode.Study())
			want := ""
			switch kind {
			case "unsupported language":
				session.Language = "python"
				data, err := json.Marshal(session)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(policy, data, 0600); err != nil {
					t.Fatal(err)
				}
				want = "not supported"
			case "missing reference":
				if err := os.RemoveAll(referenceDir); err != nil {
					t.Fatal(err)
				}
				want = "no reference solution"
			case "student syntax":
				writeDiffFile(t, session.Workspace, "life.c", "/* unterminated")
				want = "student: line 1: unterminated"
			case "reference syntax":
				writeDiffFile(t, referenceDir, "life.c", "/* unterminated")
				want = "reference: line 1: unterminated"
			case "file read":
				if err := os.Remove(filepath.Join(session.Workspace, "life.c")); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(session.Workspace, "life.c"), 0755); err != nil {
					t.Fatal(err)
				}
				want = "not a regular file"
			case "workspace missing":
				if err := os.RemoveAll(session.Workspace); err != nil {
					t.Fatal(err)
				}
				want = "workspace:"
			}
			code, out, errOut := runCLI(t, "reference-diff", "--session", policy)
			if code == 0 || out != "" || !strings.Contains(errOut, want) || strings.Contains(errOut, "reference_secret") {
				t.Fatalf("comparison error: %d %q %q, want %q", code, out, errOut, want)
			}
		})
	}
}

func TestSprintDiffRejectsInvalidPoliciesAndBypasses(t *testing.T) {
	_, policy, _ := diffFixture(t, mode.Recall())
	for _, args := range [][]string{
		{}, {"--session"}, {"--session", ""}, {"--session", policy + ".missing"},
		{"--session", policy}, {"--session", policy, "--mode", "copy"},
		{"--mode", "copy", "--session", policy}, {"--session", policy, "--workspace", "/tmp"},
		{"--session", policy, "--show-reference"}, {"--session", policy, "--json"},
		{"--session", policy, "--session", "other.json"}, {"life", "--mode", "copy"},
	} {
		code, out, errOut := runCLI(t, append([]string{"reference-diff"}, args...)...)
		if code == 0 || out != "" || strings.Contains(out+errOut, "reference_secret") || strings.Contains(out+errOut, "42") {
			t.Fatalf("bypass %v: %d %q %q", args, code, out, errOut)
		}
	}
	for _, content := range []string{
		"", "{", "null", `{}`, `{"version":1,"workspace":"/tmp","drill_id":"life"}`,
		`{"version":1,"workspace":"/tmp","drill_id":"life","options":{"show_reference":"true"}}`,
		`{"version":2,"workspace":"/tmp","drill_id":"life","options":{"show_reference":true}}`,
		`{"version":1,"workspace":"relative","drill_id":"life","options":{"show_reference":true}}`,
		`{"version":1,"workspace":"/tmp","drill_id":"life","options":{}}`,
		`{"version":1,"workspace":"/tmp","drill_id":"life","options":{"show_reference":null}}`,
	} {
		if err := os.WriteFile(policy, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		code, out, errOut := runCLI(t, "reference-diff", "--session", policy)
		if code == 0 || out != "" || (!strings.Contains(errOut, "policy is missing or invalid") && !strings.Contains(errOut, "reference is hidden")) {
			t.Fatalf("invalid policy %q: %d %q %q", content, code, out, errOut)
		}
	}
}
