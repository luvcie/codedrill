package typingsound

import (
	"reflect"
	"testing"
	"time"
)

func emittedKeys(events []Event) []byte {
	keys := make([]byte, 0, len(events))
	for _, event := range events {
		keys = append(keys, event.Key)
	}
	return keys
}

func TestInputFiltersTerminalSequencesAndPasteAcrossReads(t *testing.T) {
	input := NewInputWithMapping(false, DefaultMapping())
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	for _, chunk := range []string{
		"a{;", "\x1b[20", "0~pasted{;\r}", "\x1b[201", "~",
		"\x1b[<0;12;34M", "\x1b[M{;}", "\x1b]title;{}", "\x1b\\",
		"\x1bO", "P", "\x1b", ";", "}\r",
	} {
		input.Observe([]byte(chunk), emit)
	}
	if want := []byte{'{', ';', '}', '\r'}; !reflect.DeepEqual(emittedKeys(got), want) {
		t.Fatalf("got %q, want %q", emittedKeys(got), want)
	}
}

func TestEscapeThenKey(t *testing.T) {
	input := NewInputWithMapping(false, DefaultMapping())
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	now := time.Unix(0, 0)
	input.observe([]byte("\x1b"), emit, now)
	input.observe([]byte(";"), emit, now.Add(time.Second))
	input.observe([]byte("\x1b"), emit, now.Add(2*time.Second))
	input.observe([]byte(";"), emit, now.Add(2*time.Second+time.Millisecond))
	if want := []byte{';'}; !reflect.DeepEqual(emittedKeys(got), want) {
		t.Fatalf("got %q, want one ordinary semicolon", emittedKeys(got))
	}
}

