package ui

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
)

func searchKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func typeSearch(m *MainModel, text string) {
	for _, r := range text {
		m.Update(searchKey(string(r)))
	}
}

func selectedID(m *MainModel) string {
	it, ok := m.list.SelectedItem().(drillItem)
	if !ok || it.isAdd {
		return ""
	}
	return it.drill.ID
}

// switchToCategory moves the menu to the named tab with h or l.
func switchToCategory(t *testing.T, m *MainModel, name string) {
	t.Helper()
	for range m.categories {
		if m.categoryFilter == name {
			return
		}
		m.Update(runeKey("l"))
	}
	t.Fatalf("no category %q in %v", name, m.categories)
}

func categoryNamed(t *testing.T, m *MainModel, name string) int {
	t.Helper()
	for i, c := range m.categories {
		if c == name {
			return i
		}
	}
	t.Fatalf("no category %q in %v", name, m.categories)
	return -1
}

func quitCmd(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

func TestSearchSmartcase(t *testing.T) {
	strlen := drill.Drill{ID: "ft_strlen", Title: "ft_strlen"}
	if len(matchRunes(strlen.ID, "str")) == 0 || !drillMatches(strlen, "str") {
		t.Fatal("lowercase pattern should match ft_strlen")
	}
	if drillMatches(strlen, "Str") {
		t.Fatal("pattern with an uppercase letter should be case-sensitive")
	}
	titled := drill.Drill{ID: "reverse", Title: "Reverse Str"}
	if !drillMatches(titled, "Str") {
		t.Fatal("Str should match a title containing Str")
	}
	if !drillMatches(drill.Drill{ID: "two_sum", Title: "Two Sum"}, "two sum") {
		t.Fatal("lowercase pattern should match the title case-insensitively")
	}
	if drillMatches(strlen, "") {
		t.Fatal("empty pattern should match nothing")
	}
}

func TestSearchPromptFindsDrillInOtherTab(t *testing.T) {
	m := newMenuTestModel(t)
	switchToCategory(t, m, "42")
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if m.search.open {
		t.Fatal("Enter should close the prompt")
	}
	if m.categoryFilter != "LeetCode" || selectedID(m) != "palindrome" {
		t.Fatalf("want palindrome on LeetCode, got %q on %q", selectedID(m), m.categoryFilter)
	}
	if m.search.pattern != "palindrome" || !m.search.hl {
		t.Fatalf("pattern %q hl %v, want palindrome highlighted", m.search.pattern, m.search.hl)
	}
	if m.state != StateMenu {
		t.Fatalf("state %v, want menu", m.state)
	}
}

func TestSearchFindsByID(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	typeSearch(m, "two_sum")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if selectedID(m) != "two_sum" {
		t.Fatalf("selected %q, want two_sum", selectedID(m))
	}
}

func TestSearchNAndNWrap(t *testing.T) {
	m := newMenuTestModel(t)
	switchToCategory(t, m, "42")
	var ids []string
	for _, it := range m.list.Items() {
		if di := it.(drillItem); !di.isAdd {
			ids = append(ids, di.drill.ID)
		}
	}
	if len(ids) < 3 {
		t.Fatalf("need several 42 drills, got %v", ids)
	}
	m.Update(runeKey("/"))
	typeSearch(m, "ft_")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if selectedID(m) != ids[0] {
		t.Fatalf("Enter selected %q, want %q", selectedID(m), ids[0])
	}

	for i := 1; i < len(ids); i++ {
		m.Update(runeKey("n"))
		if selectedID(m) != ids[i] || m.menuNotice != "" {
			t.Fatalf("n step %d: selected %q notice %q, want %q", i, selectedID(m), m.menuNotice, ids[i])
		}
	}
	m.Update(runeKey("n"))
	if selectedID(m) != ids[0] {
		t.Fatalf("n should wrap to %q, got %q", ids[0], selectedID(m))
	}
	if m.menuNotice != "search hit BOTTOM, continuing at TOP" {
		t.Fatalf("notice %q", m.menuNotice)
	}
	if !strings.Contains(m.View(), "search hit BOTTOM, continuing at TOP") {
		t.Fatal("wrap notice not shown")
	}

	m.Update(runeKey("N"))
	if selectedID(m) != ids[len(ids)-1] {
		t.Fatalf("N should wrap to %q, got %q", ids[len(ids)-1], selectedID(m))
	}
	if m.menuNotice != "search hit TOP, continuing at BOTTOM" {
		t.Fatalf("notice %q", m.menuNotice)
	}
}

func TestSearchNWithoutPattern(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("n"))
	if m.menuNotice != "No previous search" {
		t.Fatalf("notice %q", m.menuNotice)
	}
}

