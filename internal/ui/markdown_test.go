package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderMarkdown(t *testing.T) {
	md := "# Hello World\n\nThis is a **test** problem."
	plain := ansi.Strip(RenderMarkdown(md, 80))
	if !strings.Contains(plain, "Hello World") || !strings.Contains(plain, "test problem") || strings.Contains(plain, "**") {
		t.Errorf("markdown was not rendered:\n%s", plain)
	}
}

func TestCleanSummary(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"# Header\n\nWrite a function that reverses bits.", "Write a function that reverses bits."},
		{"```c\nint foo();\n```\nSome description", "Some description"},
		{
			"This is a very long description that exceeds the maximum length of sixty characters and should be truncated",
			"This is a very long description that exceeds the maximum...",
		},
		{"", ""},
		{strings.Repeat("é", 70), strings.Repeat("é", 57) + "..."},
	} {
		if got := CleanSummary(tc.input); got != tc.want {
			t.Errorf("CleanSummary(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}
