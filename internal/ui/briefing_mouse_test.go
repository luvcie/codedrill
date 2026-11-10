package ui

import (
	"fmt"
	"os"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/mode"
)

// checkboxBoxX returns the x coordinate of the checkbox box for the option at
// index i on row y. Left-column options (0, 2) start at x=2; right-column
// options (1, 3) are located by finding the label and backing up past the box.
func checkboxBoxX(t *testing.T, m *MainModel, y int, i int, label string) int {
	t.Helper()
	if i%2 == 0 {
		return 2
	}
	return columnOf(t, m, y, label) - 4 // 3 for "[x]" + 1 for space
}

func TestBriefingMouseTogglingAndFocus(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	labels := []string{
		"Reference check",
		"Show reference",
		"Require exact copy",
		"Require tests",
	}

	// Initial state for ft_strlen is mode.Study(): CheckReference: true, ShowReference: true, RequireTests: true. Focus: 0.
	if m.modeFocus != 0 || m.briefingMode != mode.Study() {
		t.Fatalf("unexpected initial state: focus %d, mode %+v", m.modeFocus, m.briefingMode)
	}

	// 1. Click option 0: toggles CheckReference from true to false. Focus stays on 0.
	// Normalization clears ShowReference and RequireExact.
	y0 := rowOf(t, m, labels[0])
	x0 := checkboxBoxX(t, m, y0, 0, labels[0])
	if idx, ok := m.briefingCheckboxAt(x0, y0); !ok || idx != 0 {
		t.Fatalf("briefingCheckboxAt(%d, %d) = %d, %v; want 0, true", x0, y0, idx, ok)
	}
	m.Update(leftClick(x0, y0))
	if m.modeFocus != 0 || m.briefingMode.CheckReference || m.briefingMode.ShowReference {
		t.Fatalf("click option 0: focus %d, mode %+v", m.modeFocus, m.briefingMode)
	}

	// 2. Click option 1: toggles ShowReference from false to true. Implies CheckReference: true. Focus moves to 1.
	y1 := rowOf(t, m, labels[1])
	x1 := checkboxBoxX(t, m, y1, 1, labels[1])
	if idx, ok := m.briefingCheckboxAt(x1, y1); !ok || idx != 1 {
		t.Fatalf("briefingCheckboxAt(%d, %d) = %d, %v; want 1, true", x1, y1, idx, ok)
	}
	m.Update(leftClick(x1, y1))
	if m.modeFocus != 1 || !m.briefingMode.ShowReference || !m.briefingMode.CheckReference {
		t.Fatalf("click option 1: focus %d, mode %+v", m.modeFocus, m.briefingMode)
	}

	// 3. Subsequent keyboard control works from the mouse-updated focus.
	// Pressing Space toggles the focused option (option 1 -> ShowReference false).
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.briefingMode.ShowReference {
		t.Fatalf("Space after mouse focus did not toggle option 1: %+v", m.briefingMode)
	}
	// Pressing Tab advances focus to option 2.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.modeFocus != 2 {
		t.Fatalf("Tab after mouse focus = %d, want 2", m.modeFocus)
	}

	// 4. Click option 3: toggles RequireTests from true to false. Focus moves to 3.
	y3 := rowOf(t, m, labels[3])
	x3 := checkboxBoxX(t, m, y3, 3, labels[3])
	if idx, ok := m.briefingCheckboxAt(x3, y3); !ok || idx != 3 {
		t.Fatalf("briefingCheckboxAt(%d, %d) = %d, %v; want 3, true", x3, y3, idx, ok)
	}
	m.Update(leftClick(x3, y3))
	if m.modeFocus != 3 || m.briefingMode.RequireTests {
		t.Fatalf("click option 3: focus %d, RequireTests %v", m.modeFocus, m.briefingMode.RequireTests)
	}

	// 5. Click option 3 again: toggles RequireTests back on. Focus stays on 3.
	m.Update(leftClick(x3, y3))
	if m.modeFocus != 3 || !m.briefingMode.RequireTests {
		t.Fatalf("click option 3 again: focus %d, RequireTests %v", m.modeFocus, m.briefingMode.RequireTests)
	}
}

