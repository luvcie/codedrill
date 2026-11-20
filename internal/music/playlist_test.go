package music

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

var testSongs = []Song{{Title: "First song", Path: "first.mp3"}, {Title: "Second song", Path: "second.mp3"}, {Title: "Third song", Path: "third.mp3"}}

type playingTrack struct {
	path      string
	finish    chan error
	cancelled chan struct{}
}

func startTestPlaylist(t *testing.T, shuffle bool, songs ...Song) (string, <-chan playingTrack, context.CancelFunc, <-chan error) {
	t.Helper()
	if len(songs) == 0 {
		songs = testSongs
	}
	ctx, cancel := context.WithCancel(context.Background())
	socket := filepath.Join(t.TempDir(), "music.sock")
	started := make(chan playingTrack, 16)
	done := make(chan error, 1)
	go func() {
		done <- runPlaylist(ctx, songs, shuffle, socket, func(ctx context.Context, path string) error {
			track := playingTrack{path: path, finish: make(chan error, 1), cancelled: make(chan struct{})}
			defer close(track.cancelled)
			started <- track
			select {
			case <-ctx.Done():
				return nil
			case err := <-track.finish:
				return err
			}
		})
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("player did not stop")
		}
	})
	return socket, started, cancel, done
}

func takeTrack(t *testing.T, started <-chan playingTrack, want string) playingTrack {
	t.Helper()
	select {
	case track := <-started:
		if want != "" && track.path != want {
			t.Fatalf("playing %q, want %q", track.path, want)
		}
		return track
	case <-time.After(2 * time.Second):
		t.Fatal("player did not start the next song")
		return playingTrack{}
	}
}

func TestPlaybackControls(t *testing.T) {
	socket, started, _, _ := startTestPlaylist(t, false)
	first := takeTrack(t, started, "first.mp3")
	command := func(command Command, want string) PlaybackState {
		t.Helper()
		state, err := Control(socket, command)
		if err != nil || state.Title != want {
			t.Fatalf("%s: %+v, %v; want %q", command, state, err, want)
		}
		return state
	}
	command(GetStatus, "First song")
	command(Next, "Second song")
	takeTrack(t, started, "second.mp3")
	select {
	case <-first.cancelled:
	default:
		t.Fatal("Next did not stop the old song")
	}
	command(Previous, "First song")
	takeTrack(t, started, "first.mp3")
	command(Previous, "Third song")
	third := takeTrack(t, started, "third.mp3")
	if !command(ToggleLoop, "Third song").Loop {
		t.Fatal("loop was not enabled")
	}
	select {
	case <-third.cancelled:
		t.Fatal("toggling loop interrupted the current song")
	default:
	}
	third.finish <- nil
	takeTrack(t, started, "third.mp3")
	command(Next, "First song")
	takeTrack(t, started, "first.mp3")
	if !command(ToggleShuffle, "First song").Shuffle {
		t.Fatal("session shuffle was not enabled")
	}
	seen := map[string]bool{}
	for range len(testSongs) {
		state, err := Control(socket, Next)
		if err != nil || seen[state.Title] {
			t.Fatalf("shuffle repeated a song before completing the playlist: %+v, %v", state, err)
		}
		seen[state.Title] = true
		takeTrack(t, started, "")
	}
	if command(ToggleShuffle, "First song").Shuffle {
		t.Fatal("session shuffle was not disabled")
	}
	if command(ToggleLoop, "First song").Loop {
		t.Fatal("loop was not disabled")
	}
	command(Next, "Second song")
	second := takeTrack(t, started, "second.mp3")
	second.finish <- nil
	takeTrack(t, started, "third.mp3")
	if _, err := Control(socket, "invalid"); err == nil {
		t.Fatal("invalid command was accepted")
	}
	command(GetStatus, "Third song")
}

