package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestStore(t *testing.T) (*HistoryStore, string) {
	t.Helper()
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "history.json")
	store := &HistoryStore{
		filePath: path,
		Attempts: []Attempt{},
		Stats:    make(map[string]*DrillStats),
	}
	return store, path
}

func TestStorageRecordAttempt(t *testing.T) {
	store, path := newTestStore(t)

	isPB, delta, err := store.RecordAttempt("drill-1", 5*time.Second, true, 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isPB {
		t.Errorf("expected first successful run to be PB")
	}
	if delta != 0 {
		t.Errorf("expected delta 0 on first run, got %v", delta)
	}
	if store.GetPB("drill-1") != 5*time.Second {
		t.Errorf("expected PB to be 5s, got %v", store.GetPB("drill-1"))
	}

	isPB, delta, err = store.RecordAttempt("drill-1", 3*time.Second, true, 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isPB {
		t.Errorf("expected faster run to be PB")
	}
	if delta != 2*time.Second {
		t.Errorf("expected delta of 2s, got %v", delta)
	}
	if store.GetPB("drill-1") != 3*time.Second {
		t.Errorf("expected PB to be 3s, got %v", store.GetPB("drill-1"))
	}

	isPB, delta, err = store.RecordAttempt("drill-1", 4*time.Second, true, 1, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isPB {
		t.Errorf("slower run should not be PB")
	}
	if delta != -1*time.Second {
		t.Errorf("expected delta -1s, got %v", delta)
	}

	isPB, _, err = store.RecordAttempt("drill-1", 1*time.Second, false, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if isPB {
		t.Errorf("failed run should not be PB")
	}
	if store.GetPB("drill-1") != 3*time.Second {
		t.Errorf("expected PB to remain 3s, got %v", store.GetPB("drill-1"))
	}
	if store.CurrentStreak != 0 {
		t.Errorf("expected current streak 0 after failure, got %d", store.CurrentStreak)
	}
	if store.BestStreak != 3 {
		t.Errorf("expected best streak 3, got %d", store.BestStreak)
	}

	_, _, _ = store.RecordAttempt("drill-1", 2*time.Second, true, 1, false)
	if store.CurrentStreak != 1 {
		t.Errorf("expected current streak 1, got %d", store.CurrentStreak)
	}
	if store.BestStreak != 3 {
		t.Errorf("expected best streak 3, got %d", store.BestStreak)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("history file was not written: %v", err)
	}

	loadedData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read written history file: %v", err)
	}
	var loadedStore HistoryStore
	if err := json.Unmarshal(loadedData, &loadedStore); err != nil {
		t.Fatalf("failed to unmarshal saved json: %v", err)
	}
	if len(loadedStore.Attempts) != 5 {
		t.Errorf("expected 5 attempts in saved file, got %d", len(loadedStore.Attempts))
	}
	if loadedStore.CurrentStreak != 1 {
		t.Errorf("expected loaded current streak 1, got %d", loadedStore.CurrentStreak)
	}
	if loadedStore.BestStreak != 3 {
		t.Errorf("expected loaded best streak 3, got %d", loadedStore.BestStreak)
	}
}

func TestStorageMetricsAndStats(t *testing.T) {
	store, _ := newTestStore(t)

	if store.GetTotalRuns() != 0 {
		t.Errorf("expected 0 total runs")
	}
	if store.GetOverallWinRate() != 0 {
		t.Errorf("expected 0 win rate")
	}
	if store.ClearedCount([]string{"d1", "d2"}) != 0 {
		t.Errorf("expected 0 cleared")
	}

	_, _, _ = store.RecordAttempt("d1", 2*time.Second, true, 1, false)
	_, _, _ = store.RecordAttempt("d1", 3*time.Second, false, 1, false)
	_, _, _ = store.RecordAttempt("d2", 4*time.Second, true, 1, false)

	if store.GetTotalRuns() != 3 {
		t.Errorf("expected 3 total runs, got %d", store.GetTotalRuns())
	}
	if store.GetPassedRuns() != 2 {
		t.Errorf("expected 2 passed runs, got %d", store.GetPassedRuns())
	}
	if winRate := store.GetOverallWinRate(); winRate != 66 {
		t.Errorf("expected 66%% win rate, got %d%%", winRate)
	}
	if n := store.ClearedCount([]string{"d1", "d2", "gone"}); n != 2 {
		t.Errorf("expected 2 cleared drills, got %d", n)
	}
	if n := store.ClearedCount([]string{"d1"}); n != 1 {
		t.Errorf("cleared count should only consider the listed drills, got %d", n)
	}
	if store.GetTotalPracticeTime() != 9*time.Second {
		t.Errorf("expected 9s total practice time, got %v", store.GetTotalPracticeTime())
	}
	if runs, passed := store.RunsToday(); runs != 3 || passed != 2 {
		t.Errorf("expected 3 runs and 2 passes today, got %d and %d", runs, passed)
	}

	top := store.GetTopAttempts("d1", 5)
	if len(top) != 1 {
		t.Fatalf("expected 1 passed attempt for d1, got %d", len(top))
	}
	if top[0].Duration != 2*time.Second {
		t.Errorf("expected top attempt duration 2s, got %v", top[0].Duration)
	}

	recent := store.GetRecentAttempts("d1", 5)
	if len(recent) != 2 {
		t.Fatalf("expected 2 attempts for d1, got %d", len(recent))
	}
	if recent[0].Passed != false {
		t.Errorf("expected most recent attempt to be the failed one")
	}

	if err := store.ResetDrillStats("d1"); err != nil {
		t.Fatalf("failed to reset drill stats: %v", err)
	}
	if store.GetPB("d1") != 0 {
		t.Errorf("expected d1 PB to be 0 after reset, got %v", store.GetPB("d1"))
	}
	if store.GetStats("d1") != nil {
		t.Errorf("expected d1 stats to be nil after reset")
	}
	if store.GetTotalRuns() != 1 {
		t.Errorf("expected only 1 remaining run (for d2), got %d", store.GetTotalRuns())
	}
}

func TestConfig(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	cfg, err := LoadConfig()
	if err != nil || !cfg.IsSoundEnabled() {
		t.Errorf("expected sound to be enabled by default")
	}

	soundOff := false
	customCfg := Config{
		Editor: "nvim",
		Sound:  &soundOff,
	}
	if err := SaveConfig(customCfg); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	reloaded, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Editor != "nvim" {
		t.Errorf("expected editor nvim, got %s", reloaded.Editor)
	}
	if reloaded.IsSoundEnabled() {
		t.Errorf("expected sound to be disabled")
	}
	if reloaded.GetCountdown() != 3 {
		t.Errorf("expected default countdown 3, got %d", reloaded.GetCountdown())
	}

	zero := 0
	reloaded.Countdown = &zero
	if reloaded.GetCountdown() != 0 {
		t.Errorf("expected countdown 0, got %d", reloaded.GetCountdown())
	}
	five := 5
	reloaded.Countdown = &five
	if reloaded.GetCountdown() != 5 {
		t.Errorf("expected countdown 5, got %d", reloaded.GetCountdown())
	}
	huge := 500
	reloaded.Countdown = &huge
	if reloaded.GetCountdown() != MaxCountdown {
		t.Errorf("expected countdown capped at %d, got %d", MaxCountdown, reloaded.GetCountdown())
	}
}

func TestCorruptConfigIsReported(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := GetConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{nope"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(); err == nil {
		t.Fatal("expected an error for invalid config JSON")
	}
}

func TestAtomicWriteAndCorruptFileHandling(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "history.json")

	testData := []byte(`{"attempts":[],"stats":{}}`)
	if err := AtomicWriteFile(path, testData, 0644); err != nil {
		t.Fatalf("AtomicWriteFile failed: %v", err)
	}

	readData, err := os.ReadFile(path)
	if err != nil || string(readData) != string(testData) {
		t.Fatalf("file content mismatch after atomic write: %v", err)
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("failed to read temp dir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "history.json" {
		t.Errorf("expected only history.json, found: %v", entries)
	}

	corruptContent := []byte(`{this is invalid json bad data}`)
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	appDir := filepath.Join(tempDir, "codedrill")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "history.json"), corruptContent, 0644); err != nil {
		t.Fatal(err)
	}

	store, err := LoadHistory()
	if err == nil || store != nil {
		t.Fatalf("corrupt history must fail to load, got store %v err %v", store, err)
	}

	appEntries, _ := os.ReadDir(appDir)
	foundBackup := false
	for _, e := range appEntries {
		if strings.HasPrefix(e.Name(), "history.json.corrupt") {
			foundBackup = true
			backupData, _ := os.ReadFile(filepath.Join(appDir, e.Name()))
			if string(backupData) != string(corruptContent) {
				t.Errorf("backup content did not match corrupted content")
			}
		}
	}
	if !foundBackup {
		t.Errorf("expected backup of corrupt file, but none was found in %v", appEntries)
	}
}

func TestDailyStreak(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	store, err := LoadHistory()
	if err != nil {
		t.Fatalf("failed to load history: %v", err)
	}

	if streak := store.GetDailyStreak(); streak != 0 {
		t.Errorf("expected daily streak 0 for empty history, got %d", streak)
	}
	if best := store.GetBestDailyStreak(); best != 0 {
		t.Errorf("expected best daily streak 0 for empty history, got %d", best)
	}

	now := time.Now()
	store.Attempts = append(store.Attempts, Attempt{
		DrillID:   "ft_strlen",
		Duration:  3 * time.Second,
		Timestamp: now,
		Passed:    true,
	})
	if streak := store.GetDailyStreak(); streak != 1 {
		t.Errorf("expected daily streak 1 for today run, got %d", streak)
	}
	if best := store.GetBestDailyStreak(); best != 1 {
		t.Errorf("expected best daily streak 1, got %d", best)
	}

	store.Attempts = append(store.Attempts, Attempt{
		DrillID:   "ft_strlen",
		Duration:  2 * time.Second,
		Timestamp: now,
		Passed:    true,
	})
	if streak := store.GetDailyStreak(); streak != 1 {
		t.Errorf("expected daily streak 1 with multiple runs today, got %d", streak)
	}

	yesterday := now.AddDate(0, 0, -1)
	store.Attempts = append(store.Attempts, Attempt{
		DrillID:   "ft_strlen",
		Duration:  4 * time.Second,
		Timestamp: yesterday,
		Passed:    true,
	})
	if streak := store.GetDailyStreak(); streak != 2 {
		t.Errorf("expected daily streak 2 for today and yesterday, got %d", streak)
	}
	if best := store.GetBestDailyStreak(); best != 2 {
		t.Errorf("expected best daily streak 2, got %d", best)
	}

	twoDaysAgo := now.AddDate(0, 0, -2)
	store.Attempts = append(store.Attempts, Attempt{
		DrillID:   "ft_strlen",
		Duration:  5 * time.Second,
		Timestamp: twoDaysAgo,
		Passed:    true,
	})
	if streak := store.GetDailyStreak(); streak != 3 {
		t.Errorf("expected daily streak 3, got %d", streak)
	}
	if best := store.GetBestDailyStreak(); best != 3 {
		t.Errorf("expected best daily streak 3, got %d", best)
	}

	storeWithoutToday := &HistoryStore{
		Attempts: []Attempt{
			{Timestamp: yesterday, Passed: true},
			{Timestamp: twoDaysAgo, Passed: true},
		},
		Stats: make(map[string]*DrillStats),
	}
	if streak := storeWithoutToday.GetDailyStreak(); streak != 2 {
		t.Errorf("expected pending active streak 2 from yesterday, got %d", streak)
	}

	storeBroken := &HistoryStore{
		Attempts: []Attempt{
			{Timestamp: twoDaysAgo, Passed: true},
		},
		Stats: make(map[string]*DrillStats),
	}
	if streak := storeBroken.GetDailyStreak(); streak != 0 {
		t.Errorf("expected streak 0 when missing yesterday and today, got %d", streak)
	}
	if best := storeBroken.GetBestDailyStreak(); best != 1 {
		t.Errorf("expected best daily streak 1 for single historical day, got %d", best)
	}

	historicalStreakStore := &HistoryStore{
		Attempts: []Attempt{
			{Timestamp: now.AddDate(0, 0, -10), Passed: true},
			{Timestamp: now.AddDate(0, 0, -9), Passed: true},
			{Timestamp: now.AddDate(0, 0, -8), Passed: true},
			{Timestamp: now.AddDate(0, 0, -7), Passed: true},
			{Timestamp: now, Passed: true},
		},
		Stats: make(map[string]*DrillStats),
	}
	if streak := historicalStreakStore.GetDailyStreak(); streak != 1 {
		t.Errorf("expected current daily streak 1, got %d", streak)
	}
	if best := historicalStreakStore.GetBestDailyStreak(); best != 4 {
		t.Errorf("expected best daily streak 4, got %d", best)
	}
}
