package typingsound

import "math"

// Prototype is the generated-tone diagnostic mode. It uses the same key
// mapping and playback engine as real packs but needs no network or cache.
type Prototype struct {
	player *PackPlayer
}

// NewPrototype starts the generated tones on the audio server.
func NewPrototype(volume float64) (*Prototype, error) {
	sounds := Sounds{
		Lock: generatedTone(220),
		Quad: generatedTone(880),
	}
	for _, frequency := range []float64{330, 370, 415, 466, 523, 587, 659, 740} {
		sounds.Combos = append(sounds.Combos, generatedTone(frequency))
	}
	player, err := NewPackPlayer(sounds, volume, DefaultMapping())
	if err != nil {
		return nil, err
	}
	return &Prototype{player: player}, nil
}

func generatedTone(frequency float64) Clip {
	clip := make(Clip, sampleRate/12)
	for i := range clip {
		envelope := 1 - float64(i)/float64(len(clip))
		clip[i] = int16(7000 * envelope * math.Sin(2*math.Pi*frequency*float64(i)/sampleRate))
	}
	return clip
}

// Key plays the tone for an observed event without blocking.
func (p *Prototype) Key(event Event) {
	p.player.Key(event)
}

func (p *Prototype) Close() {
	p.player.Close()
}
