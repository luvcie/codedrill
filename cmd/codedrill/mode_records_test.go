package main

import (
	"strings"
	"testing"
	"time"

	"github.com/luvcie/codedrill/internal/storage"
)

func TestCommandsSeparateModeRecords(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	history, err := storage.LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	for category, duration := range map[storage.RecordMode]time.Duration{
		storage.ExamMode: 20 * time.Second, storage.RecallMode: 3 * time.Second,
		storage.StudyMode: time.Second,
	} {
		if _, _, err := history.RecordAttempt("ft_strlen", category, duration, true, 1, false); err != nil {
			t.Fatal(err)
		}
	}
	code, out, errOut := runCLI(t, "list")
	if code != 0 || errOut != "" {
		t.Fatalf("list failed: %d %s", code, errOut)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "ft_strlen") && (!strings.Contains(line, "Exam PB:") || !strings.Contains(line, "20.000s")) {
			t.Fatalf("main drill list uses a practice PB: %s", line)
		}
	}
	for _, command := range []string{"top", "recent", "info", "stats"} {
		args := []string{command, "ft_strlen"}
		if command == "stats" {
			args = args[:1]
		}
		code, out, errOut := runCLI(t, args...)
		if code != 0 || errOut != "" {
			t.Fatalf("%s failed: %d %s", command, code, errOut)
		}
		for _, label := range []string{"Exam", "Recall", "Study"} {
			if !strings.Contains(out, label) {
				t.Fatalf("%s hides the %s category: %s", command, label, out)
			}
		}
		if strings.Contains(out, "Copy") {
			t.Fatalf("%s still lists Copy as a record category: %s", command, out)
		}
		if strings.Index(out, "Exam") > strings.Index(out, "Study") {
			t.Fatalf("%s did not put Exam first: %s", command, out)
		}
	}
}
