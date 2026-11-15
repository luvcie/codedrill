package music

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hajimehoshi/go-mp3"
	"github.com/luvcie/codedrill/internal/download"
)

const maxMusicBytes = 64 << 20
const holidayFilename = "Holidays 2023 (Remastered) - TETR.IO Original Soundtrack (MP3).zip"

func cacheDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "codedrill", "music")
	return dir, nil
}

func Path(track Track) string {
	dir, err := cacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, track.ID+".mp3")
}

func Cached(track Track) bool {
	stat, err := os.Stat(Path(track))
	return err == nil && stat.Mode().IsRegular() && stat.Size() > 0 && stat.Size() <= maxMusicBytes
}

func saveMP3(track Track, data []byte) error {
	if _, err := mp3.NewDecoder(bytes.NewReader(data)); err != nil {
		return fmt.Errorf("%s: invalid MP3: %w", track.Name, err)
	}
	path := Path(track)
	if path == "" {
		return errors.New("music cache unavailable")
	}
	return download.Save(path, data)
}

func Download(ctx context.Context, source string, tracks []Track, report download.Report, completed func()) error {
	if source == Holidays {
		if err := downloadHolidays(ctx, report); err != nil {
			return err
		}
		for range HolidayTracks {
			if completed != nil {
				completed()
			}
		}
		return nil
	}
	var total, base int64
	known := true
	for _, track := range tracks {
		if !Cached(track) {
			total += track.Size
			known = known && track.Size > 0
		}
	}
	for _, track := range tracks {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !Cached(track) {
			expected := track.Size
			data, err := download.Fetch(ctx, track.URL, maxMusicBytes, func(p download.Progress) {
				if p.Total <= 0 {
					known = false
				} else {
					total += p.Total - expected
					expected = p.Total
				}
				aggregate := download.Progress{Bytes: base + p.Bytes}
				if known {
					aggregate.Total = total
				}
				if report != nil {
					report(aggregate)
				}
			})
			if err != nil {
				return fmt.Errorf("%s: %w", track.Name, err)
			}
			if err := saveMP3(track, data); err != nil {
				return err
			}
			base += int64(len(data))
		}
		if completed != nil {
			completed()
		}
	}
	return nil
}

// The store issues a fresh free download token, as its download button does.
func holidayURL(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://merch.tetr.io/api/generateDownloadToken.php", strings.NewReader(`{"type":"holidays-2023-ost"}`))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var result struct {
		Success bool
		Token   string
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download token: %s", res.Status)
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 8192)).Decode(&result); err != nil {
		return "", err
	}
	if !result.Success || result.Token == "" {
		return "", errors.New("store did not issue a free download token")
	}
	return "https://merch.tetr.io/api/download.php?" + url.Values{"tk": {result.Token}, "file": {holidayFilename}}.Encode(), nil
}

func downloadHolidays(ctx context.Context, report download.Report) error {
	if Cached(HolidayTracks[0]) && Cached(HolidayTracks[1]) {
		return nil
	}
	dir, err := cacheDir()
	if err != nil {
		return err
	}
	archive := filepath.Join(dir, "holidays-2023.zip")
	file, err := os.Open(archive)
	var data []byte
	if err == nil {
		data, err = io.ReadAll(io.LimitReader(file, maxMusicBytes+1))
		file.Close()
		if len(data) > maxMusicBytes {
			return errors.New("cached holiday ZIP exceeds 64 MiB")
		}
	}
	if os.IsNotExist(err) {
		link, err := holidayURL(ctx)
		if err != nil {
			return err
		}
		data, err = download.Fetch(ctx, link, maxMusicBytes, report)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	if err := extractHolidays(data); err != nil {
		_ = os.Remove(archive)
		return err
	}
	// Keep the ZIP too, so a missing extracted song never requires another fetch.
	return download.Save(archive, data)
}

func extractHolidays(data []byte) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for i, track := range HolidayTracks {
		found := false
		for _, file := range archive.File {
			name := filepath.Base(file.Name)
			if !strings.HasPrefix(name, fmt.Sprintf("%d. ", i+1)) || !strings.EqualFold(filepath.Ext(name), ".mp3") {
				continue
			}
			if file.UncompressedSize64 > maxMusicBytes {
				return errors.New("holiday track exceeds 64 MiB")
			}
			reader, err := file.Open()
			if err != nil {
				return err
			}
			audio, err := io.ReadAll(io.LimitReader(reader, maxMusicBytes+1))
			reader.Close()
			if err != nil {
				return err
			}
			if len(audio) > maxMusicBytes {
				return errors.New("holiday track exceeds 64 MiB")
			}
			if err := saveMP3(track, audio); err != nil {
				return err
			}
			found = true
			break
		}
		if !found {
			return fmt.Errorf("ZIP is missing %s", track.Name)
		}
	}
	return nil
}

// LocalTracks lists this folder's MP3 files without copying or modifying them.
func LocalTracks(folder string) ([]Track, error) {
	if strings.HasPrefix(folder, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		folder = filepath.Join(home, folder[2:])
	}
	entries, err := os.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	var tracks []Track
	for _, entry := range entries {
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(entry.Name()), ".mp3") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		path, err := filepath.Abs(filepath.Join(folder, entry.Name()))
		if err != nil {
			return nil, err
		}
		tracks = append(tracks, Track{ID: path, Name: strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name())), URL: path, Size: info.Size()})
	}
	return tracks, nil
}
