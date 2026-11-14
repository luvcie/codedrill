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
	var input Input
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
	var input Input
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
	input := NewInput(true)
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	input.Observe([]byte("vect2  "), emit)
	input.Observe([]byte("name\t"), emit)
	want := []Event{{Key: ' ', Word: true}, {Key: '\t', Word: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func TestWordCompletionSuppressesEmptyLinesAndShortWords(t *testing.T) {
	input := NewInput(true)
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	// Spaces on an empty line, after a newline, and after one letter are quiet.
	input.Observe([]byte("   \n  \n a b "), emit)
	for _, event := range got {
		if event.Word {
			t.Fatalf("got %+v, want no word sounds", got)
		}
	}
}

func TestWordCompletionResetsOnPunctuationAndBackspace(t *testing.T) {
	input := NewInput(true)
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	// A word followed by punctuation is not a finished word. A backspace also
	// makes the boundary ambiguous, so it stays quiet.
	input.Observe([]byte("foo) \x7f"), emit)
	input.Observe([]byte("bar "), emit)
	if want := []Event{{Key: ' ', Word: true}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want one word completion", got)
	}
}

func TestWordCompletionIgnoresPasteAndEscapeSequences(t *testing.T) {
	input := NewInput(true)
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	input.Observe([]byte("\x1b[200~pasted word \x1b[201~"), emit)
	input.Observe([]byte("\x1b[A "), emit)
	if len(got) != 0 {
		t.Fatalf("got %+v, want no word sounds for paste or navigation", got)
	}
}

func TestWordCompletionOffEmitsNoSpaceOrTab(t *testing.T) {
	input := NewInput(false)
	var got []Event
	emit := func(e Event) { got = append(got, e) }
	input.Observe([]byte("word \t{;}\r"), emit)
	if want := []byte{'{', ';', '}', '\r'}; !reflect.DeepEqual(emittedKeys(got), want) {
		t.Fatalf("got %q, want %q", emittedKeys(got), want)
	}
}
