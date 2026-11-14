package typingsound

import (
	"os"
	"time"

	"github.com/jfreymuth/pulse"
)

// Engine plays preloaded mono clips through one persistent PulseAudio stream.
// Play never blocks the caller, and no process is started per key.
type Engine struct {
	client *pulse.Client
	stream *pulse.PlaybackStream
	events chan int
	clips  []Clip
	voices []engineVoice
	mix    []int32
}

type engineVoice struct {
	clip Clip
	pos  int
}

const engineQueue = 32
const maxVoices = 8

// NewEngine connects to the audio server and starts a silent stream. The
// volume is applied to the clips once, so playback costs no scaling per read.
func NewEngine(clips []Clip, volume float64) (*Engine, error) {
	volume = max(0, min(1, volume))
	scaled := make([]Clip, len(clips))
	for i, clip := range clips {
		if len(clip) == 0 {
			continue
		}
		copyClip := make(Clip, len(clip))
		for j, sample := range clip {
			copyClip[j] = int16(float64(sample) * volume)
		}
		scaled[i] = copyClip
	}
	client, err := pulse.NewClient(pulse.ClientApplicationName("codedrill typing sounds"), pulse.ClientTimeout(2*time.Second))
	if err != nil {
		return nil, err
	}
	engine := &Engine{client: client, events: make(chan int, engineQueue), clips: scaled}
	opts := []pulse.PlaybackOption{
		pulse.PlaybackMono,
		pulse.PlaybackSampleRate(sampleRate),
		pulse.PlaybackLatency(.03),
		pulse.PlaybackMediaName("Typing sounds"),
	}
	if name := os.Getenv("PULSE_SINK"); name != "" {
		sink, err := client.SinkByID(name)
		if err != nil {
			client.Close()
			return nil, err
		}
		opts = append(opts, pulse.PlaybackSink(sink))
	}
	engine.stream, err = client.NewPlayback(pulse.Int16Reader(engine.read), opts...)
	if err != nil {
		client.Close()
		return nil, err
	}
	engine.stream.Start()
	return engine, nil
}

// Play queues one clip without waiting. When the queue is full the newest
// event is dropped, so input forwarding never blocks on audio.
func (e *Engine) Play(index int) {
	if index < 0 || index >= len(e.clips) || len(e.clips[index]) == 0 {
		return
	}
	select {
	case e.events <- index:
	default:
	}
}

func (e *Engine) read(buf []int16) (int, error) {
	for {
		select {
		case index := <-e.events:
			if len(e.clips[index]) == 0 {
				continue
			}
			if len(e.voices) >= maxVoices {
				e.voices = e.voices[1:]
			}
			e.voices = append(e.voices, engineVoice{clip: e.clips[index]})
		default:
			goto mix
		}
	}
mix:
	if cap(e.mix) < len(buf) {
		e.mix = make([]int32, len(buf))
	}
	mix := e.mix[:len(buf)]
	clear(mix)
	live := e.voices[:0]
	for _, voice := range e.voices {
		voice.pos += copySamples(mix, voice.clip[voice.pos:])
		if voice.pos < len(voice.clip) {
			live = append(live, voice)
		}
	}
	e.voices = live
	for i, sum := range mix {
		buf[i] = int16(max(-32768, min(32767, sum)))
	}
	return len(buf), nil
}

// copySamples adds src into dst and returns how many samples were added.
func copySamples(dst []int32, src Clip) int {
	n := min(len(dst), len(src))
	for i := 0; i < n; i++ {
		dst[i] += int32(src[i])
	}
	return n
}

func (e *Engine) Close() {
	if e.stream != nil {
		e.stream.Close()
	}
	if e.client != nil {
		e.client.Close()
	}
}
