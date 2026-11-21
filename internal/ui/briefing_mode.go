package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/reference"
	"github.com/luvcie/codedrill/internal/solutions"
)

const briefingOptionCount = 5

func modePresetName(opts mode.Options) string {
	switch opts {
	case mode.Study(), mode.Copy():
		return "Study"
	case mode.Exam():
		return "Exam"
	}
	return "Custom"
}

func referenceLanguageSupported(d *drill.Drill) bool {
	return d != nil && reference.SupportedLanguage(d.Language)
}

// loadBriefingOptions loads a drill's saved gates into the briefing. A real
// load failure enters the existing preference error screen instead of showing
// invented selections.
func (m *MainModel) loadBriefingOptions(d *drill.Drill) bool {
	if d == nil {
		return false
	}
	prefs, err := mode.LoadPreferences(d.ID)
	if err != nil {
		m.setupErr = nil
		m.prefsErr = err
		m.state = StateSetupError
		return false
	}
	m.prefsErr = nil
	m.briefingReference = prefs.ReferenceVariant
	m.briefingNoLint = prefs.NoLint
	m.setBriefingOptions(d, prefs.Options)
	return true
}

func (m *MainModel) setBriefingOptions(d *drill.Drill, opts mode.Options) {
	m.briefingMode = opts
	m.briefingModeID = d.ID
	m.modeFocus = 0
	m.briefingSaveErr = nil
	m.refreshBriefingReference(d)
}

func (m *MainModel) refreshBriefingReference(d *drill.Drill) {
	m.briefingRefOK = false
	m.briefingRefErr = nil
	if d == nil {
		return
	}
	src, err := solutions.ResolveVariant(d.ID, d.Dir, m.briefingReference)
	if err != nil {
		m.briefingRefErr = err
		return
	}
	m.briefingRefOK = src != nil && len(src.Files) > 0
}

// briefingProblem is the blocking validation shown inline. It never edits the
// selected gates; it only refuses to start the sprint.
func (m *MainModel) briefingProblem() string {
	opts := m.briefingMode.Normalize()
	if err := opts.Validate(); err != nil {
		return "Enable at least one pass gate."
	}
	if opts.RequireExact {
		if !referenceLanguageSupported(m.activeDrill) {
			return "Exact copy is only available for C and C++ drills."
		}
		if !m.briefingRefOK {
			return "Exact copy needs an attached reference."
		}
	}
	return ""
}

func (m *MainModel) briefingReferenceNotice() string {
	if m.activeDrill == nil || !m.briefingMode.CheckReference || m.briefingProblem() != "" {
		return ""
	}
	if !referenceLanguageSupported(m.activeDrill) {
		language := strings.ToUpper(strings.TrimSpace(m.activeDrill.Language))
		if language == "" {
			language = "this drill"
		}
		return "Reference unavailable for " + language + "; tests still decide success."
	}
	if m.briefingRefErr != nil {
		return "Reference unavailable: " + m.briefingRefErr.Error() + "; tests still decide success."
	}
	if !m.briefingRefOK {
		return "No reference attached; tests still decide success."
	}
	return ""
}

func (m *MainModel) focusBriefingOption(delta int) {
	count := briefingOptionCount
	if m.hasReferenceChoice() {
		count++
	}
	m.modeFocus = (m.modeFocus + delta + count) % count
}

func (m *MainModel) hasReferenceChoice() bool {
	return m.activeDrill != nil && m.activeDrill.ID == "rank05-vect2" && m.activeDrill.Dir == ""
}

func (m *MainModel) briefingReferenceChoice() string {
	label := "Header-only"
	if m.briefingReference == solutions.SplitFiles {
		label = "Split files"
	}
	return "Reference: < " + label + " >"
}

func (m *MainModel) toggleBriefingOption() {
	if m.activeDrill == nil {
		return
	}
	switch m.modeFocus {
	case 0:
		m.briefingMode.CheckReference = !m.briefingMode.CheckReference
	case 1:
		m.briefingMode.ShowReference = !m.briefingMode.ShowReference
		if m.briefingMode.ShowReference {
			m.briefingMode.CheckReference = true
		}
	case 2:
		switch m.briefingMode {
		case mode.Study():
			m.briefingMode = mode.Copy()
		case mode.Copy():
			m.briefingMode = mode.Study()
		default:
			m.briefingMode.RequireExact = !m.briefingMode.RequireExact
			if m.briefingMode.RequireExact {
				m.briefingMode.CheckReference = true
			}
		}
	case 3:
		m.briefingMode.RequireTests = !m.briefingMode.RequireTests
	case 4:
		m.briefingNoLint = !m.briefingNoLint
	case briefingOptionCount:
		if !m.hasReferenceChoice() {
			return
		}
		if m.briefingReference == solutions.SplitFiles {
			m.briefingReference = solutions.HeaderOnly
		} else {
			m.briefingReference = solutions.SplitFiles
		}
	default:
		return
	}
	m.briefingMode = m.briefingMode.Normalize()
	m.refreshBriefingReference(m.activeDrill)
	m.saveBriefingMode()
}

