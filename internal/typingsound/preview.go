package typingsound

import (
	"fmt"
	"time"
)

const (
	previewGap     = 220 * time.Millisecond
	maxPreviewWait = 5 * time.Second
)

// previewWait is how long to keep the stream open after the last preview
// trigger. It uses the longest clip that is still playing, capped so a long
// clip cannot hold the settings preview open indefinitely.
func previewWait(end, now time.Time) time.Duration {
	if !end.After(now) {
		return 0
	}
	return min(end.Sub(now), maxPreviewWait)
}

// PreviewPack loads a pack and plays its lock, first combo and quad sounds.
// It blocks while the clips play, so callers should run it off the UI thread.
func PreviewPack(id string, volume float64) error {
	if id == "" || id == SilentPackID {
		return nil
	}
	var player *PackPlayer
	var err error
	if id == PrototypePackID {
		proto, protoErr := NewPrototype(volume)
		if protoErr != nil {
			return protoErr
		}
		player = proto.player
		defer proto.Close()
	} else {
		info, ok := FindPack(id)
		if !ok {
			return fmt.Errorf("unknown sound pack %q", id)
		}
		player, err = LoadPack(info, volume, DefaultMapping())
		if err != nil {
			return err
		}
		defer player.Close()
	}
	var end time.Time
	play := func(action Action) {
		player.PreviewKey(action)
		if length := player.PreviewDuration(action); length > 0 {
			until := time.Now().Add(length)
			if until.After(end) {
				end = until
			}
		}
	}
	play(ActionLock)
	time.Sleep(previewGap)
	play(ActionCombo)
	time.Sleep(previewGap)
	play(ActionQuad)
	time.Sleep(previewWait(end, time.Now()))
	return nil
}
