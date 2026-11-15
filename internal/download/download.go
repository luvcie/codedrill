// Package download reads bounded audio downloads and reports network progress.
package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Progress struct {
	Bytes int64
	Total int64 // Zero means the server did not provide a length.
}

type Report func(Progress)

func Fetch(ctx context.Context, url string, limit int64, report Report) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "codedrill/audio")
	res, err := (&http.Client{Timeout: 5 * time.Minute}).Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", res.Status)
	}
	if res.ContentLength > limit {
		return nil, fmt.Errorf("download exceeds %d MiB", limit>>20)
	}
	p := Progress{Total: max(0, res.ContentLength)}
	if report != nil {
		report(p)
	}
	data, err := io.ReadAll(io.TeeReader(io.LimitReader(res.Body, limit+1), writerFunc(func(b []byte) (int, error) {
		p.Bytes += int64(len(b))
		if report != nil {
			report(p)
		}
		return len(b), nil
	})))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("download exceeds %d MiB", limit>>20)
	}
	return data, nil
}

type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(b []byte) (int, error) { return f(b) }

// Save replaces a cache entry only after the complete, validated download is written.
func Save(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, "audio-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(temp.Name())
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), path)
}
