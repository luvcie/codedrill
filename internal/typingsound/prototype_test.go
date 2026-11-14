package typingsound

import (
	"reflect"
	"testing"
	"time"
)

func TestPackPlayerKeyMappingAndCombo(t *testing.T) {
	now := time.Unix(100, 0)
	player := &PackPlayer{
		mapping:    DefaultMapping(),
		lockIndex:  0,
		quadIndex:  1,
		comboBase:  2,
		comboCount: 3,
		now:        func() time.Time { return now },
	}
	// Ordinary keys are silent and do not advance the combo.
	for _, key := range []byte("a1 z") {
		if got := player.index(key); got != -1 {
			t.Fatalf("key %q: got clip %d, want silence", key, got)
		}
	}
	// Opening brace and Enter play lock without advancing the combo.
	for _, key := range []byte{'{', '\r', '\n'} {
		if got := player.index(key); got != 0 {
			t.Fatalf("lock key %q: got clip %d, want 0", key, got)
		}
	}
	// Semicolons advance and cap at the last combo.
	for i, want := range []int{2, 3, 4, 4} {
		now = now.Add(time.Second)
		if got := player.index(';'); got != want {
			t.Fatalf("combo %d: got clip %d, want %d", i, got, want)
		}
	}
	if got := player.index('}'); got != 1 {
		t.Fatalf("closing brace: got clip %d, want quad 1", got)
	}
	// Lock and quad do not reset the combo.
	now = now.Add(time.Second)
	if got := player.index(';'); got != 4 {
		t.Fatalf("combo after lock/quad: got %d, want the cap", got)
	}
	// Twenty seconds without a combo action resets it.
	now = now.Add(21 * time.Second)
	if got := player.index(';'); got != 2 {
		t.Fatalf("combo after inactivity: got %d, want the first combo", got)
	}
}

func TestCustomMapping(t *testing.T) {
	m, err := ParseMapping("{", ";", "}")
	if err != nil {
		t.Fatal(err)
	}
	if m.Action('\r') == ActionLock {
		t.Fatal("a custom lock list should replace the default Enter lock")
	}
	if m.Action('[') != ActionSilent {
		t.Fatal("unmapped key is not silent")
	}
	m, err = ParseMapping("enter,space", "none", "}")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []byte{'\r', '\n', ' '} {
		if m.Action(key) != ActionLock {
			t.Fatalf("key %q is not a lock", key)
		}
	}
	if m.Action(';') != ActionSilent {
		t.Fatal("none did not clear the combo keys")
	}
	if _, err := ParseMapping("ab", "", ""); err == nil {
		t.Fatal("accepted a multi-character key token")
	}
}

func TestPrototypeUnavailableServer(t *testing.T) {
	t.Setenv("PULSE_SERVER", "unix:/nonexistent/codedrill-pulse")
	if p, err := NewPrototype(0.7); err == nil {
		p.Close()
		t.Fatal("expected connection failure")
	}
}

func TestEngineDoesNotBlockOnFullQueue(t *testing.T) {
	engine := &Engine{events: make(chan int, 1), clips: []Clip{{1}, {2}}}
	engine.Play(0)
	engine.Play(1)
	if len(engine.events) != 1 {
		t.Fatal("expected the bounded queue to retain one event")
	}
	if got := <-engine.events; got != 0 {
		t.Fatalf("queue kept event %d, want the first", got)
	}
}

func TestEngineDropsInvalidClips(t *testing.T) {
	engine := &Engine{events: make(chan int, 2), clips: []Clip{nil, {1}}}
	engine.Play(-1)
	engine.Play(2)
	engine.Play(0)
	if len(engine.events) != 0 {
		t.Fatal("queued an invalid clip")
	}
}

func TestWordCompletionPlaysLockWithoutAdvancingCombo(t *testing.T) {
	player := &PackPlayer{
		mapping:    DefaultMapping(),
		lockIndex:  3,
		quadIndex:  7,
		comboBase:  10,
		comboCount: 4,
	}
	if got := player.eventIndex(Event{Key: ' ', Word: true}); got != 3 {
		t.Fatalf("word completion played clip %d, want the lock clip 3", got)
	}
	if got := player.eventIndex(Event{Key: '\t', Word: true}); got != 3 {
		t.Fatalf("tab completion played clip %d, want the lock clip 3", got)
	}
	// A word completion does not touch the combo counter.
	if player.combo.count != 0 {
		t.Fatalf("word completion advanced the combo to %d", player.combo.count)
	}
	// Ordinary and trigger mappings still work.
	for key, want := range map[byte]int{'{': 3, '\r': 3, '}': 7, ';': 10, 'a': -1} {
		if got := player.eventIndex(Event{Key: key}); got != want {
			t.Errorf("key %q played clip %d, want %d", key, got, want)
		}
	}
}

func TestWordCompletionEmitsOneLockPerWordBoundary(t *testing.T) {
	input := NewInput(true)
	player := &PackPlayer{mapping: DefaultMapping(), lockIndex: 2, quadIndex: 5, comboBase: 8, comboCount: 4}
	var played []int
	emit := func(event Event) {
		if i := player.eventIndex(event); i >= 0 {
			played = append(played, i)
		}
	}
	input.Observe([]byte("value  other\t"), emit)
	if want := []int{2, 2}; !reflect.DeepEqual(played, want) {
		t.Fatalf("played %v, want one lock per word boundary %v", played, want)
	}
	if player.combo.count != 0 {
		t.Fatalf("word completions advanced the combo to %d", player.combo.count)
	}
}

func TestEngineMixesOverlappingVoices(t *testing.T) {
	engine := &Engine{events: make(chan int, 4), clips: []Clip{{100, 100, 100, 100, 100}, {50, 50}}}
	engine.Play(0)
	engine.Play(1)
	buf := make([]int16, 4)
	if n, err := engine.read(buf); err != nil || n != 4 {
		t.Fatalf("read: %d %v", n, err)
	}
	want := []int16{150, 150, 100, 100}
	for i := range want {
		if buf[i] != want[i] {
			t.Fatalf("mixed %v, want %v", buf, want)
		}
	}
	if len(engine.voices) != 1 {
		t.Fatalf("%d voices left, want the unfinished one", len(engine.voices))
	}
	// A new key starts immediately instead of waiting for the old clip.
	engine.Play(1)
	buf = make([]int16, 2)
	engine.read(buf)
	if buf[0] != 150 {
		t.Fatalf("new voice did not mix immediately: %v", buf)
	}
}
