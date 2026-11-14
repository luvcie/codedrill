package typingsound

import "testing"

func TestDecodedSampleLimits(t *testing.T) {
	for _, tc := range []struct{ rate, want int }{
		{8000, 4_800_000},
		{48000, 28_800_000},
		{192000, 28_800_000},
	} {
		if got := decodedSampleLimit(tc.rate); got != tc.want {
			t.Errorf("rate %d: limit %d, want %d", tc.rate, got, tc.want)
		}
	}
	if got := capDecodedLength(maxDecodedSamples + 1); got != maxDecodedSamples {
		t.Errorf("long clip capped to %d, want %d", got, maxDecodedSamples)
	}
	if got := capDecodedLength(10); got != 10 {
		t.Errorf("short clip capped to %d, want 10", got)
	}
}
