package typingsound

import (
	"fmt"
	"time"
)

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
	player.PreviewKey(ActionLock)
	time.Sleep(220 * time.Millisecond)
	player.PreviewKey(ActionCombo)
	time.Sleep(220 * time.Millisecond)
	player.PreviewKey(ActionQuad)
	time.Sleep(650 * time.Millisecond)
	return nil
}