func TestPlaybackShuffleAndSessionIsolation(t *testing.T) {
	for range 10 {
		p := newPlaylist(testSongs, true)
		order := slices.Clone(p.order)
		slices.Sort(order)
		if !reflect.DeepEqual(order, []int{0, 1, 2}) || !p.state().Shuffle {
			t.Fatalf("starting shuffle lost songs: %+v", p)
		}
		current := p.song()
		position := slices.Index(testSongs, current)
		if _, err := p.apply(ToggleShuffle); err != nil || p.song() != current || p.shuffle {
			t.Fatal("disabling shuffle changed the current song")
		}
		p.step(1)
		if p.song() != testSongs[(position+1)%len(testSongs)] {
			t.Fatal("disabling shuffle did not restore selection order")
		}
	}
	firstSocket, firstStarted, stopFirst, firstDone := startTestPlaylist(t, true)
	takeTrack(t, firstStarted, "")
	secondSocket, secondStarted, _, _ := startTestPlaylist(t, false)
	takeTrack(t, secondStarted, "first.mp3")
	state, err := Control(firstSocket, GetStatus)
	if err != nil || !state.Shuffle || state.Loop {
		t.Fatalf("saved shuffle was not applied to the new session: %+v, %v", state, err)
	}
	if _, err := Control(firstSocket, Next); err != nil {
		t.Fatal(err)
	}
	state, err = Control(secondSocket, GetStatus)
	if err != nil || state.Title != "First song" || state.Shuffle {
		t.Fatalf("controls affected another drill: %+v, %v", state, err)
	}
	stopFirst()
	select {
	case <-firstDone:
	case <-time.After(2 * time.Second):
		t.Fatal("player did not stop")
	}
	if _, err := os.Stat(firstSocket); !os.IsNotExist(err) {
		t.Fatalf("stopped player left its socket behind: %v", err)
	}
	if _, err := Control(firstSocket, GetStatus); !errors.Is(err, ErrNoPlayer) {
		t.Fatalf("stopped player was still available: %v", err)
	}
	if _, err := Control(secondSocket, GetStatus); err != nil {
		t.Fatalf("stopping one player stopped another: %v", err)
	}
}

func TestPlaybackCancelledBeforeStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	socket := filepath.Join(t.TempDir(), "music.sock")
	err := runPlaylist(ctx, testSongs, false, socket, func(context.Context, string) error {
		t.Error("stopped drill started a song")
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(socket); !os.IsNotExist(err) {
		t.Fatalf("stopped drill opened playback controls: %v", err)
	}
}

func TestPlaybackSkipsBrokenSongsEvenWhenLooping(t *testing.T) {
	socket, started, _, _ := startTestPlaylist(t, false)
	track := takeTrack(t, started, "first.mp3")
	if _, err := Control(socket, ToggleLoop); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("unreadable audio")
	track.finish <- failure
	track = takeTrack(t, started, "second.mp3")
	track.finish <- failure
	track = takeTrack(t, started, "third.mp3")
	track.finish <- failure
	deadline := time.Now().Add(2 * time.Second)
	for {
		state, err := Control(socket, GetStatus)
		if err != nil {
			t.Fatal(err)
		}
		if state.Title == "" && state.Message != "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("broken playlist did not stop playback")
		}
		time.Sleep(time.Millisecond)
	}
	select {
	case track := <-started:
		t.Fatalf("broken song retried: %s", track.path)
	default:
	}
}

