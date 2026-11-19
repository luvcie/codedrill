package music

import (
	"context"
	"errors"
	"math/rand/v2"
)

type Song struct {
	Title string
	Path  string
}

type PlaybackState struct {
	Title   string `json:"title"`
	Loop    bool   `json:"loop"`
	Shuffle bool   `json:"shuffle"`
}

type playlist struct {
	songs   []Song
	order   []int
	at      int
	loop    bool
	shuffle bool
}

func newPlaylist(songs []Song, shuffle bool) playlist {
	p := playlist{songs: songs, order: make([]int, len(songs)), shuffle: shuffle}
	for i := range p.order {
		p.order[i] = i
	}
	if shuffle {
		rand.Shuffle(len(p.order), func(i, j int) { p.order[i], p.order[j] = p.order[j], p.order[i] })
	}
	return p
}

func (p *playlist) song() Song { return p.songs[p.order[p.at]] }

func (p *playlist) state() PlaybackState {
	return PlaybackState{Title: p.song().Title, Loop: p.loop, Shuffle: p.shuffle}
}

func (p *playlist) step(delta int) {
	p.at = (p.at + delta + len(p.order)) % len(p.order)
}

func (p *playlist) apply(command Command) (restart bool, err error) {
	switch command {
	case GetStatus:
	case Previous:
		p.step(-1)
		return true, nil
	case Next:
		p.step(1)
		return true, nil
	case ToggleLoop:
		p.loop = !p.loop
	case ToggleShuffle:
		current := p.order[p.at]
		p.shuffle = !p.shuffle
		for i := range p.order {
			p.order[i] = i
		}
		p.at = current
		if p.shuffle {
			p.order[0], p.order[current] = p.order[current], p.order[0]
			rand.Shuffle(len(p.order)-1, func(i, j int) {
				p.order[i+1], p.order[j+1] = p.order[j+1], p.order[i+1]
			})
			p.at = 0
		}
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
		var trackCtx context.Context
		trackCtx, cancel = context.WithCancel(ctx)
		finished = make(chan error, 1)
		path, result := p.song().Path, finished
		go func() { result <- play(trackCtx, path) }()
	}
	start()
	failures := 0
	for {
		select {
		case <-ctx.Done():
			cancel()
			<-finished
			return nil
		case err := <-finished:
			cancel()
			if ctx.Err() != nil {
				return nil
			}
			if err != nil {
				failures++
				if failures >= len(songs) {
					return err
				}
			} else {
				failures = 0
			}
			if !p.loop || err != nil {
				p.step(1)
			}
			start()
		case request := <-requests:
			restart, err := p.apply(request.command)
			if restart {
				cancel()
				<-finished
				failures = 0
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
