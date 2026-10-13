package storage

import (
	"fmt"
	"time"
)

// FormatDuration renders sprint times as "5.123s" or "02m 05.123s".
func FormatDuration(d time.Duration) string {
	if d == 0 {
		return "0s"
	}
	d = d.Round(time.Millisecond)
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000
	if mins > 0 {
		return fmt.Sprintf("%02dm %02d.%03ds", mins, secs, ms)
	}
	return fmt.Sprintf("%d.%03ds", secs, ms)
}