func TestPlaybackFiltersDuringSong(t *testing.T) {
	songs := []Song{
		{Title: "Calm one", Path: "calm1", Label: "Calm"},
		{Title: "Calm two", Path: "calm2", Label: "Calm"},
		{Title: "Battle one", Path: "battle1", Label: "Battle"},
		{Title: "Battle two", Path: "battle2", Label: "Battle"},
		{Title: "Local", Path: "local"},
	}
	socket, started, _, _ := startTestPlaylist(t, false, songs...)
	current := takeTrack(t, started, "calm1")
	control := func(command Command, filter Filter) PlaybackState {
		t.Helper()
		state, err := Control(socket, command)
		if err != nil || state.Filter != filter || state.Message != "" {
			t.Fatalf("%s: %+v, %v", command, state, err)
		}
		return state
	}
	control(GetStatus, All)
	control(NextFilter, CalmOnly)
	select {
	case <-current.cancelled:
		t.Fatal("matching song was interrupted")
	default:
	}
	current.finish <- nil
	current = takeTrack(t, started, "calm2")
	control(ToggleLoop, CalmOnly)
	control(NextFilter, BattleOnly)
	select {
	case <-current.cancelled:
	default:
		t.Fatal("excluded song was not stopped before changing filter")
	}
	current = takeTrack(t, started, "battle1")
	current.finish <- nil
	takeTrack(t, started, "battle1")
	for _, command := range []Command{Next, Previous, ToggleShuffle, Next, Previous, Next, Next} {
		state := control(command, BattleOnly)
		if !strings.HasPrefix(state.Title, "Battle") || !state.Loop {
			t.Fatalf("control escaped active filter or lost looping: %+v", state)
		}
		if command != ToggleShuffle {
			takeTrack(t, started, "")
		}
	}
	control(PreviousFilter, CalmOnly)
	current = takeTrack(t, started, "")
	if !strings.HasPrefix(current.path, "calm") {
		t.Fatalf("Calm filter played %s", current.path)
	}
	state := control(PreviousFilter, All)
	if !state.Shuffle || !state.Loop {
		t.Fatal("filter changed shuffle or loop")
	}
	select {
	case <-current.cancelled:
		t.Fatal("All interrupted a matching song")
	default:
	}
	seen := map[string]bool{}
	for range len(songs) {
		state := control(Next, All)
		if seen[state.Title] {
			t.Fatalf("shuffle repeated before covering restored selection: %+v", state)
		}
		seen[state.Title] = true
		takeTrack(t, started, "")
	}
	if !seen["Local"] {
		t.Fatal("All did not restore local tracks")
	}
}

func TestEmptyFilterCanResumeAll(t *testing.T) {
	socket, started, _, _ := startTestPlaylist(t, false, Song{Title: "Calm", Path: "calm", Label: "Calm"})
	current := takeTrack(t, started, "calm")
	for range 2 {
		if _, err := Control(socket, NextFilter); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-current.cancelled:
	default:
		t.Fatal("empty filter did not stop the song")
	}
	for _, command := range []Command{GetStatus, Next, Previous, ToggleLoop, ToggleShuffle} {
		state, err := Control(socket, command)
		if err != nil || state.Filter != BattleOnly || state.Title != "" || state.Message == "" {
			t.Fatalf("empty filter: %+v, %v", state, err)
		}
	}
	select {
	case <-started:
		t.Fatal("empty filter started a song")
	default:
	}
	state, err := Control(socket, NextFilter)
	if err != nil || state.Filter != All || state.Title != "Calm" || !state.Shuffle || !state.Loop || state.Message != "" {
		t.Fatalf("All did not resume: %+v, %v", state, err)
	}
	takeTrack(t, started, "calm")
}

func TestFilterWithOnlyUnplayableTracks(t *testing.T) {
	socket, started, _, _ := startTestPlaylist(t, false,
		Song{Title: "Calm", Path: "calm", Label: "Calm"},
		Song{Title: "Broken Battle", Path: "broken", Label: "Battle"})
	takeTrack(t, started, "calm")
	if _, err := Control(socket, PreviousFilter); err != nil {
		t.Fatal(err)
	}
	broken := takeTrack(t, started, "broken")
	broken.finish <- errors.New("invalid MP3")
	deadline := time.Now().Add(2 * time.Second)
	for {
		state, err := Control(socket, GetStatus)
		if err != nil {
			t.Fatal(err)
		}
		if state.Message != "" && state.Title == "" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("broken filter did not stop")
		}
		time.Sleep(time.Millisecond)
	}
	if _, err := Control(socket, NextFilter); err != nil {
		t.Fatal(err)
	}
	takeTrack(t, started, "calm")
	for range 2 {
		if _, err := Control(socket, Next); err != nil {
			t.Fatal(err)
		}
		takeTrack(t, started, "calm")
	}
}
