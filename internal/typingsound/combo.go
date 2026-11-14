package typingsound

import "time"

// Combo tracks successive combo-triggering actions within the editing session.
type Combo struct {
	count int
	last  time.Time
}

// Next returns a zero-based index, capped at the last available sound.
// A zero sound count leaves the sequence untouched.
func (c *Combo) Next(now time.Time, sounds int) int {
	if sounds <= 0 {
		return -1
	}
	if c.count == 0 || now.Sub(c.last) >= 20*time.Second || now.Before(c.last) {
		c.count = 0
	}
	c.last = now
	if c.count < sounds {
		c.count++
	}
	return c.count - 1
}
