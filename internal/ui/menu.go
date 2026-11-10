package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
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

	// A group row is the header of a collapsible subcategory. It has no
	// drill of its own.
	group     string
	groupKey  string
	count     int
	collapsed bool
}

func (i drillItem) isGroup() bool { return i.group != "" }

func langTag(d drill.Drill) string {
	lang := strings.ToUpper(d.Language)
	if lang == "" {
		lang = "CODE"
	}
	return fmt.Sprintf("[%s] ", lang)
}

func (i drillItem) Title() string {
	if i.isAdd {
		return "+ Add a drill"
	}
	if i.isGroup() {
		arrow := "▾"
		if i.collapsed {
			arrow = "▸"
		}
		return arrow + " " + strings.ToUpper(i.group)
	}
	title := langTag(i.drill) + i.drill.Title
	if grade, _ := CalculateGrade(i.pb, i.drill.Difficulty); grade != "" {
		title += fmt.Sprintf("  [%s-RANK]", grade)
	}
	return title
}

func (i drillItem) Description() string {
	if i.isAdd {
		return "Create a custom drill"
	}
	if i.isGroup() {
		if i.count == 1 {
			return "1 drill"
		}
		return fmt.Sprintf("%d drills", i.count)
	}
	difficulty := strings.ToUpper(i.drill.Difficulty)
	if difficulty == "" {
		difficulty = "EASY"
	}
	pb := "PB --"
	if grade, _ := CalculateGrade(i.pb, i.drill.Difficulty); grade != "" {
		pb = fmt.Sprintf("PB %s (%s)", storage.FormatDuration(i.pb), grade)
	}
	desc := fmt.Sprintf("[%s • %s] • %s • %s", drill.CategoryOf(i.drill), difficulty, pb, CleanSummary(i.drill.Description))
	return desc
}

func (i drillItem) FilterValue() string {
	if i.isAdd {
		return "add new drill exercise create"
	}
	if i.isGroup() {
		return i.group
	}
	d := i.drill
	return strings.Join([]string{d.ID, d.Title, d.Language, d.Category, d.Difficulty}, " ")
}

func (m *MainModel) categoryTabs() []string {
	tabs := m.categories
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
	return parts
}

func (m *MainModel) renderCategoryHeader() string {
	return strings.Join(m.categoryTabs(), " ")
}

func (m *MainModel) refreshList() {
	l := list.New(m.menuItemsFor(m.categoryFilter), m.menuDelegate(), 0, 0)
	l.SetFilteringEnabled(false)
	l.SetShowHelp(false)
	l.KeyMap.PrevPage = key.NewBinding(
		key.WithKeys("pgup", "b", "u"),
		key.WithHelp("pgup/b/u", "prev page"),
	)
	l.KeyMap.NextPage = key.NewBinding(
		key.WithKeys("pgdown", "f"),
		key.WithHelp("pgdn/f", "next page"),
	)
	l.Title = m.renderCategoryHeader()
	l.Styles.Title = lipgloss.NewStyle()
	themeList(&l)
	m.list = l
	m.sizeList()
}

// groupKey identifies a subcategory independently of the case used in the
// category names.
func groupKey(parent, sub string) string {
	return strings.ToUpper(strings.TrimSpace(parent)) + "\x00" + strings.ToUpper(strings.TrimSpace(sub))
}

func (m *MainModel) groupCollapsed(key string) bool {
	return m.collapsedGroups[key]
}

