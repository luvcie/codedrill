package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestRenderCountdown(t *testing.T) {
	for _, n := range []int{1, 3, 10, 42} {
		view := ansi.Strip(renderCountdown(n, 80, 24))
		if !strings.Contains(view, "GET READY") || !strings.Contains(view, "skip") {
			t.Errorf("countdown %d lacks the header or skip hint:\n%s", n, view)
		}
		for _, row := range strings.Split(renderBlockNumber(n), "\n") {
			if !strings.Contains(view, row) {
				t.Errorf("countdown %d does not draw its digits:\n%s", n, view)
				break
			}
		}
	}
	for _, n := range []int{0, -1} {
		view := ansi.Strip(renderCountdown(n, 80, 24))
		if !strings.Contains(view, "GO") || strings.Contains(view, "skip") || strings.Contains(view, "GET READY") {
			t.Errorf("frame %d should be the bare GO frame:\n%s", n, view)
		}
	}
	if renderCountdown(1, 0, 0) == "" {
		t.Error("expected output with default dimensions")
	}
}
