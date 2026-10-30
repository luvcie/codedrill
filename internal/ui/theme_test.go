package ui

import (
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/storage"
)

// fg is the foreground SGR parameters lipgloss writes for a palette colour,
// such as "38;2;0;240;255". The checks use what lipgloss writes, not the hex
// digits, because lipgloss rounds some colours by one step.
func fg(c lipgloss.Color) string { return sgrParams(lipgloss.NewStyle().Foreground(c)) }

// bg is the background counterpart of fg.
func bg(c lipgloss.Color) string { return sgrParams(lipgloss.NewStyle().Background(c)) }

// sgrParams returns the parameters of the SGR sequence that style s opens.
func sgrParams(s lipgloss.Style) string {
	out := s.Render("x")
	start := strings.Index(out, "\x1b[") + len("\x1b[")
	return out[start : start+strings.IndexByte(out[start:], 'm')]
}

// restoreDefaultTheme returns the palette to the default when the test ends.
func restoreDefaultTheme(t *testing.T) {
	t.Helper()
	t.Cleanup(func() { ApplyTheme(Themes[0]) })
}

// useTheme applies the theme with the given ID for the rest of the test. The
// palette that was active before is put back when the test ends.
func useTheme(t *testing.T, id string) Theme {
	t.Helper()
	th, ok := ThemeByID(id)
	if !ok {
		t.Fatalf("theme %s is missing", id)
	}
	prev := CurrentTheme()
	t.Cleanup(func() { ApplyTheme(prev) })
	ApplyTheme(th)
	return th
}

// roleColours lists every role of a theme by name.
func roleColours(th Theme) map[string]lipgloss.Color {
	return map[string]lipgloss.Color{
		"Accent": th.Accent, "Success": th.Success, "Danger": th.Danger,
		"Text": th.Text, "Muted": th.Muted, "Border": th.Border,
		"Logo": th.Logo, "OnAccent": th.OnAccent, "DescSelected": th.DescSelected,
		"ListTitle": th.ListTitle, "ListDesc": th.ListDesc,
	}
}

func TestThemesHaveEveryRoleAndUniqueIDs(t *testing.T) {
	if len(Themes) != 12 {
		t.Fatalf("got %d themes, want 12", len(Themes))
	}
	if Themes[0].ID != "legacy" {
		t.Fatalf("Themes[0] is %q, want legacy", Themes[0].ID)
	}
	if Themes[1].ID != "cyan" {
		t.Fatalf("Themes[1] is %q, want cyan", Themes[1].ID)
	}
	seen := map[string]bool{}
	for _, th := range Themes {
		if th.ID == "" || th.Name == "" {
			t.Errorf("theme %+v has no ID or name", th)
		}
		if seen[th.ID] {
			t.Errorf("duplicate theme ID %q", th.ID)
		}
		seen[th.ID] = true
		for role, c := range roleColours(th) {
			if len(c) != 7 || c[0] != '#' {
				t.Errorf("%s: role %s is %q, want #RRGGBB", th.ID, role, c)
			}
		}
	}
}

func TestDefaultThemeIsLegacy(t *testing.T) {
	want := map[string]lipgloss.Color{
		"Accent": "#9CC26A", "Success": "#D4E157", "Danger": "#D9774A",
		"Text": "#E6EBD8", "Muted": "#8C9A78", "Border": "#3E4A32",
		"Logo": "#7FA05A", "OnAccent": "#1A2214", "DescSelected": "#EEF2E0",
		"ListTitle": "#E6EBD8", "ListDesc": "#8C9A78",
	}
	got := roleColours(Themes[0])
	for role, c := range want {
		if got[role] != c {
			t.Errorf("default %s = %s, want %s", role, got[role], c)
		}
	}
	if Themes[0].ID != "legacy" || Themes[0].Name != "Legacy codedrill" {
		t.Errorf("default theme is %q (%s), want legacy (Legacy codedrill)", Themes[0].ID, Themes[0].Name)
	}
	if Themes[0].Light {
		t.Error("default theme is marked light")
	}

	restoreDefaultTheme(t)
	ApplyTheme(Themes[0])
	for name, pair := range map[string][2]lipgloss.Color{
		"ColorSecondary":    {ColorSecondary, "#9CC26A"},
		"ColorSuccess":      {ColorSuccess, "#D4E157"},
		"ColorDanger":       {ColorDanger, "#D9774A"},
		"ColorMuted":        {ColorMuted, "#8C9A78"},
		"ColorBorder":       {ColorBorder, "#3E4A32"},
		"ColorBg":           {ColorBg, "#1A2214"},
		"ColorText":         {ColorText, "#E6EBD8"},
		"ColorDescSelected": {ColorDescSelected, "#EEF2E0"},
		"ColorListTitle":    {ColorListTitle, "#E6EBD8"},
		"ColorListDesc":     {ColorListDesc, "#8C9A78"},
		"ColorCroc":         {ColorCroc, "#7FA05A"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %s, want %s", name, pair[0], pair[1])
		}
	}
}

