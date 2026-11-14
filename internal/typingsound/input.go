package typingsound

import "time"

// Event is one typing-sound trigger observed in the editor input. The key is
// reported only for triggers; the wrapper forwards the original bytes to the
// editor independently. Word marks a space or tab that finishes a word.
type Event struct {
	Key  byte
	Word bool
}

// Input recognizes plain key bytes while ignoring terminal control sequences
// and bracketed paste. It retains only protocol state and a word length, never
// entered text.
type Input struct {
	state          byte
	seq            [8]byte
	n              int
	paste          bool
	x10            int
	escapeAt       time.Time
	wordCompletion bool
	wordLen        int
	mapping        Mapping
}

// NewInput returns an observer with the default key mapping. Word completion
// reports a space or tab after at least two word bytes; it is off otherwise.
func NewInput(wordCompletion bool) *Input {
	return NewInputWithMapping(wordCompletion, DefaultMapping())
}

// NewInputWithMapping returns an observer that reports the configured trigger keys.
func NewInputWithMapping(wordCompletion bool, mapping Mapping) *Input {
	return &Input{wordCompletion: wordCompletion, mapping: mapping}
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
// Space or Tab stays silent. Any punctuation, whitespace, newline, backspace or
// escape ends the word, which also suppresses repeated spaces.
func (p *Input) track(b byte, emit func(Event)) {
	action := p.mapping.Action(b)
	if isWordByte(b) {
		p.wordLen++
		if action != ActionSilent {
			emit(Event{Key: b})
		}
		return
	}
	if b == ' ' || b == '\t' {
		// An explicitly mapped whitespace key plays its action. Word completion
		// stays the default for unmapped space and tab, and never emits twice.
		if action != ActionSilent {
			emit(Event{Key: b})
		} else if p.wordCompletion && p.wordLen >= 2 {
			emit(Event{Key: b, Word: true})
		}
		p.wordLen = 0
		return
	}
	if action != ActionSilent {
		emit(Event{Key: b})
	}
	p.wordLen = 0
}

func isWordByte(b byte) bool {
	return b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' ||
		b == '_' || b >= 0x80
}
