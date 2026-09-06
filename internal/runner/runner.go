package runner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/luvcie/codedrill/internal/storage"
)

type Session struct {
	DrillID     string
	Workspace   string
	EntryFile   string
	TestCommand string
	AutoSubmit  bool
	StartTime   time.Time
	Duration    time.Duration
	Submits     int
}

func NewSession(drillID, entryFile, starterCode, testCommand, sourceDir string, autoSubmit bool) (*Session, error) {
	tempDir, err := os.MkdirTemp("", "codedrill-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp workspace: %w", err)
	}

	if sourceDir != "" {
		entries, err := os.ReadDir(sourceDir)
		if err == nil {
			for _, entry := range entries {
				if entry.Name() == "drill.json" {
					continue
				}
				src := filepath.Join(sourceDir, entry.Name())
				dst := filepath.Join(tempDir, entry.Name())
				data, err := os.ReadFile(src)
				if err == nil {
					_ = os.WriteFile(dst, data, 0755)
				}
			}
		}
	}

	filePath := filepath.Join(tempDir, entryFile)
	if starterCode != "" {
		_ = os.WriteFile(filePath, []byte(starterCode), 0644)
	} else if _, err := os.Stat(filePath); os.IsNotExist(err) {
		_ = os.WriteFile(filePath, []byte(""), 0644)
	}

	sess := &Session{
		DrillID:     drillID,
		Workspace:   tempDir,
		EntryFile:   entryFile,
		TestCommand: testCommand,
		AutoSubmit:  autoSubmit,
		StartTime:   time.Now(),
	}

	sess.setupWorkspaceHelpers()

	return sess, nil
}

func (s *Session) setupWorkspaceHelpers() {
	cmdStr := s.resolveRunCommand()

	runScript := fmt.Sprintf(`#!/usr/bin/env bash
cd "%s"
echo -e "\033[1;36m==> Compiling / Running: %s\033[0m"
%s
`, s.Workspace, cmdStr, cmdStr)
	_ = os.WriteFile(filepath.Join(s.Workspace, "run"), []byte(runScript), 0755)

	submitScript := fmt.Sprintf(`#!/usr/bin/env bash
cd "%s"
echo -e "\033[1;32m==> Submitting sprint...\033[0m"
if command -v zellij >/dev/null 2>&1; then
  if [ -n "$ZELLIJ_SESSION_NAME" ]; then
    zellij kill-session "$ZELLIJ_SESSION_NAME" >/dev/null 2>&1
  else
    zellij kill-all-sessions -y >/dev/null 2>&1
  fi
fi
exit 0
`, s.Workspace)
	_ = os.WriteFile(filepath.Join(s.Workspace, "submit"), []byte(submitScript), 0755)

	editor := DetectEditor()
	var editorLauncher string
	if s.AutoSubmit {
		editorLauncher = fmt.Sprintf(`#!/usr/bin/env bash
cd "%s"
%s "%s"
./submit
`, s.Workspace, editor, s.EntryFile)
	} else {
		editorLauncher = fmt.Sprintf(`#!/usr/bin/env bash
cd "%s"
%s "%s"
echo -e "\033[1;33m==> Editor closed. You are in the workspace shell.\033[0m"
echo -e "You can re-open your editor (\033[1;36m%s %s\033[0m), create files, or edit headers."
exec bash
`, s.Workspace, editor, s.EntryFile, editor, s.EntryFile)
	}
	_ = os.WriteFile(filepath.Join(s.Workspace, "start-editor.sh"), []byte(editorLauncher), 0755)

	autoSubmitHint := "• Run \033[1;33m./submit\033[0m (or close editor) to lock time"
	if !s.AutoSubmit {
		autoSubmitHint = "• Run \033[1;33m./submit\033[0m when done to lock time"
	}

	terminalLauncher := fmt.Sprintf(`#!/usr/bin/env bash
cd "%s"
clear
echo -e "\033[1;35m==============================================\033[0m"
echo -e "\033[1;37m CODEDRILL — SPRINT ACTIVE                    \033[0m"
echo -e "\033[1;35m==============================================\033[0m"
echo -e " • Run \033[1;32m./run\033[0m     to compile and execute code"
echo -e " %s"
echo -e "\033[1;35m----------------------------------------------\033[0m"
echo ""
exec bash
`, s.Workspace, autoSubmitHint)
	_ = os.WriteFile(filepath.Join(s.Workspace, "start-terminal.sh"), []byte(terminalLauncher), 0755)

	layoutKdl := fmt.Sprintf(`layout {
    pane split_direction="vertical" {
        pane command="bash" {
            args "%s"
            size "60%%"
        }
        pane command="bash" {
            args "%s"
            size "40%%"
        }
    }
}
`, filepath.Join(s.Workspace, "start-editor.sh"), filepath.Join(s.Workspace, "start-terminal.sh"))
	_ = os.WriteFile(filepath.Join(s.Workspace, "layout.kdl"), []byte(layoutKdl), 0644)
}

