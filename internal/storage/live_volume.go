package storage

import (
	"context"
	"math"
	"sync/atomic"
	"time"
)

type LiveVolume struct{ gain atomic.Uint64 }

// FollowVolume shares saved volume changes across the drill and editor processes.
// File reads stay off the audio callback.
func FollowVolume(ctx context.Context, initial float64, setting func(Config) float64) *LiveVolume {
	v := &LiveVolume{}
	v.gain.Store(math.Float64bits(max(0, min(1, initial))))
	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				if cfg, err := PeekConfig(); err == nil {
					v.gain.Store(math.Float64bits(setting(cfg)))
				}
			}
		}
	}()
	return v
}

func (v *LiveVolume) Gain() float64 {
	if v == nil {
		return 1
	}
	return math.Float64frombits(v.gain.Load())
}