// TestCyanThemeKeepsOldValues pins the cyan palette to the colours codedrill
// drew before the legacy palette became the default.
func TestCyanThemeKeepsOldValues(t *testing.T) {
	cyan, ok := ThemeByID("cyan")
	if !ok {
		t.Fatal("cyan theme is missing")
	}
	if cyan.Name != "Cyan" {
		t.Errorf("cyan name = %q, want Cyan", cyan.Name)
	}
	want := map[string]lipgloss.Color{
		"Accent": "#00F0FF", "Success": "#00FF66", "Danger": "#FF4D4D",
		"Text": "#FFFFFF", "Muted": "#969696", "Border": "#4A4A55",
		"Logo": "#7FA05A", "OnAccent": "#1A1A24", "DescSelected": "#EEEEEE",
		"ListTitle": "#DDDDDD", "ListDesc": "#777777",
	}
	got := roleColours(cyan)
	for role, c := range want {
		if got[role] != c {
			t.Errorf("cyan %s = %s, want %s", role, got[role], c)
		}
	}
	if cyan.Light {
		t.Error("cyan theme is marked light")
	}

	restoreDefaultTheme(t)
	useTheme(t, "cyan")
	for name, pair := range map[string][2]lipgloss.Color{
		"ColorSecondary":    {ColorSecondary, "#00F0FF"},
		"ColorSuccess":      {ColorSuccess, "#00FF66"},
		"ColorDanger":       {ColorDanger, "#FF4D4D"},
		"ColorMuted":        {ColorMuted, "#969696"},
		"ColorBorder":       {ColorBorder, "#4A4A55"},
		"ColorBg":           {ColorBg, "#1A1A24"},
		"ColorText":         {ColorText, "#FFFFFF"},
		"ColorDescSelected": {ColorDescSelected, "#EEEEEE"},
		"ColorListTitle":    {ColorListTitle, "#DDDDDD"},
		"ColorListDesc":     {ColorListDesc, "#777777"},
		"ColorCroc":         {ColorCroc, "#7FA05A"},
	} {
		if pair[0] != pair[1] {
			t.Errorf("%s = %s, want %s", name, pair[0], pair[1])
		}
	}
}

