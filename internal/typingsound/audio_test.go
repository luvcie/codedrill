package typingsound

import (
	"fmt"
	"testing"
)

func TestClipExtractionLimits(t *testing.T) {
	samples := make(Clip, maxClipSamples)
	for i := range samples {
		samples[i] = 1
	}
	atlas := Atlas{
		"at_limit":  {0, maxClipSeconds * 1000},
		"too_long":  {0, maxClipSeconds*1000 + 1},
		"overlap_1": {0, maxClipSeconds * 1000},
		"overlap_2": {0, maxClipSeconds * 1000},
	}
	if got := atlasClip(samples, atlas, "at_limit"); len(got) != maxClipSamples {
		t.Fatalf("clip at the limit has %d samples, want %d", len(got), maxClipSamples)
	}
	if got := atlasClip(samples, atlas, "too_long"); got != nil {
		t.Fatalf("accepted a %d-sample clip", len(got))
	}
	sounds := extractSounds(samples, atlas)
	if len(sounds.Lock) != 0 || len(sounds.Quad) != 0 || len(sounds.Combos) != 0 {
		t.Fatalf("unknown names produced sounds: %+v", sounds)
	}
}

func TestExtractSoundsCapsOverlappingCopies(t *testing.T) {
	samples := make(Clip, maxClipSamples)
	atlas := Atlas{
		"harddrop":  {0, maxClipSeconds * 1000},
		"clearquad": {0, maxClipSeconds * 1000},
	}
	for i := 1; i <= 16; i++ {
		atlas[fmt.Sprintf("combo_%d", i)] = [2]float64{0, maxClipSeconds * 1000}
	}
	sounds := extractSounds(samples, atlas)
	total := len(sounds.Lock) + len(sounds.Quad)
	for _, clip := range sounds.Combos {
		total += len(clip)
	}
	if total != maxTotalClipSamples {
		t.Fatalf("extracted %d samples, want the %d-sample budget", total, maxTotalClipSamples)
	}
	if len(sounds.Combos) != 10 {
		t.Fatalf("kept %d overlapping combos, want 10 within the budget", len(sounds.Combos))
	}
	if !soundsWithinLimits(sounds) {
		t.Fatal("extracted sounds exceed the playback limits")
	}
}

func TestSoundsWithinLimitsRejectsOversizedClip(t *testing.T) {
	if soundsWithinLimits(Sounds{Lock: make(Clip, maxClipSamples+1)}) {
		t.Fatal("accepted an oversized clip")
	}
	if !soundsWithinLimits(Sounds{Lock: make(Clip, maxClipSamples)}) {
		t.Fatal("rejected a clip at the limit")
	}
}

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
