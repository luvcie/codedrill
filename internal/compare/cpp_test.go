package compare

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
)

func TestCPPMultiFileOwnershipAndWeighting(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{
		ID: "vect2", Language: "cpp", EntryFile: "vect2.cpp",
		StudentFiles: []string{"vect2.cpp", "vect2.hpp"}, OptionalStudentFiles: []string{"extra.hpp"},
	}
	files := map[string]string{
		"vect2.cpp": "#include \"vect2.hpp\"\nint vect2::get() const { return x; }",
		"vect2.hpp": "struct vect2 { int x; int get() const; };",
		"extra.hpp": "", "main.cpp": "PROVIDED /* invalid source",
	}
	attachReference(t, d.ID, files)
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{
		"vect2.cpp": "# include \"vect2.hpp\"\nint vect2 :: get ( ) const { return /* comment */ x ; }",
		"vect2.hpp": files["vect2.hpp"], "extra.hpp": "",
		"main.cpp": "different provided harness", "stray.cpp": "not source @",
	})
	result, err := Compare(d, workspace)
	if err != nil || !result.Exact || result.Percent() != 100 {
		t.Fatalf("comparison: %+v, %v", result, err)
	}
	if !slices.Equal(fileRels(result), []string{"vect2.cpp", "extra.hpp", "vect2.hpp"}) {
		t.Fatalf("file order/ownership: %v", fileRels(result))
	}
	writeTree(t, workspace, map[string]string{"vect2.hpp": "struct vect2 { int y; int get() const; };"})
	partial, err := Compare(d, workspace)
	if err != nil || partial.Exact || partial.Matched != result.Matched-1 {
		t.Fatalf("partial comparison: %+v, %v", partial, err)
	}
	wantScore := float64(partial.Matched) / float64(partial.ReferenceTokens)
	if partial.Score != wantScore {
		t.Fatalf("score %v, want %v", partial.Score, wantScore)
	}
	// Even an empty optional reference file must exist to claim an exact copy.
	if err := os.Remove(filepath.Join(workspace, "extra.hpp")); err != nil {
		t.Fatal(err)
	}
	missing, err := Compare(d, workspace)
	if err != nil || missing.Exact || !missing.Files[1].Missing {
		t.Fatalf("optional counterpart: %+v, %v", missing, err)
	}
	if err := os.Remove(filepath.Join(workspace, "vect2.hpp")); err != nil {
		t.Fatal(err)
	}
	missing, err = Compare(d, workspace)
	if err != nil || missing.Exact || !missing.Files[2].Missing || missing.Percent() >= 100 {
		t.Fatalf("required counterpart: %+v, %v", missing, err)
	}
}

func TestCPPLegacyResourcesAndOverlaySelection(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "legacy", Language: "c++", EntryFile: "entry.cpp", Dir: t.TempDir(), ExtraFiles: map[string]string{"interface.hpp": "provided"}}
	files := map[string]string{
		"entry.cpp": "int f() { return 1; }", "impl.hpp": "int f();",
		"main.cpp": "provided @", "interface.hpp": "provided @",
	}
	writeTree(t, d.Dir, map[string]string{"main.cpp": "original provided"})
	bundled := make(map[string]string)
	for path, content := range files {
		bundled["solution/"+path] = content
	}
	writeTree(t, d.Dir, bundled)
	workspace := t.TempDir()
	writeTree(t, workspace, files)
	res, err := Compare(d, workspace)
	if err != nil || !res.Exact || !slices.Equal(fileRels(res), []string{"entry.cpp", "impl.hpp"}) {
		t.Fatalf("bundled files: %+v, %v", res, err)
	}
	attachReference(t, d.ID, map[string]string{"entry.cpp": "int f() { return 2; }"})
	res, err = Compare(d, workspace)
	if err != nil || res.Exact || len(res.Files) != 1 {
		t.Fatalf("overlay must win without merging alternatives: %+v, %v", res, err)
	}
	writeTree(t, workspace, map[string]string{"entry.cpp": "int f() { return 2; }"})
	res, err = Compare(d, workspace)
	if err != nil || !res.Exact {
		t.Fatalf("selected alternative: %+v, %v", res, err)
	}
}

func TestCPPIncompleteReferenceFailsClosed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "incomplete", Language: "cpp", EntryFile: "solution.cpp", StudentFiles: []string{"solution.hpp"}}
	attachReference(t, d.ID, map[string]string{"solution.cpp": "int f();", "main.cpp": "provided"})
	workspace := t.TempDir()
	writeTree(t, workspace, map[string]string{"solution.cpp": "int f();"})
	res, err := Compare(d, workspace)
	if err == nil || res.Exact || !strings.Contains(err.Error(), "missing required student file solution.hpp") {
		t.Fatalf("incomplete reference passed: %+v, %v", res, err)
	}
}

func TestReferenceCannotComeFromWorkspace(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "unsafe", Language: "cpp", EntryFile: "solution.cpp", Dir: t.TempDir()}
	writeTree(t, d.Dir, map[string]string{"solution/solution.cpp": "int f();"})
	workspace := filepath.Join(d.Dir, "solution")
	res, err := Compare(d, workspace)
	if err == nil || res.Exact || !strings.Contains(err.Error(), "outside the workspace") {
		t.Fatalf("workspace reference passed: %+v, %v", res, err)
	}
}