// menuItemsFor builds the rows of one category. A parent category with
// subcategories puts a header row before each group. Blocks follow the saved
// arrangement; anything new is appended in the old default order. The add row
// is always last. Other views of the menu, like search, use this so their row
// numbers match the list.
func (m *MainModel) menuItemsFor(category string) []list.Item {
	var items []list.Item
	if !m.reorderableCategory(category) {
		for _, d := range drill.Filter(m.drills, category) {
			items = append(items, drillItem{drill: d, pb: m.history.GetPB(d.ID)})
		}
		return append(items, drillItem{isAdd: true})
	}

	for _, block := range m.menuBlocks(category) {
		if !block.isGroup() {
			items = append(items, drillItem{drill: block.drill, pb: m.history.GetPB(block.drill.ID)})
			continue
		}
		key := groupKey(category, block.group)
		collapsed := m.groupCollapsed(key)
		items = append(items, drillItem{
			group:     block.group,
			groupKey:  key,
			count:     len(block.drills),
			collapsed: collapsed,
		})
		if collapsed {
			continue
		}
		for _, d := range block.drills {
			items = append(items, drillItem{drill: d, pb: m.history.GetPB(d.ID)})
		}
	}
	return append(items, drillItem{isAdd: true})
}

// toggleGroup folds or unfolds a subcategory and keeps the cursor on its
// header.
func (m *MainModel) toggleGroup(it drillItem) {
	if m.collapsedGroups == nil {
		m.collapsedGroups = make(map[string]bool)
	}
	m.collapsedGroups[it.groupKey] = !m.collapsedGroups[it.groupKey]
	m.refreshList()
	for i, item := range m.list.Items() {
		if g, ok := item.(drillItem); ok && g.isGroup() && g.groupKey == it.groupKey {
			m.list.Select(i)
			break
		}
	}
	m.sizeList()
}

func (m *MainModel) refreshPB(d drill.Drill) tea.Cmd {
	for i, item := range m.list.Items() {
		if it, ok := item.(drillItem); ok && !it.isAdd && it.drill.ID == d.ID {
			it.pb = m.history.GetPB(it.drill.ID)
			return m.list.SetItem(i, it)
		}
	}
	return nil
}

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
	m.setCategory(((m.categoryIndex+step)%n + n) % n)
}

func (m *MainModel) setCategory(i int) {
	m.cancelPress()
	m.categoryIndex = i
	m.categoryFilter = m.categories[i]
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
	return it.drill, ok && !it.isAdd && !it.isGroup()
}

func (m *MainModel) activateSelected() tea.Cmd {
	if it, ok := m.list.SelectedItem().(drillItem); ok {
		switch {
		case it.isGroup():
			m.toggleGroup(it)
			return nil
		case it.isAdd:
			m.state = StateAddDrill
			m.initForm()
			return m.form.Init()
		}
	}
	if d, ok := m.selectedDrill(); ok {
		m.openBriefing(&d)
	}
	return nil
}

