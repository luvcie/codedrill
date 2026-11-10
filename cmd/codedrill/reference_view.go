package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/reference"
	"github.com/luvcie/codedrill/internal/referenceview"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/solutions"
)

var errReferenceHidden = errors.New("reference is hidden for this sprint")

func visibleReferenceSession(path string) (runner.ReferenceSession, error) {
	session, err := runner.LoadReferenceSession(path)
	if err != nil {
		return runner.ReferenceSession{}, err
	}
	if !session.Options.ShowReference {
		return runner.ReferenceSession{}, errReferenceHidden
	}
	return session, nil
}

func loadViewerFiles(session runner.ReferenceSession) (map[string][]byte, error) {
	if !reference.SupportedLanguage(session.Language) {
		return nil, errors.New("the reference viewer currently supports C and C++ drills")
	}
	source, err := solutions.Resolve(session.DrillID, session.DrillDir)
	if err != nil || source == nil || len(source.Files) == 0 {
		return nil, errors.New("reference files are unavailable for this sprint")
	}
	files := make(map[string][]byte)
	for _, file := range source.Files {
		switch ext := strings.ToLower(filepath.Ext(file.Rel)); ext {
		case ".c", ".h", ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
			if strings.EqualFold(strings.TrimSpace(session.Language), "c") && ext != ".c" && ext != ".h" {
				continue
			}
			content, err := file.Read()
			if err != nil {
				return nil, errors.New("reference files are unavailable for this sprint")
			}
			files[file.Rel] = content
		}
	}
	if len(files) == 0 {
		return nil, errors.New("no reference source files are available for this sprint")
	}
	return files, nil
}

func (c *cli) referenceView(args []string) error {
	var sessionPath string
	open := false
	togglePane, sidePane := false, false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--session":
			if i+1 >= len(args) {
				return runner.ErrReferencePolicy
			}
			i++
			sessionPath = args[i]
		case "--open":
			open = true
		case "--toggle-pane":
			togglePane = true
		case "--side-pane":
			sidePane = true
		default:
			return usage("usage: codedrill reference-view --session <session-policy> [--open | --toggle-pane]")
		}
	}
	if (open && togglePane) || (sidePane && (open || togglePane)) {
		return usage("choose one reference viewer action")
	}
	session, err := visibleReferenceSession(sessionPath)
	if err != nil {
		return err
	}
	if togglePane {
		return toggleReferencePane(session, sessionPath)
	}
	files, err := loadViewerFiles(session)
	if err != nil {
		return err
	}
	if open {
		if os.Getenv("ZELLIJ") == "" {
			return errors.New("open the reference viewer from the sprint's Zellij terminal")
		}
		exe, err := os.Executable()
		if err != nil {
			return errors.New("cannot launch the reference viewer")
		}
		cmd := exec.Command("zellij", "action", "new-pane", "--floating", "--close-on-exit", "--name", "Reference", "--cwd", session.Workspace,
			"--", exe, "reference-view", "--session", sessionPath)
		if err := cmd.Run(); err != nil {
			return errors.New("could not open the floating reference pane")
		}
		return nil
	}
	score := ""
	if session.Options.CheckReference {
		if result, err := compare.Compare(session.Drill(), session.Workspace); err == nil {
			score = fmt.Sprintf("%.1f%%", result.Percent())
		}
	}
	_, err = tea.NewProgram(referenceview.New(files, score), tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithOutput(c.out)).Run()
	if err != nil {
		return errors.New("reference viewer could not start")
	}
	return nil
}
