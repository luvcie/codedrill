package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/luvcie/codedrill/internal/runner"
)

type referencePane struct {
	ID         uint32 `json:"id"`
	Plugin     bool   `json:"is_plugin"`
	Floating   bool   `json:"is_floating"`
	Suppressed bool   `json:"is_suppressed"`
	Exited     bool   `json:"exited"`
	TabID      uint32 `json:"tab_id"`
	Command    string `json:"terminal_command"`
}

func (p referencePane) paneID() string { return fmt.Sprintf("terminal_%d", p.ID) }

func referencePanes() ([]referencePane, error) {
	data, err := exec.Command("zellij", "action", "list-panes", "--json", "--all").Output()
	if err != nil {
		return nil, errors.New("cannot inspect sprint panes; side-by-side viewing requires Zellij 0.45 or newer")
	}
	var panes []referencePane
	if err := json.Unmarshal(data, &panes); err != nil {
		return nil, errors.New("cannot inspect sprint panes")
	}
	return panes, nil
}

func toggleReferencePane(session runner.ReferenceSession, policy string) error {
	if os.Getenv("ZELLIJ") == "" {
		return errors.New("toggle the reference pane from the sprint's Zellij session")
	}
	// Serialize shortcut and shell requests; pane discovery stays authoritative even
	// when the viewer is closed with q or Zellij's own pane controls.
	lock, err := os.OpenFile(filepath.Join(filepath.Dir(policy), "pane.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return errors.New("cannot lock the reference pane")
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return errors.New("cannot lock the reference pane")
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)

	exe, err := os.Executable()
	if err != nil {
		return errors.New("cannot launch the reference viewer")
	}
	viewerArgs := []string{exe, "reference-view", "--session", policy, "--side-pane"}
	panes, err := referencePanes()
	if err != nil {
		return err
	}
	var editor, terminal, viewer *referencePane
	for i := range panes {
		p := &panes[i]
		if p.Plugin || p.Exited || p.Floating {
			continue
		}
		var slot **referencePane
		switch p.Command {
		case "bash " + filepath.Join(session.Workspace, "start-editor.sh"):
			slot = &editor
		case "bash " + filepath.Join(session.Workspace, "start-terminal.sh"):
			slot = &terminal
		case strings.Join(viewerArgs, " "):
			slot = &viewer
		}
		if slot != nil {
			if *slot != nil {
				return errors.New("sprint panes are ambiguous; no panes were changed")
			}
			*slot = p
		}
	}
	if terminal == nil || editor == nil || terminal.TabID != editor.TabID {
		return errors.New("the original sprint editor and terminal panes are unavailable")
	}
	if viewer != nil {
		if viewer.Suppressed || viewer.TabID != terminal.TabID {
			return errors.New("restore the sprint pane layout before toggling the reference")
		}
		// Restore from the viewer's slot, not the shortcut's temporary floating
		// pane. Let close-on-exit restore the shell through the normal quit path.
		if err := exec.Command("zellij", "action", "focus-pane-id", viewer.paneID()).Run(); err != nil {
			return errors.New("could not focus the reference pane to close it")
		}
		if err := exec.Command("zellij", "action", "send-keys", "--pane-id", viewer.paneID(), "q").Run(); err != nil {
			return errors.New("could not close the reference pane; focus it and press q to return to the terminal")
		}
		if err := waitForReferenceTerminal(*terminal, *viewer); err != nil {
			return err
		}
		if err := exec.Command("zellij", "action", "focus-pane-id", terminal.paneID()).Run(); err != nil {
			return errors.New("terminal restored, but could not focus it")
		}
		return nil
	}
	if terminal.Suppressed || editor.Suppressed {
		return errors.New("restore the sprint pane layout before toggling the reference")
	}
	if _, err := loadViewerFiles(session); err != nil {
		return err
	}
	args := []string{"action", "new-pane", "--in-place", "--pane-id", terminal.paneID(),
		"--no-focus", "--close-on-exit", "--name", "Reference beside editor", "--cwd", session.Workspace, "--"}
	args = append(args, viewerArgs...)
	if err := exec.Command("zellij", args...).Run(); err != nil {
		return errors.New("could not open the reference pane; side-by-side viewing requires Zellij 0.45 or newer")
	}
	if err := exec.Command("zellij", "action", "focus-pane-id", editor.paneID()).Run(); err != nil {
		return errors.New("reference opened, but could not focus the editor")
	}
	return nil
}

func waitForReferenceTerminal(terminal, viewer referencePane) error {
	deadline := time.Now().Add(3 * time.Second)
	retry := time.Now().Add(100 * time.Millisecond)
	for time.Now().Before(deadline) {
		panes, err := referencePanes()
		if err != nil {
			return err
		}
		restored, viewing := false, false
		for _, p := range panes {
			if p.Plugin {
				continue
			}
			if p.ID == terminal.ID && !p.Suppressed {
				restored = true
			}
			if p.ID == viewer.ID {
				viewing = true
			}
		}
		if restored && !viewing {
			return nil
		}
		// A very fast toggle can arrive during the viewer's terminal setup.
		if viewing && time.Now().After(retry) {
			if err := exec.Command("zellij", "action", "send-keys", "--pane-id", viewer.paneID(), "q").Run(); err != nil {
				return errors.New("could not close the reference pane")
			}
			retry = time.Now().Add(100 * time.Millisecond)
		}
		time.Sleep(20 * time.Millisecond)
	}
	return errors.New("reference pane has not closed; focus it and press q to return to the terminal")
}
