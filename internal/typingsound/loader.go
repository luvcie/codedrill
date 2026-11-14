package typingsound

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const downloadTimeout = 45 * time.Second

// CacheDir is where downloaded exports are kept between sessions.
func CacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "codedrill", "typing-sounds")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// DownloadPack returns the local path of a pack's export, downloading it from
// the community host on first use. The file is validated before it is stored.
func DownloadPack(info PackInfo) (string, error) {
	dir, err := CacheDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, info.ID+".tpse")
	if file, err := os.Open(path); err == nil {
		ok := false
		if stat, statErr := file.Stat(); statErr == nil && stat.Size() > 0 && stat.Size() <= maxExportBytes {
			head := make([]byte, 1)
			if _, readErr := io.ReadFull(file, head); readErr == nil && head[0] == '{' {
				ok = true
			}
		}
		file.Close()
		if ok {
			return path, nil
		}
		_ = os.Remove(path)
	}
	client := &http.Client{Timeout: downloadTimeout}
	request, err := http.NewRequest(http.MethodGet, info.URL, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", info.Name, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: server returned %s", info.Name, response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxExportBytes+1))
	if err != nil {
		return "", fmt.Errorf("download %s: %w", info.Name, err)
	}
	if len(data) > maxExportBytes {
		return "", fmt.Errorf("download %s: export exceeds 24 MiB", info.Name)
	}
	if _, _, err := ParseExport(data); err != nil {
		return "", fmt.Errorf("download %s: %w", info.Name, err)
	}
	temp, err := os.CreateTemp(dir, info.ID+".*.tmp")
	if err != nil {
		return "", err
	}
	tempPath := temp.Name()
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return "", err
	}
	return path, nil
}

// LoadPack downloads as needed, decodes the export and extracts the lock, quad
// and combo clips. A pack with none of them is an error, so the caller can fall
// back to a plain editor session.
func LoadPack(info PackInfo, volume float64, mapping Mapping) (*PackPlayer, error) {
	sounds, err := loadSounds(info)
	if err != nil {
		return nil, err
	}
	return NewPackPlayer(sounds, volume, mapping)
}

func loadSounds(info PackInfo) (Sounds, error) {
	path, err := DownloadPack(info)
	if err != nil {
		return Sounds{}, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return Sounds{}, err
	}
	if stat.Size() <= 0 || stat.Size() > maxExportBytes {
		_ = os.Remove(path)
		_ = os.Remove(clipCachePath(path))
		return Sounds{}, fmt.Errorf("%s has an invalid export size", info.Name)
	}
	hash, err := fileChecksum(path)
	if err != nil {
		return Sounds{}, err
	}
	if sounds, ok := readClipCache(path, stat.Size(), hash); ok {
		return sounds, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Sounds{}, err
	}
	audio, atlas, err := ParseExport(data)
	if err != nil {
		_ = os.Remove(path)
		_ = os.Remove(clipCachePath(path))
		return Sounds{}, err
	}
	samples, rate, err := decodeAudio(audio)
	if err != nil {
		return Sounds{}, fmt.Errorf("%s: %w", info.Name, err)
	}
	samples = resample(samples, rate, sampleRate)
	sounds := extractSounds(samples, atlas)
	if len(sounds.Lock) == 0 && len(sounds.Quad) == 0 && len(sounds.Combos) == 0 {
		return Sounds{}, fmt.Errorf("%s has no lock, quad or combo sounds", info.Name)
	}
	writeClipCache(path, stat.Size(), hash, sounds)
	return sounds, nil
}

// extractSounds copies the gameplay clips within the per-effect and cumulative
// budgets. A long clip is skipped, not truncated, so a malformed atlas entry
// cannot consume the budget for the remaining effects.
func extractSounds(samples Clip, atlas Atlas) Sounds {
	sounds := Sounds{}
	total := 0
	take := func(names ...string) Clip {
		for _, name := range names {
			clip := atlasClip(samples, atlas, name)
			if len(clip) == 0 || total+len(clip) > maxTotalClipSamples {
				continue
			}
			total += len(clip)
			return clip
		}
		return nil
	}
	sounds.Lock = take("harddrop", "lock")
	sounds.Quad = take("clearquad", "clearline")
	for i := 1; i <= maxComboClips; i++ {
		name := fmt.Sprintf("combo_%d", i)
		if clip := take(name); len(clip) > 0 {
			sounds.Combos = append(sounds.Combos, clip)
		}
	}
	return sounds
}

// Clip caches avoid decoding a multi-megabyte recording on every editor
// launch. The header records the export size and SHA-256 so two exports of the
// same size cannot share stale decoded audio.
const clipCacheMagic = "CDTS2"
const maxClipCount = 2 + maxComboClips

const maxClipCacheBytes = len(clipCacheMagic) + 8 + sha256.Size + 2 + maxClipCount*4 + maxTotalClipSamples*2

func clipCachePath(exportPath string) string {
	return exportPath + ".pcm"
}

