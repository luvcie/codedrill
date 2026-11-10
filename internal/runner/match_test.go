package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/luvcie/codedrill/internal/drill"
)

func newMatchSession(t *testing.T) *Session {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s, err := NewSession(drill.Drill{
		ID: "ft_strlen", Title: "ft_strlen", Language: "c", EntryFile: "ft_strlen.c",
		StarterCode: "int ft_strlen(char *s) { return 0; }\n", TestCommand: "true",
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Cleanup)
	return s
}

func TestMatchHelperCreated(t *testing.T) {
	s := newMatchSession(t)
	for _, name := range []string{"m", "match"} {
		info, err := os.Stat(filepath.Join(s.Workspace, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Mode()&0111 == 0 {
			t.Fatalf("%s is not executable", name)
		}
	}
	data, err := os.ReadFile(filepath.Join(s.Workspace, "match"))
	if err != nil {
		t.Fatal(err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	exe, _ = filepath.Abs(exe)
	for _, want := range []string{
		"CODEDRILL=" + shellQuote(exe),
		"DRILL='ft_strlen'",
		"WORKSPACE=" + shellQuote(s.Workspace),
		`"$CODEDRILL" match "$DRILL" --workspace "$WORKSPACE"`,
		"-w|--watch|watch",
		"sleep 0.5",
	} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("match helper lacks %q:\n%s", want, data)
		}
	}
	banner, err := os.ReadFile(filepath.Join(s.Workspace, "start-terminal.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"m  match", "m -w watch"} {
		if !strings.Contains(string(banner), want) {
			t.Fatalf("terminal banner lacks %q:\n%s", want, banner)
		}
	}
}

func TestMatchHelperHasNoReferenceSource(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	source := t.TempDir()
	if err := os.MkdirAll(filepath.Join(source, "solution"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "solution", "secret.c"), []byte("SECRET_REFERENCE_MARKER\n"), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := NewSession(drill.Drill{
		ID: "qa", EntryFile: "main.c", StarterCode: "int main(void) { return 0; }\n",
		TestCommand: "true", Dir: source,
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Cleanup()
	for _, name := range []string{"m", "match"} {
		data, err := os.ReadFile(filepath.Join(s.Workspace, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if strings.Contains(string(data), "SECRET_REFERENCE_MARKER") {
			t.Fatalf("%s exposes reference source", name)
		}
	}
}

const matchStubOutput = `cat <<'OUT'
Match: ft_strlen (C)
Score: 91.7%
Tokens: 11 matched, 1 missing, 1 extra
First difference: ft_strlen.c:1
OUT
`

// installMatchStub points the helper at a fake codedrill binary and returns
// the file where the stub records its arguments.
func installMatchStub(t *testing.T, workspace, body string) string {
	t.Helper()
	args := filepath.Join(t.TempDir(), "args.txt")
	stub := filepath.Join(t.TempDir(), "fake codedrill")
	content := "#!/usr/bin/env bash\nprintf '%s\\n' \"$@\" > " + shellQuote(args) + "\n" + body
	if err := os.WriteFile(stub, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workspace, "match")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(data), "\n")
	replaced := false
	for i, line := range lines {
		if strings.HasPrefix(line, "CODEDRILL=") {
			lines[i] = "CODEDRILL=" + shellQuote(stub)
			replaced = true
			break
		}
	}
	if !replaced {
		t.Fatal("match helper has no CODEDRILL line")
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0755); err != nil {
		t.Fatal(err)
	}
	return args
}

func TestMatchHelperRunsMatch(t *testing.T) {
	base := filepath.Join(t.TempDir(), "work space 'quoted' $x")
	if err := os.MkdirAll(base, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TMPDIR", base)
	s := newMatchSession(t)
	args := installMatchStub(t, s.Workspace, matchStubOutput)

	cmd := exec.Command("./m")
	cmd.Dir = s.Workspace
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("m failed: %v\n%s", err, out)
	}
	want := "match 91.7% | 11 matched, 1 missing, 1 extra | ft_strlen.c:1\n"
	if string(out) != want {
		t.Fatalf("output is %q, want %q", out, want)
	}
	data, err := os.ReadFile(args)
	if err != nil {
		t.Fatal(err)
	}
	wantArgs := "match\nft_strlen\n--workspace\n" + s.Workspace + "\n"
	if string(data) != wantArgs {
		t.Fatalf("stub arguments are %q, want %q", data, wantArgs)
	}
}

func TestMatchHelperWatch(t *testing.T) {
	s := newMatchSession(t)
	installMatchStub(t, s.Workspace, matchStubOutput)

	cmd := exec.Command("./m", "-w")
	cmd.Dir = s.Workspace
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 512)
	deadline := time.Now().Add(5 * time.Second)
	for strings.Count(string(buf), "match 91.7%") < 2 && time.Now().Before(deadline) {
		n, err := stdout.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	if got := strings.Count(string(buf), "match 91.7%"); got < 2 {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatalf("watch printed %d updates, want at least 2:\n%s", got, buf)
	}
	if err := cmd.Process.Signal(os.Interrupt); err != nil {
		t.Fatalf("signal watch: %v", err)
	}
	for {
		n, err := stdout.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("watch did not exit cleanly: %v\n%s", err, buf)
	}
	for _, want := range []string{"\033[?25l", "\r\033[K", "\033[?25h"} {
		if !strings.Contains(string(buf), want) {
			t.Fatalf("watch output lacks %q:\n%q", want, buf)
		}
	}
}

func TestMatchHelperMissingReference(t *testing.T) {
	s := newMatchSession(t)
	installMatchStub(t, s.Workspace, "echo 'codedrill: no reference solution for ft_strlen' >&2\nexit 1\n")

	cmd := exec.Command("./m")
	cmd.Dir = s.Workspace
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("missing reference exited 0")
	}
	if string(out) != "match: no reference solution for ft_strlen\n" {
		t.Fatalf("output is %q", out)
	}
}

func TestMatchHelperRunsDiff(t *testing.T) {
	s := newMatchSession(t)
	args := installMatchStub(t, s.Workspace, "")
	cmd := exec.Command("./m", "-d")
	cmd.Dir = s.Workspace
	out, err := cmd.CombinedOutput()
	if err != nil || len(out) != 0 {
		t.Fatalf("m -d: %v, %q", err, out)
	}
	data, err := os.ReadFile(args)
	if err != nil {
		t.Fatal(err)
	}
	if want := "reference-diff\n--session\n" + s.referencePolicy + "\n"; string(data) != want {
		t.Fatalf("diff arguments: %q, want %q", data, want)
	}
}
