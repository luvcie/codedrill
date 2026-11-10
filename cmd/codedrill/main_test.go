package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/solutions"
	"github.com/luvcie/codedrill/internal/storage"
)

func runCLI(t *testing.T, args ...string) (code int, stdout, stderr string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code = run(args, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestCommands(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	hist, err := storage.LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := hist.RecordAttempt("ft_strlen", 3200*time.Millisecond, true, 1, false); err != nil {
		t.Fatal(err)
	}
	if _, _, err := hist.RecordAttempt("ft_strlen", 5500*time.Millisecond, false, 2, false); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"--help"}, []string{"Usage:"}},
		{[]string{"version"}, []string{"v" + Version}},
		{[]string{"list"}, []string{"ft_strlen", "two_sum", "3.200s (S)"}},
		{[]string{"list", "42"}, []string{"ft_strlen", "[42"}},
		{[]string{"list", "python"}, []string{"binary_search"}},
		{[]string{"search", "palindrome"}, []string{"palindrome"}},
		{[]string{"info", "FT_STRLEN"}, []string{"Drill:", "ft_strlen", "S ≤ 4s"}},
		{[]string{"top", "ft_strlen"}, []string{"Hall of fame", "#1", "3.200s"}},
		{[]string{"recent", "ft_strlen"}, []string{"FAIL", "PASS"}},
		{[]string{"daily", "info"}, []string{"Drill:"}},
		{[]string{"stats"}, []string{"Total Sprints:     2", "ft_strlen"}},
		{[]string{"config"}, []string{"Configuration:", "Shell:"}},
		{[]string{"config", "countdown", "off"}, []string{"Countdown off"}},
		{[]string{"config", "countdown", "5"}, []string{"5s"}},
		{[]string{"config", "sound", "off"}, []string{"Sound off."}},
		{[]string{"config", "editor", "vim"}, []string{"vim"}},
		{[]string{"config", "shell", "zsh"}, []string{"Shell set to zsh."}},
	} {
		code, out, errOut := runCLI(t, tc.args...)
		if code != 0 {
			t.Errorf("%v: exit %d, stderr %q", tc.args, code, errOut)
			continue
		}
		for _, w := range tc.want {
			if !strings.Contains(out, w) {
				t.Errorf("%v: output lacks %q:\n%s", tc.args, w, out)
			}
		}
	}

	cfg, err := storage.LoadConfig()
	if err != nil || cfg.Editor != "vim" || cfg.Shell != "zsh" || cfg.IsSoundEnabled() || cfg.GetCountdown() != 5 {
		t.Errorf("config not saved: %+v, %v", cfg, err)
	}
}

func TestFailuresGoToStderrWithExitStatus(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, args := range [][]string{
		{"info", "nope"},
		{"top", "nope"},
		{"recent", "nope"},
		{"list", "nope-nothing"},
		{"search", "zzzzqqq"},
		{"reset", "nope"},
		{"rm", "nope"},
		{"export", "nope"},
		{"import", "/does/not/exist.json"},
		{"restore", "ft_strlen"},
		{"config", "countdown", "11"},
		{"config", "countdown", "3junk"},
		{"config", "sound", "banana"},
		{"config", "shell"},
		{"config", "bogus"},
		{"start"}, {"info"}, {"top"}, {"recent"}, {"search"}, {"rm"}, {"reset"}, {"export"}, {"import"},
		{"unknown-command"},
	} {
		code, out, errOut := runCLI(t, args...)
		if code == 0 || out != "" || !strings.HasPrefix(errOut, "codedrill: ") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", args, code, out, errOut)
		}
	}
}

// An unreadable config directory used to crash list/info/top with a nil
// history and let config changes report success without saving.
func TestUnwritableConfigDirFailsCleanly(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/dev/null")
	for _, args := range [][]string{{"list"}, {"info", "ft_strlen"}, {"top", "ft_strlen"}, {"stats"}, {"config", "sound", "off"}} {
		code, _, errOut := runCLI(t, args...)
		if code == 0 || strings.Contains(errOut, "panic") {
			t.Errorf("%v: exit %d, stderr %q", args, code, errOut)
		}
	}
}

func TestCorruptConfigIsNotOverwritten(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, _ := storage.GetConfigPath()
	if err := os.WriteFile(path, []byte(`{"editor": "hx",`), 0644); err != nil {
		t.Fatal(err)
	}
	if code, _, _ := runCLI(t, "config", "sound", "off"); code == 0 {
		t.Fatal("changing a setting over a corrupt config should fail")
	}
	if data, _ := os.ReadFile(path); string(data) != `{"editor": "hx",` {
		t.Fatalf("corrupt config was overwritten: %q", data)
	}
}

func TestResetResolvesNames(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	hist, _ := storage.LoadHistory()
	if _, _, err := hist.RecordAttempt("ft_strlen", time.Second, true, 1, false); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runCLI(t, "reset", "FT_STRLEN"); code != 0 {
		t.Fatalf("reset by name: %s", errOut)
	}
	hist, _ = storage.LoadHistory()
	if hist.GetPB("ft_strlen") != 0 {
		t.Fatal("reset by name left the PB in place")
	}
	if code, _, _ := runCLI(t, "reset", "all"); code != 0 {
		t.Fatal("reset all failed")
	}
}