func TestBriefingMouseAccidentalSprintPrevention(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	y0 := rowOf(t, m, "Reference check")
	y3 := rowOf(t, m, "Require tests")

	// Click checkbox 0 multiple times.
	for i := 0; i < 5; i++ {
		cmd := send(m, leftClick(2, y0))
		if cmd != nil {
			t.Fatalf("click on checkbox produced cmd: %v", cmd)
		}
		if m.state != StateBriefing {
			t.Fatalf("click on checkbox changed state to %v", m.state)
		}
	}

	// Rapidly click checkbox 3.
	for i := 0; i < 4; i++ {
		cmd := send(m, leftClick(3, y3))
		if cmd != nil {
			t.Fatalf("click on checkbox produced cmd: %v", cmd)
		}
		if m.state != StateBriefing {
			t.Fatalf("click on checkbox 3 changed state to %v", m.state)
		}
	}

	// Clicking in empty areas outside checkboxes must not start a sprint.
	for _, pt := range [][2]int{
		{50, y0},
		{10, 1},
		{5, y3 + 1},
		{60, 10},
	} {
		cmd := send(m, leftClick(pt[0], pt[1]))
		if cmd != nil {
			t.Fatalf("click at (%d, %d) produced cmd: %v", pt[0], pt[1], cmd)
		}
		if m.state != StateBriefing {
			t.Fatalf("click at (%d, %d) changed state to %v", pt[0], pt[1], m.state)
		}
	}
}

func TestBriefingMouseNormalizationAndValidation(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Switch to Exam preset first (reference checking off, tests required).
	m.Update(runeKey("4"))
	if m.briefingMode != mode.Exam() {
		t.Fatalf("expected Exam preset: %+v", m.briefingMode)
	}

	// Clicking ShowReference (option 1) when CheckReference is false
	// must normalize and turn CheckReference on.
	y1 := rowOf(t, m, "Show reference")
	x1 := checkboxBoxX(t, m, y1, 1, "Show reference")
	m.Update(leftClick(x1, y1))
	if !m.briefingMode.ShowReference || !m.briefingMode.CheckReference {
		t.Fatalf("clicking ShowReference did not imply CheckReference: %+v", m.briefingMode)
	}

	// Clicking RequireExact (option 2) turns exact copy on and implies CheckReference.
	y2 := rowOf(t, m, "Require exact copy")
	m.Update(leftClick(2, y2))
	if !m.briefingMode.RequireExact || !m.briefingMode.CheckReference {
		t.Fatalf("clicking RequireExact did not imply CheckReference: %+v", m.briefingMode)
	}

	// Disabling CheckReference (option 0) clears both ShowReference and RequireExact.
	y0 := rowOf(t, m, "Reference check")
	m.Update(leftClick(2, y0))
	if m.briefingMode.CheckReference || m.briefingMode.ShowReference || m.briefingMode.RequireExact {
		t.Fatalf("disabling CheckReference kept reference sub-options: %+v", m.briefingMode)
	}
}

func TestBriefingMousePersistence(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Toggle option 0 off (from initial true to false).
	y0 := rowOf(t, m, "Reference check")
	m.Update(leftClick(2, y0))

	saved, err := mode.Load("ft_strlen")
	if err != nil {
		t.Fatalf("mode.Load failed: %v", err)
	}
	if saved.CheckReference {
		t.Fatalf("saved preferences did not persist CheckReference off: %+v", saved)
	}

	// Toggle option 0 back on.
	m.Update(leftClick(2, y0))
	saved, err = mode.Load("ft_strlen")
	if err != nil {
		t.Fatalf("mode.Load failed: %v", err)
	}
	if !saved.CheckReference {
		t.Fatalf("saved preferences did not persist CheckReference on: %+v", saved)
	}

	// Test persistence failure handling: make preferences path a directory.
	path, err := mode.PrefsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
	y3 := rowOf(t, m, "Require tests")
	m.Update(leftClick(2, y3))
	if m.briefingSaveErr == nil {
		t.Fatal("expected briefingSaveErr on save failure")
	}
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Preferences not saved") {
		t.Fatalf("save error message not visible in view:\n%s", view)
	}
}

