package storage

import (
	"slices"
	"strings"
	"time"
)

// RecordMode identifies a record category. Runs without a mode have no records.
type RecordMode string

const (
	ExamMode   RecordMode = "exam"
	RecallMode RecordMode = "recall"
	StudyMode  RecordMode = "study"
)

type ModeStats struct {
	PersonalBest  time.Duration
	TotalRuns     int
	Successful    int
	CurrentStreak int
	BestStreak    int
	LastAttempt   time.Time
}

func (s *HistoryStore) indexModes() {
	if s.modeStats != nil && s.indexedAttempts == len(s.Attempts) {
		return
	}
	s.modeStats = make(map[string]map[RecordMode]*ModeStats)
	for _, att := range s.Attempts {
		if att.Mode == "" {
			continue
		}
		if s.modeStats[att.DrillID] == nil {
			s.modeStats[att.DrillID] = make(map[RecordMode]*ModeStats)
		}
		st := s.modeStats[att.DrillID][att.Mode]
		if st == nil {
			st = &ModeStats{}
			s.modeStats[att.DrillID][att.Mode] = st
		}
		st.TotalRuns++
		st.LastAttempt = att.Timestamp
		if att.Passed {
			st.Successful++
			st.CurrentStreak++
			st.BestStreak = max(st.BestStreak, st.CurrentStreak)
			if st.PersonalBest == 0 || att.Duration < st.PersonalBest {
				st.PersonalBest = att.Duration
			}
		} else {
			st.CurrentStreak = 0
		}
	}
	s.indexedAttempts = len(s.Attempts)
}

func (s *HistoryStore) GetModeStats(drillID string, category RecordMode) *ModeStats {
	s.indexModes()
	return s.modeStats[drillID][category]
}

// RecordModes always starts with Exam, even before the first Exam run.
func (s *HistoryStore) RecordModes(drillID string) []RecordMode {
	s.indexModes()
	records := s.modeStats[drillID]
	if drillID == "" {
		records = make(map[RecordMode]*ModeStats)
		for _, modes := range s.modeStats {
			for category, stats := range modes {
				records[category] = stats
			}
		}
	}
	modes := []RecordMode{ExamMode, RecallMode, StudyMode}
	var custom []RecordMode
	for category := range records {
		if strings.HasPrefix(string(category), "custom-") {
			custom = append(custom, category)
		}
	}
	slices.Sort(custom)
	modes = append(modes, custom...)
	return modes
}
