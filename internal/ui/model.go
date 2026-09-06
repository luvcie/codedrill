package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/storage"
)

type AppState int

const (
	StateMenu AppState = iota
	StateAddDrill
	StateBriefing
	StateCountdown
	StateSprinting
	StateEvaluating
	StateResults
)

type drillItem struct {
	drill drill.Drill
	isAdd bool
}

func (i drillItem) Title() string {
	if i.isAdd {
		return "+ [Add New Drill / Exercise]"
	}
	langBadge := strings.ToUpper(i.drill.Language)
	if langBadge == "" {
		langBadge = "CODE"
	}
	return fmt.Sprintf("[%s] %s", langBadge, i.drill.Title)
}

func (i drillItem) Description() string {
	if i.isAdd {
		return "Create a custom coding challenge or paste a drill template"
	}
	desc := i.drill.Description
	if len(desc) > 60 {
		desc = desc[:57] + "..."
	}
	if i.drill.AutoSubmit {
		desc += " [Auto-Submit]"
	}
	return desc
}

func (i drillItem) FilterValue() string {
	if i.isAdd {
		return "add new drill exercise create"
	}
	return i.drill.Title + " " + i.drill.Language + " " + i.drill.Category
}

type countdownTickMsg struct{}

func countdownTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return countdownTickMsg{}
	})
}

type sprintFinishedMsg struct {
	err error
}

type evalFinishedMsg struct {
	passed bool
	output string
	err    error
}

type MainModel struct {
	state       AppState
	list        list.Model
	history     *storage.HistoryStore
	drills      []drill.Drill
	activeDrill *drill.Drill
	session     *runner.Session

	sprintStartTime time.Time
	accumulatedTime time.Duration

	countdownCount int

	form *huh.Form

	lastDuration time.Duration
	isPB         bool
	pbDelta      time.Duration
	evalOutput   string
	evalPassed   bool

	width  int
	height int
}

func NewMainModel() (*MainModel, error) {
	hist, err := storage.LoadHistory()
	if err != nil {
		return nil, err
	}

	drills, _ := drill.LoadDrills()

	m := &MainModel{
		state:   StateMenu,
		history: hist,
		drills:  drills,
	}

	m.refreshList()
	return m, nil
}

func (m *MainModel) refreshList() {
	var items []list.Item
	for _, d := range m.drills {
		items = append(items, drillItem{drill: d, isAdd: false})
	}
	items = append(items, drillItem{isAdd: true})

	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(ColorSecondary).
		BorderLeftForeground(ColorSecondary)
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(lipgloss.Color("#EEEEEE"))

	l := list.New(items, delegate, 70, 16)
	l.Title = "CODEDRILL — SPEEDRUN SELECTION"
	l.Styles.Title = TitleStyle
	m.list = l
}