func TestBriefingMouseScrollOffsets(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	// Make description long so the briefing can be deeply scrolled.
	longDrill := *m.activeDrill
	longDrill.Description = strings.Repeat("Long description paragraph to test scrolling behavior.\n\n", 40)
	m.activeDrill = &longDrill
	m.briefingKey = ""
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	initialY0 := rowOf(t, m, "Reference check")

	// Scroll down by 2 lines.
	m.scrollBy(2)
	scrolledY0 := rowOf(t, m, "Reference check")
	if scrolledY0 != initialY0-2 {
		t.Fatalf("after scrolling 2 lines, rowOf is %d, want %d", scrolledY0, initialY0-2)
	}

	// Clicking at the new position toggles option 0 off.
	m.Update(leftClick(2, scrolledY0))
	if m.briefingMode.CheckReference {
		t.Fatalf("click at scrolled position failed to toggle CheckReference off: %+v", m.briefingMode)
	}

	// Clicking at the old position initialY0 now hits a different row (option 2), not option 0.
	if idx, ok := m.briefingCheckboxAt(2, initialY0); ok && idx == 0 {
		t.Fatalf("click at old position %d hit option 0 while scrolled", initialY0)
	}

	// Scroll down far enough so that all checkboxes are scrolled out of view.
	m.scrollBy(30)
	lines := stripLines(m)
	for _, label := range briefingOptionLabels {
		for _, line := range lines {
			if strings.Contains(line, label) {
				t.Fatalf("label %q is still visible after scrolling far down: %q", label, line)
			}
		}
	}
	// Clicking across rows when scrolled off screen must hit no checkbox.
	for y := 0; y < 15; y++ {
		if idx, ok := m.briefingCheckboxAt(2, y); ok {
			t.Fatalf("row %d hit checkbox %d while off screen", y, idx)
		}
	}

	// Scroll back up to the top.
	m.scrollBy(-50)
	restoredY0 := rowOf(t, m, "Reference check")
	if restoredY0 != initialY0 {
		t.Fatalf("restored rowOf is %d, want %d", restoredY0, initialY0)
	}
	m.Update(leftClick(2, restoredY0))
	if !m.briefingMode.CheckReference {
		t.Fatalf("click at restored position failed to toggle CheckReference on: %+v", m.briefingMode)
	}
}

func TestBriefingMouseWrappedContent(t *testing.T) {
	restoreDefaultTheme(t)
	for _, size := range []tea.WindowSizeMsg{
		{Width: 60, Height: 20},
		{Width: 80, Height: 24},
	} {
		t.Run(fmt.Sprintf("%dx%d", size.Width, size.Height), func(t *testing.T) {
			m := newBriefingModeModel(t, "ft_strlen")
			m.Update(size)

			for i, label := range briefingOptionLabels {
				y := rowOf(t, m, label)
				x := checkboxBoxX(t, m, y, i, label)
				idx, ok := m.briefingCheckboxAt(x, y)
				if !ok || idx != i {
					t.Fatalf("%dx%d: rowOf(%q)=%d gave briefingCheckboxAt = %d, %v; want %d, true",
						size.Width, size.Height, label, y, idx, ok, i)
				}
				// Verify clicking toggles it.
				before := m.briefingMode
				m.Update(leftClick(x, y))
				if m.briefingMode == before {
					t.Fatalf("%dx%d: clicking option %d (%s) at row %d did not change mode",
						size.Width, size.Height, i, label, y)
				}
				if m.modeFocus != i {
					t.Fatalf("%dx%d: clicking option %d at row %d set focus %d",
						size.Width, size.Height, i, y, m.modeFocus)
				}
			}
		})
	}
}

