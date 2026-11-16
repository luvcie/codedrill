package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func TestModePersonalBestsAreIndependent(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	s, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		category RecordMode
		time     time.Duration
	}{
		{ExamMode, 20 * time.Second}, {RecallMode, 10 * time.Second},
		{CopyMode, 5 * time.Second}, {StudyMode, time.Second},
	} {
		pb, delta, err := s.RecordAttempt("drill", tc.category, tc.time, true, 1, false)
		if err != nil || !pb || delta != 0 {
			t.Fatalf("first %s run: PB=%v delta=%v error=%v", tc.category, pb, delta, err)
		}
		if got := s.GetPB("drill", tc.category); got != tc.time {
			t.Fatalf("%s PB = %v, want %v", tc.category, got, tc.time)
		}
	}
	if pb, delta, err := s.RecordAttempt("drill", ExamMode, 15*time.Second, true, 1, false); err != nil || !pb || delta != 5*time.Second {
		t.Fatalf("Exam improvement compared with another mode: %v %v %v", pb, delta, err)
	}
	if pb, delta, err := s.RecordAttempt("drill", ExamMode, time.Millisecond, false, 1, false); err != nil || pb || delta != 0 {
		t.Fatalf("failed Exam attempt set a record: %v %v %v", pb, delta, err)
	}
	reloaded, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	for category, want := range map[RecordMode]time.Duration{ExamMode: 15 * time.Second, RecallMode: 10 * time.Second, CopyMode: 5 * time.Second, StudyMode: time.Second} {
		if got := reloaded.GetPB("drill", category); got != want {
			t.Fatalf("reloaded %s PB = %v, want %v", category, got, want)
		}
		for _, attempt := range reloaded.GetTopAttempts("drill", category, 10) {
			if attempt.Mode != category || !attempt.Passed {
				t.Fatalf("%s leaderboard includes %+v", category, attempt)
			}
		}
	}
}

func TestOldHistoryDoesNotSetModeRecords(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := GetHistoryPath()
	if err != nil {
		t.Fatal(err)
	}
	const old = `{"attempts":[{"drill_id":"old","duration":2000000000,"timestamp":"2026-01-01T12:00:00Z","passed":true,"attempts":1}],"stats":{"old":{"drill_id":"old","personal_best":2000000000,"total_runs":1,"successful":1}}}`
	if err := os.WriteFile(path, []byte(old), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	wantModes := []RecordMode{ExamMode, RecallMode, CopyMode, StudyMode}
	if got := s.RecordModes("old"); !slices.Equal(got, wantModes) {
		t.Fatalf("old runs added a record category: %v", got)
	}
	for _, category := range wantModes {
		if got := s.GetPB("old", category); got != 0 {
			t.Fatalf("old PB counted as a %s record: %v", category, got)
		}
	}
	if s.GetModeStats("old", "") != nil {
		t.Fatal("old runs created an unnamed record category")
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != old {
		t.Fatal("loading history rewrote the user's file")
	}
	if pb, delta, err := s.RecordAttempt("old", ExamMode, 20*time.Second, true, 1, false); err != nil || !pb || delta != 0 {
		t.Fatalf("first Exam run competed with old PB: %v %v %v", pb, delta, err)
	}
	reloaded, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Attempts) != 2 || reloaded.GetPB("old", ExamMode) != 20*time.Second || reloaded.GetStats("old").TotalRuns != 2 {
		t.Fatalf("save changed practice history or the Exam PB: %+v", reloaded)
	}
	if data, err := os.ReadFile(path); err != nil || bytes.Contains(data, []byte("personal_best")) {
		t.Fatalf("save retained the old shared PB: %s, %v", data, err)
	}
	if err := reloaded.ResetDrillStats("old"); err != nil {
		t.Fatal(err)
	}
	if reloaded.GetPB("old", ExamMode) != 0 || len(reloaded.Attempts) != 0 {
		t.Fatal("reset kept a category's records")
	}
}

func TestOldSummaryDoesNotSetModeRecords(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := GetHistoryPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"stats":{"old":{"drill_id":"old","personal_best":1000000000,"total_runs":2,"successful":2}}}`), 0644); err != nil {
		t.Fatal(err)
	}
	s, err := LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if got := s.RecordModes("old"); len(got) != 4 || s.GetModeStats("old", "") != nil {
		t.Fatalf("old summary added a record category: %v", got)
	}
	if pb, delta, err := s.RecordAttempt("old", ExamMode, 5*time.Second, true, 1, false); err != nil || !pb || delta != 0 {
		t.Fatalf("first Exam run competed with old summary: %v %v %v", pb, delta, err)
	}
	s, err = LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if s.GetPB("old", ExamMode) != 5*time.Second || s.GetStats("old").TotalRuns != 3 {
		t.Fatalf("save changed the Exam PB or practice totals: %+v", s)
	}
}

func TestModeSaveFailureDoesNotPublishRecord(t *testing.T) {
	s, path := newTestStore(t)
	if _, _, err := s.RecordAttempt("drill", ExamMode, 20*time.Second, true, 1, false); err != nil {
		t.Fatal(err)
	}
	backup := filepath.Join(t.TempDir(), "previous.json")
	if err := os.Rename(path, backup); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	if pb, _, err := s.RecordAttempt("drill", CopyMode, time.Second, true, 1, false); err == nil || pb {
		t.Fatal("failed save published a Copy PB")
	}
	if s.GetPB("drill", CopyMode) != 0 || s.GetPB("drill", ExamMode) != 20*time.Second || len(s.Attempts) != 1 {
		t.Fatal("failed save changed mode records")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(backup, path); err != nil {
		t.Fatal(err)
	}
	if pb, delta, err := s.RecordAttempt("drill", CopyMode, time.Second, true, 1, false); err != nil || !pb || delta != 0 || len(s.Attempts) != 2 {
		t.Fatalf("retry did not record exactly one first Copy PB: %v %v %v", pb, delta, err)
	}
}
