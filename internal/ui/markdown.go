package ui

import (
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"
)

// RenderMarkdown renders a drill description for a box of the given width.
// If glamour fails, the raw text is shown instead.
func RenderMarkdown(content string, width int) string {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(width-10, 40)),
	)
	if err != nil {
		return content
	}
	out, err := r.Render(content)
	if err != nil {
		return content
	}
	return strings.Trim(out, "\n")
}

// CleanSummary returns the first line of prose in a markdown description,
// skipping headings and code blocks, shortened to 60 columns.
func CleanSummary(desc string) string {
	inCode := false
	for _, line := range strings.Split(desc, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCode = !inCode
			continue
		}
		if !inCode && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			return shorten(trimmed, 60)
		}
	}
	return shorten(strings.TrimSpace(desc), 60)
}

func shorten(s string, width int) string {
	if ansi.StringWidth(s) <= width {
		return s
	}
	return strings.TrimRight(ansi.Truncate(s, width-3, ""), " ") + "..."
}
