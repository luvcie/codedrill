package runner

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

type Session struct {
	SessionName string
	Workspace   string
	EntryFile   string
	TestCommand string
	StartTime   time.Time
	Submits     int

	drill   drill.Drill
	pb      time.Duration
	logoRGB string
}

const CrocArt = ":==8-;;;;;;;;,,,    ,\n" +
	"`\"\"-=,)===,)=\"''''''\n" +
	"   '''  '''"

var helperFiles = []string{
	"pace", "p", "run", "r", "submit", "s", "info", "i", "diff", "d",
	"start-editor.sh", "start-terminal.sh", "layout.kdl",
	".sprint_start", ".starter_code",
}

const defaultLogoRGB = "127;160;90"

func NewSession(d drill.Drill, pb time.Duration) (*Session, error) {
	return NewSessionWithLogo(d, pb, "")
}

func NewSessionWithLogo(d drill.Drill, pb time.Duration, logoRGB string) (_ *Session, err error) {
	tempDir, err := os.MkdirTemp("", "codedrill-*")
	if err != nil {
		return nil, fmt.Errorf("create temp workspace: %w", err)
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tempDir)
		}
	}()

	if d.Dir != "" {
		if err := copyDrillDir(d.Dir, tempDir); err != nil {
			return nil, fmt.Errorf("copy drill files: %w", err)
		}
	}

	if !filepath.IsLocal(d.EntryFile) || filepath.Clean(d.EntryFile) == "." {
		return nil, fmt.Errorf("invalid entry filename: %q", d.EntryFile)
	}
	for name, content := range d.ExtraFiles {
		if err := writeWorkspaceFile(tempDir, name, []byte(content), 0755); err != nil {
			return nil, fmt.Errorf("write extra file %q: %w", name, err)
		}
	}

	entryPath := filepath.Join(tempDir, d.EntryFile)
	info, statErr := os.Stat(entryPath)
	if statErr != nil && !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("check entry file: %w", statErr)
	}
	if info != nil && !info.Mode().IsRegular() {
		return nil, fmt.Errorf("entry is not a regular file: %s", d.EntryFile)
	}
	if d.StarterCode != "" || os.IsNotExist(statErr) {
		if err := writeWorkspaceFile(tempDir, d.EntryFile, []byte(d.StarterCode), 0644); err != nil {
			return nil, fmt.Errorf("write entry file: %w", err)
		}
	}

	reserved := helperFiles
	if d.Description != "" {
		reserved = append(reserved[:len(reserved):len(reserved)], "PROBLEM.md")
	}
	for _, name := range reserved {
		if _, err := os.Lstat(filepath.Join(tempDir, name)); err == nil {
			return nil, fmt.Errorf("drill file %q clashes with a workspace helper", name)
		}
	}

	starter, err := os.ReadFile(entryPath)
	if err != nil {
		return nil, fmt.Errorf("read entry file: %w", err)
	}

	sess := &Session{
		SessionName: fmt.Sprintf("codedrill-%d", time.Now().UnixMilli()),
		Workspace:   tempDir,
		EntryFile:   d.EntryFile,
		TestCommand: d.TestCommand,
		StartTime:   time.Now(),
		drill:       d,
		pb:          pb,
		logoRGB:     logoRGB,
	}
	if err := sess.writeHelpers(starter); err != nil {
		return nil, fmt.Errorf("write workspace helpers: %w", err)
	}
	return sess, nil
}

func copyDrillDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if name == "." && !entry.IsDir() {
			return fmt.Errorf("source is not a directory: %s", path)
		}
		if name == "." || name == "drill.json" {
			return nil
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(dst, name), 0755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("unsupported source file: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return writeWorkspaceFile(dst, name, data, 0755)
	})
}

func writeWorkspaceFile(root, name string, data []byte, mode os.FileMode) error {
	if !filepath.IsLocal(name) || filepath.Clean(name) == "." {
		return fmt.Errorf("invalid workspace filename: %q", name)
	}
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, mode)
}

// shellQuote wraps s in single quotes so bash treats it as one literal word.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// script builds a helper that runs from the workspace with the given
// variables set. vars alternates name, value.
func script(body string, vars ...string) []byte {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\ncd \"$(dirname \"$0\")\" || exit 1\n")
	for i := 0; i+1 < len(vars); i += 2 {
		fmt.Fprintf(&b, "%s=%s\n", vars[i], shellQuote(vars[i+1]))
	}
	b.WriteString(body)
	return []byte(b.String())
}

func ms(d time.Duration) string { return strconv.FormatInt(d.Milliseconds(), 10) }

