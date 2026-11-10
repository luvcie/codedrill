package referenceview

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestFilesScrollAndClose(t *testing.T) {
	m := New(map[string][]byte{
		"life.c": []byte(strings.Repeat("int line;\n", 50) + "int last_line;"),
		"life.h": []byte("#define HEADER_MARKER 1\n"),
	}, "42.5%")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	m = next.(Model)
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "REFERENCE: life.c | Match: 42.5%") || strings.Contains(view, "last_line") {
		t.Fatalf("initial view: %q", view)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	m = next.(Model)
	if !strings.Contains(ansi.Strip(m.View()), "last_line") {
		t.Fatal("could not scroll to the end of the source")
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = next.(Model)
	if view := ansi.Strip(m.View()); !strings.Contains(view, "REFERENCE: life.h") || !strings.Contains(view, "HEADER_MARKER") {
		t.Fatalf("second file: %q", view)
	}
	next, _ = m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	m = next.(Model)
	if m.current != 0 || m.viewport.YOffset != 0 {
		t.Fatal("file selection did not return to the first file")
	}
	for _, key := range []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune("q")}, {Type: tea.KeyEsc}, {Type: tea.KeyCtrlC}} {
		_, cmd := m.Update(key)
		if cmd == nil {
			t.Fatalf("%s did not close the viewer", key)
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatalf("%s did not quit", key)
		}
	}
}

func TestSmallSizesAndControlCharacters(t *testing.T) {
	m := New(map[string][]byte{"very/long/life.c": []byte("int main(void) {\x1b]52;c;bad\a return 0; }\n")}, "100.0%")
	if strings.Contains(ansi.Strip(m.source[m.paths[0]]), "\x1b") {
		t.Fatal("source terminal escape was not sanitized")
	}
	for _, size := range [][2]int{{1, 1}, {5, 2}, {12, 3}, {20, 6}, {80, 24}} {
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
}

func TestMultiFileNavigationInSidePane(t *testing.T) {
	m := New(map[string][]byte{
		"main.c": []byte(strings.Repeat("int line;\n", 40)),
		"main.h": []byte("#define SIDE_HEADER 1\n"),
	}, "50.0%")
	update := func(msg tea.Msg) {
		t.Helper()
		next, _ := m.Update(msg)
		m = next.(Model)
		for _, line := range strings.Split(m.View(), "\n") {
			if ansi.StringWidth(line) > 22 {
				t.Fatalf("reference overflowed narrow pane: %q", line)
			}
		}
	}
	update(tea.WindowSizeMsg{Width: 22, Height: 8})
	update(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.viewport.YOffset == 0 {
		t.Fatal("narrow pane did not scroll")
	}
	update(tea.KeyMsg{Type: tea.KeyTab})
	if !strings.Contains(ansi.Strip(m.View()), "SIDE_HEADER") {
		t.Fatal("second file is not readable in narrow pane")
	}
	update(tea.KeyMsg{Type: tea.KeyTab})
	if m.current != 0 || m.viewport.YOffset != 0 {
		t.Fatal("file navigation did not wrap and reset scroll")
	}
	update(tea.KeyMsg{Type: tea.KeyShiftTab})
	if m.current != 1 {
		t.Fatal("reverse navigation did not wrap")
	}
}
