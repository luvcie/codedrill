package main

import (
	"errors"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/referenceview"
)

func sprintDiff(policy string) (compare.Result, error) {
	session, err := visibleReferenceSession(policy)
	if err != nil {
		return compare.Result{}, err
	}
	return compare.Compare(session.Drill(), session.Workspace)
}

func (c *cli) referenceDiff(args []string) error {
	if len(args) != 2 || args[0] != "--session" || args[1] == "" {
		return usage("usage: codedrill reference-diff --session <session-policy>")
	}
	result, err := sprintDiff(args[1])
	if err != nil {
		return err
	}
	if result.Exact {
		c.println("Normalized sources match.")
		return nil
	}
	_, err = tea.NewProgram(referenceview.NewDiff(result), tea.WithAltScreen(), tea.WithMouseCellMotion(), tea.WithOutput(c.out)).Run()
	if err != nil {
		return errors.New("reference diff viewer could not start")
	}
	return nil
}
