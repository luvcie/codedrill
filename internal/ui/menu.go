package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
)

type drillItem struct {
	drill drill.Drill
	pb    time.Duration
	isAdd bool
}

func (i drillItem) Title() string {
	if i.isAdd {
		return "+ Add a drill"
	}
	lang := strings.ToUpper(i.drill.Language)
	if lang == "" {
		lang = "CODE"
	}
	title := fmt.Sprintf("[%s] %s", lang, i.drill.Title)
	if grade, _ := CalculateGrade(i.pb, i.drill.Difficulty); grade != "" {
		title += fmt.Sprintf("  [%s-RANK]", grade)
	}
	return title
}

func (i drillItem) Description() string {
	if i.isAdd {
		return "Create a custom drill"
	}
	difficulty := strings.ToUpper(i.drill.Difficulty)
	if difficulty == "" {
		difficulty = "EASY"
	}
	pb := "PB: --"
	if grade, _ := CalculateGrade(i.pb, i.drill.Difficulty); grade != "" {
		pb = fmt.Sprintf("PB: %s (%s)", storage.FormatDuration(i.pb), grade)
	}
	desc := fmt.Sprintf("[%s • %s] • %s • %s", drill.CategoryOf(i.drill), difficulty, pb, CleanSummary(i.drill.Description))
	if i.drill.AutoSubmit {
		desc += " [Auto-Submit]"
	}
	return desc
}

func (i drillItem) FilterValue() string {
	if i.isAdd {
		return "add new drill exercise create"
	}
	d := i.drill
	return strings.Join([]string{d.ID, d.Title, d.Language, d.Category, d.Difficulty}, " ")
}

var (
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(ColorBg).Background(ColorSecondary).Padding(0, 1)
	inactiveTabStyle = SubtextStyle.Padding(0, 1)
)

// renderCategoryHeader draws the categories as tabs. Only the active one is
// filled, so the highlight moves when h/l or Tab switch category.
func (m *MainModel) renderCategoryHeader() string {
	tabs := m.categories
	// A filter from "codedrill random <filter>" may not be a category name.
	if !slices.ContainsFunc(tabs, func(c string) bool { return strings.EqualFold(c, m.categoryFilter) }) && m.categoryFilter != "" {
		tabs = append(tabs[:len(tabs):len(tabs)], m.categoryFilter)
	}
	parts := make([]string, len(tabs))
	for i, c := range tabs {
		style := inactiveTabStyle
		if strings.EqualFold(c, m.categoryFilter) {
			style = activeTabStyle
		}
		parts[i] = style.Render(strings.ToUpper(c))
	}
	return strings.Join(parts, " ")
}

