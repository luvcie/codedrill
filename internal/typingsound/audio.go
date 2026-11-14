package typingsound

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/hajimehoshi/go-mp3"
	"github.com/jfreymuth/oggvorbis"
)

// Clip is mono 16-bit PCM at the playback sample rate.
type Clip []int16

const sampleRate = 48000

// Atlas timings are capped at ten minutes, so audio after that can never play.
// The limit also keeps a highly compressed or malformed recording from using
// unbounded memory while it decodes.
const (
	maxDecodedSeconds = 600
	maxDecodedSamples = maxDecodedSeconds * sampleRate
)

func decodedSampleLimit(rate int) int {
	return min(maxDecodedSamples, rate*maxDecodedSeconds)
}

func capDecodedLength(length int) int {
	return min(length, maxDecodedSamples)
}

// decodeAudio turns an Ogg Vorbis, MP3 or WAV recording into mono samples at
// its original sample rate. Unknown formats are rejected rather than guessed.
func decodeAudio(data []byte) (Clip, int, error) {
	if len(data) < 12 {
		return nil, 0, errors.New("audio is too short")
	}
	switch {
	case bytes.HasPrefix(data, []byte("OggS")):
		return decodeOgg(data)
	case bytes.HasPrefix(data, []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WAVE")):
		return decodeWAV(data)
	case bytes.HasPrefix(data, []byte("ID3")) || (data[0] == 0xff && data[1]&0xe0 == 0xe0):
		return decodeMP3(data)
	default:
		return nil, 0, fmt.Errorf("unsupported audio format")
	}
}

func decodeOgg(data []byte) (Clip, int, error) {
	reader, err := oggvorbis.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, 0, fmt.Errorf("decode ogg: %w", err)
	}
	rate, channels := reader.SampleRate(), reader.Channels()
	if rate <= 0 || channels <= 0 || channels > 8 {
		return nil, 0, errors.New("invalid ogg stream")
	}
	buf := make([]float32, 4096*channels)
	limit := decodedSampleLimit(rate)
	var out Clip
	for {
		n, readErr := reader.Read(buf)
		for i := 0; i+channels <= n; i += channels {
			var sum float32
			for c := 0; c < channels; c++ {
				sum += buf[i+c]
			}
			out = append(out, toInt16(float64(sum)/float64(channels)))
		}
		if len(out) >= limit {
			out = out[:limit]
			break
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, 0, fmt.Errorf("decode ogg: %w", readErr)
		}
		if n == 0 {
			break
		}
	}
	if len(out) == 0 {
		return nil, 0, errors.New("ogg stream is empty")
	}
	return out, rate, nil
}

func decodeMP3(data []byte) (Clip, int, error) {
	decoder, err := mp3.NewDecoder(bytes.NewReader(data))
	if err != nil {
		return nil, 0, fmt.Errorf("decode mp3: %w", err)
	}
	rate := decoder.SampleRate()
	if rate <= 0 {
		return nil, 0, errors.New("invalid mp3 stream")
	}
	// go-mp3 always decodes to interleaved stereo 16-bit little-endian.
	buf := make([]byte, 4096*4)
	limit := decodedSampleLimit(rate)
	var out Clip
	for {
		n, readErr := decoder.Read(buf)
		for i := 0; i+3 < n; i += 4 {
			left := int16(binary.LittleEndian.Uint16(buf[i:]))
			right := int16(binary.LittleEndian.Uint16(buf[i+2:]))
			out = append(out, int16((int32(left)+int32(right))/2))
		}
		if len(out) >= limit {
			out = out[:limit]
			break
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, 0, fmt.Errorf("decode mp3: %w", readErr)
		}
		if n == 0 {
			break
		}
	}
	if len(out) == 0 {
		return nil, 0, errors.New("mp3 stream is empty")
	}
	return out, rate, nil
}