var briefingPresets = []mode.Options{
	mode.Study(),
	mode.Exam(),
}

func (m *MainModel) cycleBriefingPreset(delta int) {
	if m.activeDrill == nil {
		return
	}
	current := -1
	for i, p := range briefingPresets {
		if m.briefingMode == p || (p == mode.Study() && m.briefingMode == mode.Copy()) {
			current = i
			break
		}
	}
	var next int
	if current < 0 {
		if delta > 0 {
			next = 0
		} else {
			next = len(briefingPresets) - 1
		}
	} else {
		next = (current + delta + len(briefingPresets)) % len(briefingPresets)
	}
	m.applyBriefingPreset(briefingPresets[next])
}

func (m *MainModel) applyBriefingPreset(opts mode.Options) {
	if m.activeDrill == nil {
		return
	}
	if opts == mode.Study() && m.briefingMode == mode.Copy() {
		opts = mode.Copy()
	}
	m.briefingMode = opts
	m.refreshBriefingReference(m.activeDrill)
	m.saveBriefingMode()
}

func (m *MainModel) saveBriefingMode() {
	if m.activeDrill == nil {
		return
	}
	m.briefingSaveErr = mode.SavePreferences(m.activeDrill.ID, mode.Preferences{
		Options: m.briefingMode, ReferenceVariant: m.briefingReference, NoLint: m.briefingNoLint,
	})
}

// briefingColumnLayout sizes the two-column checkbox panel. colWidth fits the
// widest checkbox cell ("[x] Copy reference exactly"); gap is the space between
// columns. Returns gap < 0 when the panel is too narrow for two columns.
func briefingColumnLayout(width int) (colWidth, gap int) {
	panelW := boxWidth(width, 40) - 4 // indentLines adds two spaces per side
	const minGap = 3
	const widestCell = 26 // "[x] Copy reference exactly"
	if panelW < 2*widestCell+minGap {
		return panelW, -1
	}
	return widestCell, minGap
}

// padCell right-pads s with spaces to reach the given display width.
func padCell(s string, width int) string {
	if ansi.StringWidth(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-ansi.StringWidth(s))
}

type briefingCheckbox struct {
	label   string
	checked bool
}

func briefingCheckboxText(opt briefingCheckbox) string {
	if opt.checked {
		return "[x] " + opt.label
	}
	return "[ ] " + opt.label
}

func briefingCheckboxCell(opt briefingCheckbox, focused bool) string {
	text := briefingCheckboxText(opt)
	if focused {
		return AccentStyle.Render(text)
	}
	return text
}

func (m *MainModel) briefingModePanel() string {
	options := []briefingCheckbox{
		{"Reference check", m.briefingMode.CheckReference},
		{"Show reference", m.briefingMode.ShowReference},
		{"Copy reference exactly", m.briefingMode.RequireExact},
		{"Require tests", m.briefingMode.RequireTests},
		{"Linter off (Helix)", m.briefingNoLint},
	}

	lines := make([]string, 0, 8)
	lines = append(lines, labelValue("Mode:", fmt.Sprintf("< %s >", modePresetName(m.briefingMode)), AccentStyle))

	w, _ := m.screenSize()
	colWidth, gap := briefingColumnLayout(w)

	if gap < 0 {
		// Vertical fallback for narrow terminals.
		for i, opt := range options {
			lines = append(lines, briefingCheckboxCell(opt, i == m.modeFocus))
		}
	} else {
		// Two columns: row 0 holds options 0 and 1, row 1 holds options 2 and 3; option 4 has its own row.
		for row := 0; row < 2; row++ {
			var cells []string
			for col := 0; col < 2; col++ {
				i := row*2 + col
				cell := padCell(briefingCheckboxText(options[i]), colWidth)
				if i == m.modeFocus {
					cell = AccentStyle.Render(cell)
				}
				cells = append(cells, cell)
			}
			lines = append(lines, strings.Join(cells, strings.Repeat(" ", gap)))
		}
		lines = append(lines, briefingCheckboxCell(options[4], m.modeFocus == 4))
	}

	if m.hasReferenceChoice() {
		choice := m.briefingReferenceChoice()
		if m.modeFocus == briefingOptionCount {
			choice = AccentStyle.Render(choice)
		}
		lines = append(lines, choice)
	}

	if problem := m.briefingProblem(); problem != "" {
		lines = append(lines, DangerStyle.Render(problem))
	} else if notice := m.briefingReferenceNotice(); notice != "" {
		lines = append(lines, SubtextStyle.Render(notice))
	}
	if m.briefingSaveErr != nil {
		lines = append(lines, DangerStyle.Render("Preferences not saved: "+m.briefingSaveErr.Error()))
	}
	return strings.Join(lines, "\n")
}
