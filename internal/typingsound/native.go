package typingsound

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// nativeSounds extracts gameplay clips from a native TETR.IO sound-pack ZIP.
// These packs store each effect as its own file, unlike .tpse exports, which
// pack one recording and an atlas.
func nativeSounds(data []byte) (Sounds, error) {
	files, err := nativeFiles(data)
	if err != nil {
		return Sounds{}, err
	}
	sounds := Sounds{}
	total := 0
	take := func(names ...string) (Clip, error) {
		for _, name := range names {
			file := files[name]
			if file == nil {
				continue
			}
			clip, err := decodeNativeClip(file)
			if err != nil {
				return nil, err
			}
			if len(clip) == 0 || len(clip) > maxClipSamples || total+len(clip) > maxTotalClipSamples {
				continue
			}
			total += len(clip)
			return clip, nil
		}
		return nil, nil
	}

	if sounds.Lock, err = take("harddrop", "lock"); err != nil {
		return Sounds{}, err
	}
	if sounds.Quad, err = take("clearquad", "clearline"); err != nil {
		return Sounds{}, err
	}
	for i := 1; i <= maxComboClips; i++ {
		clip, err := take(fmt.Sprintf("combo_%d", i))
		if err != nil {
			return Sounds{}, err
		}
		if len(clip) > 0 {
			sounds.Combos = append(sounds.Combos, clip)
		}
	}
	if len(sounds.Lock) == 0 && len(sounds.Quad) == 0 && len(sounds.Combos) == 0 {
		return Sounds{}, errors.New("no lock, quad or combo sounds")
	}
	return sounds, nil
}

// nativeFiles indexes the supported effect files by their base name. Packs can
// be flat or have one folder, and some contain unrelated music or artwork.
func nativeFiles(data []byte) (map[string]*zip.File, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid ZIP pack: %w", err)
	}
	files := make(map[string]*zip.File)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() || file.UncompressedSize64 > maxAudioBytes {
			continue
		}
		name := strings.ToLower(path.Base(file.Name))
		ext := path.Ext(name)
		if ext != ".wav" && ext != ".mp3" && ext != ".ogg" {
			continue
		}
		name = strings.TrimSuffix(name, ext)
		if !soundName.MatchString(name) {
			continue
		}
		if _, exists := files[name]; !exists {
			files[name] = file
		}
	}
	return files, nil
}

func decodeNativeClip(file *zip.File) (Clip, error) {
	if file.UncompressedSize64 > maxAudioBytes {
		return nil, fmt.Errorf("%s exceeds 16 MiB", file.Name)
	}
	reader, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", file.Name, err)
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, maxAudioBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", file.Name, err)
	}
	if len(data) == 0 || len(data) > maxAudioBytes {
		return nil, fmt.Errorf("%s has an invalid audio size", file.Name)
	}
	samples, rate, err := decodeAudio(data)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", file.Name, err)
	}
	return resample(samples, rate, sampleRate), nil
}

// validateNativePack checks a download before it is cached. Full decoding
// happens when the pack is loaded, so a corrupt clip does not keep a bad file.
func validateNativePack(data []byte) error {
	files, err := nativeFiles(data)
	if err != nil {
		return err
	}
	for _, name := range []string{"harddrop", "lock", "clearquad", "clearline"} {
		if files[name] != nil {
			return nil
		}
	}
	for i := 1; i <= maxComboClips; i++ {
		if files[fmt.Sprintf("combo_%d", i)] != nil {
			return nil
		}
	}
	return errors.New("ZIP has no lock, quad or combo sounds")
}