const paceBody = `fmt_ms() {
  if [ "$1" -ge 60000 ]; then
    printf '%dm %02d.%03ds' $(($1 / 60000)) $(($1 % 60000 / 1000)) $(($1 % 1000))
  else
    printf '%d.%03ds' $(($1 / 1000)) $(($1 % 1000))
  fi
}

now_ms() {
  local t
  t=$(date +%s%3N 2>/dev/null)
  case "$t" in
    *N*)
      if command -v python3 >/dev/null 2>&1; then
        t=$(python3 -c 'import time; print(int(time.time()*1000))')
      else
        t=$(($(date +%s) * 1000))
      fi
      ;;
  esac
  echo "$t"
}

pace_line() {
  [ -f .sprint_start ] || return 1
  local elapsed=$(( $(now_ms) - $(cat .sprint_start) ))
  local rank=C
  if [ "$elapsed" -le "$S_MS" ]; then rank=S
  elif [ "$elapsed" -le "$A_MS" ]; then rank=A
  elif [ "$elapsed" -le "$B_MS" ]; then rank=B
  fi
  local ref=$PB_MS label=PB
  if [ "$ref" -le 0 ]; then ref=$S_MS label="S pace"; fi
  local color='\033[1;32m' delta
  if [ "$elapsed" -lt "$ref" ]; then
    delta="-$(fmt_ms $((ref - elapsed))) vs $label"
  else
    delta="+$(fmt_ms $((elapsed - ref))) vs $label"
    color='\033[1;33m'
  fi
  printf "${color}pace %s  %s rank  (%s)\033[0m" "$(fmt_ms "$elapsed")" "$rank" "$delta"
}

case "$1" in
  -w|--watch|watch)
    printf '\033[?25l'
    trap 'printf "\033[?25h\033[0m\n"' EXIT
    trap 'exit 0' INT TERM
    while true; do
      printf '\r\033[K'
      pace_line
      sleep 0.1
    done
    ;;
  *)
    pace_line && echo
    ;;
esac
`

const runBody = `./pace
START=$(date +%s%N 2>/dev/null)
eval "$TEST_CMD"
STATUS=$?
END=$(date +%s%N 2>/dev/null)
ELAPSED=""
case "$START" in *N*) ;; *) ELAPSED=" ($(( (END - START) / 1000000 ))ms)" ;; esac
if [ $STATUS -eq 0 ]; then
  [ "$SOUND" = 1 ] && printf '\a'
  echo -e "\033[1;32mPASS${ELAPSED}\033[0m  s to submit"
else
  echo -e "\033[1;31mFAIL (exit $STATUS)\033[0m  i for the problem"
fi
exit $STATUS
`

const submitBody = `if [ -n "$ZELLIJ_SESSION_NAME" ] && command -v zellij >/dev/null 2>&1; then
  zellij kill-session "$ZELLIJ_SESSION_NAME" >/dev/null 2>&1
fi
exit 0
`

const infoBody = `if [ ! -f PROBLEM.md ]; then
  echo "$TITLE"
elif command -v glow >/dev/null 2>&1; then
  glow -s dark PROBLEM.md
elif command -v bat >/dev/null 2>&1; then
  bat --paging=never PROBLEM.md
else
  cat PROBLEM.md
fi
`

// The shell pane puts "." first in PATH so "d" runs this script, which means
// a bare "diff" here would call itself. find_bin skips workspace copies.
const diffBody = `STARTER=.starter_code

find_bin() {
  local p
  for p in $(type -aP "$1"); do
    [ "$p" -ef "./$1" ] || { echo "$p"; return; }
  done
}

case "$1" in
  -r|--reset|reset)
    resp=y
    if [ "$2" != "-y" ]; then
      read -r -p "Reset $ENTRY to the starter code? [y/N] " resp
    fi
    case "$resp" in
      [yY]|[yY][eE][sS])
        cp "$STARTER" "$ENTRY"
        echo "Reset $ENTRY to the starter code."
        ;;
      *) echo "Cancelled." ;;
    esac
    exit 0
    ;;
esac

if cmp -s "$STARTER" "$ENTRY"; then
  echo "No changes in $ENTRY yet."
  exit 0
fi

echo -e "\033[1;36mChanges in $ENTRY:\033[0m"
GIT=$(find_bin git)
DIFF=$(find_bin diff)
if [ -n "$GIT" ]; then
  "$GIT" --no-pager diff --no-index --color=always -- "$STARTER" "$ENTRY" | tail -n +5
elif [ -n "$DIFF" ]; then
  "$DIFF" -u "$STARTER" "$ENTRY" | tail -n +3
else
  echo "Neither git nor diff is installed."
fi
exit 0
`