func TestBriefingMouseWithLongWrappedTitle(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	// Set an active drill with a very long title and description that wraps.
	longDrill := *m.activeDrill
	longDrill.Title = "very_long_drill_title_that_will_cause_wrapping_in_narrow_terminals"
	m.activeDrill = &longDrill
	m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})

	for i, label := range briefingOptionLabels {
		y := rowOf(t, m, label)
		x := checkboxBoxX(t, m, y, i, label)
		idx, ok := m.briefingCheckboxAt(x, y)
		if !ok || idx != i {
			t.Fatalf("long title: rowOf(%q)=%d hit %d, %v; want %d, true", label, y, idx, ok, i)
		}
	}
}

func TestBriefingMouseClickOutsidePreservesBehavior(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	y0 := rowOf(t, m, "Reference check")
	textWidth := ansi.StringWidth("  [x] Reference check")

	// 1. Right beyond the checkbox text.
	if idx, ok := m.briefingCheckboxAt(textWidth, y0); ok {
		t.Fatalf("clicking at x=%d on checkbox row hit checkbox %d", textWidth, idx)
	}
	// 2. Far to the right edge.
	if idx, ok := m.briefingCheckboxAt(70, y0); ok {
		t.Fatalf("clicking at x=70 on checkbox row hit checkbox %d", idx)
	}
	// 3. Out of bounds.
	if idx, ok := m.briefingCheckboxAt(-1, y0); ok {
		t.Fatalf("negative x hit checkbox %d", idx)
	}
	if idx, ok := m.briefingCheckboxAt(80, y0); ok {
		t.Fatalf("x=80 hit checkbox %d", idx)
	}
	if idx, ok := m.briefingCheckboxAt(2, -1); ok {
		t.Fatalf("negative y hit checkbox %d", idx)
	}
	if idx, ok := m.briefingCheckboxAt(2, 24); ok {
		t.Fatalf("y=24 hit checkbox %d", idx)
	}

	// 4. Clicking on the header.
	if idx, ok := m.briefingCheckboxAt(5, 1); ok {
		t.Fatalf("header row hit checkbox %d", idx)
	}

	// 5. Clicking on the mode label row.
	modeY := rowOf(t, m, "< Study >")
	if idx, ok := m.briefingCheckboxAt(5, modeY); ok {
		t.Fatalf("mode label row hit checkbox %d", idx)
	}

	// Send clicks outside and verify mode and focus are completely preserved.
	initialMode := m.briefingMode
	initialFocus := m.modeFocus
	m.Update(leftClick(textWidth+5, y0))
	m.Update(leftClick(5, modeY))
	m.Update(leftClick(10, 1))
	if m.briefingMode != initialMode || m.modeFocus != initialFocus {
		t.Fatalf("clicks outside changed mode %+v or focus %d", m.briefingMode, m.modeFocus)
	}
}

func TestBriefingMouseClickBetweenColumnsDoesNothing(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// The checkbox row holds both columns with a gap between them.
	y0 := rowOf(t, m, "Reference check")

	// Left column box is at x=2. Find where the right column starts.
	rightColStart := columnOf(t, m, y0, "Show reference") - 4 // box starts 4 cells before its label
	// The gap runs from the end of the left column cell to rightColStart.
	colWidth, _ := briefingColumnLayout(80)
	leftCellEnd := 2 + colWidth // indent 2 + cell width

	initialMode := m.briefingMode
	initialFocus := m.modeFocus

	// Click each gap cell: none should toggle anything.
	for x := leftCellEnd; x < rightColStart; x++ {
		if idx, ok := m.briefingCheckboxAt(x, y0); ok {
			t.Fatalf("gap click at x=%d on row %d hit checkbox %d", x, y0, idx)
		}
		m.Update(leftClick(x, y0))
		if m.briefingMode != initialMode || m.modeFocus != initialFocus {
			t.Fatalf("gap click at x=%d changed mode %+v or focus %d", x, m.briefingMode, m.modeFocus)
		}
	}
}

