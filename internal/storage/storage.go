package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Attempt struct {
	ID        string        `json:"id"`
	DrillID   string        `json:"drill_id"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Passed    bool          `json:"passed"`
	Attempts  int           `json:"attempts"`
}

type DrillStats struct {
	DrillID      string        `json:"drill_id"`
	PersonalBest time.Duration `json:"personal_best"`
	TotalRuns    int           `json:"total_runs"`
	Successful   int           `json:"successful"`
	LastAttempt  time.Time     `json:"last_attempt"`
}

type HistoryStore struct {
	filePath string
	Attempts []Attempt              `json:"attempts"`
	Stats    map[string]*DrillStats `json:"stats"`
}

func GetHistoryPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	dir := filepath.Join(configDir, "codedrill")
	if err := os.MkdirAll(dir, 0755); err != nil {
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

	if err := json.Unmarshal(data, store); err != nil {
		return store, nil
	}
	if store.Stats == nil {
		store.Stats = make(map[string]*DrillStats)
	}

	return store, nil
}

func (s *HistoryStore) RecordAttempt(drillID string, duration time.Duration, passed bool, submitCount int) (isPB bool, delta time.Duration, err error) {
	att := Attempt{
		ID:        time.Now().Format("20060102150405"),
		DrillID:   drillID,
		Duration:  duration,
		Timestamp: time.Now(),
		Passed:    passed,
		Attempts:  submitCount,
	}
	s.Attempts = append(s.Attempts, att)

	stat, exists := s.Stats[drillID]
	if !exists {
		stat = &DrillStats{
			DrillID: drillID,
		}
		s.Stats[drillID] = stat
	}

	stat.TotalRuns++
	stat.LastAttempt = att.Timestamp

	if passed {
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
	}

	err = s.Save()
	return isPB, delta, err
}

func (s *HistoryStore) GetPB(drillID string) time.Duration {
	if stat, exists := s.Stats[drillID]; exists {
		return stat.PersonalBest
	}
	return 0
}

func (s *HistoryStore) Save() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0644)
}
