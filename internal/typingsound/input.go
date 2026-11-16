package typingsound

import "time"

// Event is one typing-sound trigger observed in the editor input. The key is
// reported only for triggers; the wrapper forwards the original bytes to the
// editor independently. Word marks an optional word or punctuation completion.
type Event struct {
	Key  byte
	Word bool
}

// Input recognizes plain key bytes while ignoring terminal control sequences
// and bracketed paste. It retains only protocol state and completion counters,
// never entered text.
type Input struct {
	state          byte
	seq            [8]byte
	n              int
	paste          bool
	x10            int
	escapeAt       time.Time
	wordCompletion bool
	wordLen        int
	colonCount     int
	colonSeparated bool
	mapping        Mapping
}

// NewInput returns an observer with the default key mapping. When enabled,
// word completion reports words, closing parentheses and separate colon tokens.
func NewInput(wordCompletion bool) *Input {
	return NewInputWithMapping(wordCompletion, DefaultMapping())
}

// NewInputWithMapping returns an observer that reports the configured trigger keys.
func NewInputWithMapping(wordCompletion bool, mapping Mapping) *Input {
	return &Input{wordCompletion: wordCompletion, colonSeparated: true, mapping: mapping}
}

// Observe reports sound-triggering events. The caller must forward the original
// input unchanged, independently of this observer.
func (p *Input) Observe(data []byte, emit func(Event)) {
	p.observe(data, emit, time.Now())
}

func (p *Input) observe(data []byte, emit func(Event), now time.Time) {
	if p.state == 1 && !p.paste && now.Sub(p.escapeAt) > 50*time.Millisecond {
		p.state = 0
	}
	for _, b := range data {
		if p.x10 > 0 {
			p.x10--
			continue
		}
		switch p.state {
		case 0:
			if b == 0x1b {
				p.state = 1
				p.escapeAt = now
				// A control sequence is navigation, not typing.
				p.wordLen = 0
				p.colonCount = 0
				p.colonSeparated = false
			} else if !p.paste {
				p.track(b, emit)
			}
		case 1:
			switch b {
			case '[':
				p.state, p.n = 2, 0
			case ']', 'P', '_', '^':
				p.state = 3
			case 'O':
				p.state = 5
			case 0x1b:
				// A second escape can begin a new control sequence.
				p.escapeAt = now
			default:
				p.state = 0
			}
		case 2:
			if p.n < len(p.seq) {
				p.seq[p.n] = b
				p.n++
			}
			if b >= 0x40 && b <= 0x7e {
				if p.n == 4 && string(p.seq[:4]) == "200~" {
					p.paste = true
				} else if p.n == 4 && string(p.seq[:4]) == "201~" {
					p.paste = false
				} else if p.n == 1 && b == 'M' {
					p.x10 = 3
				}
				p.state = 0
			}
		case 3: // OSC, DCS, APC and PM strings end in BEL or ST.
			if b == 7 {
				p.state = 0
			} else if b == 0x1b {
				p.state = 4
			}
		case 4:
			if b == '\\' {
				p.state = 0
			} else if b != 0x1b {
				p.state = 3
			}
		case 5: // SS3 function-key sequence
			p.state = 0
		}
	}
}

// track updates the word heuristic for one plain byte and emits its trigger.
// A word needs at least two word bytes so a single normal-mode command before
// Space or Tab stays silent. Reference and pointer suffixes preserve the word;
// other punctuation, whitespace, backspace or escape ends it.
func (p *Input) track(b byte, emit func(Event)) {
	// Wait for another key to distinguish separate colon tokens from scope resolution.
	if b != ':' {
		whitespace := b == ' ' || b == '\t' || b == '\r' || b == '\n'
		singleColon := p.colonCount == 1 && (b >= ' ' && b != 0x7f || whitespace)
		spacedScope := p.colonCount == 2 && p.colonSeparated && whitespace
		if p.wordCompletion && (singleColon || spacedScope) && p.mapping.Action(':') == ActionSilent {
			emit(Event{Key: ':', Word: true})
		}
		p.colonCount = 0
		p.colonSeparated = whitespace
	} else if p.colonCount < 3 {
		p.colonCount++
	}

	action := p.mapping.Action(b)
	if isWordByte(b) {
		p.wordLen++
		if action != ActionSilent {
			emit(Event{Key: b})
		}
		return
	}
	if b == ' ' || b == '\t' || b == ')' {
		// Explicit mappings take priority over optional completion sounds.
		if action != ActionSilent {
			emit(Event{Key: b})
		} else if p.wordCompletion && (b == ')' || p.wordLen >= 2) {
			emit(Event{Key: b, Word: true})
		}
		p.wordLen = 0
		return
	}
	if action != ActionSilent {
		emit(Event{Key: b})
	}
	if b != '&' && b != '*' {
		p.wordLen = 0
	}
}

func isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' ||
		b == '_' || b >= 0x80
}