func TestWordCompletionTriggersOnceAfterAWord(t *testing.T) {
	input := NewInputWithMapping(true, DefaultMapping())
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	input.Observe([]byte("vect2  "), emit)
	input.Observe([]byte("name\t"), emit)
	want := []Event{{Key: ' ', Word: true}, {Key: '\t', Word: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWordCompletionAllowsReferenceAndPointerSuffixes(t *testing.T) {
	for _, word := range []string{"vect2&", "vect2&&", "vect2*", "vect2**", "vect2*&"} {
		for _, boundary := range []byte{' ', '\t'} {
			t.Run(word+string(boundary), func(t *testing.T) {
				input := NewInputWithMapping(true, DefaultMapping())
				var got []Event
				emit := func(e Event) { got = append(got, e) }
				for _, b := range []byte(word) {
					input.Observe([]byte{b}, emit)
				}
				input.Observe([]byte{boundary, boundary}, emit)
				if want := []Event{{Key: boundary, Word: true}}; !reflect.DeepEqual(got, want) {
					t.Fatalf("got %+v, want %+v", got, want)
				}
			})
		}
	}
}

func TestWordCompletionAtClosingParenthesesAndStandaloneColon(t *testing.T) {
	for _, name := range []string{"vect2", "Point", "make_value", "f"} {
		t.Run(name, func(t *testing.T) {
			input := NewInputWithMapping(true, DefaultMapping())
			var got []Event
			emit := func(e Event) { got = append(got, e) }
			for _, b := range []byte(name + "::" + name + "()") {
				input.Observe([]byte{b}, emit)
			}
			if want := []Event{{Key: ')', Word: true}}; !reflect.DeepEqual(got, want) {
				t.Fatalf("after parentheses: got %+v, want %+v", got, want)
			}
			input.Observe([]byte(" : coordinates{0,0}"), emit)
			want := []Event{{Key: ')', Word: true}, {Key: ':', Word: true}, {Key: '{'}, {Key: '}'}}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestWordCompletionKeepsScopeResolutionSilentAcrossReads(t *testing.T) {
	input := NewInputWithMapping(true, DefaultMapping())
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	for _, chunk := range []string{"::", "Widget", ":", ":", "Widget", "()", "  \t"} {
		input.Observe([]byte(chunk), emit)
	}
	if want := []Event{{Key: ')', Word: true}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWordCompletionAtStandaloneColon(t *testing.T) {
	for _, next := range []byte{' ', '\t', 'v'} {
		input := NewInputWithMapping(true, DefaultMapping())
		var got []Event
		emit := func(e Event) { got = append(got, e) }
		input.Observe([]byte{':'}, emit)
		if len(got) != 0 {
			t.Fatalf("got %+v before a second colon could be ruled out", got)
		}
		input.Observe([]byte{next}, emit)
		if want := []Event{{Key: ':', Word: true}}; !reflect.DeepEqual(got, want) {
			t.Fatalf("after %q: got %+v, want %+v", next, got, want)
		}
	}
}

func TestWordCompletionAtSeparatedScopeOperator(t *testing.T) {
	for _, prefix := range []string{"", " ", "\t"} {
		for _, next := range []byte{' ', '\t'} {
			input := NewInputWithMapping(true, DefaultMapping())
			var got []Event
			emit := func(e Event) { got = append(got, e) }
			for _, chunk := range []string{prefix, ":", ":"} {
				input.Observe([]byte(chunk), emit)
			}
			if len(got) != 0 {
				t.Fatalf("got %+v before the following whitespace", got)
			}
			input.Observe([]byte{next, next}, emit)
			if want := []Event{{Key: ':', Word: true}}; !reflect.DeepEqual(got, want) {
				t.Fatalf("after %q: got %+v, want %+v", prefix+"::"+string(next), got, want)
			}
		}
	}
}

func TestWordCompletionIgnoresAttachedScopeOperators(t *testing.T) {
	for _, keys := range []string{"Widget:: ", " ::Widget", " ::: "} {
		input := NewInputWithMapping(true, DefaultMapping())
		var got []Event
		input.Observe([]byte(keys), func(e Event) { got = append(got, e) })
		if len(got) != 0 {
			t.Fatalf("after %q: got %+v, want no word sounds", keys, got)
		}
	}
}

func TestWordCompletionCancelsColonOnEditingAndNavigation(t *testing.T) {
	for _, keys := range []string{":\x7f ", ":\b ", ":\x01 ", ":\x1b[A ", ":\x1b[200~pasted() : \x1b[201~ "} {
		t.Run(keys, func(t *testing.T) {
			input := NewInputWithMapping(true, DefaultMapping())
			var got []Event
			input.Observe([]byte(keys), func(e Event) { got = append(got, e) })
			if len(got) != 0 {
				t.Fatalf("got %+v, want no word sounds for edits, navigation or paste", got)
			}
		})
	}
}

func TestWordCompletionSuppressesEmptyLinesAndShortWords(t *testing.T) {
	input := NewInputWithMapping(true, DefaultMapping())
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	// Spaces on an empty line, after a newline, and after one letter are quiet.
	input.Observe([]byte("   \n  \n a b & && * ** a& a&& a* "), emit)
	for _, event := range got {
		if event.Word {
			t.Fatalf("got %+v, want no word sounds", got)
		}
	}
}

func TestWordCompletionResetsOnPunctuationAndBackspace(t *testing.T) {
	input := NewInputWithMapping(true, DefaultMapping())
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	// A word followed by punctuation is not a finished word. A backspace also
	// makes the boundary ambiguous, so it stays quiet.
	input.Observe([]byte("foo] vect2&\x7f vect2*\b "), emit)
	input.Observe([]byte("bar "), emit)
	if want := []Event{{Key: ' ', Word: true}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want one word completion", got)
	}
}

func TestWordCompletionIgnoresPasteAndEscapeSequences(t *testing.T) {
	input := NewInputWithMapping(true, DefaultMapping())
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	input.Observe([]byte("\x1b[200~pasted vect2& call() : \x1b[201~"), emit)
	input.Observe([]byte("vect2&\x1b[A "), emit)
	if len(got) != 0 {
		t.Fatalf("got %+v, want no word sounds for paste or navigation", got)
	}
}

func TestInputUsesConfiguredMapping(t *testing.T) {
	mapping, err := ParseMapping(":", "!", "?")
	if err != nil {
		t.Fatal(err)
	}
	input := NewInputWithMapping(false, mapping)
	var got []Event
	input.Observe([]byte("a:b!c?d{;}\r"), func(event Event) { got = append(got, event) })
	if want := []Event{{Key: ':'}, {Key: '!'}, {Key: '?'}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestMappedWhitespaceDoesNotDuplicateWordCompletion(t *testing.T) {
	mapping, err := ParseMapping("", "space", "")
	if err != nil {
		t.Fatal(err)
	}
	input := NewInputWithMapping(true, mapping)
	var got []Event
	input.Observe([]byte("vect2& "), func(event Event) { got = append(got, event) })
	if want := []Event{{Key: ' '}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want one mapped space event", got)
	}
}

func TestMappedPunctuationDoesNotDuplicateWordCompletion(t *testing.T) {
	mapping, err := ParseMapping(")", ":", "")
	if err != nil {
		t.Fatal(err)
	}
	input := NewInputWithMapping(true, mapping)
	var got []Event
	input.Observe([]byte("Widget::Widget() : "), func(e Event) { got = append(got, e) })
	if want := []Event{{Key: ':'}, {Key: ':'}, {Key: ')'}, {Key: ':'}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want only the configured actions %+v", got, want)
	}
}

func TestWordCompletionOffEmitsOnlyMappedKeys(t *testing.T) {
	input := NewInputWithMapping(false, DefaultMapping())
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	input.Observe([]byte("word \tvect2& vect2*\tWidget::Widget() : {;}\r"), emit)
	if want := []byte{'{', ';', '}', '\r'}; !reflect.DeepEqual(emittedKeys(got), want) {
		t.Fatalf("got %q, want %q", emittedKeys(got), want)
	}
}