func TestBriefingMouseDisabled(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.mouseOff = true

	y0 := rowOf(t, m, "Reference check")
	initialMode := m.briefingMode
	initialFocus := m.modeFocus

	// Mouse click must do nothing when mouse is off.
	cmd := send(m, leftClick(2, y0))
	if cmd != nil {
		t.Fatalf("mouse click with mouseOff produced cmd: %v", cmd)
	}
	if m.briefingMode != initialMode || m.modeFocus != initialFocus {
		t.Fatalf("mouse click with mouseOff changed mode or focus")
	}

	// Keyboard navigation must continue working normally.
	m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m.modeFocus != 1 {
		t.Fatalf("Tab key with mouseOff: focus = %d, want 1", m.modeFocus)
	}
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if m.briefingMode.ShowReference {
		t.Fatalf("Space key with mouseOff did not toggle ShowReference off: %+v", m.briefingMode)
	}
	m.Update(runeKey("4"))
	if m.briefingMode != mode.Exam() {
		t.Fatalf("Preset key '4' with mouseOff did not apply Exam: %+v", m.briefingMode)
	}
}

func TestBriefingMouseScrollingAndFooterPreserved(t *testing.T) {
	restoreDefaultTheme(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// 1. Wheel scrolling works.
	initOffset := m.scrollOffset
	m.Update(wheel(false, 10, 10)) // wheel down
	if m.scrollOffset != initOffset+wheelStep {
		t.Fatalf("wheel down offset = %d, want %d", m.scrollOffset, initOffset+wheelStep)
	}
	m.Update(wheel(true, 10, 10)) // wheel up
	if m.scrollOffset != initOffset {
		t.Fatalf("wheel up offset = %d, want %d", m.scrollOffset, initOffset)
	}

	// 2. Footer items can still be clicked.
	footerRow := rowOf(t, m, "[Esc/q] back")
	idx := strings.Index(stripLines(m)[footerRow], "[Esc/q]")
	if idx < 0 {
		t.Fatal("could not find [Esc/q] in footer row")
	}
	cmd := send(m, leftClick(idx+1, footerRow))
	if cmd == nil {
		t.Fatal("clicking footer item produced no cmd")
	}
	if m.helpPress == nil || m.helpPress.key.String() != "esc" {
		t.Fatalf("clicking footer item did not set helpPress: %+v", m.helpPress)
	}
}

func TestBriefingMouseThemeStyling(t *testing.T) {
	restoreDefaultTheme(t)
	forceTrueColor(t)
	m := newBriefingModeModel(t, "ft_strlen")
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	colWidth, _ := briefingColumnLayout(80)

	// Focus 0 is styled with AccentStyle (left column, padded to colWidth).
	view := m.View()
	want0 := AccentStyle.Render(padCell("[x] Reference check", colWidth))
	if !strings.Contains(view, want0) {
		t.Fatalf("view does not contain styled option 0:\n%s", view)
	}

	// Click option 2: focus moves to option 2 and it gets AccentStyle.
	y2 := rowOf(t, m, "Require exact copy")
	x2 := checkboxBoxX(t, m, y2, 2, "Require exact copy")
	m.Update(leftClick(x2, y2))
	view = m.View()
	want2 := AccentStyle.Render(padCell("[x] Require exact copy", colWidth))
	if !strings.Contains(view, want2) {
		t.Fatalf("view does not contain styled option 2:\n%s", view)
	}
	// Option 0 must no longer be styled with AccentStyle.
	if strings.Contains(view, want0) {
		t.Fatalf("view still contains styled option 0 after focus moved to 2")
	}
}
