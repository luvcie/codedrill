package music

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync/atomic"
	"testing"

	"github.com/luvcie/codedrill/internal/download"
	"github.com/luvcie/codedrill/internal/storage"
)

// A silent MPEG-1 layer III frame; enough to exercise the decoder without bundled audio.
func silentMP3() []byte {
	frame := make([]byte, 417)
	copy(frame, []byte{0xff, 0xfb, 0x90, 0x00})
	return append(frame, frame...)
}

func TestDownloadsCacheAndComplete(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write(silentMP3())
	}))
	defer server.Close()
	tracks := []Track{{ID: "one", Name: "One", URL: server.URL}, {ID: "two", Name: "Two", URL: server.URL}}
	for _, source := range []string{SprintZen, TetraLeague} {
		completed := 0
		var progress download.Progress
		if err := Download(context.Background(), source, tracks, func(p download.Progress) { progress = p }, func() { completed++ }); err != nil {
			t.Fatal(err)
		}
		if completed != 2 {
			t.Fatalf("completed %d", completed)
		}
		if !Cached(tracks[0]) || !Cached(tracks[1]) {
			t.Fatal("cache missing")
		}
		if hits.Load() != 2 {
			t.Fatalf("cache fetched again: %d", hits.Load())
		}
		if progress.Bytes != 0 && progress.Bytes != 2*int64(len(silentMP3())) {
			t.Fatalf("progress %v", progress)
		}
	}
}

func TestBattleCatalogAndCachedPlayback(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", root)
	wantIDs := []string{
		"kmk-033", "kmk-024", "kmk-038", "kmk-047", "tmk-060", "mkk-033", "kmk-036",
		"kmk-017", "kmk-032", "kmk-004", "kmk-041", "kmk-039", "kmk-051",
	}
	var ids []string
	for _, track := range Tracks(TetraLeague) {
		ids = append(ids, track.ID)
		if track.Name == "" || track.Artist == "" || track.URL == "" || track.Size <= 0 {
			t.Fatalf("incomplete Battle track: %+v", track)
		}
	}
	if !reflect.DeepEqual(ids, wantIDs) {
		t.Fatalf("Battle track IDs = %v, want %v", ids, wantIDs)
	}
	seen := make(map[string]bool)
	for _, source := range []string{SprintZen, TetraLeague, Holidays} {
		for _, track := range Tracks(source) {
			if seen[track.ID] {
				t.Fatalf("playlists share cache ID %q", track.ID)
			}
			seen[track.ID] = true
		}
	}
	for _, track := range []Track{Calm[0], Battle[0], HolidayTracks[0]} {
		if err := saveMP3(track, silentMP3()); err != nil {
			t.Fatal(err)
		}
	}
	songs := SelectedSongs([]string{"kvx-016", "kmk-033", "kmk-024", "christmas-city"}, "")
	wantSongs := []Song{
		{Title: Calm[0].Name, Path: filepath.Join(root, "codedrill", "music", "kvx-016.mp3")},
		{Title: Battle[0].Name, Path: filepath.Join(root, "codedrill", "music", "kmk-033.mp3")},
		{Title: HolidayTracks[0].Name, Path: filepath.Join(root, "codedrill", "music", "christmas-city.mp3")},
	}
	if !reflect.DeepEqual(songs, wantSongs) {
		t.Fatalf("cached playback = %v, want %v", songs, wantSongs)
	}
}

func TestHolidayZIPIsReusedAndSelectionDoesNotDiscardOtherSong(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for _, name := range []string{"Album/1. クリスマス都市 (Remastered).mp3", "Album/2. Carol of the Blocks (Remastered).mp3"} {
		f, _ := writer.Create(name)
		_, _ = f.Write(silentMP3())
	}
	_ = writer.Close()
	dir, err := cacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "holidays-2023.zip"), buf.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	completed := 0
	if err := Download(context.Background(), Holidays, HolidayTracks[:1], nil, func() { completed++ }); err != nil {
		t.Fatal(err)
	}
	if completed != 2 || !Cached(HolidayTracks[0]) || !Cached(HolidayTracks[1]) {
		t.Fatal("ZIP did not supply both songs")
	}
	_ = os.Remove(Path(HolidayTracks[1]))
	if err := downloadHolidays(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if !Cached(HolidayTracks[1]) {
		t.Fatal("missing track was not restored from cached ZIP")
	}
	songs := SelectedSongs([]string{HolidayTracks[1].ID}, "")
	if !reflect.DeepEqual(songs, []Song{{Title: HolidayTracks[1].Name, Path: Path(HolidayTracks[1])}}) {
		t.Fatalf("wrong selection: %v", songs)
	}
	if err := extractHolidays([]byte("not a ZIP")); err == nil {
		t.Fatal("accepted invalid ZIP")
	}
}

func TestLocalFolderAndVolume(t *testing.T) {
	folder := t.TempDir()
	for _, name := range []string{"a.MP3", "b.mp3", "ignore.txt"} {
		if err := os.WriteFile(filepath.Join(folder, name), silentMP3(), 0600); err != nil {
			t.Fatal(err)
		}
	}
	tracks, err := LocalTracks(folder)
	if err != nil || len(tracks) != 2 {
		t.Fatalf("tracks %v: %v", tracks, err)
	}
	songs := SelectedSongs([]string{tracks[1].ID}, folder)
	if !reflect.DeepEqual(songs, []Song{{Title: "b", Path: filepath.Join(folder, "b.mp3")}}) {
		t.Fatal(songs)
	}
	data := binary.LittleEndian.AppendUint16(nil, uint16(1000))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reader := &pcmReader{reader: bytes.NewReader(data), volume: storage.FollowVolume(ctx, .25, storage.Config.MusicVolume)}
	out := make([]int16, 2)
	n, err := reader.read(out)
	if n != 1 || err != nil || out[0] != 250 {
		t.Fatalf("read %v %d %v", out, n, err)
	}
	if _, err = reader.read(out); err == nil || err == io.EOF {
		t.Fatalf("expected pulse end of data: %v", err)
	}
}
