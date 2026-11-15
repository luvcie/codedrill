package download

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchProgressAndLimits(t *testing.T) {
	for _, known := range []bool{true, false} {
		t.Run(map[bool]string{true: "known", false: "unknown"}[known], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if known {
					w.Header().Set("Content-Length", "6")
				} else {
					w.(http.Flusher).Flush()
				}
				_, _ = w.Write([]byte("abc"))
				w.(http.Flusher).Flush()
				_, _ = w.Write([]byte("def"))
			}))
			defer server.Close()
			var updates []Progress
			data, err := Fetch(context.Background(), server.URL, 6, func(p Progress) { updates = append(updates, p) })
			if err != nil || string(data) != "abcdef" {
				t.Fatalf("%q %v", data, err)
			}
			if len(updates) < 2 || updates[len(updates)-1].Bytes != 6 {
				t.Fatalf("progress: %v", updates)
			}
			wantTotal := int64(0)
			if known {
				wantTotal = 6
			}
			if updates[len(updates)-1].Total != wantTotal {
				t.Fatalf("progress: %v", updates)
			}
			if _, err := Fetch(context.Background(), server.URL, 4, nil); err == nil {
				t.Fatal("accepted oversized download")
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := Fetch(ctx, server.URL, 6, nil); err == nil {
				t.Fatal("ignored cancellation")
			}
		})
	}
}