func TestSearchNNotFoundKeepsPattern(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.search.pattern = "zzqx"
	m.Update(runeKey("n"))
	if m.menuNotice != "Pattern not found: zzqx" {
		t.Fatalf("notice %q", m.menuNotice)
	}
}

func TestSearchLiveJumpBeforeEnter(t *testing.T) {
	m := newMenuTestModel(t)
	switchToCategory(t, m, "42")
	m.Update(runeKey("/"))
	typeSearch(m, "pal")
	if !m.search.open || selectedID(m) != "palindrome" || m.categoryFilter != "LeetCode" {
		t.Fatalf("live jump: open %v selected %q on %q", m.search.open, selectedID(m), m.categoryFilter)
	}
	if m.search.pattern != "" {
		t.Fatal("pattern should only be stored on Enter")
	}
}

func TestSearchEscRestoresOrigin(t *testing.T) {
	m := newMenuTestModel(t)
	switchToCategory(t, m, "42")
	m.list.Select(2)
	origin, index := m.categoryIndex, m.list.Index()

	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	if m.categoryIndex == origin {
		t.Fatal("live jump should have moved to another tab")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.search.open || m.categoryIndex != origin || m.list.Index() != index {
		t.Fatalf("esc: open %v tab %d index %d, want tab %d index %d", m.search.open, m.categoryIndex, m.list.Index(), origin, index)
	}
	if m.search.pattern != "" {
		t.Fatalf("esc should not store a pattern, got %q", m.search.pattern)
	}
}

func TestSearchEscKeepsPreviousPattern(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("/"))
	typeSearch(m, "ft_s")
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.search.pattern != "palindrome" {
		t.Fatalf("pattern %q, want palindrome", m.search.pattern)
	}
}

