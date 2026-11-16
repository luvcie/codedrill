package mode

import (
	"testing"

	"github.com/luvcie/codedrill/internal/storage"
)

func TestRecordModesKeepCustomOptionsSeparate(t *testing.T) {
	seen := make(map[storage.RecordMode]Options)
	for bits := 0; bits < 16; bits++ {
		opts := (Options{CheckReference: bits&1 != 0, ShowReference: bits&2 != 0, RequireExact: bits&4 != 0, RequireTests: bits&8 != 0}).Normalize()
		if opts.Validate() != nil {
			continue
		}
		category := opts.RecordMode()
		if opts == Copy() && category == storage.StudyMode {
			opts = Study()
		}
		if previous, exists := seen[category]; exists && previous != opts {
			t.Fatalf("different options share %s: %+v and %+v", category, previous, opts)
		}
		seen[category] = opts
		if RecordLabel(category) == "Unknown mode" {
			t.Fatalf("mode has no readable label: %+v", opts)
		}
	}
	for opts, want := range map[Options]storage.RecordMode{Exam(): storage.ExamMode, Recall(): storage.RecallMode, Copy(): storage.StudyMode, Study(): storage.StudyMode} {
		if got := opts.RecordMode(); got != want {
			t.Fatalf("preset category = %s, want %s", got, want)
		}
	}
}