// relLuminance is the WCAG relative luminance of a #RRGGBB colour.
func relLuminance(t *testing.T, c lipgloss.Color) float64 {
	t.Helper()
	v, err := strconv.ParseUint(strings.TrimPrefix(string(c), "#"), 16, 32)
	if err != nil || len(c) != 7 {
		t.Fatalf("%q is not #RRGGBB", c)
	}
	lin := func(x uint64) float64 {
		s := float64(x) / 255
		if s <= 0.03928 {
			return s / 12.92
		}
		return math.Pow((s+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(v>>16&0xFF) + 0.7152*lin(v>>8&0xFF) + 0.0722*lin(v&0xFF)
}

// contrastRatio is the WCAG contrast ratio between two palette colours.
func contrastRatio(t *testing.T, a, b lipgloss.Color) float64 {
	t.Helper()
	la, lb := relLuminance(t, a), relLuminance(t, b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

// themeBackground is the terminal background a theme is drawn on: #1E1E1E for
// the dark themes and #FFFFFF for Light.
func themeBackground(th Theme) lipgloss.Color {
	if th.Light {
		return "#FFFFFF"
	}
	return "#1E1E1E"
}

func TestThemeContrast(t *testing.T) {
	for _, th := range Themes {
		bg := themeBackground(th)
		if r := contrastRatio(t, th.Muted, bg); r < 4.5 {
			t.Errorf("%s: Muted %s on %s is %.2f, want >= 4.5", th.ID, th.Muted, bg, r)
		}
		if r := contrastRatio(t, th.Text, bg); r < 7 {
			t.Errorf("%s: Text %s on %s is %.2f, want >= 7", th.ID, th.Text, bg, r)
		}
	}
}

func TestMonochromeRolesAreDistinct(t *testing.T) {
	mono, ok := ThemeByID("monochrome")
	if !ok {
		t.Fatal("monochrome theme is missing")
	}
	// Grey levels, so one channel is enough for the distance.
	level := func(c lipgloss.Color) int {
		v, _ := strconv.ParseUint(strings.TrimPrefix(string(c), "#"), 16, 32)
		return int(v & 0xFF)
	}
	roles := map[string]lipgloss.Color{"Success": mono.Success, "Text": mono.Text, "Danger": mono.Danger, "Muted": mono.Muted}
	for _, pair := range [][2]string{{"Success", "Text"}, {"Text", "Danger"}, {"Danger", "Muted"}} {
		a, b := roles[pair[0]], roles[pair[1]]
		if d := int(math.Abs(float64(level(a) - level(b)))); d < 20 {
			t.Errorf("monochrome %s %s and %s %s are %d apart, want >= 20", pair[0], a, pair[1], b, d)
		}
	}
	if r := contrastRatio(t, mono.Danger, themeBackground(mono)); r < 4.5 {
		t.Errorf("monochrome Danger %s on the dark background is %.2f, want >= 4.5", mono.Danger, r)
	}
}

func TestThemeByID(t *testing.T) {
	for _, tc := range []struct {
		id   string
		want string
		ok   bool
	}{
		{"gruvbox", "gruvbox", true},
		{"GRUVBOX", "gruvbox", true},
		{"Tokyo-Night", "tokyo-night", true},
		{"cyan", "cyan", true},
		{"legacy", "legacy", true},
		{"default", "cyan", true},
		{"DEFAULT", "cyan", true},
		{"nope", "", false},
		{"", "", false},
	} {
		got, ok := ThemeByID(tc.id)
		if ok != tc.ok || got.ID != tc.want {
			t.Errorf("ThemeByID(%q) = %q, %v; want %q, %v", tc.id, got.ID, ok, tc.want, tc.ok)
		}
	}
}

func TestThemeOrDefaultFallsBack(t *testing.T) {
	for _, id := range []string{"", "nope", "bogus-theme"} {
		if got := ThemeOrDefault(id); got.ID != "legacy" {
			t.Errorf("ThemeOrDefault(%q) = %q, want legacy", id, got.ID)
		}
	}
	if got := ThemeOrDefault("Nord"); got.ID != "nord" {
		t.Errorf("ThemeOrDefault(Nord) = %q, want nord", got.ID)
	}
}

// The old ID "default" meant the cyan palette, so a saved "default" keeps it.
func TestThemeOrDefaultAliasesOldDefaultToCyan(t *testing.T) {
	for _, id := range []string{"default", "Default"} {
		if got := ThemeOrDefault(id); got.ID != "cyan" {
			t.Errorf("ThemeOrDefault(%q) = %q, want cyan", id, got.ID)
		}
	}
}

func TestApplyThemeSetsCurrentTheme(t *testing.T) {
	restoreDefaultTheme(t)
	ApplyTheme(Themes[2])
	if got := CurrentTheme(); got.ID != "gruvbox" {
		t.Fatalf("CurrentTheme() = %q, want gruvbox", got.ID)
	}
	if ColorSecondary != Themes[2].Accent || ColorCroc != Themes[2].Logo {
		t.Fatalf("colours did not follow gruvbox: accent %s, logo %s", ColorSecondary, ColorCroc)
	}
	if AccentStyle.GetForeground() != Themes[2].Accent {
		t.Fatalf("AccentStyle kept the old colour: %v", AccentStyle.GetForeground())
	}
}

// themeScreens renders every screen the theme colours, keyed by name. The
// add-drill form is built from the palette at open time, so it is rebuilt
// with applyTheme and not with ApplyTheme alone.
func themeScreens(t *testing.T, m *MainModel) map[string]string {
	t.Helper()
	views := map[string]string{}
	render := func(name string, state AppState) {
		m.state = state
		m.scrollOffset = 0
		views[name] = m.View()
	}
	render("menu", StateMenu)
	render("briefing", StateBriefing)
	render("results", StateResults)
	render("profile", StateStats)
	render("hall of fame", StateLeaderboard)
	m.countdownCount = 3
	render("countdown", StateCountdown)
	render("add drill", StateAddDrill)
	return views
}

// newThemeTestModel is a model at 120x200 with one drill that has runs, and
// an open add-drill form. Its description has no inline code, so glamour's
// code background does not show up in the checks.
func newThemeTestModel(t *testing.T) *MainModel {
	t.Helper()
	m := newEvaluationTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 200})
	m.activeDrill.Description = "# ft_strlen\n\nWrite a function that returns the length of a string.\n\n## Constraints\n\n- Reproduce standard behavior."
	if _, _, err := m.history.RecordAttempt("ft_strlen", 3200*time.Millisecond, true, 1, false); err != nil {
		t.Fatal(err)
	}
	m.lastDuration = 6 * time.Second
	m.resultDrill = *m.activeDrill
	m.isPB = true
	m.pbDelta = 500 * time.Millisecond
	m.lastSubmits = 1
	m.earnedXP = 125
	m.levelInfo = storage.GetLevelInfo(125)
	m.evalPassed = true
	m.evalOutput = "ok\tsolution\t0.01s"
	m.state = StateMenu
	m.initForm()
	// Enter on the add item calls Init, which focuses the first field.
	m.form.Init()
	return m
}

func TestThemeRecoloursEveryScreen(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	m := newThemeTestModel(t)
	ApplyTheme(Themes[0])

	prev := Themes[0]
	for _, th := range Themes {
		m.applyTheme(th)
		views := themeScreens(t, m)
		if !strings.Contains(views["menu"], fg(th.Accent)) {
			t.Errorf("%s: menu lacks the accent %s", th.ID, fg(th.Accent))
		}
		for name, view := range views {
			if !strings.Contains(view, fg(th.Accent)) {
				t.Errorf("%s: %s lacks the accent %s", th.ID, name, fg(th.Accent))
			}
			// The old accent must be gone unless the new theme reuses it.
			if th.ID != prev.ID && !sharesRole(prev.Accent, th) {
				for _, old := range []string{fg(prev.Accent), bg(prev.Accent)} {
					if strings.Contains(view, old) {
						t.Errorf("%s: %s still has the %s accent %s", th.ID, name, prev.ID, old)
					}
				}
			}
		}
		prev = th
	}
}

// sharesRole reports whether colour c is also a role of th.
func sharesRole(c lipgloss.Color, th Theme) bool {
	for _, other := range roleColours(th) {
		if other == c {
			return true
		}
	}
	return false
}

func TestBriefingCacheFollowsTheme(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	m := newThemeTestModel(t)
	ApplyTheme(Themes[0])
	m.state = StateBriefing
	_ = m.View()
	if m.briefingCache == "" {
		t.Fatal("briefing was not rendered")
	}

	gruvbox, _ := ThemeByID("gruvbox")
	m.applyTheme(gruvbox)
	if m.briefingCache != "" || m.briefingKey != "" {
		t.Fatal("applyTheme kept the briefing cache")
	}
	// The heading text is in the theme's text colour, so it shows the change.
	if view := m.View(); !strings.Contains(view, fg(gruvbox.Text)) {
		t.Fatalf("briefing did not re-render in gruvbox text colour %s", fg(gruvbox.Text))
	}
}

func TestApplyThemeKeepsListPlaceAndCategory(t *testing.T) {
	restoreDefaultTheme(t)
	m := newMenuTestModel(t)
	m.Update(runeKey("l"))
	if len(m.list.Items()) < 2 {
		t.Fatal("test needs at least two rows in the first category")
	}
	m.list.Select(1)
	category, index := m.categoryFilter, m.categoryIndex

	gruvbox, _ := ThemeByID("gruvbox")
	m.applyTheme(gruvbox)
	ApplyTheme(Themes[0])
	if m.categoryFilter != category || m.categoryIndex != index {
		t.Errorf("category changed to %q (%d), want %q (%d)", m.categoryFilter, m.categoryIndex, category, index)
	}
	if m.list.Index() != 1 {
		t.Errorf("list index is %d after a theme change, want 1", m.list.Index())
	}
}

func TestAppliedThemeReachesOpenForm(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	m := openAddDrillForm(t)
	nord, _ := ThemeByID("nord")
	m.applyTheme(nord)
	if view := m.View(); !strings.Contains(view, fg(nord.Accent)) {
		t.Fatalf("open add drill form lacks the nord accent %s", fg(nord.Accent))
	}
}

func TestNewFormUsesCurrentTheme(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	ApplyTheme(Themes[2])
	m := openAddDrillForm(t)
	if view := m.View(); !strings.Contains(view, fg(Themes[2].Accent)) {
		t.Fatalf("form opened after a theme change lacks the gruvbox accent %s", fg(Themes[2].Accent))
	}
}

func TestLightThemeBriefingMarkdown(t *testing.T) {
	forceTrueColor(t)
	restoreDefaultTheme(t)
	light, ok := ThemeByID("light")
	if !ok || !light.Light {
		t.Fatal("light theme is missing or not marked light")
	}
	ApplyTheme(light)
	out := RenderMarkdown("# Title\n\nBody with **bold** text.\n\n## Notes", 80)
	plain := ansi.Strip(out)
	for _, want := range []string{"Title", "Body with bold text", "Notes"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("light briefing lacks %q:\n%s", want, plain)
		}
	}
	if !strings.Contains(out, fg(light.Text)) {
		t.Fatalf("light briefing lacks the light text colour %s:\n%q", fg(light.Text), out)
	}
}

func TestBriefingStyleCopiesGlamourBase(t *testing.T) {
	before := styles.DarkStyleConfig.Heading.Color
	beforeLight := styles.LightStyleConfig.Heading.Color
	for _, th := range []Theme{Themes[0], Themes[len(Themes)-1]} {
		_ = briefingStyle(th)
	}
	if styles.DarkStyleConfig.Heading.Color != before || styles.LightStyleConfig.Heading.Color != beforeLight {
		t.Fatal("building the briefing style changed glamour's global styles")
	}
}
