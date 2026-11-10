package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/referenceview"
	"github.com/luvcie/codedrill/internal/runner"
)

const cppReference = "int vect2::get() const {\nreturn x;\n}"

func cppFixture(t *testing.T, options mode.Options) (runner.ReferenceSession, string) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "vect2", Title: "vect2", Language: "cpp", EntryFile: "vect2.cpp", StudentFiles: []string{"vect2.hpp"}}
	if err := drill.SaveDrill(d); err != nil {
		t.Fatal(err)
	}
	attachMatchReference(t, d.ID, map[string]string{
		"vect2.cpp": cppReference, "vect2.hpp": "struct vect2 { int x; int get() const; };",
		"main.cpp": "provided test harness",
	})
	workspace := matchWorkspace(t, map[string]string{
		"vect2.cpp": cppReference, "vect2.hpp": "struct vect2 { int x; int get() const; };",
		"main.cpp": "provided harness outside comparison", "extra.cpp": "@ unrelated",
	})
	session := runner.ReferenceSession{Version: 1, Workspace: workspace, DrillID: d.ID, EntryFile: d.EntryFile, Language: d.Language, StudentFiles: d.StudentFiles, Options: &options}
	policy := filepath.Join(t.TempDir(), "session.json")
	data, err := json.Marshal(session)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policy, data, 0600); err != nil {
		t.Fatal(err)
	}
	return session, policy
}

func TestMatchCPPAndJSON(t *testing.T) {
	session, _ := cppFixture(t, mode.Copy())
	for _, partial := range []bool{false, true} {
		if partial {
			writeDiffFile(t, session.Workspace, "vect2.cpp", strings.Replace(cppReference, "return x", "return y", 1))
		}
		code, out, errOut := runCLI(t, "match", session.DrillID, "--workspace", session.Workspace, "--json")
		if code != 0 {
			t.Fatalf("match: %d, %s", code, errOut)
		}
		var report matchReport
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		if report.Exact == partial || len(report.Files) != 2 || report.Files[0].Path != "vect2.cpp" || report.Files[1].Path != "vect2.hpp" {
			t.Fatalf("report: %+v", report)
		}
		if partial && (report.Percent >= 100 || report.FirstDifference == nil || report.FirstDifference.Line != 2) {
			t.Fatalf("partial report: %+v", report)
		}
		if strings.Contains(out, "return") || strings.Contains(out, "struct vect2") {
			t.Fatal("scoring exposed reference contents")
		}
	}
}

func TestCPPDiffAndViewerPolicies(t *testing.T) {
	cases := []mode.Options{mode.Study(), mode.Copy(), mode.Recall(), mode.Exam()}
	for bits := 0; bits < 16; bits++ {
		cases = append(cases, mode.Options{CheckReference: bits&1 != 0, ShowReference: bits&2 != 0, RequireExact: bits&4 != 0, RequireTests: bits&8 != 0})
	}
	for _, options := range cases {
		session, policy := cppFixture(t, options)
		writeDiffFile(t, session.Workspace, "vect2.cpp", strings.Replace(cppReference, "return x", "return y", 1))
		result, err := sprintDiff(policy)
		if !options.ShowReference {
			if err != errReferenceHidden {
				t.Fatalf("hidden diff: %+v, %v", result, err)
			}
			for _, action := range [][]string{{"reference-diff", "--session", policy}, {"reference-view", "--session", policy, "--open"}, {"reference-view", "--session", policy, "--toggle-pane"}} {
				code, out, errOut := runCLI(t, action...)
				if code == 0 || out != "" || !strings.Contains(errOut, "reference is hidden") {
					t.Fatalf("hidden action %v: %d %q %q", action, code, out, errOut)
				}
			}
			continue
		}
		if err != nil || result.Exact || len(result.Files) != 2 {
			t.Fatalf("visible comparison: %+v, %v", result, err)
		}
		view := ansi.Strip(referenceview.NewDiff(result).View())
		if !strings.Contains(view, "- 2: x") || !strings.Contains(view, "+ 2: y") {
			t.Fatalf("C++ diff: %s", view)
		}
		files, err := loadViewerFiles(session)
		if err != nil || string(files["vect2.cpp"]) != cppReference || len(files["vect2.hpp"]) == 0 {
			t.Fatalf("C++ viewer files: %v, %v", files, err)
		}
		view = ansi.Strip(referenceview.New(files, "91%").View())
		if !strings.Contains(view, "main.cpp") {
			t.Fatalf("existing viewer did not retain source file navigation: %s", view)
		}
		writeDiffFile(t, session.Workspace, "vect2.cpp", cppReference)
		code, out, errOut := runCLI(t, "reference-diff", "--session", policy)
		if code != 0 || out != "Normalized sources match.\n" || errOut != "" {
			t.Fatalf("exact C++ diff command: %d %q %q", code, out, errOut)
		}
	}
}

// This builds and invokes the real CLI through generated sprint helpers.
// No comparator or CLI stub participates in this test.
func TestCPPRealSprintHelpers(t *testing.T) {
	snapshot, _ := cppFixture(t, mode.Copy())
	binary := filepath.Join(t.TempDir(), "codedrill")
	build := exec.Command("go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	d := snapshot.Drill()
	options := mode.Copy()
	s, err := runner.NewSessionWithOptions(d, 0, "", runner.Options{ReferenceOptions: &options})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Cleanup)
	for _, name := range []string{"vect2.cpp", "vect2.hpp"} {
		data, err := os.ReadFile(filepath.Join(snapshot.Workspace, name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(s.Workspace, name), data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	path := filepath.Join(s.Workspace, "match")
	helper, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(helper), "\n")
	for i, line := range lines {
		if strings.HasPrefix(line, "CODEDRILL=") {
			lines[i] = "CODEDRILL='" + binary + "'"
		}
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0755); err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{"", "-d"} {
		cmd := exec.Command("./m")
		if flag != "" {
			cmd = exec.Command("./m", flag)
		}
		cmd.Dir = s.Workspace
		out, err := cmd.CombinedOutput()
		want := "match 100.0%"
		if flag == "-d" {
			want = "Normalized sources match."
		}
		if err != nil || !strings.Contains(string(out), want) {
			t.Fatalf("m %s: %v, %s", flag, err, out)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1800*time.Millisecond)
	defer cancel()
	watch := exec.CommandContext(ctx, "./m", "-w")
	watch.Dir = s.Workspace
	watch.Cancel = func() error { return watch.Process.Signal(os.Interrupt) }
	watch.WaitDelay = time.Second
	out, _ := watch.CombinedOutput()
	if strings.Count(string(out), "match 100.0%") < 2 || !strings.Contains(string(out), "\033[?25h") {
		t.Fatalf("real watch did not refresh and restore cursor: %q", out)
	}
}
