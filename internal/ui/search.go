package ui

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/luvcie/codedrill/internal/drill"
)

// menuSearch is the vim-style search in the menu. Search never narrows the
// list. It moves the selection and underlines matches in the titles.
type menuSearch struct {
	// open means the / prompt is showing and keys are typed into text.
	open bool
	text string
	// origin is where the selection was when the prompt opened. Esc and an
	// unmatched pattern go back there.
	origin searchSpot
	// pattern is the last committed search, used by n and N.
	pattern string
	// hl means matches of the search are underlined. Esc turns it off
	// without forgetting the pattern.
	hl bool
}

// searchSpot is a position in the menu: a category tab and a row in its list.
type searchSpot struct {
	tab   int
	index int
}

// highlight is the text whose matches are underlined, or "" for none. While
// the prompt is open, the text being typed is highlighted live.
func (s menuSearch) highlight() string {
	switch {
	case s.open:
		return s.text
	case s.hl:
		return s.pattern
	}
	return ""
}

// caseSensitive reports whether pattern is matched exactly. Smartcase: a
// pattern with an uppercase letter is case-sensitive, anything else is not.
func caseSensitive(pattern string) bool {
	for _, r := range pattern {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

// matchRunes returns the rune positions in text covered by any occurrence of
// pattern, in increasing order. It returns nil when pattern is empty or absent.
func matchRunes(text, pattern string) []int {
	if pattern == "" {
		return nil
	}
	t, p := []rune(text), []rune(pattern)
	fold := !caseSensitive(pattern)
	marked := make([]bool, len(t))
	for i := 0; i+len(p) <= len(t); i++ {
		found := true
		for j := range p {
			a, b := t[i+j], p[j]
			if fold {
				a, b = unicode.ToLower(a), unicode.ToLower(b)
			}
			if a != b {
				found = false
				break
			}
		}
		if found {
			for j := range p {
				marked[i+j] = true
			}
		}
	}
	var out []int
	for i, ok := range marked {
		if ok {
			out = append(out, i)
		}
	}
	return out
}

func drillMatches(d drill.Drill, pattern string) bool {
	return len(matchRunes(d.ID, pattern)) > 0 || len(matchRunes(d.Title, pattern)) > 0
}

// findMatch walks the menu from origin and returns the first matching spot.
// hits[t][i] reports whether row i of tab t matches. The walk goes through the
// rest of the origin tab, then the other category tabs in order (skipping
// ALL, which is index 0), and ends with the start of the origin tab. An origin
// on ALL searches ALL only.
//
// include makes the origin row itself the first candidate, which is what the
// live jump of the prompt needs. Without it the walk starts after the origin,
// which is what n and N need.
//
// wrapped is true when the walk has gone past an end of the menu. The tab
// ring is the buffer: forward, a match in a tab before the origin tab means the
// walk went past the last category and started again at the first. Backward,
// a match in a tab after the origin tab means the reverse. Coming back to the
// origin tab from its far side also counts.
func findMatch(hits [][]bool, origin searchSpot, forward, include bool) (spot searchSpot, wrapped, ok bool) {
	n := len(hits)
	if origin.tab < 0 || origin.tab >= n || len(hits[origin.tab]) == 0 {
		return searchSpot{}, false, false
	}
	own := hits[origin.tab]
	type candidate struct {
		spot    searchSpot
		wrapped bool
	}
	var order []candidate
	add := func(tab, i int, wrap bool) {
		order = append(order, candidate{searchSpot{tab, i}, wrap})
	}

	// Other category tabs in walk order. ALL is never one of them, and an
	// origin on ALL has none at all.
	var others []int
	if origin.tab != 0 {
		step := 1
		if !forward {
			step = -1
		}
		for k := 1; k < n; k++ {
			if t := ((origin.tab+step*k)%n + n) % n; t != 0 {
				others = append(others, t)
			}
		}
	}
	// crossed reports whether reaching tab t from the origin went past the
	// end of the tab ring.
	crossed := func(t int) bool {
		if forward {
			return t < origin.tab
		}
		return t > origin.tab
	}

	if forward {
		for i := origin.index; i < len(own); i++ {
			if i != origin.index || include {
				add(origin.tab, i, false)
			}
		}
		for _, t := range others {
			for i := range hits[t] {
				add(t, i, crossed(t))
			}
		}
		last := origin.index
		if include {
			last--
		}
		for i := 0; i <= last && i < len(own); i++ {
			add(origin.tab, i, true)
		}
	} else {
		for i := origin.index; i >= 0 && i < len(own); i-- {
			if i != origin.index || include {
				add(origin.tab, i, false)
			}
		}
		for _, t := range others {
			for i := len(hits[t]) - 1; i >= 0; i-- {
				add(t, i, crossed(t))
			}
		}
		first := origin.index
		if include {
			first++
		}
		for i := len(own) - 1; i >= 0 && i >= first; i-- {
			add(origin.tab, i, true)
		}
	}

	for _, c := range order {
		if hits[c.spot.tab][c.spot.index] {
			return c.spot, c.wrapped, true
		}
	}
	return searchSpot{}, false, false
}

// drillDelegate renders drill rows like list.DefaultDelegate and underlines
// the matches of highlight in the drill's title.
//
// The matched runs are styled with the row's own title style, as bubbles does
// for its filter matches. The style is applied per run, not around the whole
// line, so the underline never sits inside a reset that cuts off the selected
// colour.
type drillDelegate struct {
	list.DefaultDelegate
	highlight string
	// drop is the row that shows the drag drop marker, or -1 for none.
	drop int
}

func newDrillDelegate(highlight string) drillDelegate {
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(ColorSecondary).BorderLeftForeground(ColorSecondary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(ColorDescSelected).BorderLeftForeground(ColorSecondary)
	delegate.Styles.NormalTitle = delegate.Styles.NormalTitle.Foreground(ColorListTitle)
	delegate.Styles.NormalDesc = delegate.Styles.NormalDesc.Foreground(ColorListDesc)
	delegate.Styles.DimmedTitle = delegate.Styles.DimmedTitle.Foreground(ColorListDesc)
	delegate.Styles.DimmedDesc = delegate.Styles.DimmedDesc.Foreground(ColorBorder)
	return drillDelegate{DefaultDelegate: delegate, highlight: highlight, drop: -1}
}

// titleMatches returns the rune positions of highlight in the rendered title
// of a row. Only the drill's title counts. The [LANG] tag and the rank badge
// are not searched and not underlined.
func titleMatches(it drillItem, highlight string) []int {
	if it.isAdd {
		return nil
	}
	offset := utf8.RuneCountInString(langTag(it.drill))
	idx := matchRunes(it.drill.Title, highlight)
	for i := range idx {
		idx[i] += offset
	}
	return idx
}

func (d drillDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	it, ok := item.(drillItem)
	if !ok || m.Width() <= 0 {
		return
	}
	s := &d.Styles
	titleStyle, descStyle := s.NormalTitle, s.NormalDesc
	if index == m.Index() {
		titleStyle, descStyle = s.SelectedTitle, s.SelectedDesc
	}

	textWidth := m.Width() - s.NormalTitle.GetPaddingLeft() - s.NormalTitle.GetPaddingRight()
	title := ansi.Truncate(it.Title(), textWidth, "…")
	desc := it.Description()
	if index == d.drop {
		desc = "drop here"
		descStyle = s.SelectedDesc.Foreground(ColorSecondary)
	}
	if d.ShowDescription {
		var lines []string
		for i, line := range strings.Split(desc, "\n") {
			if i >= d.Height()-1 {
				break
			}
			lines = append(lines, ansi.Truncate(line, textWidth, "…"))
		}
		desc = strings.Join(lines, "\n")
	}

	if idx := titleMatches(it, d.highlight); len(idx) > 0 {
		unmatched := titleStyle.Inline(true)
		matched := unmatched.Underline(true)
		title = lipgloss.StyleRunes(title, idx, matched, unmatched)
	}
	title = titleStyle.Render(title)
	desc = descStyle.Render(desc)

	if d.ShowDescription {
		fmt.Fprintf(w, "%s\n%s", title, desc) //nolint: errcheck
		return
	}
	fmt.Fprintf(w, "%s", title) //nolint: errcheck
}

func (m *MainModel) currentSpot() searchSpot {
	return searchSpot{tab: m.categoryIndex, index: m.list.Index()}
}

// searchHits marks the matching rows of every category tab. The current tab
// is read from the list, so the marks match what is on screen. Other tabs use
// the same row builder as the list, which keeps group headers and collapsed
// subcategories in step. Headers and the "+ Add a drill" row never match.
func (m *MainModel) searchHits(pattern string) [][]bool {
	hits := make([][]bool, len(m.categories))
	for t, c := range m.categories {
		items := m.menuItemsFor(c)
		if t == m.categoryIndex {
			items = m.list.Items()
		}
		hits[t] = make([]bool, len(items))
		for i, it := range items {
			if di, ok := it.(drillItem); ok && !di.isAdd && !di.isGroup() {
				hits[t][i] = drillMatches(di.drill, pattern)
			}
		}
	}
	return hits
}

func (m *MainModel) goTo(spot searchSpot) {
	if spot.tab != m.categoryIndex || m.categoryFilter != m.categories[spot.tab] {
		m.categoryIndex = spot.tab
		m.categoryFilter = m.categories[spot.tab]
		m.refreshList()
	}
	m.list.Select(spot.index)
}

func (m *MainModel) searchFrom(pattern string, from searchSpot, forward, include bool) (wrapped, ok bool) {
	spot, wrapped, ok := findMatch(m.searchHits(pattern), from, forward, include)
	if ok {
		m.goTo(spot)
	}
	return wrapped, ok
}

// syncSearch brings the list in line with the search state after it changed:
// the underlines. Callers resize the list afterwards.
func (m *MainModel) syncSearch() {
	m.list.SetDelegate(m.menuDelegate())
	m.list.SetShowHelp(false)
}

func (m *MainModel) openSearch() {
	m.search.open = true
	m.search.text = ""
	m.search.origin = m.currentSpot()
	m.syncSearch()
	m.sizeList()
}

// closeSearch leaves the prompt and puts the selection back where it was
// when the prompt opened.
func (m *MainModel) closeSearch() {
	m.search.open = false
	m.goTo(m.search.origin)
	m.syncSearch()
	m.sizeList()
}

// liveSearch moves the selection to the first match from the origin while
// the text is typed. Nothing matching puts the selection back at the origin.
func (m *MainModel) liveSearch() {
	m.goTo(m.search.origin)
	if m.search.text != "" {
		m.searchFrom(m.search.text, m.search.origin, true, true)
	}
	m.syncSearch()
}

// commitSearch leaves the prompt with its text. A match becomes the last
// pattern. Empty text repeats the last pattern.
func (m *MainModel) commitSearch() {
	m.search.open = false
	m.goTo(m.search.origin)
	text := m.search.text
	if text == "" {
		text = m.search.pattern
	}
	if text == "" {
		m.syncSearch()
		m.sizeList()
		return
	}
	if _, ok := m.searchFrom(text, m.search.origin, true, true); !ok {
		m.menuNotice = "Pattern not found: " + text
	} else {
		m.search.pattern = text
		m.search.hl = true
	}
	m.syncSearch()
	m.sizeList()
}

func (m *MainModel) stepSearch(forward bool) {
	if m.search.pattern == "" {
		m.menuNotice = "No previous search"
		m.syncSearch()
		m.sizeList()
		return
	}
	// Only a real pattern turns the highlight on. Otherwise an invisible
	// highlight would swallow the next Esc.
	m.search.hl = true
	wrapped, ok := m.searchFrom(m.search.pattern, m.currentSpot(), forward, false)
	switch {
	case !ok:
		m.menuNotice = "Pattern not found: " + m.search.pattern
	case wrapped && forward:
		m.menuNotice = "search hit BOTTOM, continuing at TOP"
	case wrapped:
		m.menuNotice = "search hit TOP, continuing at BOTTOM"
	}
	m.syncSearch()
	m.sizeList()
}

// updateSearchPrompt handles keys while the / prompt is open. Every printable
// key is text, so q, x, n and the rest never run a command here.
func (m *MainModel) updateSearchPrompt(key tea.KeyMsg) tea.Cmd {
	switch key.Type {
	case tea.KeyCtrlC:
		return tea.Quit
	case tea.KeyEsc:
		m.closeSearch()
	case tea.KeyEnter:
		m.commitSearch()
	case tea.KeyBackspace:
		if m.search.text == "" {
			m.closeSearch()
			return nil
		}
		r := []rune(m.search.text)
		m.search.text = string(r[:len(r)-1])
		m.liveSearch()
		m.sizeList()
	case tea.KeySpace:
		m.search.text += " "
		m.liveSearch()
		m.sizeList()
	case tea.KeyRunes:
		m.search.text += string(key.Runes)
		m.liveSearch()
		m.sizeList()
	}
	return nil
}
