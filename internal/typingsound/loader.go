package typingsound

import (
	"encoding/binary"
	"fmt"
	"io"
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
	if sounds, ok := readClipCache(path, stat.Size()); ok {
		return sounds, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Sounds{}, err
	}
	audio, atlas, err := ParseExport(data)
	if err != nil {
		_ = os.Remove(path)
		return Sounds{}, err
	}
	samples, rate, err := decodeAudio(audio)
	if err != nil {
		return Sounds{}, fmt.Errorf("%s: %w", info.Name, err)
	}
	samples = resample(samples, rate, sampleRate)
	sounds := Sounds{}
	for _, name := range []string{"harddrop", "lock"} {
		if clip := atlasClip(samples, atlas, name); len(clip) > 0 {
			sounds.Lock = clip
			break
		}
	}
	for _, name := range []string{"clearquad", "clearline"} {
		if clip := atlasClip(samples, atlas, name); len(clip) > 0 {
			sounds.Quad = clip
			break
		}
	}
	for i := 1; i <= 16; i++ {
		clip := atlasClip(samples, atlas, fmt.Sprintf("combo_%d", i))
		if len(clip) == 0 {
			continue
		}
		sounds.Combos = append(sounds.Combos, clip)
	}
	if len(sounds.Lock) == 0 && len(sounds.Quad) == 0 && len(sounds.Combos) == 0 {
		return Sounds{}, fmt.Errorf("%s has no lock, quad or combo sounds", info.Name)
	}
	writeClipCache(path, stat.Size(), sounds)
	return sounds, nil
}

// Clip caches avoid decoding a multi-megabyte recording on every editor
// launch. The header records the export size so a different download is not
// mistaken for the cached one.
const clipCacheMagic = "CDTS1"

func clipCachePath(exportPath string) string {
	return exportPath + ".pcm"
}

func readClipCache(exportPath string, sourceSize int64) (Sounds, bool) {
	data, err := os.ReadFile(clipCachePath(exportPath))
	if err != nil || len(data) < len(clipCacheMagic)+10 || string(data[:len(clipCacheMagic)]) != clipCacheMagic {
		return Sounds{}, false
	}
	pos := len(clipCacheMagic)
	if int64(binary.LittleEndian.Uint64(data[pos:])) != sourceSize {
		return Sounds{}, false
	}
	pos += 8
	count := int(binary.LittleEndian.Uint16(data[pos:]))
	pos += 2
	if count < 2 || count > 2+16 {
		return Sounds{}, false
	}
	readClip := func() (Clip, bool) {
		if pos+4 > len(data) {
			return nil, false
		}
		length := int(binary.LittleEndian.Uint32(data[pos:]))
		pos += 4
		if length < 0 || pos+length*2 > len(data) {
			return nil, false
		}
		clip := make(Clip, length)
		for i := range clip {
			clip[i] = int16(binary.LittleEndian.Uint16(data[pos+i*2:]))
		}
		pos += length * 2
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
	return sounds, true
}

func writeClipCache(exportPath string, sourceSize int64, sounds Sounds) {
	clips := []Clip{sounds.Lock, sounds.Quad}
	clips = append(clips, sounds.Combos...)
	var out []byte
	out = append(out, clipCacheMagic...)
	out = binary.LittleEndian.AppendUint64(out, uint64(sourceSize))
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
	if !ok {
		return nil
	}
	start := int(timing[0] / 1000 * sampleRate)
	end := start + int(timing[1]/1000*sampleRate)
	if start < 0 || start >= len(samples) || end <= start {
		return nil
	}
	if end > len(samples) {
		end = len(samples)
	}
	clip := make(Clip, end-start)
	copy(clip, samples[start:end])
	return clip
}