// decodeWAV reads uncompressed or float PCM from a RIFF/WAVE file. Compressed
// WAV codecs are not supported.
func decodeWAV(data []byte) (Clip, int, error) {
	if len(data) < 12 {
		return nil, 0, errors.New("wav is too short")
	}
	var (
		format, channels, bits int
		rate                   int
		samples                Clip
	)
	for pos := 12; pos+8 <= len(data); {
		id := string(data[pos : pos+4])
		size := int(binary.LittleEndian.Uint32(data[pos+4:]))
		body := pos + 8
		if size < 0 || body+size > len(data) {
			size = len(data) - body
		}
		switch id {
		case "fmt ":
			if size < 16 {
				return nil, 0, errors.New("invalid wav format chunk")
			}
			format = int(binary.LittleEndian.Uint16(data[body:]))
			channels = int(binary.LittleEndian.Uint16(data[body+2:]))
			rate = int(binary.LittleEndian.Uint32(data[body+4:]))
			bits = int(binary.LittleEndian.Uint16(data[body+14:]))
			if format == 0xfffe && size >= 26 {
				format = int(binary.LittleEndian.Uint16(data[body+24:]))
			}
		case "data":
			samples = decodeWAVData(data[body:body+size], format, channels, bits)
		}
		pos = body + size + size%2
	}
	if channels <= 0 || channels > 8 || rate <= 0 {
		return nil, 0, errors.New("invalid wav header")
	}
	if len(samples) == 0 {
		return nil, 0, errors.New("wav has no audio data")
	}
	// decodeWAVData already mixed channels; drop any trailing partial group.
	usable := len(samples) / channels * channels
	if usable == 0 {
		return nil, 0, errors.New("wav has no audio data")
	}
	out := make(Clip, 0, usable/channels)
	for i := 0; i < usable; i += channels {
		var sum int32
		for c := 0; c < channels; c++ {
			sum += int32(samples[i+c])
		}
		out = append(out, int16(sum/int32(channels)))
	}
	return out, rate, nil
}

func decodeWAVData(data []byte, format, channels, bits int) Clip {
	if channels <= 0 {
		return nil
	}
	width := bits / 8
	if width <= 0 {
		return nil
	}
	count := len(data) / width
	out := make(Clip, 0, count)
	for i := 0; i+width <= len(data); i += width {
		var sample int16
		switch {
		case format == 3 && bits == 32:
			sample = toInt16(float64(math.Float32frombits(binary.LittleEndian.Uint32(data[i:]))))
		case format == 3 && bits == 64:
			sample = toInt16(math.Float64frombits(binary.LittleEndian.Uint64(data[i:])))
		case format == 1 && bits == 8:
			sample = int16(int(data[i])-128) << 8
		case format == 1 && bits == 16:
			sample = int16(binary.LittleEndian.Uint16(data[i:]))
		case format == 1 && bits == 24:
			v := int32(data[i]) | int32(data[i+1])<<8 | int32(data[i+2])<<16
			if v&0x800000 != 0 {
				v -= 1 << 24
			}
			sample = int16(v >> 8)
		case format == 1 && bits == 32:
			sample = int16(int32(binary.LittleEndian.Uint32(data[i:])) >> 16)
		default:
			return nil
		}
		out = append(out, sample)
	}
	return out
}

func toInt16(v float64) int16 {
	v = math.Max(-1, math.Min(1, v))
	return int16(v * 32767)
}

// resample linearly interpolates mono samples from one rate to another.
func resample(in Clip, from, to int) Clip {
	if from <= 0 || to <= 0 || len(in) == 0 {
		return nil
	}
	if from == to {
		return in
	}
	length := capDecodedLength(int(int64(len(in)) * int64(to) / int64(from)))
	if length <= 0 {
		return nil
	}
	out := make(Clip, length)
	ratio := float64(from) / float64(to)
	for i := range out {
		pos := float64(i) * ratio
		left := int(pos)
		frac := pos - float64(left)
		if left+1 >= len(in) {
			out[i] = in[len(in)-1]
			continue
		}
		out[i] = int16(float64(in[left])*(1-frac) + float64(in[left+1])*frac)
	}
	return out
}
