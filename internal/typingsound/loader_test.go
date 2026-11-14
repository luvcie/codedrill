package typingsound

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

// makeWAV builds a mono 16-bit PCM WAV from the given samples.
func makeWAV(samples []int16, rate int) []byte {
	data := make([]byte, 0, len(samples)*2)
	for _, sample := range samples {
		data = binary.LittleEndian.AppendUint16(data, uint16(sample))
	}
	var out []byte
	out = append(out, "RIFF"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(36+len(data)))
	out = append(out, "WAVEfmt "...)
	out = binary.LittleEndian.AppendUint32(out, 16)
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint16(out, 1)
	out = binary.LittleEndian.AppendUint32(out, uint32(rate))
	out = binary.LittleEndian.AppendUint32(out, uint32(rate*2))
	out = binary.LittleEndian.AppendUint16(out, 2)
	out = binary.LittleEndian.AppendUint16(out, 16)
	out = append(out, "data"...)
	out = binary.LittleEndian.AppendUint32(out, uint32(len(data)))
	return append(out, data...)
}

// makeExport builds a TETR.IO+ export whose recording holds four 100 ms
// segments, one for each atlas entry.
func makeExport(t *testing.T) []byte {
	t.Helper()
	rate := 8000
	var samples []int16
	for _, value := range []int16{1000, 2000, 3000, 4000} {
		for range rate / 10 {
			samples = append(samples, value)
		}
	}
	atlas := map[string][2]float64{
		"harddrop":  {0, 100},
		"clearquad": {100, 100},
		"combo_1":   {200, 100},
		"combo_2":   {300, 100},
	}
	data, err := json.Marshal(map[string]any{
		"customSounds":     "data:audio/wav;base64," + base64.StdEncoding.EncodeToString(makeWAV(samples, rate)),
		"customSoundAtlas": atlas,
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDecodeWAVAndResample(t *testing.T) {
	clips, rate, err := decodeAudio(makeWAV([]int16{1000, -1000, 2000, -2000}, 8000))
	if err != nil {
		t.Fatal(err)
	}
	if rate != 8000 || len(clips) != 4 || clips[0] != 1000 || clips[1] != -1000 {
		t.Fatalf("decoded %v at %d Hz", clips, rate)
	}
	out := resample(clips, 8000, 16000)
	if len(out) != 8 {
		t.Fatalf("resampled to %d samples, want 8", len(out))
	}
	if out[0] != 1000 || out[2] != -1000 {
		t.Fatalf("resampled values %v", out[:4])
	}
	if got := resample(nil, 0, 48000); got != nil {
		t.Fatalf("resampled an empty clip: %v", got)
	}
}

func TestLoadSoundsFromDownloadedPackAndCache(t *testing.T) {
	export := makeExport(t)
	var hits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write(export)
	}))
	defer server.Close()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	info := PackInfo{ID: "sfx-test", Name: "Test pack", URL: server.URL + "/test.tpse"}

	sounds, err := loadSounds(info)
	if err != nil {
		t.Fatal(err)
	}
	if len(sounds.Lock) == 0 || len(sounds.Quad) == 0 || len(sounds.Combos) != 2 {
		t.Fatalf("sounds: lock %d, quad %d, combos %d", len(sounds.Lock), len(sounds.Quad), len(sounds.Combos))
	}
	want := sampleRate / 10
	if len(sounds.Lock) != want || len(sounds.Combos[0]) != want {
		t.Fatalf("clip lengths: lock %d, combo %d, want %d", len(sounds.Lock), len(sounds.Combos[0]), want)
	}
	if sounds.Lock[0] <= 0 || sounds.Quad[0] <= sounds.Lock[0] || sounds.Combos[1][0] <= sounds.Combos[0][0] {
		t.Fatal("clips were sliced out of order")
	}
	// The second load must read the cache instead of the server.
	if _, err := loadSounds(info); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Fatalf("server hit %d times, want 1", hits.Load())
	}
	path, err := CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(path, info.ID+".tpse")); err != nil {
		t.Fatalf("export missing from cache: %v", err)
	}
}

func TestLoadSoundsRejectsBadDownloads(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/404" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"customSounds":"data:audio/ogg;base64,YQ==","customSoundAtlas":{}}`))
	}))
	defer server.Close()
	for _, tc := range []struct {
		name string
		url  string
	}{
		{"status", server.URL + "/404"},
		{"invalid export", server.URL + "/bad"},
	} {
		info := PackInfo{ID: "sfx-" + tc.name, Name: tc.name, URL: tc.url}
		if _, err := loadSounds(info); err == nil {
			t.Fatalf("%s: loaded a bad pack", tc.name)
		}
		if _, err := os.Stat(filepath.Join(mustCache(t), "sfx-"+tc.name+".tpse")); !os.IsNotExist(err) {
			t.Fatalf("%s: stored a failed download (stat err %v)", tc.name, err)
		}
	}
	// A pack whose export has no gameplay clips is rejected.
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(makeExport(t))
	}))
	defer server2.Close()
	// Remove the gameplay entries by rewriting the atlas.
	export := makeExport(t)
	var parsed map[string]any
	if err := json.Unmarshal(export, &parsed); err != nil {
		t.Fatal(err)
	}
	parsed["customSoundAtlas"] = map[string]any{"menu": []float64{0, 10}}
	trimmed, _ := json.Marshal(parsed)
	server3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(trimmed)
	}))
	defer server3.Close()
	if _, err := loadSounds(PackInfo{ID: "sfx-empty", Name: "empty", URL: server3.URL}); err == nil {
		t.Fatal("accepted a pack without gameplay sounds")
	}
}

func mustCache(t *testing.T) string {
	t.Helper()
	dir, err := CacheDir()
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFindPackAndLabels(t *testing.T) {
	if len(Catalog) < 20 {
		t.Fatalf("catalog has %d packs", len(Catalog))
	}
	if pack, ok := FindPack("sfx-Starcat_JP-8Bit_Puzzler"); !ok || pack.Author != "Starcat_JP" {
		t.Fatalf("catalog lookup failed: %+v %v", pack, ok)
	}
	if _, ok := FindPack("missing"); ok {
		t.Fatal("found a pack that is not in the catalog")
	}
	if PackLabel("") != "Silent" || PackLabel(SilentPackID) != "Silent" {
		t.Fatal("silent label is wrong")
	}
	if PackLabel("missing") != fmt.Sprintf("Unknown pack (%s)", "missing") {
		t.Fatalf("unknown label %q", PackLabel("missing"))
	}
}