func (s *Session) Cleanup() {
	if s.Workspace != "" {
		_ = os.RemoveAll(s.Workspace)
	}
}

func DetectEditor() string {
	cfg := storage.LoadConfig()
	if cfg.Editor != "" {
		if _, err := exec.LookPath(cfg.Editor); err == nil {
			return cfg.Editor
		}
	}

	if envCustom := os.Getenv("CODEDRILL_EDITOR"); envCustom != "" {
		return envCustom
	}

	for _, modern := range []string{"hx", "nvim", "vim", "emacs", "micro"} {
		if _, err := exec.LookPath(modern); err == nil {
			return modern
		}
	}

	if envEditor := os.Getenv("EDITOR"); envEditor != "" {
		return envEditor
	}
	if envVisual := os.Getenv("VISUAL"); envVisual != "" {
		return envVisual
	}

	for _, fallback := range []string{"vim", "nano"} {
		if _, err := exec.LookPath(fallback); err == nil {
			return fallback
		}
	}

	return "vim"
}

func (s *Session) GetSprintCmd() *exec.Cmd {
	if _, err := exec.LookPath("zellij"); err == nil {
		layoutPath := filepath.Join(s.Workspace, "layout.kdl")
		cmd := exec.Command("zellij", "-l", layoutPath)
		cmd.Dir = s.Workspace
		return cmd
	}

	editor := DetectEditor()
	targetFile := filepath.Join(s.Workspace, s.EntryFile)
	cmd := exec.Command(editor, targetFile)
	cmd.Dir = s.Workspace
	return cmd
}

func (s *Session) resolveRunCommand() string {
	cmdStr := strings.TrimSpace(s.TestCommand)
	if cmdStr != "" {
		return cmdStr
	}

	ext := filepath.Ext(s.EntryFile)
	switch ext {
	case ".c":
		return fmt.Sprintf("if grep -q 'main(' %s; then gcc -Wall -Wextra -Werror %s -o a.out && ./a.out; else gcc -Wall -Wextra -Werror -c %s; fi", s.EntryFile, s.EntryFile, s.EntryFile)
	case ".cpp", ".cc":
		return fmt.Sprintf("if grep -q 'main(' %s; then g++ -Wall -Wextra -Werror %s -o a.out && ./a.out; else g++ -Wall -Wextra -Werror -c %s; fi", s.EntryFile, s.EntryFile, s.EntryFile)
	case ".go":
		return fmt.Sprintf("go run %s", s.EntryFile)
	case ".rs":
		return fmt.Sprintf("rustc %s -o a.out && ./a.out", s.EntryFile)
	case ".py":
		return fmt.Sprintf("python3 %s", s.EntryFile)
	}

	return fmt.Sprintf("echo 'No test command defined for %s'", s.EntryFile)
}

func (s *Session) RunTests() (bool, string, error) {
	s.Submits++
	cmdStr := s.resolveRunCommand()

	cmd := exec.Command("bash", "-c", cmdStr)
	cmd.Dir = s.Workspace

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

	if err == nil {
		if output == "" {
			output = fmt.Sprintf("OK: '%s' completed with exit code 0.", cmdStr)
		}
		return true, output, nil
	}

	if output == "" {
		output = fmt.Sprintf("Command '%s' exited with error: %v", cmdStr, err)
	}
	return false, output, nil
}
