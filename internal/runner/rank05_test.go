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

// This also runs in the Nix build, without the developer's config or source paths.
func TestRank05FreshInstall(t *testing.T) {
	for _, command := range []string{"gcc", "g++", "python3", "bash"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("rank 5 runtime checks need %s", command)
		}
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]struct{ file, before, after string }{
		"rank05-vect2":   {"vect2.hpp", "coords[0] = x;", "coords[0] = y;"},
		"rank05-bigint":  {"bigint.cpp", "return decimal == rhs.decimal;", "return decimal != rhs.decimal;"},
		"rank05-polyset": {"searchable_tree_bag.cpp", "return position != nullptr;", "return false;"},
		"rank05-life":    {"life.c", "? 'O' : ' '", "? '0' : ' '"},
		"rank05-bsq":     {"bsq.c", "map->rows[y][x] = map->fill", "map->rows[y][x] = map->empty"},
	}
	for id, mutation := range mutations {
		t.Run(id, func(t *testing.T) {
			d := drill.Find(drills, id)
			if d == nil || !d.IsPreset || d.Dir != "" || d.Category != "42/rank 5" {
				t.Fatalf("built-in drill unavailable: %+v", d)
			}
			source, err := solutions.Resolve(d.ID, d.Dir)
			if err != nil || source == nil || source.Local || len(source.Files) == 0 {
				t.Fatalf("built-in reference unavailable: %+v, %v", source, err)
			}
			for _, options := range []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()} {
				session, err := NewSessionWithOptions(*d, 0, "", Options{ReferenceOptions: &options})
				if err != nil {
					t.Fatal(err)
				}
				defer session.Cleanup()
				policy, err := LoadReferenceSession(session.referencePolicy)
				if err != nil || *policy.Options != options {
					t.Fatalf("mode policy changed: %+v, %v", policy, err)
				}
				required, optional, err := d.StudentPaths()
				if err != nil {
					t.Fatal(err)
				}
				for _, name := range required {
					data, err := os.ReadFile(filepath.Join(session.Workspace, name))
					if err != nil || len(data) != 0 {
						t.Fatalf("%s leaked student code: %q, %v", name, data, err)
					}
				}
				for _, name := range append(optional, "solution", "drill.json") {
					if _, err := os.Stat(filepath.Join(session.Workspace, name)); !os.IsNotExist(err) {
						t.Fatalf("unexpected workspace file %s: %v", name, err)
					}
				}
				if options != mode.Study() {
					continue
				}
				if passed, output := session.RunTests(); passed {
					t.Fatalf("empty submission passed: %s", output)
				}
				for _, file := range source.Files {
					if file.Path != "" {
						t.Fatalf("reference requires external files: %s", file.Path)
					}
					data, err := file.Read()
					if err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(session.Workspace, file.Rel), data, 0600); err != nil {
						t.Fatal(err)
					}
				}
				if passed, output := session.RunTests(); !passed {
					t.Fatalf("reference failed drill command: %s", output)
				}
				exact, err := compare.Compare(*d, session.Workspace)
				if err != nil || !exact.Exact {
					t.Fatalf("reference cannot satisfy Copy/Recall: %+v, %v", exact, err)
				}
				entry := filepath.Join(session.Workspace, d.EntryFile)
				data, err := os.ReadFile(entry)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(entry, append(data, []byte("\n#define CODEDRILL_ALTERNATIVE 1\n")...), 0600); err != nil {
					t.Fatal(err)
				}
				if passed, output := session.RunTests(); !passed {
					t.Fatalf("equivalent implementation failed tests: %s", output)
				}
				alternative, err := compare.Compare(*d, session.Workspace)
				if err != nil || alternative.Exact {
					t.Fatalf("alternative was an exact match: %+v, %v", alternative, err)
				}
				for _, gate := range []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()} {
					if !mode.Evaluate(gate, true, &exact, nil).Passed {
						t.Fatalf("exact passing solution rejected in %+v", gate)
					}
					if mode.Evaluate(gate, true, &alternative, nil).Passed == gate.RequireExact {
						t.Fatalf("alternative evaluated incorrectly in %+v", gate)
					}
					if mode.Evaluate(gate, false, &exact, nil).Passed == gate.RequireTests {
						t.Fatalf("tests gate evaluated incorrectly in %+v", gate)
					}
				}
				broken := filepath.Join(session.Workspace, mutation.file)
				data, err = os.ReadFile(broken)
				if err != nil || !strings.Contains(string(data), mutation.before) {
					t.Fatalf("behavior mutation has no target in %s: %v", mutation.file, err)
				}
				if err := os.WriteFile(broken, []byte(strings.Replace(string(data), mutation.before, mutation.after, 1)), 0600); err != nil {
					t.Fatal(err)
				}
				if passed, output := session.RunTests(); passed || strings.Contains(output, "error:") {
					t.Fatalf("behavior mutation must compile and fail at runtime: %s", output)
				}
			}
		})
	}
}

