// Package typingsound contains the formats and key mapping for editor sounds.
package typingsound

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
)

const maxExportBytes = 24 << 20
const maxAudioBytes = 16 << 20

var soundName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Atlas timings are in milliseconds, as in TETR.IO+ exports.
type Atlas map[string][2]float64

// ParseExport reads the sound recording and atlas from a TETR.IO+ export.
func ParseExport(data []byte) ([]byte, Atlas, error) {
	if len(data) > maxExportBytes {
		return nil, nil, errors.New("sound export exceeds 24 MiB")
	}
	var export struct {
		Sounds string                     `json:"customSounds"`
		Atlas  map[string]json.RawMessage `json:"customSoundAtlas"`
	}
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, nil, fmt.Errorf("invalid sound export: %w", err)
	}
	if len(export.Atlas) == 0 || len(export.Atlas) > 512 {
		return nil, nil, errors.New("sound atlas must contain 1 to 512 entries")
	}
	atlas := make(Atlas, len(export.Atlas))
	for name, raw := range export.Atlas {
		var timing []float64
		if !soundName.MatchString(name) || json.Unmarshal(raw, &timing) != nil || len(timing) != 2 ||
			math.IsNaN(timing[0]) || math.IsNaN(timing[1]) || math.IsInf(timing[0], 0) || math.IsInf(timing[1], 0) ||
			timing[0] < 0 || timing[1] <= 0 || timing[0]+timing[1] > 600000 {
			return nil, nil, fmt.Errorf("invalid sound atlas entry %q", name)
		}
		atlas[name] = [2]float64{timing[0], timing[1]}
	}
	prefix, encoded, ok := strings.Cut(export.Sounds, ",")
	if !ok || (prefix != "data:audio/ogg;base64" && prefix != "data:audio/mpeg;base64" && prefix != "data:audio/mp3;base64" && prefix != "data:audio/wav;base64" && prefix != "data:audio/webm;base64") || len(encoded) == 0 || len(encoded) > ((maxAudioBytes+2)/3)*4 {
		return nil, nil, errors.New("invalid or oversized sound recording")
	}
	audio, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(audio) == 0 || len(audio) > maxAudioBytes {
		return nil, nil, errors.New("invalid or oversized sound recording")
	}
	return audio, atlas, nil
}
