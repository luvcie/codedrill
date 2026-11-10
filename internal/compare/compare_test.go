package compare

import (
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/solutions"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func attachReference(t *testing.T, drillID string, files map[string]string) {
	t.Helper()
	dir, err := solutions.OverlayDir(drillID)
	if err != nil {
		t.Fatal(err)
	}
	writeTree(t, dir, files)
}

func fileRels(res Result) []string {
	out := make([]string, len(res.Files))
	for i, f := range res.Files {
		out[i] = f.Rel
	}
	return out
}

func TestPercentRoundingAndExactMatch(t *testing.T) {
	for _, tc := range []struct {
		score float64
		exact bool
		want  float64
	}{
		{0, false, 0},
		{0.1234, false, 12.3},
		{0.1236, false, 12.4},
		{0.9994, false, 99.9},
		{0.9999, false, 99.9},
		{1, false, 99.9},
		{1, true, 100},
	} {
		got := (Result{Score: tc.score, Exact: tc.exact}).Percent()
		if got != tc.want {
			t.Errorf("score %v exact %v: got %v%%, want %v%%", tc.score, tc.exact, got, tc.want)
		}
	}
}

func TestSingleFileExactMatch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "ft_strlen", Language: "c", EntryFile: "ft_strlen.c"}
	attachReference(t, d.ID, map[string]string{
		"ft_strlen.c": "int ft_strlen(char *s)\n{\n\treturn 0;\n}\n",
	})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{
		"ft_strlen.c": "// solution\nint ft_strlen(char *s) { return (0); }\n",
	})

	res, err := Compare(d, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Exact || res.Score != 1 || res.Percent() != 100 {
		t.Fatalf("not an exact match: %+v", res)
	}
	if len(res.Files) != 1 || res.Files[0].Rel != "ft_strlen.c" || res.Files[0].Missing {
		t.Fatalf("files are %+v", res.Files)
	}
}

func TestMultipleFilesExactMatch(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "custom", Language: "c", EntryFile: "main.c"}
	attachReference(t, d.ID, map[string]string{
		"main.c":       "int main(void) { return 0; }\n",
		"lib/helper.h": "int helper(void);\n",
	})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{
		"main.c":       "int main(void)\n{\n\treturn 0;\n}\n",
		"lib/helper.h": "int helper(void);\n",
	})

	res, err := Compare(d, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Exact || res.Percent() != 100 {
		t.Fatalf("not an exact match: %+v", res)
	}
	if got := fileRels(res); !slices.Equal(got, []string{"main.c", "lib/helper.h"}) {
		t.Fatalf("files are %v", got)
	}
}

func TestOneMismatchingFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "pair", Language: "c", EntryFile: "a.c"}
	attachReference(t, d.ID, map[string]string{
		"a.c": "int a(void) { return 0; }\n",
		"b.c": "int b(void) { return 2; }\n",
	})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{
		"a.c": "int a(void) { return 0; }\n",
		"b.c": "int b(void) { return 3; }\n",
	})

	res, err := Compare(d, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if res.Exact || res.Percent() >= 100 {
		t.Fatalf("mismatch reported as exact: %+v", res)
	}
	if !res.Files[0].Result.Exact {
		t.Fatalf("a.c should match: %+v", res.Files[0])
	}
	if res.Files[1].Result.Exact {
		t.Fatalf("b.c should not match: %+v", res.Files[1])
	}
}

func TestMissingWorkspaceFileFailsExact(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "missing", Language: "c", EntryFile: "main.c"}
	attachReference(t, d.ID, map[string]string{
		"main.c":     "int main(void) { return 0; }\n",
		"lib/util.c": "int util(void) { return 0; }\n",
	})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{"main.c": "int main(void) { return 0; }\n"})

	res, err := Compare(d, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if res.Exact || res.Percent() >= 100 {
		t.Fatalf("missing file passed: %+v", res)
	}
	last := res.Files[len(res.Files)-1]
	if last.Rel != "lib/util.c" || !last.Missing || last.Result.Exact {
		t.Fatalf("missing file result is %+v", last)
	}
}

