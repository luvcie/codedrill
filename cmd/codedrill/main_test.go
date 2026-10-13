package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		{[]string{"config"}, []string{"Configuration:"}},
		{[]string{"config", "countdown", "off"}, []string{"Countdown off"}},
		{[]string{"config", "countdown", "5"}, []string{"5s"}},
		{[]string{"config", "sound", "off"}, []string{"Sound off."}},
		{[]string{"config", "editor", "vim"}, []string{"vim"}},
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
	if err != nil || cfg.Editor != "vim" || cfg.IsSoundEnabled() || cfg.GetCountdown() != 5 {
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
