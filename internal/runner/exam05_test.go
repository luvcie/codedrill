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
func TestExam05FreshInstall(t *testing.T) {
	for _, command := range []string{"cc", "c++", "python3", "bash"} {
		if _, err := exec.LookPath(command); err != nil {
			t.Skipf("Exam 05 runtime checks need %s", command)
		}
	}
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	drills, err := drill.LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	mutations := map[string]struct{ file, before, after string }{
		"exam05-vect2":   {"vect2.cpp", "coordinates{x, y}", "coordinates{y, x}"},
		"exam05-bigint":  {"bigint.cpp", "return decimal == rhs.decimal;", "return decimal != rhs.decimal;"},
		"exam05-polyset": {"searchable_tree_bag.cpp", "return position != nullptr;", "return false;"},
		"exam05-life":    {"life.c", "? 'O' : ' '", "? '0' : ' '"},
		"exam05-bsq":     {"bsq.c", "map->rows[y][x] = map->fill", "map->rows[y][x] = map->empty"},
	}
	for id, mutation := range mutations {
		t.Run(id, func(t *testing.T) {
			d := drill.Find(drills, id)
			if d == nil || !d.IsPreset || d.Dir != "" || d.Category != "42/Exam 05" {
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
