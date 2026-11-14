package typingsound

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMergeAutoPairsPreservesOtherSettings(t *testing.T) {
	source := `theme = "gruvbox"

[editor]
line-number = "relative"
auto-pairs = true # folded by default
bufferline = "multiple"

[editor.cursor-shape]
insert = "bar"

[keys.normal]
C-s = ":w"
`
	got := mergeAutoPairs(source)
	if strings.Contains(got, "auto-pairs = true") {
		t.Fatalf("auto-pairs was not disabled:\n%s", got)
	}
	if !strings.Contains(got, "auto-pairs = false") {
		t.Fatalf("auto-pairs is missing:\n%s", got)
	}
	for _, want := range []string{
		`theme = "gruvbox"`, `line-number = "relative"`, `bufferline = "multiple"`,
		`[editor.cursor-shape]`, `insert = "bar"`, `[keys.normal]`, `C-s = ":w"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("merged config lost %q:\n%s", want, got)
		}
	}
}

func TestMergeAutoPairsAddsEditorSection(t *testing.T) {
	got := mergeAutoPairs("theme = \"dark\"\n")
	if !strings.Contains(got, "[editor]\nauto-pairs = false") {
		t.Fatalf("missing editor override:\n%s", got)
	}
	if !strings.HasPrefix(got, "theme = \"dark\"") {
		t.Fatalf("existing setting moved:\n%s", got)
	}
	if got := mergeAutoPairs(""); got != "[editor]\nauto-pairs = false\n" {
		t.Fatalf("empty config merged as %q", got)
	}
}

func TestMergeAutoPairsHandlesDottedKey(t *testing.T) {
	got := mergeAutoPairs("editor.auto-pairs = true\ntheme = \"dark\"\n")
	if strings.Contains(got, "true") || !strings.Contains(got, "editor.auto-pairs = false") {
		t.Fatalf("dotted key was not disabled:\n%s", got)
	}
	if !strings.Contains(got, `theme = "dark"`) {
		t.Fatalf("dotted key merge lost another setting:\n%s", got)
	}
}

func TestMergeAutoPairsHandlesQuotedKeysAndTables(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
		old    string
	}{
		{
			name:   "quoted table and key",
			source: "[\"editor\"]\n\"auto-pairs\" = true # keep this comment\nline-number = \"relative\"\n",
			want:   "auto-pairs = false # keep this comment",
			old:    "auto-pairs\" = true",
		},
		{
			name:   "literal table and key",
			source: "[ 'editor' ]\n'auto-pairs' = true\n",
			want:   "auto-pairs = false",
			old:    "'auto-pairs' = true",
		},
		{
			name:   "quoted dotted key",
			source: "\"editor\".\"auto-pairs\" = true\ntheme = \"dark\"\n",
			want:   "editor.auto-pairs = false",
			old:    "auto-pairs\" = true",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeAutoPairs(tc.source)
			if strings.Contains(got, tc.old) {
				t.Fatalf("old key remains:\n%s", got)
			}
			if !strings.Contains(got, tc.want) {
				t.Fatalf("missing %q:\n%s", tc.want, got)
			}
			if strings.Count(got, "auto-pairs") != 1 {
				t.Fatalf("created a duplicate key:\n%s", got)
			}
		})
	}
}

func TestMergeAutoPairsInsertsUnderQuotedEditorAndIgnoresOtherTables(t *testing.T) {
	source := "[\"editor\"]\nline-number = \"relative\"\n\n[keys.normal]\neditor.auto-pairs = true\n"
	got := mergeAutoPairs(source)
	if !strings.Contains(got, "[\"editor\"]\nauto-pairs = false\nline-number = \"relative\"") {
		t.Fatalf("did not insert under the existing editor table:\n%s", got)
	}
	if strings.Count(got, "auto-pairs") != 2 {
		t.Fatalf("changed the key inside keys.normal or created another table:\n%s", got)
	}
}

func TestMergeAutoPairsEditsInlineEditorTable(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "existing key",
			source: "theme = \"dark\"\neditor = { line-number = \"relative\", \"auto-pairs\" = true } # keep\n",
			want:   "editor = { line-number = \"relative\", \"auto-pairs\" = false } # keep",
		},
		{
			name:   "missing key",
			source: "editor = { line-number = \"relative\" }\n",
			want:   "editor = { line-number = \"relative\", auto-pairs = false }",
		},
		{
			name:   "empty table",
			source: "editor = {}\n",
			want:   "editor = { auto-pairs = false }",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeAutoPairs(tc.source)
			if !strings.Contains(got, tc.want) {
				t.Fatalf("missing %q:\n%s", tc.want, got)
			}
			if strings.Contains(got, "[editor]") || strings.Count(got, "auto-pairs") != 1 {
				t.Fatalf("inline table became a duplicate table or key:\n%s", got)
			}
		})
	}
}

func TestMergeAutoPairsDoesNotMatchCommentedTable(t *testing.T) {
	got := mergeAutoPairs("# [editor]\ntheme = \"dark\"\n")
	if !strings.Contains(got, "\n\n[editor]\nauto-pairs = false") {
		t.Fatalf("commented table changed insertion:\n%s", got)
	}
}

func TestHelixArgsWritesSessionConfigAndCleansUp(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	configDir := filepath.Join(dir, "helix")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte("theme = \"gruvbox\"\n\n[editor]\nauto-pairs = true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	args, cleanup, err := HelixArgs([]string{"hx", "main.c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(args) != 4 || args[1] != "-c" || args[3] != "main.c" {
		t.Fatalf("args %q, want hx -c CONFIG main.c", args)
	}
	data, err := os.ReadFile(args[2])
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, `theme = "gruvbox"`) || !strings.Contains(text, "auto-pairs = false") {
		t.Fatalf("session config did not merge the user's file:\n%s", text)
	}
	// The user's own file is untouched.
	original, _ := os.ReadFile(filepath.Join(configDir, "config.toml"))
	if !strings.Contains(string(original), "auto-pairs = true") {
		t.Fatalf("the user config was modified:\n%s", original)
	}
	cleanup()
	if _, err := os.Stat(filepath.Dir(args[2])); !os.IsNotExist(err) {
		t.Fatalf("temporary config directory survived cleanup: %v", err)
	}
}

func TestHelixArgsLeavesOtherEditorsAndExplicitConfigAlone(t *testing.T) {
	for _, args := range [][]string{
		{"nvim", "main.c"},
		{"vim", "main.c"},
		{"hx", "-c", "/existing/config.toml", "main.c"},
	} {
		got, cleanup, err := HelixArgs(args)
		if err != nil {
			t.Fatal(err)
		}
		defer cleanup()
		if strings.Join(got, " ") != strings.Join(args, " ") {
			t.Errorf("%q was rewritten to %q", args, got)
		}
	}
}
