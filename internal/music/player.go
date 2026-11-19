package music

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/pulse"
	"github.com/luvcie/codedrill/internal/storage"
)

// Player belongs to the drill, outside the editor and its Zellij panes.
type Player struct {
	cancel context.CancelFunc
	done   chan struct{}
	err    error // Read only after done closes.
}

func SelectedSongs(ids []string, folder string) []Song {
	var songs []Song
	for _, source := range []string{SprintZen, TetraLeague, Holidays} {
		for _, track := range Tracks(source) {
			if slices.Contains(ids, track.ID) && Cached(track) {
				songs = append(songs, Song{Title: track.Name, Path: Path(track)})
			}
		}
	}
	if folder != "" {
		tracks, _ := LocalTracks(folder)
		for _, track := range tracks {
			if slices.Contains(ids, track.ID) {
				songs = append(songs, Song{Title: track.Name, Path: track.URL})
			}
		}
	}
	return songs
}

func Start(songs []Song, volume float64, shuffle bool, socket string) *Player {
	ctx, cancel := context.WithCancel(context.Background())
	p := &Player{cancel: cancel, done: make(chan struct{})}
	go func() { defer close(p.done); p.err = play(ctx, songs, max(0, min(1, volume)), shuffle, socket) }()
	return p
}

func (p *Player) Close() error {
	p.cancel()
	<-p.done
	return p.err
}

func play(ctx context.Context, songs []Song, volume float64, shuffle bool, socket string) error {
	if len(songs) == 0 {
		return errors.New("no selected tracks are available; choose or download songs in Settings → Music")
	}
	client, err := pulse.NewClient(pulse.ClientApplicationName("codedrill music"), pulse.ClientTimeout(2*time.Second))
	if err != nil {
		return err
	}
	defer client.Close()
	live := storage.FollowVolume(ctx, volume, storage.Config.MusicVolume)
	return runPlaylist(ctx, songs, shuffle, socket, func(trackCtx context.Context, path string) error {
		return playTrack(trackCtx, client, path, live)
	})
}

func playTrack(ctx context.Context, client *pulse.Client, path string, volume *storage.LiveVolume) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	decoder, err := mp3.NewDecoder(file)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	reader := &pcmReader{reader: decoder, volume: volume}
	opts := []pulse.PlaybackOption{pulse.PlaybackStereo, pulse.PlaybackSampleRate(decoder.SampleRate()), pulse.PlaybackLatency(.1), pulse.PlaybackMediaName("Drill music")}
	if name := os.Getenv("PULSE_SINK"); name != "" {
		sink, err := client.SinkByID(name)
		if err != nil {
			return err
		}
		opts = append(opts, pulse.PlaybackSink(sink))
	}
	stream, err := client.NewPlayback(pulse.Int16Reader(reader.read), opts...)
	if err != nil {
		return err
	}
	defer stream.Close()
	stream.Start()
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for stream.Running() {
		select {
		case <-ctx.Done():
			stream.Pause()
			return nil
		case <-tick.C:
		}
	}
	// Let the final buffered samples play before replacing this stream.
	timer := time.NewTimer(150 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
	return stream.Error()
}

type pcmReader struct {
	reader io.Reader
	volume *storage.LiveVolume
	buf    []byte
}

func (r *pcmReader) read(out []int16) (int, error) {
	if cap(r.buf) < len(out)*2 {
		r.buf = make([]byte, len(out)*2)
	}
	n, err := io.ReadFull(r.reader, r.buf[:len(out)*2])
	gain := r.volume.Gain()
	for i := 0; i < n/2; i++ {
		out[i] = int16(float64(int16(binary.LittleEndian.Uint16(r.buf[i*2:]))) * gain)
	}
	// Pulse drops data returned alongside an error; deliver the last samples first.
	if n > 0 {
		return n / 2, nil
	}
	if err == io.EOF {
		err = pulse.EndOfData
	}
	return 0, err
}
