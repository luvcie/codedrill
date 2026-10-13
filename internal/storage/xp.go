package storage

import "time"

type LevelInfo struct {
	Level       int
	Title       string
	TotalXP     int
	NextLevelXP int
	Progress    float64
}

const (
	failXP       = 10
	clearXP      = 50
	firstTryXP   = 25
	DailyBonusXP = 50
)

// AttemptXP is the XP a run earns. isDaily marks the first clear of the day's
// daily drill.
func AttemptXP(duration time.Duration, passed bool, submitCount int, isDaily bool) int {
	if !passed {
		return failXP
	}

	xp := clearXP

	if duration > 0 {
		switch {
		case duration <= 5*time.Second:
			xp += 50
		case duration <= 15*time.Second:
			xp += 30
		case duration <= 45*time.Second:
			xp += 15
		default:
			xp += 5
		}
	}

	if submitCount <= 1 {
		xp += firstTryXP
	}
	if isDaily {
		xp += DailyBonusXP
	}

	return xp
}

func (s *HistoryStore) GetTotalXP() int {
	total := 0
	for _, a := range s.Attempts {
		total += AttemptXP(a.Duration, a.Passed, a.Attempts, a.IsDaily)
	}
	return total
}

func xpForLevel(l int) int {
	if l <= 1 {
		return 0
	}
	return 25*l*l + 75*l - 100
}

func GetLevelInfo(totalXP int) LevelInfo {
	if totalXP < 0 {
		totalXP = 0
	}

	level := 1
	for {
		nextXP := xpForLevel(level + 1)
		if totalXP < nextXP {
			break
		}
		level++
	}

	startXP := xpForLevel(level)
	nextXP := xpForLevel(level + 1)
	xpNeeded := nextXP - startXP
	xpInLevel := totalXP - startXP
	if xpNeeded <= 0 {
		xpNeeded = 1
	}

	progress := float64(xpInLevel) / float64(xpNeeded)
	if progress > 1.0 {
		progress = 1.0
	}
	if progress < 0.0 {
		progress = 0.0
	}

	return LevelInfo{
		Level:       level,
		Title:       levelTitle(level),
		TotalXP:     totalXP,
		NextLevelXP: nextXP,
		Progress:    progress,
	}
}

func levelTitle(level int) string {
	switch {
	case level < 3:
		return "Syntax Novice"
	case level < 5:
		return "Script Sprinter"
	case level < 8:
		return "Byte Dasher"
	case level < 12:
		return "Algorithm Ace"
	case level < 16:
		return "Terminal Prodigy"
	case level < 20:
		return "Executioner"
	case level < 25:
		return "Keyboard Virtuoso"
	case level < 30:
		return "Speed Demon"
	case level < 40:
		return "Godspeed Runner"
	default:
		return "Grandmaster"
	}
}
