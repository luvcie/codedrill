package storage

import (
	"testing"
	"time"
)

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d        time.Duration
		expected string
	}{
		{0, "0s"},
		{500 * time.Millisecond, "0.500s"},
		{1500 * time.Millisecond, "1.500s"},
		{59*time.Second + 999*time.Millisecond, "59.999s"},
		{60 * time.Second, "01m 00.000s"},
		{75*time.Second + 250*time.Millisecond, "01m 15.250s"},
		{10 * time.Minute, "10m 00.000s"},
		{time.Hour - time.Millisecond, "59m 59.999s"},
		{time.Hour - 500*time.Microsecond, "1h 00m 00.000s"},
		{time.Hour, "1h 00m 00.000s"},
		{time.Hour + time.Minute + 2*time.Second + 3*time.Millisecond, "1h 01m 02.003s"},
		{170*time.Minute + 15*time.Second + 230*time.Millisecond, "2h 50m 15.230s"},
		{24 * time.Hour, "24h 00m 00.000s"},
	}

	for _, c := range cases {
		got := FormatDuration(c.d)
		if got != c.expected {
			t.Errorf("FormatDuration(%v) = %q; want %q", c.d, got, c.expected)
		}
	}
}