func (m *MainModel) updateMenu(msg tea.Msg) tea.Cmd {
	if k, ok := msg.(tea.KeyMsg); ok {
		m.cancelPress()
		hadNotice := m.menuNotice != ""
		m.menuNotice = ""
		if m.pendingDelete != nil {
			if k.String() == "y" || k.String() == "Y" {
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

	k, ok := msg.(tea.KeyMsg)
	if !ok {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return cmd
	}
	if m.search.open {
		return m.updateSearchPrompt(k)
	}

	switch k.String() {
	case "/":
		m.openSearch()
		return nil
	case "n", "N":
		m.stepSearch(k.String() == "n")
		return nil
	case "esc":
		if m.search.hl {
			m.search.hl = false
			m.syncSearch()
			m.sizeList()
			return nil
		}
	case "x":
		if d, ok := m.selectedDrill(); ok {
			m.pendingDelete = &d
			m.sizeList()
		}
		return nil
	case "enter", " ":
		return m.activateSelected()
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
	case "shift+up":
		return m.keyboardMenuMove(-1)
	case "shift+down":
		return m.keyboardMenuMove(1)
	case "tab", "l", "right":
		m.switchCategory(1)
		return nil
	case "shift+tab", "h", "left":
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
	case "o":
		m.openSettings()
		return nil
	case "e":
		if d, ok := m.selectedDrill(); ok {
			if d.IsPreset {
				m.menuNotice = "Built-in drills cannot be edited."
			} else {
				m.editDrill = &d
				m.state = StateEditDrill
				m.initEditForm()
				return m.form.Init()
			}
		}
		return nil
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return cmd
}

const newCategoryOption = "\x00new"

func (m *MainModel) categoryOptions() []huh.Option[string] {
	names := []string{"Custom", "42", "Algorithms", "LeetCode"}
	add := func(c string) {
		c = drill.NormalizeCategory(c)
		if c == "" {
			return
		}
		for _, n := range names {
			if strings.EqualFold(n, c) {
				return
			}
		}
		names = append(names, c)
	}
	for _, c := range m.categories {
		if !strings.EqualFold(c, "ALL") {
			add(c)
		}
	}
	// Subcategories are choices too: "42/Exam 05" must be selectable when
	// editing a drill that already lives there.
	for _, d := range m.drills {
		add(drill.CategoryOf(d))
	}
	drill.SortCategories(names)
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
	if drill.NormalizeCategory(s) == "" {
		return fmt.Errorf("category name cannot be empty")
	}
	return nil
}

// canonicalCategory normalizes a category path and keeps the spelling a drill
// already uses. A new "42 exam 05" becomes the existing "42/Exam 05".
func (m *MainModel) canonicalCategory(category string) string {
	category = drill.NormalizeCategory(category)
	for _, d := range m.drills {
		if c := drill.CategoryOf(d); strings.EqualFold(c, category) {
			return c
		}
	}
	for _, c := range m.categories {
		if strings.EqualFold(c, category) {
			return c
		}
	}
	return category
}

func formTheme() *huh.Theme {
	t := huh.ThemeBase()
	cyan := ColorSecondary
	dim := ColorMuted

	t.Form.Base = lipgloss.NewStyle().MarginLeft(2)
	t.Group.Title = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	t.Group.Description = lipgloss.NewStyle().Foreground(dim)

	t.Focused.Base = t.Focused.Base.BorderForeground(cyan)
	t.Focused.Card = t.Focused.Base
	t.Focused.Title = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	t.Focused.Description = lipgloss.NewStyle().Foreground(dim)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(ColorDanger)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(ColorDanger)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(cyan)
	t.Focused.NextIndicator = t.Focused.NextIndicator.Foreground(cyan)
	t.Focused.PrevIndicator = t.Focused.PrevIndicator.Foreground(cyan)
	t.Focused.Option = t.Focused.Option.Foreground(ColorText)
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(ColorText)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(cyan)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(cyan)
	t.Focused.SelectedPrefix = t.Focused.SelectedPrefix.Foreground(cyan)
	t.Focused.UnselectedPrefix = t.Focused.UnselectedPrefix.Foreground(dim)
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(ColorBg).Background(cyan)
	t.Focused.Next = t.Focused.FocusedButton
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(dim).UnsetBackground()
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(cyan)
	t.Focused.TextInput.CursorText = t.Focused.TextInput.CursorText.Foreground(ColorText)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(dim)
	t.Focused.TextInput.Prompt = t.Focused.TextInput.Prompt.Foreground(cyan)
	t.Focused.TextInput.Text = t.Focused.TextInput.Text.Foreground(ColorText)

	t.Blurred = t.Focused
	t.Blurred.Base = t.Focused.Base.BorderStyle(lipgloss.HiddenBorder())
	t.Blurred.Card = t.Blurred.Base
	t.Blurred.Title = lipgloss.NewStyle().Foreground(dim)
	t.Blurred.SelectSelector = lipgloss.NewStyle().SetString("  ")
	t.Blurred.NextIndicator = lipgloss.NewStyle()
	t.Blurred.PrevIndicator = lipgloss.NewStyle()
	t.Blurred.TextInput.Prompt = t.Blurred.TextInput.Prompt.Foreground(dim)
	t.Blurred.FocusedButton = t.Blurred.BlurredButton.Underline(true)

	t.Help.ShortKey = lipgloss.NewStyle().Foreground(ColorText)
	t.Help.FullKey = t.Help.ShortKey
	t.Help.ShortDesc = lipgloss.NewStyle().Foreground(dim)
	t.Help.FullDesc = t.Help.ShortDesc
	t.Help.ShortSeparator = lipgloss.NewStyle().Foreground(ColorBorder)
	t.Help.FullSeparator = t.Help.ShortSeparator
	t.Help.Ellipsis = t.Help.ShortSeparator
	return t
}

func formKeyMap() *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("esc", "ctrl+c"), key.WithHelp("esc", "cancel"))
	return km
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
	).
		WithShowHelp(false).
		WithTheme(formTheme()).
		WithKeyMap(formKeyMap()).
		WithWidth(max(m.width-2, 0))
}

func (m *MainModel) updateAddDrill(msg tea.Msg) tea.Cmd {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	switch m.form.State {
	case huh.StateAborted:
		m.form = nil
		m.state = StateMenu
		m.menuNotice = "Cancelled."
		m.sizeList()
		return nil
	case huh.StateCompleted:
		if m.form.GetBool("confirm") {
			m.saveFormDrill()
		} else {
			m.menuNotice = "Cancelled."
			m.sizeList()
		}
		m.form = nil
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
	}
	category = m.canonicalCategory(category)

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

func (m *MainModel) initEditForm() {
	d := m.editDrill
	m.formCategory = d.Category
	title := d.Title
	difficulty := d.Difficulty
	language := d.Language
	entryFile := d.EntryFile
	testCmd := d.TestCommand
	description := d.Description
	autoSubmit := d.AutoSubmit
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
				Value(&title).
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
				).
				Value(&difficulty),
			huh.NewInput().
				Key("language").
				Title("Language").
				Value(&language).
				Placeholder("e.g. c, go, rust, python, cpp..."),
			huh.NewInput().
				Key("entry_file").
				Title("Entry Filename").
				Value(&entryFile).
				Placeholder("e.g. ft_putchar.c or solution.go"),
			huh.NewInput().
				Key("test_cmd").
				Title("Test Command (optional)").
				Value(&testCmd).
				Placeholder("e.g. gcc -Wall -Wextra -Werror -c ft_putchar.c"),
			huh.NewText().
				Key("description").
				Title("Problem Description").
				Value(&description).
				Placeholder("Prototype and instructions. Markdown works."),
			huh.NewConfirm().
				Key("auto_submit").
				Title("Submit automatically when closing editor? (No = keep shell open)").
				Affirmative("Yes (auto-submit)").
				Negative("No (default)").
				Value(&autoSubmit),
			huh.NewConfirm().
				Key("confirm").
				Title("Save changes to this drill?").
				Affirmative("Yes").
				Negative("Cancel"),
		),
	).
		WithShowHelp(false).
		WithTheme(formTheme()).
		WithKeyMap(formKeyMap()).
		WithWidth(max(m.width-2, 0))
}

