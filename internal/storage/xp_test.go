package storage

import (
	"testing"
	"time"
)

func TestAttemptXP(t *testing.T) {
	if xp := AttemptXP(5*time.Second, false, 1, false); xp != 10 {
		t.Errorf("expected 10 XP for failed attempt, got %d", xp)
	}

	if xp := AttemptXP(3*time.Second, true, 1, false); xp != 125 {
		t.Errorf("expected 125 XP for 3s first-try, got %d", xp)
	}

	if xp := AttemptXP(10*time.Second, true, 3, false); xp != 80 {
		t.Errorf("expected 80 XP for 10s multi-submit, got %d", xp)
	}

	if xp := AttemptXP(30*time.Second, true, 2, false); xp != 65 {
		t.Errorf("expected 65 XP, got %d", xp)
	}

	if xp := AttemptXP(60*time.Second, true, 2, false); xp != 55 {
		t.Errorf("expected 55 XP, got %d", xp)
	}

	if xp := AttemptXP(3*time.Second, true, 1, true); xp != 175 {
		t.Errorf("expected 175 XP with daily bonus, got %d", xp)
	}
}

func TestGetLevelInfo(t *testing.T) {
	info0 := GetLevelInfo(0)
	if info0.Level != 1 {
		t.Errorf("expected level 1 for 0 XP, got %d", info0.Level)
	}
	if info0.Title != "Syntax Novice" {
		t.Errorf("expected Syntax Novice, got %s", info0.Title)
	}
	if info0.Progress != 0.0 {
		t.Errorf("expected 0 progress, got %f", info0.Progress)
	}

	info150 := GetLevelInfo(150)
	if info150.Level != 2 {
		t.Errorf("expected level 2 for 150 XP, got %d", info150.Level)
	}
	if info150.Progress != 0.0 {
		t.Errorf("expected 0 progress at exact level start, got %f", info150.Progress)
	}

	info250 := GetLevelInfo(250)
	if info250.Level != 2 {
		t.Errorf("expected level 2 for 250 XP, got %d", info250.Level)
	}
	if info250.Progress != 0.5 {
		t.Errorf("expected 0.5 progress for 250 XP, got %f", info250.Progress)
	}

	infoGrandmaster := GetLevelInfo(100000)
	if infoGrandmaster.Level < 40 {
		t.Errorf("expected high level for 100000 XP, got %d", infoGrandmaster.Level)
	}
	if infoGrandmaster.Title != "Grandmaster" {
		t.Errorf("expected Grandmaster title, got %s", infoGrandmaster.Title)
	}
}

func TestHistoryStoreTotalXP(t *testing.T) {
	hist := &HistoryStore{
		Attempts: []Attempt{
			{Duration: 3 * time.Second, Passed: true, Attempts: 1},
			{Duration: 10 * time.Second, Passed: true, Attempts: 2},
			{Duration: 5 * time.Second, Passed: false, Attempts: 1},
		},
	}
	expectedXP := 125 + 80 + 10
	if total := hist.GetTotalXP(); total != expectedXP {
		t.Errorf("expected total XP %d, got %d", expectedXP, total)
	}
}
