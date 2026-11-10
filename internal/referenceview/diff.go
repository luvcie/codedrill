package referenceview

import (
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/luvcie/codedrill/internal/compare"
)

// NewDiff preserves comparison order: entry file first, then sorted paths.
func NewDiff(result compare.Result) Model {
	m := Model{
		source: make(map[string]string), viewport: viewport.New(80, 20),
		width: 80, height: 24, title: "DIFF", wrap: true,
	}
	for _, file := range result.Files {
		var text strings.Builder
		text.WriteString("File: " + file.Rel + "\n\n")
		if file.Missing {
			text.WriteString("Required workspace file is missing.\n\n")
		}
		if file.Result.Exact && !file.Missing {
			text.WriteString("Normalized sources match.")
		} else {
			text.WriteString("- missing: reference line\n+ extra: student line\n  unchanged tokens (shortened)\n\n")
			text.WriteString(file.Result.DiffText())
		}
		m.paths = append(m.paths, file.Rel)
		m.source[file.Rel] = strings.ReplaceAll(safeText(text.String()), "\t", "    ")
	}
	m.selectFile()
	return m
}
