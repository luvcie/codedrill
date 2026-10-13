package storage

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"time"
)

type Attempt struct {
	DrillID   string        `json:"drill_id"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Passed    bool          `json:"passed"`
	Attempts  int           `json:"attempts"`
	IsDaily   bool          `json:"is_daily,omitempty"`
}

type DrillStats struct {
	DrillID       string        `json:"drill_id"`
	PersonalBest  time.Duration `json:"personal_best"`
	TotalRuns     int           `json:"total_runs"`
	Successful    int           `json:"successful"`
	CurrentStreak int           `json:"current_streak,omitempty"`
	BestStreak    int           `json:"best_streak,omitempty"`
	LastAttempt   time.Time     `json:"last_attempt"`
}

type HistoryStore struct {
	filePath      string
	Attempts      []Attempt              `json:"attempts"`
	Stats         map[string]*DrillStats `json:"stats"`
	CurrentStreak int                    `json:"current_streak,omitempty"`
	BestStreak    int                    `json:"best_streak,omitempty"`
}

func GetHistoryPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

func LoadHistory() (*HistoryStore, error) {
	path, err := GetHistoryPath()
	if err != nil {
		return nil, err
	}

	store := &HistoryStore{
		filePath: path,
		Attempts: []Attempt{},
		Stats:    make(map[string]*DrillStats),
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}

	if len(data) == 0 {
		return store, nil
	}

	if err := json.Unmarshal(data, store); err != nil {
		// Keep a copy before anything can save over the file.
		backupPath := fmt.Sprintf("%s.corrupt-%s", path, time.Now().Format("20060102-150405"))
		if werr := os.WriteFile(backupPath, data, 0644); werr != nil {
			return nil, fmt.Errorf("%s is corrupt (%v) and could not be backed up: %w", path, err, werr)
		}
		return nil, fmt.Errorf("%s is corrupt and was copied to %s; move it aside to start fresh: %w", path, filepath.Base(backupPath), err)
	}
	if store.Stats == nil {
		store.Stats = make(map[string]*DrillStats)
	}
	// Files written before streaks were tracked have none stored.
	if store.BestStreak == 0 && len(store.Attempts) > 0 {
		store.recomputeStreaks()
	}
	return store, nil
}

func (s *HistoryStore) recomputeStreaks() {
	cur := 0
	best := 0
	for _, a := range s.Attempts {
		if a.Passed {
			cur++
			if cur > best {
				best = cur
			}
		} else {
			cur = 0
		}
	}
	s.CurrentStreak = cur
	s.BestStreak = best

	for drillID, stat := range s.Stats {
		dCur := 0
		dBest := 0
		for _, a := range s.Attempts {
			if a.DrillID == drillID {
				if a.Passed {
					dCur++
					if dCur > dBest {
						dBest = dCur
					}
				} else {
					dCur = 0
				}
			}
		}
		stat.CurrentStreak = dCur
		stat.BestStreak = dBest
	}
}

// RecordAttempt saves a finished sprint. isDaily marks the first clear of the
// day's daily drill, which earns bonus XP.
func (s *HistoryStore) RecordAttempt(drillID string, duration time.Duration, passed bool, submitCount int, isDaily bool) (isPB bool, delta time.Duration, err error) {
	// Publish progress only after saving, so a failed write can be retried.
	original := s
	next := *s
	next.Attempts = slices.Clone(s.Attempts)
	next.Stats = maps.Clone(s.Stats)
	s = &next
	att := Attempt{
		DrillID:   drillID,
		Duration:  duration,
		Timestamp: time.Now(),
		Passed:    passed,
		Attempts:  submitCount,
		IsDaily:   isDaily,
	}
	s.Attempts = append(s.Attempts, att)

	stat, exists := s.Stats[drillID]
	if !exists {
		stat = &DrillStats{
			DrillID: drillID,
		}
		s.Stats[drillID] = stat
	} else {
		updated := *stat
		stat = &updated
		s.Stats[drillID] = stat
	}

	stat.TotalRuns++
	stat.LastAttempt = att.Timestamp

	if passed {
		s.CurrentStreak++
		if s.CurrentStreak > s.BestStreak {
			s.BestStreak = s.CurrentStreak
		}
		stat.CurrentStreak++
		if stat.CurrentStreak > stat.BestStreak {
			stat.BestStreak = stat.CurrentStreak
		}
		stat.Successful++
		if stat.PersonalBest == 0 || duration < stat.PersonalBest {
			isPB = true
			if stat.PersonalBest != 0 {
				delta = stat.PersonalBest - duration
			}
			stat.PersonalBest = duration
		} else {
			delta = stat.PersonalBest - duration
		}
	} else {
		s.CurrentStreak = 0
		stat.CurrentStreak = 0
	}

	if err := s.Save(); err != nil {
		return false, 0, err
	}
	*original = next
	return isPB, delta, nil
}

func (s *HistoryStore) IsDailyDrillPassedToday(drillID string, t time.Time) bool {
	if s == nil || drillID == "" {
		return false
	}
	todayStr := t.Local().Format("2006-01-02")
	for _, a := range s.Attempts {
		if a.DrillID == drillID && a.Passed && a.Timestamp.Local().Format("2006-01-02") == todayStr {
			return true
		}
	}
	return false
}

func (s *HistoryStore) GetPB(drillID string) time.Duration {
	if stat, exists := s.Stats[drillID]; exists {
		return stat.PersonalBest
	}
	return 0
}

func (s *HistoryStore) GetStats(drillID string) *DrillStats {
	if stat, exists := s.Stats[drillID]; exists {
		return stat
	}
	return nil
}

func (s *HistoryStore) GetTopAttempts(drillID string, limit int) []Attempt {
	var passed []Attempt
	for _, a := range s.Attempts {
		if a.DrillID == drillID && a.Passed {
			passed = append(passed, a)
		}
	}

	slices.SortStableFunc(passed, func(a, b Attempt) int { return int(a.Duration - b.Duration) })

	if len(passed) > limit {
		return passed[:limit]
	}
	return passed
}

func (s *HistoryStore) GetRecentAttempts(drillID string, limit int) []Attempt {
	var list []Attempt
	for i := len(s.Attempts) - 1; i >= 0; i-- {
		if s.Attempts[i].DrillID == drillID {
			list = append(list, s.Attempts[i])
			if len(list) >= limit {
				break
			}
		}
	}
	return list
}

func (s *HistoryStore) GetTotalPracticeTime() time.Duration {
	var total time.Duration
	for _, a := range s.Attempts {
		total += a.Duration
	}
	return total
}

func (s *HistoryStore) GetTotalRuns() int {
	return len(s.Attempts)
}

func (s *HistoryStore) GetPassedRuns() int {
	passed := 0
	for _, a := range s.Attempts {
		if a.Passed {
			passed++
		}
	}
	return passed
}

func (s *HistoryStore) GetOverallWinRate() int {
	total := len(s.Attempts)
	if total == 0 {
		return 0
	}
	return (s.GetPassedRuns() * 100) / total
}

// ClearedCount counts the given drills with at least one passing run.
func (s *HistoryStore) ClearedCount(drillIDs []string) int {
	n := 0
	for _, id := range drillIDs {
		if stat := s.Stats[id]; stat != nil && stat.Successful > 0 {
			n++
		}
	}
	return n
}

// RunsToday counts today's runs, and how many of them passed.
func (s *HistoryStore) RunsToday() (runs, passed int) {
	today := time.Now().Local().Format("2006-01-02")
	for _, a := range s.Attempts {
		if a.Timestamp.Local().Format("2006-01-02") == today {
			runs++
			if a.Passed {
				passed++
			}
		}
	}
	return runs, passed
}

// GetDailyStreak counts consecutive practice days ending today or yesterday.
func (s *HistoryStore) GetDailyStreak() int {
	if len(s.Attempts) == 0 {
		return 0
	}
	daySet := make(map[string]bool)
	for _, a := range s.Attempts {
		daySet[a.Timestamp.Local().Format("2006-01-02")] = true
	}

	today := time.Now().Local()
	todayStr := today.Format("2006-01-02")
	yesterday := today.AddDate(0, 0, -1)
	yesterdayStr := yesterday.Format("2006-01-02")

	var checkDate time.Time
	streak := 0
	if daySet[todayStr] {
		streak = 1
		checkDate = yesterday
	} else if daySet[yesterdayStr] {
		streak = 1
		checkDate = yesterday.AddDate(0, 0, -1)
	} else {
		return 0
	}

	for {
		dateStr := checkDate.Format("2006-01-02")
		if daySet[dateStr] {
			streak++
			checkDate = checkDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}
	return streak
}

func (s *HistoryStore) GetBestDailyStreak() int {
	if len(s.Attempts) == 0 {
		return 0
	}
	dayMap := make(map[string]bool)
	for _, a := range s.Attempts {
		dayMap[a.Timestamp.Local().Format("2006-01-02")] = true
	}

	var days []time.Time
	for dStr := range dayMap {
		t, err := time.Parse("2006-01-02", dStr)
		if err == nil {
			days = append(days, t)
		}
	}

	if len(days) == 0 {
		return 0
	}

	slices.SortFunc(days, func(a, b time.Time) int { return a.Compare(b) })

	best := 1
	current := 1
	for i := 1; i < len(days); i++ {
		if days[i].Equal(days[i-1].AddDate(0, 0, 1)) {
			current++
			if current > best {
				best = current
			}
		} else {
			current = 1
		}
	}
	return best
}

// ResetDrillStats removes a drill's runs. Like RecordAttempt, it only changes
// the store once the save succeeded.
func (s *HistoryStore) ResetDrillStats(drillID string) error {
	next := *s
	next.Stats = maps.Clone(s.Stats)
	delete(next.Stats, drillID)
	next.Attempts = slices.DeleteFunc(slices.Clone(s.Attempts), func(a Attempt) bool { return a.DrillID == drillID })
	for id, stat := range next.Stats {
		copied := *stat
		next.Stats[id] = &copied
	}
	next.recomputeStreaks()
	if err := next.Save(); err != nil {
		return err
	}
	*s = next
	return nil
}

func (s *HistoryStore) ResetAll() error {
	next := HistoryStore{filePath: s.filePath, Attempts: []Attempt{}, Stats: map[string]*DrillStats{}}
	if err := next.Save(); err != nil {
		return err
	}
	*s = next
	return nil
}

func (s *HistoryStore) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWriteFile(s.filePath, data, 0644)
}

// AtomicWriteFile writes data to a temp file in the same directory and renames
// it over path, so readers never see a half-written file.
func AtomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(dir, fmt.Sprintf(".%s-*.tmp", filepath.Base(path)))
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()

	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return err
	}

	if err := tmpFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	cleanup = false
	return nil
}
