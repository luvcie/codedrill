package referenceview

import (
	"bytes"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/reference"
)

func TestDiffFilesLinesAndNavigation(t *testing.T) {
	result, err := reference.Compare("int f(void) {\nreturn 42;\n}", "int f(void) {\n\nreturn 7;\n}")
	if err != nil {
		t.Fatal(err)
	}
	m := NewDiff(compare.Result{Files: []compare.FileResult{
		{Rel: "z.c", Result: result},
		{Rel: "a.h", Missing: true},
		{Rel: "b.c", Result: reference.Result{Exact: true}},
	}})
	for _, want := range []string{"DIFF: z.c", "- missing: reference line", "+ extra: student line", "- 2: 42", "+ 3: 7"} {
		if !strings.Contains(m.View(), want) {
			t.Fatalf("diff missing %q: %s", want, m.View())
		}
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(Model)
	if !strings.Contains(m.View(), "DIFF: a.h") || !strings.Contains(m.View(), "Required workspace file is missing.") {
		t.Fatalf("missing file: %s", m.View())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(Model)
	if !strings.Contains(m.View(), "Normalized sources match.") {
		t.Fatalf("exact file: %s", m.View())
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(Model)
	if m.paths[m.current] != "z.c" || m.viewport.YOffset != 0 {
		t.Fatal("file navigation lost comparison order or scroll reset")
	}
}

func TestDiffNarrowWrappingScrollAndExit(t *testing.T) {
	result, err := reference.Compare(strings.Repeat("int missing_token;\n", 100), "")
	if err != nil {
		t.Fatal(err)
	}
	m := NewDiff(compare.Result{Files: []compare.FileResult{{Rel: "nested/very_long_file_name.c", Missing: true, Result: result}}})
	for _, size := range [][2]int{{1, 1}, {5, 2}, {12, 3}, {20, 8}, {80, 24}} {
		next, _ := m.Update(tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = next.(Model)
		lines := strings.Split(m.View(), "\n")
		if len(lines) > size[1] {
			t.Fatalf("%v: %d rows", size, len(lines))
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size[0] {
				t.Fatalf("%v: overflowing line %q", size, line)
			}
		}
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	m = next.(Model)
	if m.viewport.YOffset == 0 {
		t.Fatal("diff did not scroll")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = next.(Model)
	if !strings.Contains(m.View(), "- 100:") {
		t.Fatalf("last source line is inaccessible: %s", m.View())
	}
	for _, key := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("q")}, {Type: tea.KeyEsc}, {Type: tea.KeyCtrlC}} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Fatalf("%s did not close diff", key)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%s did not quit", key)
		}
	}
}

func TestDiffSanitizesTerminalControls(t *testing.T) {
	m := NewDiff(compare.Result{Files: []compare.FileResult{{Rel: "bad\x1b]52;c;payload\a.c", Result: reference.Result{
		Ops: []reference.DiffOp{{Op: reference.Missing, Token: reference.Token{Text: "\"\x1b[2J\t\"", Line: 1}}},
	}}}})
	if strings.ContainsAny(m.View(), "\x1b\a\t") {
		t.Fatalf("diff contains terminal controls: %q", m.View())
	}
}

func TestDiffProgramRestoresAlternateScreen(t *testing.T) {
	result, err := reference.Compare("int reference_token;", "int student_token;")
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	_, err = tea.NewProgram(NewDiff(compare.Result{Files: []compare.FileResult{{Rel: "main.c", Result: result}}}),
		tea.WithAltScreen(), tea.WithInput(strings.NewReader("q")), tea.WithOutput(&out), tea.WithoutSignalHandler()).Run()
	if err != nil {
		t.Fatal(err)
	}
	for _, sequence := range []string{"\x1b[?1049h", "\x1b[?1049l", "\x1b[?25h"} {
		if !strings.Contains(out.String(), sequence) {
			t.Fatalf("terminal lifecycle lacks %q: %q", sequence, out.String())
		}
	}
}
