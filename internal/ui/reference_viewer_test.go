package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/runner"
)

func TestViewerUsesRunModeSnapshot(t *testing.T) {
	m := newBriefingModeModel(t, "ft_strlen")
	m.runMode = mode.Copy()
	m.briefingMode = mode.Recall()
	if err := mode.Save("ft_strlen", mode.Exam()); err != nil {
		t.Fatal(err)
	}
	m.launchSprintSession()
	if m.session == nil {
		t.Fatalf("session did not start: %v", m.setupErr)
	}
	t.Cleanup(m.session.Cleanup)
	m.runMode = mode.Exam()
	helper, err := os.ReadFile(filepath.Join(m.session.Workspace, "match"))
	if err != nil {
		t.Fatal(err)
	}
	var path string
	for _, line := range strings.Split(string(helper), "\n") {
		if strings.HasPrefix(line, "POLICY='") {
			path = strings.TrimSuffix(strings.TrimPrefix(line, "POLICY='"), "'")
		}
	}
	policy, err := runner.LoadReferenceSession(path)
	if err != nil || *policy.Options != mode.Copy() {
		t.Fatalf("viewer did not capture runMode: %+v, %v", policy, err)
	}
}