func TestExtraWorkspaceFilesAreIgnored(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "extra", Language: "c", EntryFile: "a.c"}
	attachReference(t, d.ID, map[string]string{"a.c": "int a(void) { return 0; }\n"})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{
		"a.c":         "int a(void) { return 0; }\n",
		"stray.c":     "this is not compared\n",
		"lib/stray.h": "nor is this\n",
	})

	res, err := Compare(d, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Exact || len(res.Files) != 1 || res.Files[0].Rel != "a.c" {
		t.Fatalf("extra workspace files affected the result: %+v", res)
	}
}

func TestScoreIsWeightedByTokens(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "weighted", Language: "c", EntryFile: "main.c"}
	attachReference(t, d.ID, map[string]string{
		"main.c": "int main(void) { return 0; }\n",
		"big.c":  strings.Repeat("int x;\n", 30),
	})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{"main.c": "int main(void) { return 0; }\n"})

	res, err := Compare(d, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if res.Matched != 10 || res.ReferenceTokens != 100 {
		t.Fatalf("aggregates are %+v", res)
	}
	if math.Abs(res.Score-0.1) > 1e-9 {
		t.Fatalf("score is %v, want 0.1 (an average would be 0.5)", res.Score)
	}
	if res.Percent() != 10 {
		t.Fatalf("percent is %v, want 10", res.Percent())
	}
}

func TestEntryFileFirstThenAlphabetical(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "order", Language: "c", EntryFile: "entry.c"}
	body := "int f(void) { return 0; }\n"
	attachReference(t, d.ID, map[string]string{
		"entry.c": body,
		"z.c":     body,
		"a.c":     body,
		"m.c":     body,
	})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{
		"entry.c": body,
		"z.c":     body,
		"a.c":     body,
		"m.c":     body,
	})

	res, err := Compare(d, workspace)
	if err != nil {
		t.Fatal(err)
	}
	if got := fileRels(res); !slices.Equal(got, []string{"entry.c", "a.c", "m.c", "z.c"}) {
		t.Fatalf("files are %v", got)
	}
}

func TestUnsupportedLanguage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, tc := range []struct{ language, want string }{
		{"python", "not supported"},
	} {
		d := drill.Drill{ID: "lang", Language: tc.language}
		if _, err := Compare(d, t.TempDir()); err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("language %q: error is %v, want one containing %q", tc.language, err, tc.want)
		}
	}
}

func TestCppReferenceFileIsUnsupported(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "mixed", Language: "c", EntryFile: "main.c"}
	attachReference(t, d.ID, map[string]string{
		"main.c":     "int main(void) { return 0; }\n",
		"helper.cpp": "int helper() { return 0; }\n",
	})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{
		"main.c":     "int main(void) { return 0; }\n",
		"helper.cpp": "int helper() { return 0; }\n",
	})
	if _, err := Compare(d, workspace); err == nil || !strings.Contains(err.Error(), "C++") {
		t.Fatalf("error is %v, want a C++ unsupported error", err)
	}
}

func TestRejectsUnsafeDrillID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "../escape", Language: "c"}
	if _, err := Compare(d, t.TempDir()); err == nil || !strings.Contains(err.Error(), "invalid drill id") {
		t.Fatalf("error is %v, want an invalid drill id error", err)
	}
}

func TestRejectsSymlinkedWorkspaceFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	outside := t.TempDir()
	writeTree(t, outside, map[string]string{"a.c": "int a(void) { return 0; }\n"})
	d := drill.Drill{ID: "linked", Language: "c", EntryFile: "a.c"}
	attachReference(t, d.ID, map[string]string{"a.c": "int a(void) { return 0; }\n"})
	workspace := t.TempDir()
	if err := os.Symlink(filepath.Join(outside, "a.c"), filepath.Join(workspace, "a.c")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := Compare(d, workspace); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error is %v, want a symlink error", err)
	}
}

func TestReferenceSymlinkLeavesNoSolution(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	outside := t.TempDir()
	writeTree(t, outside, map[string]string{"a.c": "int a(void) { return 0; }\n"})
	overlay, err := solutions.OverlayDir("linked")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(overlay, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "a.c"), filepath.Join(overlay, "a.c")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	d := drill.Drill{ID: "linked", Language: "c", EntryFile: "a.c"}
	if _, err := Compare(d, t.TempDir()); err == nil || !strings.Contains(err.Error(), "no reference solution") {
		t.Fatalf("error is %v, want a missing solution error", err)
	}
}
