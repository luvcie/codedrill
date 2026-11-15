package drill

import (
	"strings"
	"time"
)

// PaceTargets returns the S, A and B rank cut-offs for a difficulty.
// A time at or under a cut-off earns that rank.
func PaceTargets(difficulty string) (s, a, b time.Duration) {
	switch strings.ToLower(difficulty) {
	case "hard":
		return 25 * time.Second, 50 * time.Second, 90 * time.Second
	case "medium":
		return 12 * time.Second, 25 * time.Second, 50 * time.Second
	default:
		return 4 * time.Second, 8 * time.Second, 15 * time.Second
	}
}

type RankTarget struct {
	Rank      string
	Time      time.Duration
	Inclusive bool
}

func (r RankTarget) Reached(d time.Duration) bool {
	return d > 0 && (r.Time == 0 || d < r.Time || r.Inclusive && d == r.Time)
}

// RankTargets lists public ranks from lowest to highest. Hidden ranks never
// appear in progression hints or live pace, even after they have been earned.
func RankTargets(d Drill) []RankTarget {
	if seconds, ok := rankSeconds[d.ID]; d.IsPreset && ok {
		targets := make([]RankTarget, len(publicRanks))
		targets[0] = RankTarget{Rank: publicRanks[0]}
		for i, rank := range publicRanks[1:] {
			targets[i+1] = RankTarget{Rank: rank, Time: time.Duration(seconds[i]) * time.Second}
		}
		return targets
	}
	s, a, b := PaceTargets(d.Difficulty)
	return []RankTarget{
		{Rank: "C"},
		{Rank: "B", Time: b, Inclusive: true},
		{Rank: "A", Time: a, Inclusive: true},
		{Rank: "S", Time: s, Inclusive: true},
	}
}

// CalculateRank grades a successful completion or a saved personal best.
func CalculateRank(duration time.Duration, d Drill) string {
	if duration <= 0 {
		return ""
	}
	if seconds, ok := rankSeconds[d.ID]; d.IsPreset && ok && duration < time.Duration(seconds[len(seconds)-1])*time.Second {
		return "X+"
	}
	targets := RankTargets(d)
	for i := len(targets) - 1; i >= 0; i-- {
		if targets[i].Reached(duration) {
			return targets[i].Rank
		}
	}
	return ""
}
