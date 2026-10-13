package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecordAttemptSaveFailureCanRetry(t *testing.T) {
	for _, passed := range []bool{true, false} {
		for _, id := range []string{"existing", "new"} {
			t.Run(id+map[bool]string{true: "/pass", false: "/forfeit"}[passed], func(t *testing.T) {
				s, path := newTestStore(t)
				if _, _, err := s.RecordAttempt("existing", 10*time.Second, true, 1, false); err != nil {
					t.Fatal(err)
				}
				before, _ := json.Marshal(s)
				oldStats := s.Stats["existing"]
				backup := path + ".previous"
				if err := os.Rename(path, backup); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(path, 0755); err != nil {
					t.Fatal(err)
				}
				for range 2 {
					pb, delta, err := s.RecordAttempt(id, 5*time.Second, passed, 2, true)
					if err == nil || pb || delta != 0 {
						t.Fatal("failed write reported a recorded result")
					}
					after, _ := json.Marshal(s)
					if string(after) != string(before) || oldStats.TotalRuns != 1 {
						t.Fatal("failed save changed history, stats, streaks or XP inputs")
					}
					leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".history.json-*.tmp"))
					if len(leftovers) != 0 {
						t.Fatal("failed save left temporary files")
					}
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(backup, path); err != nil {
					t.Fatal(err)
				}
				if _, _, err := s.RecordAttempt(id, 5*time.Second, passed, 2, true); err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var disk HistoryStore
				if err := json.Unmarshal(data, &disk); err != nil {
					t.Fatal(err)
				}
				if len(disk.Attempts) != 2 || disk.GetTotalXP() != s.GetTotalXP() || disk.GetPB(id) != s.GetPB(id) {
					t.Fatal("retry did not persist exactly one new attempt")
				}
			})
		}
	}
}