func (m *MainModel) initForm() {
	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Key("title").
				Title("Exercise Title").
				Placeholder("e.g. ft_putchar, ft_split, Two Sum"),
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
			huh.NewInput().
				Key("description").
				Title("Problem Description").
				Placeholder("Write function prototype or instructions..."),
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

func (m *MainModel) Init() tea.Cmd {
	return nil
}

func (m *MainModel) startOrResumeSprint() tea.Cmd {
	m.state = StateSprinting
	m.sprintStartTime = time.Now()
	sprintCmd := m.session.GetSprintCmd()
	return tea.ExecProcess(sprintCmd, func(err error) tea.Msg {
		return sprintFinishedMsg{err: err}
	})
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-6)
		return m, nil

	case countdownTickMsg:
		if m.state == StateCountdown {
			m.countdownCount--
			if m.countdownCount <= 0 {
				sess, err := runner.NewSession(
					m.activeDrill.ID,
					m.activeDrill.EntryFile,
					m.activeDrill.StarterCode,
					m.activeDrill.TestCommand,
					m.activeDrill.Dir,
					m.activeDrill.AutoSubmit,
				)
				if err != nil {
					return m, nil
				}
				m.session = sess
				m.accumulatedTime = 0
				return m, m.startOrResumeSprint()
			}
			return m, countdownTick()
		}

	case sprintFinishedMsg:
		if m.state == StateSprinting {
			runTime := time.Since(m.sprintStartTime)
			m.accumulatedTime += runTime
			m.lastDuration = m.accumulatedTime

			m.state = StateEvaluating
			return m, func() tea.Msg {
				passed, output, err := m.session.RunTests()
				return evalFinishedMsg{passed: passed, output: output, err: err}
			}
		}

	case evalFinishedMsg:
		if m.state == StateEvaluating {
			m.evalPassed = msg.passed
			m.evalOutput = msg.output
			return m, nil
		}
	}

	switch m.state {
	case StateMenu:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter":
				selected, ok := m.list.SelectedItem().(drillItem)
				if ok {
					if selected.isAdd {
						m.state = StateAddDrill
						m.initForm()
						return m, m.form.Init()
					}
					m.activeDrill = &selected.drill
					m.state = StateBriefing
					return m, nil
				}
			}
		}
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	case StateAddDrill:
		form, cmd := m.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.form = f
		}
		if m.form.State == huh.StateCompleted {
			confirmed := m.form.GetBool("confirm")
			if !confirmed {
				m.state = StateMenu
				return m, nil
			}

			title := strings.TrimSpace(m.form.GetString("title"))
			if title == "" {
				title = "Untitled Drill"
			}
			newID := strings.ToLower(strings.ReplaceAll(title, " ", "-"))

			entry := strings.TrimSpace(m.form.GetString("entry_file"))
			if entry == "" {
				entry = "solution.txt"
			}

			autoSubmit := m.form.GetBool("auto_submit")

			newDrill := drill.Drill{
				ID:          newID,
				Title:       title,
				Language:    strings.ToLower(strings.TrimSpace(m.form.GetString("language"))),
				EntryFile:   entry,
				TestCommand: strings.TrimSpace(m.form.GetString("test_cmd")),
				Description: strings.TrimSpace(m.form.GetString("description")),
				AutoSubmit:  autoSubmit,
			}
			_ = drill.SaveDrill(newDrill)
			m.drills, _ = drill.LoadDrills()
			m.refreshList()
			m.state = StateMenu
			return m, nil
		}
		if m.form.State == huh.StateAborted {
			m.state = StateMenu
			return m, nil
		}
		return m, cmd

	case StateBriefing:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter", " ":
				m.state = StateCountdown
				m.countdownCount = 3
				return m, countdownTick()
			case "esc", "q":
				m.state = StateMenu
				return m, nil
			}
		}

	case StateEvaluating:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "enter", "s":
				if m.evalPassed {
					m.state = StateResults
					isPB, delta, _ := m.history.RecordAttempt(
						m.activeDrill.ID,
						m.lastDuration,
						true,
						m.session.Submits,
					)
					m.isPB = isPB
					m.pbDelta = delta
					if m.session != nil {
						m.session.Cleanup()
					}
					return m, nil
				}
				return m, m.startOrResumeSprint()

			case "e":
				return m, m.startOrResumeSprint()

			case "q", "esc":
				if m.session != nil {
					m.session.Cleanup()
				}
				m.state = StateMenu
				return m, nil
			}
		}

	case StateResults:
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch msg.String() {
			case "r":
				m.state = StateCountdown
				m.countdownCount = 3
				return m, countdownTick()
			case "q", "esc", "enter":
				m.state = StateMenu
				return m, nil
			}
		}
	}

	return m, nil
}