func TestSolutionAttachAndList(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "answer.c"), []byte("int answer(void) { return 1; }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "helper.h"), []byte("#define HELPER 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	code, out, errOut := runCLI(t, "solution", "ft_strlen", filepath.Join(dir, "answer.c"))
	if code != 0 || !strings.Contains(out, "Attached 1 solution file") || !strings.Contains(out, "answer.c") {
		t.Fatalf("attach: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
	if code, out, errOut := runCLI(t, "solution", "FT_STRLEN", filepath.Join(dir, "helper.h")); code != 0 || !strings.Contains(out, "Attached 1 solution file") {
		t.Fatalf("attach by title: exit %d, stdout %q, stderr %q", code, out, errOut)
	}

	code, out, errOut = runCLI(t, "solution", "ft_strlen")
	if code != 0 {
		t.Fatalf("list: exit %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"answer.c", "helper.h", "Destination:"} {
		if !strings.Contains(out, want) {
			t.Fatalf("list lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "return 1") || strings.Contains(out, "HELPER") {
		t.Fatalf("list exposed file contents:\n%s", out)
	}

	tree := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tree, "lib"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "lib", "util.h"), []byte("int util(void);\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tree, "main.c"), []byte("int main(void) { return 0; }\n"), 0644); err != nil {
		t.Fatal(err)
	}
	code, out, errOut = runCLI(t, "solution", "ft_strlen", tree)
	if code != 0 || !strings.Contains(out, filepath.Join("lib", "util.h")) || !strings.Contains(out, "main.c") {
		t.Fatalf("attach directory: exit %d, stdout %q, stderr %q", code, out, errOut)
	}

	code, out, _ = runCLI(t, "solution", "ft_strlen")
	if code != 0 || !strings.Contains(out, "answer.c") || !strings.Contains(out, filepath.Join("lib", "util.h")) {
		t.Fatalf("directory attach replaced earlier files:\n%s", out)
	}

	code, out, errOut = runCLI(t, "solution", "two_sum")
	if code != 0 || !strings.Contains(out, "No solution files attached") {
		t.Fatalf("empty list: exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}

func TestSolutionFailures(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, args := range [][]string{
		{"solution"},
		{"solution", "nope"},
		{"solution", "ft_strlen", filepath.Join(t.TempDir(), "missing.c")},
	} {
		code, out, errOut := runCLI(t, args...)
		if code == 0 || out != "" || !strings.HasPrefix(errOut, "codedrill: ") {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", args, code, out, errOut)
		}
	}
}

func TestImportExportRemoveRestore(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	code, out, _ := runCLI(t, "export", "ft_strlen")
	if code != 0 {
		t.Fatal("export failed")
	}
	var d map[string]any
	if err := json.Unmarshal([]byte(out), &d); err != nil || d["id"] != "ft_strlen" {
		t.Fatalf("export is not the drill's JSON: %v", err)
	}
	if _, ok := d["is_preset"]; ok {
		t.Error("export leaks the internal preset flag")
	}

	d["id"], d["title"], d["category"] = "qa-custom", "QA custom", "Custom"
	data, _ := json.Marshal(d)
	path := filepath.Join(t.TempDir(), "drill.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runCLI(t, "import", path); code != 0 || !strings.Contains(out, "qa-custom") {
		t.Fatalf("import: %d %q", code, out)
	}
	if _, out, _ := runCLI(t, "list", "custom"); !strings.Contains(out, "qa-custom") {
		t.Fatal("imported drill not listed")
	}
	if code, out, _ := runCLI(t, "rm", "QA custom"); code != 0 || !strings.Contains(out, "Deleted") {
		t.Fatalf("rm custom: %d %q", code, out)
	}

	if code, out, _ := runCLI(t, "rm", "ft_strlen"); code != 0 || !strings.Contains(out, "Hid") {
		t.Fatalf("rm preset: %d %q", code, out)
	}
	if _, out, _ := runCLI(t, "restore"); !strings.Contains(out, "ft_strlen") {
		t.Fatal("hidden drill not listed")
	}
	if code, _, _ := runCLI(t, "restore", "ft_strlen"); code != 0 {
		t.Fatal("restore failed")
	}
	if _, out, _ := runCLI(t, "list"); !strings.Contains(out, "ft_strlen") {
		t.Fatal("restored drill missing")
	}
}

func attachMatchReference(t *testing.T, drillID string, files map[string]string) {
	t.Helper()
	dir, err := solutions.OverlayDir(drillID)
	if err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}

func matchWorkspace(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestMatchExact(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	attachMatchReference(t, "ft_strlen", map[string]string{
		"ft_strlen.c": "int ft_strlen(char *s) { return 0; }\n",
	})
	workspace := matchWorkspace(t, map[string]string{
		"ft_strlen.c": "// student\nint ft_strlen(char *s)\n{\n\treturn (0);\n}\n",
	})
	code, out, errOut := runCLI(t, "match", "ft_strlen", "--workspace", workspace)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"Score: 100.0%", "0 missing, 0 extra"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "First difference") {
		t.Fatalf("exact match reports a difference:\n%s", out)
	}
}

func TestMatchPartial(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	attachMatchReference(t, "ft_strlen", map[string]string{
		"ft_strlen.c": "int ft_strlen(char *s) { return 0; }\n",
	})
	workspace := matchWorkspace(t, map[string]string{
		"ft_strlen.c": "int ft_strlen(char *s) { return 1; }\n",
	})
	code, out, errOut := runCLI(t, "match", "ft_strlen", "--workspace="+workspace)
	if code != 0 {
		t.Fatalf("mismatch should exit 0, got %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"Score: 91.7%", "1 missing, 1 extra", "First difference: ft_strlen.c:1"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "100.0%") {
		t.Fatalf("non-exact match shows 100%%:\n%s", out)
	}
}

func TestMatchMultipleFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	attachMatchReference(t, "ft_strlen", map[string]string{
		"ft_strlen.c": "int ft_strlen(char *s) { return 0; }\n",
		"helper.h":    "int helper(void);\n",
	})
	workspace := matchWorkspace(t, map[string]string{
		"ft_strlen.c": "int ft_strlen(char *s) { return 0; }\n",
		"helper.h":    "int helper(void);\n",
	})
	code, out, errOut := runCLI(t, "match", "ft_strlen", "--workspace", workspace)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"Files:", "ft_strlen.c", "helper.h", "100.0%"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestMatchMissingWorkspaceFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	attachMatchReference(t, "ft_strlen", map[string]string{
		"ft_strlen.c": "int ft_strlen(char *s) { return 0; }\n",
		"helper.h":    "int helper(void);\n",
	})
	workspace := matchWorkspace(t, map[string]string{
		"ft_strlen.c": "int ft_strlen(char *s) { return 0; }\n",
	})
	code, out, errOut := runCLI(t, "match", "ft_strlen", "--workspace", workspace)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	for _, want := range []string{"helper.h", "missing from workspace", "First difference: helper.h"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output lacks %q:\n%s", want, out)
		}
	}
}

func TestMatchMissingReference(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	code, out, errOut := runCLI(t, "match", "ft_swap", "--workspace", t.TempDir())
	if code == 0 || out != "" || !strings.Contains(errOut, "no reference solution") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}

func TestMatchUnsupportedLanguage(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := drill.SaveDrill(drill.Drill{
		ID: "unsupported-drill", Title: "Python drill", Category: "Custom",
		Language: "python", EntryFile: "main.py",
	}); err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runCLI(t, "match", "unsupported-drill", "--workspace", t.TempDir())
	if code == 0 || out != "" || !strings.Contains(errOut, "not supported") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}

func TestMatchJSON(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	attachMatchReference(t, "ft_strlen", map[string]string{
		"ft_strlen.c": "int reference_only_symbol(char *s) { return 0; }\n",
	})
	workspace := matchWorkspace(t, map[string]string{
		"ft_strlen.c": "int reference_only_symbol(char *s) { return 1; }\n",
	})
	code, out, errOut := runCLI(t, "match", "ft_strlen", "--json", "--workspace", workspace)
	if code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if strings.Contains(out, "reference_only_symbol") || strings.Contains(out, "return 0") {
		t.Fatalf("JSON exposed reference source:\n%s", out)
	}
	var report struct {
		Drill           string  `json:"drill"`
		Exact           bool    `json:"exact"`
		Percent         float64 `json:"percent"`
		MissingTokens   int     `json:"missing_tokens"`
		ExtraTokens     int     `json:"extra_tokens"`
		FirstDifference *struct {
			Path string `json:"path"`
			Line int    `json:"line"`
		} `json:"first_difference"`
		Files []struct {
			Path    string  `json:"path"`
			Missing bool    `json:"missing"`
			Exact   bool    `json:"exact"`
			Percent float64 `json:"percent"`
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if report.Drill != "ft_strlen" || report.Exact || report.Percent >= 100 {
		t.Fatalf("report is %+v", report)
	}
	if report.MissingTokens != 1 || report.ExtraTokens != 1 {
		t.Fatalf("token counts are %+v", report)
	}
	if report.FirstDifference == nil || report.FirstDifference.Path != "ft_strlen.c" || report.FirstDifference.Line != 1 {
		t.Fatalf("first difference is %+v", report.FirstDifference)
	}
	if len(report.Files) != 1 || report.Files[0].Path != "ft_strlen.c" || report.Files[0].Exact || report.Files[0].Missing {
		t.Fatalf("files are %+v", report.Files)
	}
}

func TestMatchOperationalFailureExitsNonzero(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	code, out, errOut := runCLI(t, "match", "ft_strlen", "--workspace", filepath.Join(t.TempDir(), "missing"))
	if code == 0 || out != "" || !strings.HasPrefix(errOut, "codedrill: ") {
		t.Fatalf("exit %d, stdout %q, stderr %q", code, out, errOut)
	}
}