func (m *MainModel) refreshList() {
	var items []list.Item
	for _, d := range drill.Filter(m.drills, m.categoryFilter) {
		items = append(items, drillItem{drill: d, pb: m.history.GetPB(d.ID)})
	}
	items = append(items, drillItem{isAdd: true})

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.Foreground(ColorSecondary).BorderLeftForeground(ColorSecondary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.Foreground(lipgloss.Color("#EEEEEE")).BorderLeftForeground(ColorSecondary)

	l := list.New(items, delegate, 0, 0)
	// h/l switch categories and d starts the daily, so keep them out of paging.
	l.KeyMap.PrevPage.SetKeys("left", "pgup", "b", "u")
	l.KeyMap.NextPage.SetKeys("right", "pgdown", "f")
	l.Title = m.renderCategoryHeader()
	l.Styles.Title = lipgloss.NewStyle()
	m.list = l
	m.sizeList()
}

// refreshPB updates one drill's PB in the list without rebuilding it.
func (m *MainModel) refreshPB(d drill.Drill) tea.Cmd {
	for i, item := range m.list.Items() {
		if it, ok := item.(drillItem); ok && !it.isAdd && it.drill.ID == d.ID {
			it.pb = m.history.GetPB(it.drill.ID)
			return m.list.SetItem(i, it)
		}
	}
	return nil
}

// reloadDrills rereads drills from disk and keeps the current category if it
// still exists. The list cursor stays at the same position where possible.
func (m *MainModel) reloadDrills() {
	cursor := m.list.Index()
	drills, err := drill.LoadDrills()
	if drills != nil {
		m.drills = drills
	}
	if err != nil && m.menuNotice == "" {
		m.menuNotice = "Some drills could not be loaded: " + err.Error()
	}
	m.categories = getAvailableCategories(m.drills)
	m.categoryIndex = 0
	found := false
	for i, c := range m.categories {
		if strings.EqualFold(c, m.categoryFilter) {
			m.categoryIndex, found = i, true
		}
	}
	if !found {
		m.categoryFilter = "ALL"
	}
	m.refreshList()
	m.list.Select(min(cursor, len(m.list.Items())-1))
}

func (m *MainModel) switchCategory(step int) {
	n := len(m.categories)
	m.categoryIndex = ((m.categoryIndex+step)%n + n) % n
	m.categoryFilter = m.categories[m.categoryIndex]
	m.refreshList()
}

func (m *MainModel) confirmDelete() {
	d := m.pendingDelete
	m.pendingDelete = nil
	removed, hidden, err := drill.RemoveDrill(d.ID)
	switch {
	case err != nil:
		m.menuNotice = "Could not remove drill: " + err.Error()
	case hidden:
		m.menuNotice = fmt.Sprintf("Hid %q. Bring it back: codedrill restore %s", removed.Title, removed.ID)
	default:
		m.menuNotice = fmt.Sprintf("Deleted %q.", removed.Title)
		if err := m.history.ResetDrillStats(removed.ID); err != nil {
			m.menuNotice = fmt.Sprintf("Deleted %q, but could not clear its runs: %v", removed.Title, err)
		}
	}
	m.reloadDrills()
}

func (m *MainModel) selectedDrill() (drill.Drill, bool) {
	it, ok := m.list.SelectedItem().(drillItem)
	return it.drill, ok && !it.isAdd
}

func (m *MainModel) updateMenu(msg tea.Msg) tea.Cmd {
	if key, ok := msg.(tea.KeyMsg); ok {
		hadNotice := m.menuNotice != ""
		m.menuNotice = ""
		if m.pendingDelete != nil {
			if key.String() == "y" || key.String() == "Y" {
				m.confirmDelete()
			} else {
				m.pendingDelete = nil
			}
			m.sizeList()
			return nil
		}
		if hadNotice {
			m.sizeList()
		}
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok || m.list.FilterState() == list.Filtering {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return cmd
	}

	switch key.String() {
	case "x":
		if d, ok := m.selectedDrill(); ok {
			m.pendingDelete = &d
			m.sizeList()
		}
		return nil
	case "enter", " ":
		if it, ok := m.list.SelectedItem().(drillItem); ok && it.isAdd {
			m.state = StateAddDrill
			m.initForm()
			return m.form.Init()
		}
		if d, ok := m.selectedDrill(); ok {
			m.openBriefing(&d)
		}
		return nil
	case "d":
		if daily := drill.GetDailyDrill(m.drills, time.Now()); daily != nil {
			m.openBriefing(daily)
		}
		return nil
	case "r":
		if d := m.randomDrill(""); d != nil {
			m.openBriefing(d)
		}
		return nil
	case "tab", "l":
		m.switchCategory(1)
		return nil
	case "shift+tab", "h":
		m.switchCategory(-1)
		return nil
	case "s":
		m.state = StateStats
		return nil
	case "v":
		if d, ok := m.selectedDrill(); ok {
			m.activeDrill = &d
			m.openLeaderboard(StateMenu)
		}
		return nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

const newCategoryOption = "\x00new"

// categoryOptions lists the built-in categories first, then any the user made.
func (m *MainModel) categoryOptions() []huh.Option[string] {
	names := []string{"Custom", "42", "Algorithms", "LeetCode"}
	for _, c := range m.categories {
		if c == "ALL" {
			continue
		}
		known := false
		for _, n := range names {
			known = known || strings.EqualFold(n, c)
		}
		if !known {
			names = append(names, c)
		}
	}
	opts := make([]huh.Option[string], 0, len(names)+1)
	for _, n := range names {
		opts = append(opts, huh.NewOption(n, n))
	}
	return append(opts, huh.NewOption("+ New category...", newCategoryOption))
}

func validateCategoryName(s string) error {
	s = strings.TrimSpace(s)
	if s == "" {
		return fmt.Errorf("category name cannot be empty")
	}
	if strings.EqualFold(s, "ALL") {
		return fmt.Errorf(`"ALL" is reserved`)
	}
	return nil
}

func (m *MainModel) initForm() {
	m.formCategory = "Custom"
	s, a, b := drill.PaceTargets("easy")
	ms, ma, mb := drill.PaceTargets("medium")
	hs, ha, hb := drill.PaceTargets("hard")
	paceLabel := func(name string, s, a, b time.Duration) string {
		return fmt.Sprintf("%s (S ≤ %v, A ≤ %v, B ≤ %v)", name, s, a, b)
	}
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("title").
				Title("Exercise Title").
				Placeholder("e.g. ft_putchar, ft_split, Two Sum"),
			huh.NewSelect[string]().
				Key("category").
				Title("Category").
				Options(m.categoryOptions()...).
				Value(&m.formCategory),
		),
		huh.NewGroup(
			huh.NewInput().
				Key("new_category").
				Title("New Category Name").
				Placeholder("e.g. Interview Prep, Rust, Piscine").
				Validate(validateCategoryName),
		).WithHideFunc(func() bool { return m.formCategory != newCategoryOption }),
		huh.NewGroup(
			huh.NewSelect[string]().
				Key("difficulty").
				Title("Difficulty & Target Pace").
				Options(
					huh.NewOption(paceLabel("Easy", s, a, b), "easy"),
					huh.NewOption(paceLabel("Medium", ms, ma, mb), "medium"),
					huh.NewOption(paceLabel("Hard", hs, ha, hb), "hard"),
				),
			huh.NewInput().
				Key("language").
				Title("Language").
				Placeholder("e.g. c, go, rust, python, cpp..."),
			huh.NewInput().
				Key("entry_file").
				Title("Entry Filename").
				Placeholder("e.g. ft_putchar.c or solution.go"),
			huh.NewInput().
				Key("test_cmd").
				Title("Test Command (optional)").
				Placeholder("e.g. gcc -Wall -Wextra -Werror -c ft_putchar.c"),
			huh.NewText().
				Key("description").
				Title("Problem Description").
				Placeholder("Prototype and instructions. Markdown works."),
			huh.NewConfirm().
				Key("auto_submit").
				Title("Submit automatically when closing editor? (No = keep shell open)").
				Affirmative("Yes (auto-submit)").
				Negative("No (default)"),
			huh.NewConfirm().
				Key("confirm").
				Title("Save and create this drill?").
				Affirmative("Yes").
				Negative("Cancel"),
		),
	).WithShowHelp(true)
}

func (m *MainModel) updateAddDrill(msg tea.Msg) tea.Cmd {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	switch m.form.State {
	case huh.StateAborted:
		m.state = StateMenu
		return nil
	case huh.StateCompleted:
		if m.form.GetBool("confirm") {
			m.saveFormDrill()
		}
		m.state = StateMenu
		return nil
	}
	return cmd
}

func (m *MainModel) saveFormDrill() {
	title := strings.TrimSpace(m.form.GetString("title"))
	if title == "" {
		title = "Untitled Drill"
	}
	entry := strings.TrimSpace(m.form.GetString("entry_file"))
	if entry == "" {
		entry = "solution.txt"
	}
	category := m.formCategory
	if category == newCategoryOption {
		category = strings.TrimSpace(m.form.GetString("new_category"))
		// Reuse the existing spelling if the name only differs in case.
		for _, c := range m.categories {
			if strings.EqualFold(c, category) {
				category = c
			}
		}
	}

	d := drill.Drill{
		ID:          drill.UniqueID(m.drills, title),
		Title:       title,
		Category:    category,
		Difficulty:  m.form.GetString("difficulty"),
		Language:    strings.ToLower(strings.TrimSpace(m.form.GetString("language"))),
		EntryFile:   entry,
		TestCommand: strings.TrimSpace(m.form.GetString("test_cmd")),
		Description: strings.TrimSpace(m.form.GetString("description")),
		AutoSubmit:  m.form.GetBool("auto_submit"),
	}
	if err := drill.SaveDrill(d); err != nil {
		m.menuNotice = "Could not save drill: " + err.Error()
	} else {
		m.menuNotice = fmt.Sprintf("Created %q in %s.", title, category)
	}
	m.reloadDrills()
}