const editorBody = `if ! command -v "${EDITOR_CMD%% *}" >/dev/null 2>&1; then
  echo -e "\033[1;31mEditor '$EDITOR_CMD' not found in PATH.\033[0m"
  echo "Set one with: codedrill config editor <name>, or export CODEDRILL_EDITOR."
  exec bash
fi
$EDITOR_CMD "$ENTRY"
if [ "$AUTO_SUBMIT" = 1 ]; then
  exec ./submit
fi
echo -e "\033[1;33mEditor closed.\033[0m Reopen it with: $EDITOR_CMD $ENTRY"
exec bash
`

const terminalBody = `export PATH=".:$PATH"
clear
columns=$(tput cols 2>/dev/null || echo 40)
if [ "${#TITLE}" -gt "$columns" ]; then TITLE="${TITLE:0:$((columns-1))}…"; fi
printf '\033[38;2;%sm%s\n\033[1mcodedrill\033[0m\n' "$LOGO_RGB" "$CROC"
echo "$TITLE"
echo -e "\033[0;37mS rank <$S_TARGET\nPB: $PB\033[0m"
echo
echo -e "\033[1;32mr  run tests\033[0m"
echo 'p  pace   p -w watch'
echo 'd  diff   d -r reset'
echo 'i  problem'
echo -e "\033[1;36ms  submit\033[0m"
echo "$HINT"
echo
exec bash
`

type helperFile struct {
	name string
	data []byte
	mode os.FileMode
}

