package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

func TestRenderCountdown(t *testing.T) {
	for _, n := range []int{1, 3, 10, 42} {
		view := ansi.Strip(renderCountdown(n, 80, 20))
		if !strings.Contains(view, "skip") {
			t.Errorf("countdown %d lacks the skip hint:\n%s", n, view)
		}
		if strings.Contains(view, "get ready") {
			t.Errorf("countdown %d draws its own context line; the header owns it:\n%s", n, view)
		}
		if lipgloss.Height(view) != 20 {
			t.Errorf("countdown %d is %d rows, want the 20 it was given:\n%s", n, lipgloss.Height(view), view)
		}
		for _, row := range strings.Split(renderBlockNumber(n), "\n") {
			if !strings.Contains(view, row) {
				t.Errorf("countdown %d does not draw its digits:\n%s", n, view)
				break
			}
		}
	}
	for _, n := range []int{0, -1} {
		view := ansi.Strip(renderCountdown(n, 80, 20))
		if !strings.Contains(view, strings.Split(goArt, "\n")[0]) || strings.Contains(view, "skip") {
			t.Errorf("frame %d should be the bare GO frame:\n%s", n, view)
		}
		if lipgloss.Height(view) != 20 {
			t.Errorf("frame %d is %d rows, want 20", n, lipgloss.Height(view))
		}
	}
	// Short rows drop the box padding and still keep the hint in view.
	short := ansi.Strip(renderCountdown(3, 80, 9))
	if lipgloss.Height(short) != 9 || !strings.Contains(short, "skip") {
		t.Errorf("countdown in 9 rows lost the hint or changed height:\n%s", short)
	}
	// Anything below the box height is clipped, never grown past the window.
	if h := lipgloss.Height(renderCountdown(3, 80, 4)); h != 4 {
		t.Errorf("countdown in 4 rows is %d rows", h)
	}
	if renderCountdown(1, 0, 0) == "" {
		t.Error("expected output with default dimensions")
	}
}

func TestCountdownScreenHeaderAndFit(t *testing.T) {
	for _, size := range []struct{ w, h int }{{60, 20}, {80, 24}, {120, 40}} {
		m := newEvaluationTestModel(t)
		m.Update(tea.WindowSizeMsg{Width: size.w, Height: size.h})
		m.state = StateCountdown

		m.countdownCount = 3
		view := m.View()
		plain := ansi.Strip(view)
		if lipgloss.Width(view) > size.w || lipgloss.Height(view) > size.h {
			t.Fatalf("%dx%d countdown is %dx%d", size.w, size.h, lipgloss.Width(view), lipgloss.Height(view))
		}
		if lipgloss.Height(view) != size.h {
			t.Errorf("%dx%d countdown fills %d rows, want %d", size.w, size.h, lipgloss.Height(view), size.h)
		}
		if !strings.Contains(plain, "ft_strlen · get ready") || !strings.Contains(plain, "skip") {
			t.Errorf("%dx%d counting frame lacks the header context or skip hint:\n%s", size.w, size.h, plain)
		}
		assertWordmarkNearTop(t, size.w, size.h, plain)

		m.countdownCount = 0
		plain = ansi.Strip(m.View())
		if !strings.Contains(plain, "ft_strlen · go") || strings.Contains(plain, "skip") {
			t.Errorf("%dx%d GO frame should show the go context and no skip hint:\n%s", size.w, size.h, plain)
		}
		assertWordmarkNearTop(t, size.w, size.h, plain)
	}
}

// assertWordmarkNearTop checks that the codedrill wordmark is one of the first
// two non-blank rows of a screen.
func assertWordmarkNearTop(t *testing.T, w, h int, plain string) {
	t.Helper()
	var first []string
	for _, line := range strings.Split(plain, "\n") {
		if strings.TrimSpace(line) != "" {
			first = append(first, line)
		}
		if len(first) == 2 {
			break
		}
	}
	for _, line := range first {
		if strings.Contains(line, "codedrill") {
			return
		}
	}
	t.Errorf("%dx%d: codedrill is not on the first two non-blank rows:\n%s", w, h, plain)
}

// The skip hint is a help line: it sits at column 2 under the box, not
// centred in the frame.
func TestCountdownHintIsLeftAlignedUnderBox(t *testing.T) {
	view := ansi.Strip(renderCountdown(3, 80, 20))
	var hint string
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "skip") {
			hint = line
		}
	}
	if !strings.HasPrefix(hint, "  [Space/Enter] skip  [Esc] cancel") {
		t.Fatalf("hint is not at column 2: %q", hint)
	}
}

func TestCountdownFitsNarrowWidths(t *testing.T) {
	for _, width := range []int{20, 30} {
		for _, n := range []int{0, 3, 10} {
			view := renderCountdown(n, width, 12)
			if lipgloss.Height(view) != 12 {
				t.Errorf("width %d count %d is %d rows, want 12", width, n, lipgloss.Height(view))
			}
			for _, line := range strings.Split(view, "\n") {
				if w := lipgloss.Width(line); w > width {
					t.Errorf("width %d count %d row is %d columns: %q", width, n, w, ansi.Strip(line))
				}
			}
		}
	}
}