func (m *MainModel) View() string {
	switch m.state {
	case StateMenu:
		return "\n" + m.list.View()

	case StateAddDrill:
		return TitleStyle.Render("CREATE NEW DRILL") + "\n\n" + m.form.View()

	case StateBriefing:
		pb := m.history.GetPB(m.activeDrill.ID)
		pbStr := "None yet"
		if pb > 0 {
			pbStr = formatDuration(pb)
		}

		autoSubmitBadge := ""
		if m.activeDrill.AutoSubmit {
			autoSubmitBadge = BadgeStyle.Render("AUTO-SUBMIT ON EXIT") + " Yes\n"
		}

		content := fmt.Sprintf(
			"%s\n\n%s: %s\n%s: %s\n%s\n%s\n\n%s\n%s",
			TitleStyle.Render("DRILL: "+m.activeDrill.Title),
			BadgeStyle.Render("LANGUAGE"), strings.ToUpper(m.activeDrill.Language),
			BadgeStyle.Render("PERSONAL BEST"), pbStr,
			autoSubmitBadge,
			BoxStyle.Render(m.activeDrill.Description),
			TimerStyle.Render("Press [ENTER] or [SPACE] to START SPRINT"),
			SubtextStyle.Render("Press [Esc] or [Q] to back"),
		)
		return "\n" + content

	case StateCountdown:
		return renderCountdown(m.countdownCount, m.width, m.height)

	case StateSprinting:
		return "\n\n\t" + TitleStyle.Render("SPRINT IN PROGRESS") + "\n\tZellij environment active..."

	case StateEvaluating:
		currentTimeStr := formatDuration(m.accumulatedTime)
		if m.evalPassed {
			status := DeltaMinusStyle.Render("[TESTS PASSED]")
			return fmt.Sprintf(
				"\n%s\n\n%s  Current Time: %s\n\n%s\n\n%s\n%s\n%s",
				TitleStyle.Render("SPRINT EVALUATION"),
				status,
				TimerStyle.Render(currentTimeStr),
				BoxStyle.Render(m.evalOutput),
				TimerStyle.Render("Press [ENTER] to Finish Sprint & Lock Time"),
				SubtextStyle.Render("Press [E] to return to Editor and keep coding"),
				SubtextStyle.Render("Press [Q] to forfeit sprint"),
			)
		}

		status := DeltaPlusStyle.Render("[TESTS / COMPILATION FAILED]")
		return fmt.Sprintf(
			"\n%s\n\n%s  Current Time: %s\n\n%s\n\n%s\n%s",
			TitleStyle.Render("SPRINT EVALUATION"),
			status,
			TimerStyle.Render(currentTimeStr),
			BoxStyle.Render(m.evalOutput),
			TimerStyle.Render("Press [E] or [ENTER] to return to Editor and fix errors"),
			SubtextStyle.Render("Press [Q] to forfeit this sprint"),
		)

	case StateResults:
		pbNotice := ""
		if m.isPB {
			pbNotice = PBStyle.Render("[NEW PERSONAL BEST]") + "\n"
		}
		deltaNotice := ""
		if m.pbDelta > 0 {
			deltaNotice = DeltaMinusStyle.Render(fmt.Sprintf("-%s faster than previous PB", formatDuration(m.pbDelta)))
		}

		timeStr := formatDuration(m.lastDuration)

		content := fmt.Sprintf(
			"%s\n\n%s%s: %s\n%s\n\n%s\n%s",
			TitleStyle.Render("SPRINT FINISHED"),
			pbNotice,
			BadgeStyle.Render("FINAL TIME"), PBStyle.Render(timeStr),
			deltaNotice,
			TimerStyle.Render("Press [R] to Retry Sprint immediately"),
			SubtextStyle.Render("Press [ENTER] or [Q] to Return to Menu"),
		)
		return "\n" + BoxStyle.Render(content)
	}

	return ""
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Millisecond)
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	ms := int(d.Milliseconds()) % 1000

	if mins > 0 {
		return fmt.Sprintf("%02dm %02d.%03ds", mins, secs, ms)
	}
	return fmt.Sprintf("%d.%03ds", secs, ms)
}
