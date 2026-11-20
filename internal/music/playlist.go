package music

import (
	"context"
	"errors"
	"math/rand/v2"
	"slices"
)

type Song struct {
	Title string
	Path  string
	Label string
}

type Filter int

const (
	All Filter = iota
	CalmOnly
	BattleOnly
)

func (f Filter) String() string {
	switch f {
	case CalmOnly:
		return "Calm only"
	case BattleOnly:
		return "Battle only"
	}
	return "All"
}

type PlaybackState struct {
	Title   string `json:"title"`
	Loop    bool   `json:"loop"`
	Shuffle bool   `json:"shuffle"`
	Filter  Filter `json:"filter"`
	Message string `json:"message,omitempty"`
}

type playlist struct {
	songs      []Song
	order      []int
	at         int
	loop       bool
	shuffle    bool
	filter     Filter
	unplayable []bool
}

func newPlaylist(songs []Song, shuffle bool) playlist {
	p := playlist{songs: songs, shuffle: shuffle, unplayable: make([]bool, len(songs))}
	p.rebuild(-1)
	return p
}

func (p *playlist) current() int {
	if len(p.order) == 0 {
		return -1
	}
	return p.order[p.at]
}

func (p *playlist) song() Song {
	if current := p.current(); current >= 0 {
		return p.songs[current]
	}
	return Song{}
}

func (p *playlist) rebuild(current int) {
	p.order = p.order[:0]
	for i, song := range p.songs {
		if p.unplayable[i] || p.filter == CalmOnly && song.Label != "Calm" || p.filter == BattleOnly && song.Label != "Battle" {
			continue
		}
		p.order = append(p.order, i)
	}
	if p.shuffle {
		rand.Shuffle(len(p.order), func(i, j int) { p.order[i], p.order[j] = p.order[j], p.order[i] })
	}
	p.at = max(0, slices.Index(p.order, current))
}

func (p *playlist) state() PlaybackState {
	state := PlaybackState{Title: p.song().Title, Loop: p.loop, Shuffle: p.shuffle, Filter: p.filter}
	if len(p.order) == 0 {
		state.Message = "No playable tracks match this filter."
	}
	return state
}

func (p *playlist) step(delta int) {
	if len(p.order) > 0 {
		p.at = (p.at + delta + len(p.order)) % len(p.order)
	}
}

func (p *playlist) apply(command Command) (restart bool, err error) {
	switch command {
	case GetStatus:
	case Previous:
		p.step(-1)
		return len(p.order) > 0, nil
	case Next:
		p.step(1)
		return len(p.order) > 0, nil
	case ToggleLoop:
		p.loop = !p.loop
	case ToggleShuffle:
		current := p.current()
		p.shuffle = !p.shuffle
		p.rebuild(current)
	case NextFilter, PreviousFilter:
		current := p.current()
		delta := 1
		if command == PreviousFilter {
			delta = -1
		}
		p.filter = (p.filter + Filter(delta) + 3) % 3
		p.rebuild(current)
		return p.current() != current, nil
	default:
		return false, errors.New("unknown music command")
	}
	return false, nil
}

// One loop owns the queue; a track is stopped before another starts.
func runPlaylist(ctx context.Context, songs []Song, shuffle bool, socket string, play func(context.Context, string) error) error {
	if len(songs) == 0 {
		return errors.New("no selected tracks are available; choose or download songs in Settings → Music")
	}
	if ctx.Err() != nil {
		return nil
	}
	requests := make(chan controlRequest)
	stop, err := serveControls(ctx, socket, requests)
	if err != nil {
		return err
	}
	defer stop()
	p := newPlaylist(songs, shuffle)
	var cancel context.CancelFunc
	var finished chan error
	start := func() {
		if len(p.order) == 0 {
			return
		}
		var trackCtx context.Context
		trackCtx, cancel = context.WithCancel(ctx)
		finished = make(chan error, 1)
		path, result := p.song().Path, finished
		go func() { result <- play(trackCtx, path) }()
	}
	stopTrack := func() {
		if cancel != nil {
			cancel()
			<-finished
			cancel, finished = nil, nil
		}
	}
	start()
	for {
		select {
		case <-ctx.Done():
			stopTrack()
			return nil
		case err := <-finished:
			cancel()
			cancel, finished = nil, nil
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				p.unplayable[p.current()] = true
				p.step(1)
				p.rebuild(p.current())
			} else if !p.loop {
				p.step(1)
			}
			start()
		case request := <-requests:
			restart, err := p.apply(request.command)
			if restart {
				stopTrack()
				start()
			}
			response := controlResponse{State: p.state()}
			if err != nil {
				response.Error = err.Error()
			}
			request.reply <- response
		}
	}
}
