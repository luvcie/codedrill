package typingsound

import (
	"fmt"
	"strings"
)

// Action is what a key does for the typing sounds.
type Action int

const (
	ActionSilent Action = iota
	ActionLock
	ActionCombo
	ActionQuad
)

// Mapping assigns key bytes to actions. Several keys can share an action.
type Mapping struct {
	lock  map[byte]bool
	combo map[byte]bool
	quad  map[byte]bool
}

// DefaultMapping implements the documented behavior: braces play lock and
// quad, Enter plays lock, and semicolon advances the combo.
func DefaultMapping() Mapping {
	return Mapping{
		lock:  map[byte]bool{'{': true, '\r': true, '\n': true},
		combo: map[byte]bool{';': true},
		quad:  map[byte]bool{'}': true},
	}
}

// ParseMapping builds a mapping from three comma-separated key lists. An empty
// list keeps the default for that action. Tokens may be a single character,
// "enter", "tab", "space", or "none" to clear the action.
func ParseMapping(lock, combo, quad string) (Mapping, error) {
	m := Mapping{lock: map[byte]bool{}, combo: map[byte]bool{}, quad: map[byte]bool{}}
	defaults := DefaultMapping()
	for _, entry := range []struct {
		target   map[byte]bool
		spec     string
		fallback map[byte]bool
	}{
		{m.lock, lock, defaults.lock},
		{m.combo, combo, defaults.combo},
		{m.quad, quad, defaults.quad},
	} {
		keys, err := parseKeys(entry.spec)
		if err != nil {
			return Mapping{}, err
		}
		if len(keys) == 0 && strings.TrimSpace(entry.spec) == "" {
			for key := range entry.fallback {
				entry.target[key] = true
			}
			continue
		}
		for _, key := range keys {
			entry.target[key] = true
		}
	}
	return m, nil
}

func parseKeys(spec string) ([]byte, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	var keys []byte
	for _, token := range strings.Split(spec, ",") {
		token = strings.TrimSpace(token)
		switch strings.ToLower(token) {
		case "":
			continue
		case "none":
			return nil, nil
		case "enter", "return", "newline":
			keys = append(keys, '\r', '\n')
		case "tab":
			keys = append(keys, '\t')
		case "space":
			keys = append(keys, ' ')
		default:
			if len([]rune(token)) != 1 {
				return nil, fmt.Errorf("key %q must be one character or enter/tab/space", token)
			}
			keys = append(keys, token[0])
		}
	}
	return keys, nil
}

// Action reports what key does.
func (m Mapping) Action(key byte) Action {
	switch {
	case m.lock[key]:
		return ActionLock
	case m.quad[key]:
		return ActionQuad
	case m.combo[key]:
		return ActionCombo
	}
	return ActionSilent
}
