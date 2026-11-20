package runner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/typingsound"
)

func TestFreeWorkPanesAndEditorAfterCD(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	bin := t.TempDir()
	for _, name := range []string{"zellij", "hx"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	log := filepath.Join(t.TempDir(), "editor.log")
	stub := filepath.Join(bin, "codedrill")
	if err := os.WriteFile(stub, []byte("#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\" >> "+shellQuote(log)+"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	previous := executablePath
	executablePath = func() string { return stub }
	t.Cleanup(func() { executablePath = previous })
	on := true
	if err := storage.SaveConfig(storage.Config{Editor: "hx", Shell: "bash", Typing: &storage.TypingSettings{Enabled: &on, Pack: typingsound.PrototypePackID}}); err != nil {
		t.Fatal(err)
	}
	project := filepath.Join(t.TempDir(), "project with spaces 'quoted'")
	nested := filepath.Join(project, "nested dir")
	if err := os.MkdirAll(nested, 0700); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"layout.kdl", "shell.sh", ".codedrill-editor", "keep.txt"} {
		if err := os.WriteFile(filepath.Join(project, name), []byte("user file"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	for panes := 1; panes <= 4; panes++ {
		t.Run(fmt.Sprint(panes), func(t *testing.T) {
			s, err := NewFreeWorkSession(panes, project)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(s.Cleanup)
			layout, err := os.ReadFile(filepath.Join(s.tempDir, "layout.kdl"))
			if err != nil || strings.Count(string(layout), `pane command="bash"`) != panes {
				t.Fatalf("wrong pane count: %s, %v", layout, err)
			}
			if cmd := s.Cmd(); cmd.Dir != project || !strings.Contains(strings.Join(cmd.Args, " "), s.SessionName) {
				t.Fatalf("wrong session command: %v (dir %q)", cmd.Args, cmd.Dir)
			}
			cmd := exec.Command("bash", filepath.Join(s.tempDir, "shell.sh"))
			cmd.Dir = project
			cmd.Stdin = strings.NewReader("pwd\ncd 'nested dir'\nhx 'file with spaces.go'\nexit\n")
			output, err := cmd.CombinedOutput()
			if err != nil || !strings.Contains(string(output), project) {
				t.Fatalf("shell: %s, %v", output, err)
			}
			s.Cleanup()
			if _, err := os.Stat(s.tempDir); !os.IsNotExist(err) {
				t.Fatalf("temporary files survived: %v", err)
			}
		})
	}
	data, err := os.ReadFile(log)
	want := nested + "\n__typing-sound-editor\n" + filepath.Join(bin, "hx") + "\nfile with spaces.go\n"
	if err != nil || string(data) != strings.Repeat(want, 4) {
		t.Fatalf("editor wrapper after cd: %q, %v", data, err)
	}
	for _, name := range []string{"layout.kdl", "shell.sh", ".codedrill-editor", "keep.txt"} {
		if data, err := os.ReadFile(filepath.Join(project, name)); err != nil || string(data) != "user file" {
			t.Fatalf("project file %s changed: %q, %v", name, data, err)
		}
	}
}

func TestFreeWorkDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	for _, value := range []string{"", "  ", "~", "~/"} {
		got, err := FreeWorkDirectory(value)
		if err != nil || got != home {
			t.Errorf("%q: got %q, %v; want home %q", value, got, err, home)
		}
	}
	file := filepath.Join(home, "file")
	if err := os.WriteFile(file, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{file, filepath.Join(home, "missing")} {
		if _, err := FreeWorkDirectory(value); err == nil {
			t.Errorf("accepted invalid directory %q", value)
		}
	}
	for _, panes := range []int{0, 5} {
		if _, err := NewFreeWorkSession(panes, ""); err == nil {
			t.Errorf("accepted %d panes", panes)
		}
	}
}
