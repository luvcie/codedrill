package typingsound

import (
	"time"
)

// Sounds is the subset of a pack used by the editor triggers.
type Sounds struct {
	Lock   Clip
	Quad   Clip
	Combos []Clip
}

// PackPlayer maps editor keys to a pack's clips through one audio engine.
// Only semicolons advance the combo counter.
type PackPlayer struct {
	engine     *Engine
	mapping    Mapping
	combo      Combo
	lockIndex  int
	quadIndex  int
	comboBase  int
	comboCount int
	now        func() time.Time
}

// NewPackPlayer preloads the clips and starts the audio stream.
func NewPackPlayer(sounds Sounds, volume float64, mapping Mapping) (*PackPlayer, error) {
	clips := make([]Clip, 0, 2+len(sounds.Combos))
	player := &PackPlayer{mapping: mapping, lockIndex: -1, quadIndex: -1, comboBase: -1}
	add := func(clip Clip) int {
		if len(clip) == 0 {
			return -1
		}
		clips = append(clips, clip)
		return len(clips) - 1
	}
	player.lockIndex = add(sounds.Lock)
	player.quadIndex = add(sounds.Quad)
	if len(sounds.Combos) > 0 {
		player.comboBase = len(clips)
		player.comboCount = len(sounds.Combos)
		for _, clip := range sounds.Combos {
			add(clip)
		}
	}
	engine, err := NewEngine(clips, volume)
	if err != nil {
		return nil, err
	}
	player.engine = engine
	return player, nil
}

// Key plays the sound for an observed event without blocking the caller. A word
// completion plays the lock clip and never advances the combo.
func (p *PackPlayer) Key(event Event) {
	if p.engine == nil {
		return
	}
	p.engine.Play(p.eventIndex(event))
}

func (p *PackPlayer) eventIndex(event Event) int {
	if event.Word {
		return p.lockIndex
	}
	return p.index(event.Key)
}

// index decides which clip a key plays. It advances the combo only for combo
// keys; lock and quad keys leave the counter alone.
func (p *PackPlayer) index(key byte) int {
	switch p.mapping.Action(key) {
	case ActionLock:
		return p.lockIndex
	case ActionQuad:
		return p.quadIndex
	case ActionCombo:
		if p.comboCount > 0 {
			now := time.Now
			if p.now != nil {
				now = p.now
			}
			return p.comboBase + p.combo.Next(now(), p.comboCount)
		}
	}
	return -1
}

// PreviewKey plays one action directly, for settings previews.
func (p *PackPlayer) PreviewKey(action Action) {
	switch action {
	case ActionLock:
		p.engine.Play(p.lockIndex)
	case ActionQuad:
		p.engine.Play(p.quadIndex)
	case ActionCombo:
		if p.comboCount > 0 {
			p.engine.Play(p.comboBase)
		}
	}
}

func (p *PackPlayer) Close() {
	p.engine.Close()
}