func (s *Session) writeHelpers(starter []byte) error {
	d := s.drill
	sMax, aMax, bMax := drill.PaceTargets(d.Difficulty)

	sound := "0"
	if cfg, _ := storage.LoadConfig(); cfg.IsSoundEnabled() {
		sound = "1"
	}
	title := d.Title
	if title == "" {
		title = d.ID
	}
	pb := "None yet"
	if s.pb > 0 {
		pb = storage.FormatDuration(s.pb)
	}
	autoSubmit, hint := "0", "Auto-submit off."
	if d.AutoSubmit {
		autoSubmit, hint = "1", "Close the editor to submit."
	}
	editor := DetectEditor()
	ws := s.Workspace
	logo := s.logoRGB
	if logo == "" {
		logo = defaultLogoRGB
	}

	files := []helperFile{
		{".sprint_start", []byte(strconv.FormatInt(s.StartTime.UnixMilli(), 10)), 0644},
		{".starter_code", starter, 0644},
		{"pace", script(paceBody, "S_MS", ms(sMax), "A_MS", ms(aMax), "B_MS", ms(bMax), "PB_MS", ms(s.pb)), 0755},
		{"run", script(runBody, "TEST_CMD", s.resolveRunCommand(), "SOUND", sound), 0755},
		{"submit", script(submitBody), 0755},
		{"info", script(infoBody, "TITLE", title), 0755},
		{"diff", script(diffBody, "ENTRY", s.EntryFile), 0755},
		{"start-editor.sh", script(editorBody, "EDITOR_CMD", editor, "ENTRY", s.EntryFile, "AUTO_SUBMIT", autoSubmit), 0755},
		{"start-terminal.sh", script(terminalBody, "LOGO_RGB", logo, "CROC", CrocArt, "TITLE", title, "S_TARGET", fmt.Sprintf("%ds", int(sMax.Seconds())), "PB", pb, "HINT", hint), 0755},
		{"layout.kdl", []byte(fmt.Sprintf(`layout {
    pane split_direction="vertical" {
        pane command="bash" {
            args %s
            size "60%%"
        }
        pane command="bash" {
            args %s
            size "40%%"
        }
    }
}
`, strconv.Quote(filepath.Join(ws, "start-editor.sh")), strconv.Quote(filepath.Join(ws, "start-terminal.sh")))), 0644},
	}
	if d.Description != "" {
		category := d.Category
		if category == "" {
			category = "Custom"
		}
		difficulty := d.Difficulty
		if difficulty == "" {
			difficulty = "easy"
		}
		problem := fmt.Sprintf("# %s\n\n- **Category:** %s\n- **Language:** %s\n- **Difficulty:** %s\n\n%s\n",
			title, category, d.Language, difficulty, d.Description)
		files = append(files, helperFile{"PROBLEM.md", []byte(problem), 0644})
	}

	for _, f := range files {
		if err := os.WriteFile(filepath.Join(ws, f.name), f.data, f.mode); err != nil {
			return err
		}
	}
	for short, long := range map[string]string{"p": "pace", "r": "run", "s": "submit", "i": "info", "d": "diff"} {
		if err := os.Symlink(long, filepath.Join(ws, short)); err != nil {
			if err := os.WriteFile(filepath.Join(ws, short), []byte("#!/usr/bin/env bash\nexec \"$(dirname \"$0\")/"+long+"\" \"$@\"\n"), 0755); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Session) Cleanup() {
	if s.SessionName != "" {
		_ = exec.Command("zellij", "kill-session", s.SessionName).Run()
	}
	if s.Workspace != "" {
		_ = os.RemoveAll(s.Workspace)
	}
}

func DetectEditor() string {
	cfg, _ := storage.LoadConfig()
	if cfg.Editor != "" {
		if _, err := exec.LookPath(cfg.Editor); err == nil {
			return cfg.Editor
		}
	}
	if env := os.Getenv("CODEDRILL_EDITOR"); env != "" {
		return env
	}
	for _, name := range []string{"hx", "nvim", "vim", "emacs", "micro"} {
		if _, err := exec.LookPath(name); err == nil {
			return name
		}
	}
	if env := os.Getenv("EDITOR"); env != "" {
		return env
	}
	if env := os.Getenv("VISUAL"); env != "" {
		return env
	}
	if _, err := exec.LookPath("nano"); err == nil {
		return "nano"
	}
	return "vim"
}

func (s *Session) GetSprintCmd() *exec.Cmd {
	if _, err := exec.LookPath("zellij"); err == nil {
		layoutPath := filepath.Join(s.Workspace, "layout.kdl")
		cmd := exec.Command("zellij", "attach", "-c", s.SessionName, "options", "--default-layout", layoutPath)
		cmd.Dir = s.Workspace
		return cmd
	}

	args := strings.Fields(DetectEditor())
	cmd := exec.Command(args[0], append(args[1:], filepath.Join(s.Workspace, s.EntryFile))...)
	cmd.Dir = s.Workspace
	return cmd
}

func (s *Session) resolveRunCommand() string {
	if cmd := strings.TrimSpace(s.TestCommand); cmd != "" {
		return cmd
	}

	f := shellQuote(s.EntryFile)
	switch filepath.Ext(s.EntryFile) {
	case ".c":
		return fmt.Sprintf("if grep -q 'main(' %[1]s; then gcc -Wall -Wextra -Werror %[1]s -o a.out && ./a.out; else gcc -Wall -Wextra -Werror -c %[1]s; fi", f)
	case ".cpp", ".cc":
		return fmt.Sprintf("if grep -q 'main(' %[1]s; then g++ -Wall -Wextra -Werror %[1]s -o a.out && ./a.out; else g++ -Wall -Wextra -Werror -c %[1]s; fi", f)
	case ".go":
		return fmt.Sprintf("if grep -q 'func main(' %[1]s; then go run %[1]s; else [ -f go.mod ] || go mod init drill >/dev/null 2>&1; go test -v; fi", f)
	case ".rs":
		return fmt.Sprintf("if [ -f Cargo.toml ]; then cargo test; elif grep -q 'fn main(' %[1]s; then rustc %[1]s -o a.out && ./a.out; else rustc --test %[1]s -o a.out && ./a.out; fi", f)
	case ".py":
		return fmt.Sprintf("if ls test_*.py >/dev/null 2>&1; then python3 -m unittest discover -s . -p 'test_*.py'; else python3 %s; fi", f)
	case ".js":
		return "node " + f
	case ".ts":
		return fmt.Sprintf("if command -v bun >/dev/null 2>&1; then bun run %[1]s; elif command -v tsx >/dev/null 2>&1; then tsx %[1]s; elif command -v deno >/dev/null 2>&1; then deno run %[1]s; else node %[1]s; fi", f)
	case ".sh":
		return "bash " + f
	}
	return fmt.Sprintf("echo %s", shellQuote("No test command defined for "+s.EntryFile))
}

var testTimeout = 15 * time.Second

// RunTests runs the drill's test command in the workspace and reports whether
// it passed, along with its combined output.
func (s *Session) RunTests() (bool, string) {
	s.Submits++
	cmdStr := s.resolveRunCommand()

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", cmdStr)
	cmd.Dir = s.Workspace
	configureTestProcess(cmd)
	cmd.WaitDelay = 250 * time.Millisecond

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	if ctx.Err() == context.DeadlineExceeded {
		msg := fmt.Sprintf("Timed out after %s. Check for infinite loops or blocking reads.", testTimeout)
		if output != "" {
			msg += "\n\nOutput before the timeout:\n" + output
		}
		return false, msg
	}
	if err == nil {
		if output == "" {
			output = fmt.Sprintf("%s exited 0.", cmdStr)
		}
		return true, output
	}
	if output == "" {
		output = fmt.Sprintf("%s failed: %v", cmdStr, err)
	}
	return false, output
}