func TestVect2ReferenceVariants(t *testing.T) {
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("vect2 checks need g++")
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	d := drill.Find(drills, "rank05-vect2")
	if d == nil {
		t.Fatal("vect2 drill is missing")
	}
	for _, variant := range []solutions.Variant{solutions.HeaderOnly, solutions.SplitFiles} {
		name := "Header-only"
		if variant == solutions.SplitFiles {
			name = "Split files"
		}
		t.Run(name, func(t *testing.T) {
			options := mode.Exam()
			session, err := NewSessionWithOptions(*d, 0, "", Options{ReferenceOptions: &options, ReferenceVariant: variant})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(session.Cleanup)
			policy, err := LoadReferenceSession(session.referencePolicy)
			if err != nil || policy.ReferenceVariant != variant || *policy.Options != options {
				t.Fatalf("reference selection was not captured: %+v, %v", policy, err)
			}
			source, err := solutions.ResolveVariant(d.ID, d.Dir, variant)
			if err != nil || source == nil {
				t.Fatalf("reference unavailable: %v", err)
			}
			for _, file := range source.Files {
				data, err := file.Read()
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(session.Workspace, file.Rel), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			passed, output := session.RunTests()
			if !passed || !mode.Evaluate(options, passed, nil, nil).Passed {
				t.Fatalf("reference failed Exam tests: %s", output)
			}
			for _, selected := range []solutions.Variant{solutions.HeaderOnly, solutions.SplitFiles} {
				result, err := compare.CompareVariant(*d, session.Workspace, selected)
				if err != nil || result.Exact != (selected == variant) {
					t.Fatalf("selection %q against %q: exact %v, error %v", selected, variant, result.Exact, err)
				}
				for _, gates := range []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()} {
					want := !gates.RequireExact || selected == variant
					if mode.Evaluate(gates, passed, &result, nil).Passed != want {
						t.Fatalf("selection %q against %q used wrong mode gates: %+v", selected, variant, gates)
					}
				}
			}
			if variant == solutions.HeaderOnly {
				cmd := exec.Command("g++", "-std=c++98", "-Wall", "-Wextra", "-Werror", "-pedantic-errors", "vect2.cpp", "main.cpp", "-o", "vect2_cxx98")
				cmd.Dir = session.Workspace
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("strict C++98 compile: %v\n%s", err, out)
				}
				cmd = exec.Command(filepath.Join(session.Workspace, "vect2_cxx98"))
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("strict C++98 main: %v\n%s", err, out)
				}
				baseline, err := exec.Command(filepath.Join(session.Workspace, "vect2_test")).CombinedOutput()
				if err != nil || string(out) != string(baseline) {
					t.Fatalf("strict C++98 main changed output: %v\n%s\nexpected:\n%s", err, out, baseline)
				}
			}
			for _, file := range source.Files {
				path := filepath.Join(session.Workspace, file.Rel)
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if passed, output := session.RunTests(); passed {
					t.Fatalf("missing required %s passed: %s", file.Rel, output)
				}
				if err := os.WriteFile(path, data, 0600); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