func (m *MainModel) updateEditDrill(msg tea.Msg) tea.Cmd {
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	switch m.form.State {
	case huh.StateAborted:
		m.form = nil
		m.editDrill = nil
		m.state = StateMenu
		m.menuNotice = "Cancelled."
		m.sizeList()
		return nil
	case huh.StateCompleted:
		if m.form.GetBool("confirm") {
			m.saveEditDrill()
		} else {
			m.menuNotice = "Cancelled."
		}
		m.form = nil
		m.editDrill = nil
		m.state = StateMenu
		return nil
	}
	return cmd
}

func (m *MainModel) saveEditDrill() {
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
	}
	category = m.canonicalCategory(category)

	d := *m.editDrill
	d.Title = title
	d.Category = category
	d.Difficulty = m.form.GetString("difficulty")
	d.Language = strings.ToLower(strings.TrimSpace(m.form.GetString("language")))
	d.EntryFile = entry
	d.TestCommand = strings.TrimSpace(m.form.GetString("test_cmd"))
	d.Description = strings.TrimSpace(m.form.GetString("description"))
	d.AutoSubmit = m.form.GetBool("auto_submit")
	if err := drill.SaveDrill(d); err != nil {
		m.menuNotice = "Could not save drill: " + err.Error()
	} else {
		m.menuNotice = fmt.Sprintf("Updated %q in %s.", title, category)
	}
	m.reloadDrills()
}
