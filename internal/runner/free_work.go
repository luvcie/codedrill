package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type FreeWorkSession struct {
	SessionName string
	directory   string
	tempDir     string
}

func FreeWorkDirectory(directory string) (string, error) {
	directory = strings.TrimSpace(directory)
	if directory == "" || directory == "~" || strings.HasPrefix(directory, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if directory == "" || directory == "~" {
			directory = home
		} else {
			directory = filepath.Join(home, directory[2:])
		}
	}
	directory, err := filepath.Abs(directory)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(directory)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s is not a directory", directory)
	}
	return directory, nil
}

func NewFreeWorkSession(panes int, directory string) (_ *FreeWorkSession, err error) {
	if panes < 1 || panes > 4 {
		return nil, fmt.Errorf("choose 1–4 panes")
	}
	directory, err = FreeWorkDirectory(directory)
	if err != nil {
		return nil, err
	}
	if _, err := exec.LookPath("zellij"); err != nil {
		return nil, fmt.Errorf("Free Work needs Zellij: %w", err)
	}
	shell, err := exec.LookPath(DetectShell())
	if err != nil {
		return nil, fmt.Errorf("configured shell: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "codedrill-free-*")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = os.RemoveAll(tempDir)
		}
	}()
	s := &FreeWorkSession{SessionName: filepath.Base(tempDir), directory: directory, tempDir: tempDir}
	pack := sessionTypingPack()
	if typingSoundsFlag(pack) == "1" {
		if err := writeEditorShim(tempDir, DetectEditor(), executablePath(), pack); err != nil {
			return nil, err
		}
	}
	// The absolute shim path survives cd; only the editor goes through the observer.
	body := `export PATH="$PWD/` + editorShimDir + `:$PATH"
export CODEDRILL_MUSIC_SOCKET="$MUSIC_SOCKET"
cd -- "$DIRECTORY" || exit 1
exec "$SHELL_CMD"
`
	if err := os.WriteFile(filepath.Join(tempDir, "shell.sh"), script(body,
		"DIRECTORY", directory, "SHELL_CMD", shell, "MUSIC_SOCKET", s.MusicSocket()), 0700); err != nil {
		return nil, err
	}
	var layout strings.Builder
	layout.WriteString("layout {\n    pane split_direction=\"vertical\" {\n")
	for _, count := range []int{(panes + 1) / 2, panes / 2} {
		if count == 0 {
			continue
		}
		layout.WriteString("        pane split_direction=\"horizontal\" {\n")
		for range count {
			fmt.Fprintf(&layout, "            pane command=\"bash\" { args %s; }\n", strconv.Quote(filepath.Join(tempDir, "shell.sh")))
		}
		layout.WriteString("        }\n")
	}
	layout.WriteString("    }\n}\n")
	if err := os.WriteFile(filepath.Join(tempDir, "layout.kdl"), []byte(layout.String()), 0600); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *FreeWorkSession) Cmd() *exec.Cmd {
	return zellijCmd(s.SessionName, filepath.Join(s.tempDir, "layout.kdl"), s.directory)
}

func (s *FreeWorkSession) MusicSocket() string {
	return filepath.Join(s.tempDir, ".codedrill-music.sock")
}

func (s *FreeWorkSession) Cleanup() {
	if s.SessionName != "" {
		_ = exec.Command("zellij", "kill-session", s.SessionName).Run()
	}
	_ = os.RemoveAll(s.tempDir)
}
