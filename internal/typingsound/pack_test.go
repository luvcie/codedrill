package typingsound

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseExport(t *testing.T) {
	data := []byte(`{"customSounds":"data:audio/ogg;base64,YWJj","customSoundAtlas":{"harddrop":[100,250],"combo_1":[400,50]}}`)
	audio, atlas, err := ParseExport(data)
	if err != nil || string(audio) != "abc" || atlas["harddrop"] != [2]float64{100, 250} {
		t.Fatalf("ParseExport: %q %v %v", audio, atlas, err)
	}
	for _, invalid := range []string{
		`{"customSounds":"data:audio/ogg;base64,%%%","customSoundAtlas":{"harddrop":[0,1]}}`,
		`{"customSounds":"data:audio/ogg;base64,YQ==","customSoundAtlas":{"harddrop":[0,0]}}`,
		`{"customSounds":"data:audio/ogg;base64,YQ==","customSoundAtlas":{"harddrop":[0,1,2]}}`,
		`{"customSounds":"data:audio/ogg;base64,YQ==","customSoundAtlas":{"../x":[0,1]}}`,
		`{"customSounds":"data:text/plain;base64,YQ==","customSoundAtlas":{"harddrop":[0,1]}}`,
	} {
		if _, _, err := ParseExport([]byte(invalid)); err == nil {
			t.Errorf("accepted %s", invalid)
		}
	}
	// A missing atlas must not turn unrelated TETR.IO+ settings into a pack.
	data, _ = json.Marshal(map[string]string{"customSounds": "data:audio/ogg;base64,YQ=="})
	if _, _, err := ParseExport(data); err == nil {
		t.Fatal("accepted export without an atlas")
	}
}

func TestComboProgressionAndReset(t *testing.T) {
	var c Combo
	now := time.Unix(100, 0)
	for i, want := range []int{0, 1, 2, 2, 2} {
		if got := c.Next(now.Add(time.Duration(i)*time.Second), 3); got != want {
			t.Fatalf("combo %d: got %d, want %d", i, got, want)
		}
	}
	if got := c.Next(now.Add(24*time.Second), 3); got != 0 {
		t.Fatalf("after inactivity got %d, want 0", got)
	}
	if got := c.Next(now, 0); got != -1 {
		t.Fatalf("empty sequence got %d, want -1", got)
	}
}