func fileChecksum(path string) ([sha256.Size]byte, error) {
	var sum [sha256.Size]byte
	file, err := os.Open(path)
	if err != nil {
		return sum, err
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return sum, err
	}
	copy(sum[:], hash.Sum(nil))
	return sum, nil
}

func readClipCache(exportPath string, sourceSize int64, sourceHash [sha256.Size]byte) (Sounds, bool) {
	path := clipCachePath(exportPath)
	stat, err := os.Stat(path)
	if err != nil || stat.Size() > int64(maxClipCacheBytes) {
		return Sounds{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) < len(clipCacheMagic)+8+sha256.Size+2 || string(data[:len(clipCacheMagic)]) != clipCacheMagic {
		return Sounds{}, false
	}
	pos := len(clipCacheMagic)
	if int64(binary.LittleEndian.Uint64(data[pos:])) != sourceSize {
		return Sounds{}, false
	}
	pos += 8
	if !bytes.Equal(data[pos:pos+sha256.Size], sourceHash[:]) {
		return Sounds{}, false
	}
	pos += sha256.Size
	count := int(binary.LittleEndian.Uint16(data[pos:]))
	pos += 2
	if count < 2 || count > maxClipCount {
		return Sounds{}, false
	}
	total := 0
	readClip := func() (Clip, bool) {
		if pos+4 > len(data) {
			return nil, false
		}
		length := uint64(binary.LittleEndian.Uint32(data[pos:]))
		pos += 4
		if length > uint64(maxClipSamples) || length > uint64(maxTotalClipSamples-total) {
			return nil, false
		}
		n := int(length)
		if n > (len(data)-pos)/2 {
			return nil, false
		}
		clip := make(Clip, n)
		for i := range clip {
			clip[i] = int16(binary.LittleEndian.Uint16(data[pos+i*2:]))
		}
		pos += n * 2
		total += n
		return clip, true
	}
	lock, ok := readClip()
	if !ok {
		return Sounds{}, false
	}
	quad, ok := readClip()
	if !ok {
		return Sounds{}, false
	}
	sounds := Sounds{Lock: lock, Quad: quad}
	for i := 2; i < count; i++ {
		clip, ok := readClip()
		if !ok {
			return Sounds{}, false
		}
		if len(clip) > 0 {
			sounds.Combos = append(sounds.Combos, clip)
		}
	}
	if pos != len(data) || (len(lock) == 0 && len(quad) == 0 && len(sounds.Combos) == 0) {
		return Sounds{}, false
	}
	return sounds, true
}

func writeClipCache(exportPath string, sourceSize int64, sourceHash [sha256.Size]byte, sounds Sounds) {
	clips := []Clip{sounds.Lock, sounds.Quad}
	clips = append(clips, sounds.Combos...)
	if !soundsWithinLimits(sounds) {
		return
	}
	size := len(clipCacheMagic) + 8 + sha256.Size + 2
	for _, clip := range clips {
		size += 4 + len(clip)*2
	}
	out := make([]byte, 0, size)
	out = append(out, clipCacheMagic...)
	out = binary.LittleEndian.AppendUint64(out, uint64(sourceSize))
	out = append(out, sourceHash[:]...)
	out = binary.LittleEndian.AppendUint16(out, uint16(len(clips)))
	for _, clip := range clips {
		out = binary.LittleEndian.AppendUint32(out, uint32(len(clip)))
		for _, sample := range clip {
			out = binary.LittleEndian.AppendUint16(out, uint16(sample))
		}
	}
	temp, err := os.CreateTemp(filepath.Dir(exportPath), filepath.Base(exportPath)+".*.tmp")
	if err != nil {
		return
	}
	tempPath := temp.Name()
	if _, err := temp.Write(out); err != nil {
		temp.Close()
		_ = os.Remove(tempPath)
		return
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return
	}
	if err := os.Rename(tempPath, clipCachePath(exportPath)); err != nil {
		_ = os.Remove(tempPath)
	}
}

// atlasClip copies one timed region of the decoded recording. Timings in the
// export are milliseconds; out-of-range or oversized entries are ignored.
func atlasClip(samples Clip, atlas Atlas, name string) Clip {
	timing, ok := atlas[name]
	if !ok || math.IsNaN(timing[0]) || math.IsNaN(timing[1]) ||
		math.IsInf(timing[0], 0) || math.IsInf(timing[1], 0) ||
		timing[0] < 0 || timing[0] > maxDecodedSeconds*1000 || timing[1] <= 0 {
		return nil
	}
	start := int(timing[0] / 1000 * sampleRate)
	length := int(timing[1] / 1000 * sampleRate)
	if start < 0 || length <= 0 || length > maxClipSamples || start >= len(samples) {
		return nil
	}
	end := start + length
	if end > len(samples) {
		end = len(samples)
	}
	if end <= start {
		return nil
	}
	clip := make(Clip, end-start)
	copy(clip, samples[start:end])
	return clip
}
