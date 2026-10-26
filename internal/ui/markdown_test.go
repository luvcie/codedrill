package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

func TestDropTitleHeading(t *testing.T) {
	for _, tc := range []struct{ input, want string }{
		{"# ft_strlen\n\nWrite it.", "Write it."},
		{"\n\n# ft_strlen\nWrite it.", "Write it."},
		{"Write it.\n\n# Notes", "Write it.\n\n# Notes"},
		{"## Prototype\nint f();", "## Prototype\nint f();"},
	} {
		if got := dropTitleHeading(tc.input); got != tc.want {
			t.Errorf("dropTitleHeading(%q) = %q; want %q", tc.input, got, tc.want)
		}
	}
}

func TestRenderMarkdownHeadingsHaveNoPrefix(t *testing.T) {
	plain := ansi.Strip(RenderMarkdown("# Title\n\n## Prototype\n\n### Constraints\n\nBody text.", 80))
	for _, want := range []string{"Title", "Prototype", "Constraints"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("heading %q missing:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "# ") {
		t.Fatalf("headings keep their # prefixes:\n%s", plain)
	}
	for _, line := range strings.Split(plain, "\n") {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "  ") {
			t.Fatalf("line starts left of column 2: %q", line)
		}
	}
}

// themedMarkdownDescription has a paragraph, inline code and a C code block.
const themedMarkdownDescription = "Write `int` helpers.\n\n```c\nint main(void) { return 0; }\n```"

// hexFG is the foreground SGR parameters for a palette colour, read from its
// hex digits. glamour writes these exactly, while fg goes through lipgloss.
func hexFG(c lipgloss.Color) string {
	var r, g, b int
	_, _ = fmt.Sscanf(string(c), "#%02x%02x%02x", &r, &g, &b)
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

func TestRenderMarkdownFollowsTheme(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	for _, id := range []string{"legacy", "cyan", "gruvbox", "monochrome", "light"} {
		th, ok := ThemeByID(id)
		if !ok {
			t.Fatalf("theme %s is missing", id)
		}
		ApplyTheme(th)
		out := RenderMarkdown(themedMarkdownDescription, 80)
		if !strings.Contains(out, hexFG(th.Text)) {
			t.Errorf("%s: briefing lacks the text colour %s:\n%q", id, hexFG(th.Text), out)
		}
		if !strings.Contains(out, "\x1b["+hexFG(th.Accent)+"m") {
			t.Errorf("%s: briefing lacks the accent colour %s:\n%q", id, hexFG(th.Accent), out)
		}
		if strings.Contains(out, "38;5;") {
			t.Errorf("%s: briefing has an ANSI-256 foreground colour:\n%q", id, out)
		}
		if id == "light" && (strings.Contains(out, "38;5;187") || strings.Contains(out, "38;5;251")) {
			t.Errorf("light briefing has glamour's code colours:\n%q", out)
		}
	}
}

func TestRenderMarkdownCodeKeywordUsesAccent(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	th, _ := ThemeByID("gruvbox")
	ApplyTheme(th)
	out := RenderMarkdown("```c\nint main(void) { return 0; }\n```", 80)
	for _, kw := range []string{"int", "return"} {
		if !strings.Contains(out, "\x1b["+hexFG(th.Accent)+"m"+kw) {
			t.Errorf("keyword %q is not in the accent colour %s:\n%q", kw, hexFG(th.Accent), out)
		}
	}
}

func TestRenderMarkdownCodeFollowsThemeChange(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	ApplyTheme(Themes[0])
	_ = RenderMarkdown("```c\nint x;\n```", 80)
	light, _ := ThemeByID("light")
	ApplyTheme(light)
	out := RenderMarkdown("```c\nint x;\n```", 80)
	if !strings.Contains(out, "\x1b["+hexFG(light.Accent)+"m") {
		t.Fatalf("code kept the previous theme's colours; light accent %s missing:\n%q", hexFG(light.Accent), out)
	}
}

func TestRenderMarkdownInlineCodeChip(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	th, _ := ThemeByID("nord")
	ApplyTheme(th)
	out := RenderMarkdown("Use `strlen` here.", 80)
	chip := "48;2;" + strings.ReplaceAll(hexFG(th.Border), "38;2;", "")
	if !strings.Contains(out, chip) {
		t.Fatalf("inline code lacks the border background %s:\n%q", chip, out)
	}
}

func TestRenderMarkdownCodeBlockHasNoBackground(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	ApplyTheme(Themes[0])
	out := RenderMarkdown("```c\nint x;\n```", 80)
	if strings.Contains(out, "48;2;") || strings.Contains(out, "48;5;") {
		t.Fatalf("code block has a background colour:\n%q", out)
	}
}
