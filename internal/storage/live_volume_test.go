package storage

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveVolumeFollowsSavedSettings(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	music := FollowVolume(ctx, .3, Config.MusicVolume)
	sfx := FollowVolume(ctx, .9, Config.TypingVolume)
	for _, volumes := range [][2]int{{0, 90}, {75, 10}} {
		mv, sv := volumes[0], volumes[1]
		if err := SaveConfig(Config{Music: &MusicSettings{Volume: &mv}, Typing: &TypingSettings{Volume: &sv}}); err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(time.Second)
		for music.Gain() != float64(mv)/100 || sfx.Gain() != float64(sv)/100 {
			if time.Now().After(deadline) {
				t.Fatalf("live gains: music %g, SFX %g", music.Gain(), sfx.Gain())
			}
			time.Sleep(5 * time.Millisecond)
		}
	}
	path, _ := GetConfigPath()
	if err := os.WriteFile(path, []byte("invalid"), 0600); err != nil {
		t.Fatal(err)
	}
	// Cross a polling interval to ensure a failed read keeps the previous gains.
	time.Sleep(150 * time.Millisecond)
	if music.Gain() != .75 || sfx.Gain() != .1 {
		t.Fatal("a failed config read reset the volume")
	}
}
