package mode

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/luvcie/codedrill/internal/storage"
)

// RecordMode groups Study and Copy together. Custom configurations each have
// their own records, separate from the named presets.
func (o Options) RecordMode() storage.RecordMode {
	o = o.Normalize()
	switch o {
	case Exam():
		return storage.ExamMode
	case Recall():
		return storage.RecallMode
	case Study(), Copy():
		return storage.StudyMode
	}
	var bits uint8
	for i, on := range []bool{o.CheckReference, o.ShowReference, o.RequireExact, o.RequireTests} {
		if on {
			bits |= 1 << i
		}
	}
	return storage.RecordMode(fmt.Sprintf("custom-%02x", bits))
}

func RecordLabel(category storage.RecordMode) string {
	switch category {
	case storage.ExamMode:
		return "Exam"
	case storage.RecallMode:
		return "Recall"
	case storage.StudyMode:
		return "Study"
	}
	if !strings.HasPrefix(string(category), "custom-") {
		return "Unknown mode"
	}
	bits, err := strconv.ParseUint(strings.TrimPrefix(string(category), "custom-"), 16, 4)
	if err != nil {
		return "Unknown mode"
	}
	var parts []string
	if bits&8 != 0 {
		parts = append(parts, "tests")
	}
	if bits&4 != 0 {
		parts = append(parts, "exact match")
	}
	if bits&2 != 0 {
		parts = append(parts, "reference visible")
	} else {
		parts = append(parts, "reference hidden")
	}
	if bits&1 != 0 && bits&4 == 0 {
		parts = append(parts, "reference scored")
	}
	return "Custom (" + strings.Join(parts, ", ") + ")"
}
