//go:build unix

package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTimeoutStopsChildWithOutputPipes(t *testing.T) {
	previous := testTimeout
	testTimeout = 300 * time.Millisecond
	t.Cleanup(func() { testTimeout = previous })
	s := &Session{Workspace: t.TempDir(), EntryFile: "main.sh", TestCommand: "bash -c 'echo child-started; sleep 1; echo leaked > survived'; echo finished"}
	if err := os.WriteFile(filepath.Join(s.Workspace, s.EntryFile), []byte("echo work"), 0644); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	passed, output := s.RunTests()
	if elapsed := time.Since(start); elapsed > 800*time.Millisecond {
		t.Fatalf("timeout waited for child: %s", elapsed)
	}
	if passed || !strings.Contains(output, "Timed out") || !strings.Contains(output, "300ms") || !strings.Contains(output, "child-started") {
		t.Fatalf("lost timeout or partial output: passed=%v output=%q", passed, output)
	}
	time.Sleep(time.Second)
	if _, err := os.Stat(filepath.Join(s.Workspace, "survived")); !os.IsNotExist(err) {
		t.Fatalf("child survived cancellation: %v", err)
	}
}
