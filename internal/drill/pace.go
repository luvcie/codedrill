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