func TestSearchBackspaceOnEmptyRestoresOrigin(t *testing.T) {
	m := newMenuTestModel(t)
	switchToCategory(t, m, "42")
	origin, index := m.categoryIndex, m.list.Index()

	m.Update(runeKey("/"))
	typeSearch(m, "two")
	if selectedID(m) != "two_sum" {
		t.Fatalf("live jump selected %q", selectedID(m))
	}
	for range 3 {
		m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}
	if !m.search.open || m.search.text != "" {
		t.Fatalf("backspace should keep the prompt open with empty text, open %v text %q", m.search.open, m.search.text)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	if m.search.open || m.categoryIndex != origin || m.list.Index() != index {
		t.Fatalf("backspace on empty should close and restore origin, open %v tab %d index %d", m.search.open, m.categoryIndex, m.list.Index())
	}
}

func TestSearchNotFoundRestoresOrigin(t *testing.T) {
	m := newMenuTestModel(t)
	switchToCategory(t, m, "42")
	m.list.Select(1)
	origin, index := m.categoryIndex, m.list.Index()

	m.Update(runeKey("/"))
	typeSearch(m, "zzqx")
	if m.menuNotice != "" {
		t.Fatalf("no error should show while typing, got %q", m.menuNotice)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.categoryIndex != origin || m.list.Index() != index {
		t.Fatalf("origin not restored: tab %d index %d", m.categoryIndex, m.list.Index())
	}
	if m.menuNotice != "Pattern not found: zzqx" {
		t.Fatalf("notice %q", m.menuNotice)
	}
	if !strings.Contains(m.View(), "Pattern not found: zzqx") {
		t.Fatal("notice not shown")
	}
}

func TestSearchPromptKeysAreText(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	categories := m.categoryFilter
	for _, k := range []string{"q", "x", "n", "j", "k", "h", "l", "d", "r"} {
		_, cmd := m.Update(runeKey(k))
		if cmd != nil {
			t.Fatalf("key %q in the prompt returned a command", k)
		}
	}
	if m.search.text != "qxnjkhldr" || !m.search.open {
		t.Fatalf("text %q open %v", m.search.text, m.search.open)
	}
	if m.pendingDelete != nil || m.categoryFilter != categories || m.state != StateMenu {
		t.Fatal("typed keys ran menu commands")
	}
}

func TestSearchPromptCtrlCQuits(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !quitCmd(cmd) {
		t.Fatal("ctrl+c should quit from the prompt")
	}
}

func TestMenuQStillQuitsOutsidePrompt(t *testing.T) {
	m := newMenuTestModel(t)
	_, cmd := m.Update(runeKey("q"))
	if !quitCmd(cmd) {
		t.Fatal("q should quit in browse state")
	}
}

func TestSearchEscClearsHighlightThenQuits(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if quitCmd(cmd) || m.search.hl {
		t.Fatalf("first esc should clear the highlight without quitting, hl %v", m.search.hl)
	}
	if m.search.pattern != "palindrome" {
		t.Fatal("clearing the highlight should keep the pattern")
	}
	if !strings.Contains(m.View(), "[n/N] next/prev") || strings.Contains(m.View(), "clear highlight") {
		t.Fatalf("help should keep n/N but not the esc hint:\n%s", m.View())
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !quitCmd(cmd) {
		t.Fatal("second esc should quit")
	}
}

func TestSearchNReenablesHighlight(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m.Update(runeKey("n"))
	if !m.search.hl {
		t.Fatal("n should turn the highlight back on")
	}
}

func TestSearchEmptyEnterRepeatsLastPattern(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	m.Update(runeKey("j"))
	if selectedID(m) == "palindrome" {
		t.Fatal("j should move away from palindrome")
	}
	m.Update(runeKey("/"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if selectedID(m) != "palindrome" {
		t.Fatalf("empty enter should repeat the last pattern, selected %q", selectedID(m))
	}
}

func TestSearchEmptyEnterWithoutPatternJustCloses(t *testing.T) {
	m := newMenuTestModel(t)
	m.list.Select(1)
	index := m.list.Index()
	m.Update(runeKey("/"))
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.search.open || m.list.Index() != index || m.menuNotice != "" {
		t.Fatalf("open %v index %d notice %q", m.search.open, m.list.Index(), m.menuNotice)
	}
}

func TestSearchHighlightSurvivesTabSwitch(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m.Update(runeKey("l"))
	if m.search.pattern != "palindrome" || !m.search.hl {
		t.Fatalf("tab switch dropped the search: pattern %q hl %v", m.search.pattern, m.search.hl)
	}
}

func TestSearchAddDrillNeverMatches(t *testing.T) {
	m := newMenuTestModel(t)
	for tab := range m.categories {
		m.categoryIndex = tab
		m.categoryFilter = m.categories[tab]
		m.refreshList()
		hits := m.searchHits("add")
		if last := hits[tab][len(hits[tab])-1]; last {
			t.Fatalf("tab %s: the add row matched", m.categories[tab])
		}
	}

	m.Update(runeKey("/"))
	typeSearch(m, "add")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	for range 3 {
		m.Update(runeKey("n"))
		if it := m.list.SelectedItem().(drillItem); it.isAdd {
			t.Fatal("search selected the add row")
		}
	}
}

func TestFindMatchWalk(t *testing.T) {
	hits := [][]bool{
		{false, false},      // ALL
		{true, false, true}, // first category
		{false, true},       // second category
	}
	cases := []struct {
		name    string
		origin  searchSpot
		forward bool
		include bool
		want    searchSpot
		wrapped bool
		ok      bool
	}{
		{"rest of tab", searchSpot{1, 0}, true, false, searchSpot{1, 2}, false, true},
		{"next tab", searchSpot{1, 2}, true, false, searchSpot{2, 1}, false, true},
		{"forward past last tab wraps", searchSpot{2, 1}, true, false, searchSpot{1, 0}, true, true},
		{"backward past first tab wraps", searchSpot{1, 0}, false, false, searchSpot{2, 1}, true, true},
		{"backward within ring does not wrap", searchSpot{2, 0}, false, false, searchSpot{1, 2}, false, true},
		{"include origin", searchSpot{1, 2}, true, true, searchSpot{1, 2}, false, true},
		{"backward", searchSpot{1, 2}, false, false, searchSpot{1, 0}, false, true},
		{"backward into tab", searchSpot{2, 0}, false, false, searchSpot{1, 2}, false, true},
	}
	for _, c := range cases {
		got, wrapped, ok := findMatch(hits, c.origin, c.forward, c.include)
		if got != c.want || wrapped != c.wrapped || ok != c.ok {
			t.Errorf("%s: got %v wrapped %v ok %v, want %v %v %v", c.name, got, wrapped, ok, c.want, c.wrapped, c.ok)
		}
	}

	solo := [][]bool{{false}, {false, true}, {false, false}}
	got, wrapped, ok := findMatch(solo, searchSpot{1, 1}, true, false)
	if !ok || !wrapped || got != (searchSpot{1, 1}) {
		t.Fatalf("sole match should wrap onto itself, got %v %v %v", got, wrapped, ok)
	}
	if _, _, ok := findMatch([][]bool{{false}, {false}}, searchSpot{1, 0}, true, true); ok {
		t.Fatal("no match should report not found")
	}
	allTab := [][]bool{{false, true, true}, {true}}
	if got, wrapped, ok := findMatch(allTab, searchSpot{0, 2}, true, false); !ok || !wrapped || got != (searchSpot{0, 1}) {
		t.Fatalf("ALL should search only ALL and wrap, got %v %v %v", got, wrapped, ok)
	}
}

func TestTitleMatchesIgnoreLanguageTag(t *testing.T) {
	it := drillItem{drill: drill.Drill{ID: "palindrome", Title: "Valid Palindrome"}}
	if got := titleMatches(it, "code"); len(got) != 0 {
		t.Fatalf("language tag matched: %v", got)
	}
	// "[CODE] " is 7 runes, "Valid " is 6 more.
	if got := titleMatches(it, "pal"); len(got) != 3 || got[0] != 13 {
		t.Fatalf("pal matched at %v, want 13..15", got)
	}
}

// underlinedText returns the visible runes that an SGR underline covers. It
// reads the escape sequences the way a terminal would: 4 turns underline on,
// 0 and 24 turn it off, and the arguments of 38 and 48 are colours, not
// attributes.
func underlinedText(s string) string {
	var out strings.Builder
	under := false
	for i := 0; i < len(s); {
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			end := strings.IndexByte(s[i:], 'm')
			if end < 0 {
				break
			}
			params := strings.Split(s[i+2:i+end], ";")
			for j := 0; j < len(params); j++ {
				switch params[j] {
				case "", "0", "24":
					under = false
				case "4":
					under = true
				case "38", "48":
					if j+1 < len(params) && params[j+1] == "5" {
						j += 2
					} else if j+1 < len(params) && params[j+1] == "2" {
						j += 4
					}
				}
			}
			i += end + 1
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if under {
			out.WriteRune(r)
		}
		i += size
	}
	return out.String()
}

func TestSearchDelegateRendersTitle(t *testing.T) {
	forceTrueColor(t)

	m := newMenuTestModel(t)
	switchToCategory(t, m, "LeetCode")
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	row := -1
	for i, it := range m.list.Items() {
		if di := it.(drillItem); !di.isAdd && di.drill.ID == "palindrome" {
			row = i
		}
	}
	if row < 0 {
		t.Fatal("palindrome not in list")
	}
	// Render the same row selected and not selected: both must underline
	// the match and nothing else. The description also contains "palindrome",
	// so it would show up here if it were highlighted.
	for _, selected := range []bool{true, false} {
		if selected {
			m.list.Select(row)
		} else {
			m.list.Select((row + 1) % len(m.list.Items()))
		}
		var buf bytes.Buffer
		item := m.list.Items()[row]
		newDrillDelegate("pal").Render(&buf, m.list, row, item)
		out := buf.String()
		if got := ansi.Strip(out); !strings.Contains(got, "[PYTHON] Valid Palindrome") {
			t.Fatalf("selected=%v: rendered title lost its text: %q", selected, got)
		}
		if got := underlinedText(out); got != "Pal" {
			t.Fatalf("selected=%v: underlined %q, want only the match %q\nraw: %q", selected, got, "Pal", out)
		}
		if !strings.Contains(out, "\x1b[4") {
			t.Fatalf("selected=%v: no underline sequence in %q", selected, out)
		}
	}
}

func TestSearchPromptHidesListHelp(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	// The bubbles footer is always off. Our help line covers the keys.
	if view := m.View(); strings.Contains(view, "q quit") || strings.Contains(view, "up/k") {
		t.Fatalf("list footer shows in browse state:\n%s", view)
	}
	browseHeight := lipgloss.Height(m.View())

	m.Update(runeKey("/"))
	typeSearch(m, "pal")
	view := m.View()
	if strings.Contains(view, "q quit") || strings.Contains(view, "up/k") {
		t.Fatalf("list footer shows while the prompt is open:\n%s", view)
	}
	if !strings.Contains(view, "/pal_") {
		t.Fatalf("prompt missing:\n%s", view)
	}
	if w, h := lipgloss.Width(view), lipgloss.Height(view); w > 80 || h > 24 {
		t.Fatalf("prompt view is %dx%d at 80x24", w, h)
	}
	if h := lipgloss.Height(view); h != browseHeight {
		t.Fatalf("view height changed from %d to %d when the prompt opened", browseHeight, h)
	}

	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if view := m.View(); strings.Contains(view, "q quit") || strings.Contains(view, "up/k") {
		t.Fatalf("list footer shows after the prompt closed:\n%s", view)
	}
	if h := lipgloss.Height(m.View()); h != browseHeight {
		t.Fatalf("view height changed from %d to %d after the prompt closed", browseHeight, h)
	}
}

func TestSearchNWithoutPatternDoesNotArmHighlight(t *testing.T) {
	m := newMenuTestModel(t)
	m.Update(runeKey("n"))
	if m.search.hl {
		t.Fatal("n with no pattern must not turn the highlight on")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if !quitCmd(cmd) {
		t.Fatal("esc after n with no previous search should quit")
	}
}

func TestSearchWrapsAcrossTabRing(t *testing.T) {
	m := newMenuTestModel(t)
	// The category ring is Algorithms, LeetCode, 42; ALL is not in it.
	// reverse_bits only matches on Algorithms, palindrome only on LeetCode.
	first, middle, last := "Algorithms", "LeetCode", m.categories[len(m.categories)-1]
	if last != "42" {
		t.Fatalf("42 should be the last tab, got %v", m.categories)
	}

	// Forward: from the last category, n goes past the end of the ring and
	// lands on the first one.
	switchToCategory(t, m, last)
	m.Update(runeKey("/"))
	typeSearch(m, "reverse_bits")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.categoryFilter != first {
		t.Fatalf("Enter should land on %q, got %q", first, m.categoryFilter)
	}
	switchToCategory(t, m, last)
	m.Update(runeKey("n"))
	if m.categoryFilter != first || m.menuNotice != "search hit BOTTOM, continuing at TOP" {
		t.Fatalf("n: tab %q notice %q, want %q and the BOTTOM notice", m.categoryFilter, m.menuNotice, first)
	}

	// Backward: from the first category, N goes back past the start of the
	// ring and lands on the last category that has the match.
	m.Update(runeKey("/"))
	typeSearch(m, "palindrome")
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.categoryFilter != middle {
		t.Fatalf("Enter should land on %q, got %q", middle, m.categoryFilter)
	}
	switchToCategory(t, m, first)
	m.Update(runeKey("N"))
	if m.categoryFilter != middle || selectedID(m) != "palindrome" || m.menuNotice != "search hit TOP, continuing at BOTTOM" {
		t.Fatalf("N: tab %q id %q notice %q", m.categoryFilter, selectedID(m), m.menuNotice)
	}
}

func TestSearchHitsMatchPerTab(t *testing.T) {
	m := newMenuTestModel(t)
	switchToCategory(t, m, "Algorithms")
	hits := m.searchHits("binary")
	algo := hits[categoryNamed(t, m, "Algorithms")]
	if !slices.Contains(algo, true) {
		t.Fatalf("Algorithms tab should match binary_search: %v", algo)
	}
	if slices.Contains(hits[categoryNamed(t, m, "42")], true) {
		t.Fatal("42 tab should not match binary")
	}
}
