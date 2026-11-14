package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/storage"
)

type AppState int

const (
	StateMenu AppState = iota
	StateAddDrill
	StateEditDrill
	StateBriefing
	StateLeaderboard
	StateStats
	StateCountdown
	StateSprinting
	StateEvaluating
	StateResults
	StateSetupError
	StateSettings
	StateTypingSounds
)

type countdownTickMsg struct{ id uint64 }

type sprintFinishedMsg struct {
	err    error
	zellij bool
}

type evalFinishedMsg struct {
	id           uint64
	session      *runner.Session
	passed       bool
	output       string
	reference    *compare.Result
	referenceErr error
	options      mode.Options
	drill        drill.Drill
	reviewable   bool
}

type MainModel struct {
	state         AppState
	previousState AppState

	list            list.Model
	history         *storage.HistoryStore
	drills          []drill.Drill
	activeDrill     *drill.Drill
	session         *runner.Session
	categoryFilter  string
	categories      []string
	categoryIndex   int
	collapsedGroups map[string]bool
	layout          storage.MenuLayout
	press           *menuPress
	search          menuSearch

	sprintStartTime time.Time
	accumulatedTime time.Duration
	countdownCount  int
	countdownID     uint64

	form         *huh.Form
	formCategory string

	editDrill     *drill.Drill
	pendingDelete *drill.Drill
	menuNotice    string

	briefingMode    mode.Options
	briefingModeID  string
	modeFocus       int
	briefingRefOK   bool
	briefingRefErr  error
	briefingSaveErr error
	runMode         mode.Options
	evalOutput      string
	evalPassed      bool
	evalRef         *compare.Result
	evalRefErr      error
	evalPending     bool
	evalID          uint64
	evalReadyTime   time.Time
	evalReview      *referenceReview
	resultReview    *referenceReview
	referenceDiff   *referenceDiffView
	launchFailed    bool
	saveErr         error
	setupErr        error
	prefsErr        error

	resultDrill       drill.Drill
	lastDuration      time.Duration
	lastSubmits       int
	isPB              bool
	pbDelta           time.Duration
	earnedXP          int
	leveledUp         bool
	levelInfo         storage.LevelInfo
	isDailyBonus      bool
	newBestStreak     bool
	briefingCache     string
	briefingKey       string
	leaderboardRecent bool

	settingsCursor       int
	settingsOrigin       Theme
	settingsSaved        string
	settingsErr          error
	settingsThemesClosed bool
	reset                *factoryResetFlow

	typingCursor  int
	typingChoices []typingChoice
	typingOn      bool
	typingWords   bool
	typingVolume  int
	typingPack    string
	typingNotice  string
	typingErr     error

	mouseOff bool
	shell    string

	helpPress *helpPressState

	width        int
	height       int
	scrollOffset int
}

func getAvailableCategories(drills []drill.Drill) []string {
	cats := []string{"ALL"}
	seen := make(map[string]bool)
	for _, d := range drills {
		cat := drill.ParentCategoryOf(d)
		if key := strings.ToUpper(cat); !seen[key] {
			seen[key] = true
			cats = append(cats, cat)
		}
	}
	drill.SortCategories(cats[1:])
	return cats
}

func NewMainModel() (*MainModel, error) {
	hist, err := storage.LoadHistory()
	if err != nil {
		return nil, err
	}
	drills, loadErr := drill.LoadDrills()
	if drills == nil {
		return nil, loadErr
	}
	cfg, _ := storage.PeekConfig()
	layout, layoutErr := storage.LoadLayout()
	m := &MainModel{
		state:          StateMenu,
		history:        hist,
		drills:         drills,
		categories:     getAvailableCategories(drills),
		categoryFilter: "ALL",
		mouseOff:       !cfg.MouseOn(),
		shell:          cfg.ShellName(),
		layout:         layout,
		briefingMode:   mode.Default(),
		runMode:        mode.Default(),
	}
	if loadErr != nil {
		m.menuNotice = "Some drills could not be loaded: " + loadErr.Error()
	} else if layoutErr != nil {
		m.menuNotice = "Could not read the menu arrangement: " + layoutErr.Error()
	}
	m.refreshList()
	return m, nil
}

func NewMainModelWithTarget(idOrTitle string) (*MainModel, error) {
	m, err := NewMainModel()
	if err != nil {
		return nil, err
	}
	found := drill.Find(m.drills, idOrTitle)
	if found == nil {
		return nil, fmt.Errorf("drill %q not found; run 'codedrill list' to see available drills", idOrTitle)
	}
	m.openBriefing(found)
	return m, nil
}

func NewMainModelWithRandomFiltered(filter string) (*MainModel, error) {
	m, err := NewMainModel()
	if err != nil {
		return nil, err
	}
	filter = strings.TrimSpace(filter)
	pool := drill.Filter(m.drills, filter)
	if len(pool) == 0 {
		if filter == "" {
			return nil, fmt.Errorf("no drills available")
		}
		return nil, fmt.Errorf("no drills match %q", filter)
	}
	if filter != "" {
		m.categoryFilter = filter
		for i, c := range m.categories {
			if strings.EqualFold(c, filter) {
				m.categoryIndex, m.categoryFilter = i, c
			}
		}
		m.refreshList()
	}
	chosen := pool[rand.Intn(len(pool))]
	m.openBriefing(&chosen)
	return m, nil
}

func NewMainModelWithAddDrill() (*MainModel, error) {
	m, err := NewMainModel()
	if err != nil {
		return nil, err
	}
	m.state = StateAddDrill
	m.initForm()
	return m, nil
}

func NewMainModelWithDaily() (*MainModel, error) {
	m, err := NewMainModel()
	if err != nil {
		return nil, err
	}
	daily := drill.GetDailyDrill(m.drills, time.Now())
	if daily == nil {
		return nil, fmt.Errorf("no drills available for the daily challenge")
	}
	m.openBriefing(daily)
	return m, nil
}

func (m *MainModel) Init() tea.Cmd {
	var form tea.Cmd
	if m.state == StateAddDrill && m.form != nil {
		form = m.form.Init()
	}
	return tea.Batch(form, m.mouseCmd())
}

func (m *MainModel) Cleanup() {
	if m.session != nil {
		m.session.Cleanup()
		m.session = nil
	}
}

// openBriefing shows a drill's briefing; Esc from there goes to the menu.
func (m *MainModel) openBriefing(d *drill.Drill) {
	m.activeDrill = d
	m.previousState = StateMenu
	if !m.loadBriefingOptions(d) {
		return
	}
	m.state = StateBriefing
}

func (m *MainModel) openLeaderboard(from AppState) {
	m.previousState = from
	m.leaderboardRecent = false
	m.state = StateLeaderboard
}

func (m *MainModel) isDaily(d drill.Drill) bool {
	daily := drill.GetDailyDrill(m.drills, time.Now())
	return daily != nil && daily.ID == d.ID
}

// filteredDrills is the pool n/p/r pick from: drills in the current category,
// or every drill if the category is empty.
func (m *MainModel) filteredDrills() []drill.Drill {
	if pool := m.menuDrills(m.categoryFilter); len(pool) > 0 {
		return pool
	}
	return m.drills
}

// stepDrill moves to the next (delta 1) or previous (delta -1) drill in the
// pool, starting from from.
func (m *MainModel) stepDrill(from drill.Drill, delta int) *drill.Drill {
	pool := m.filteredDrills()
	if len(pool) == 0 {
		return nil
	}
	next := 0
	if delta < 0 {
		next = len(pool) - 1
	}
	for i, d := range pool {
		if d.ID == from.ID {
			next = (i + delta + len(pool)) % len(pool)
		}
	}
	return &pool[next]
}

// randomDrill picks from the pool, avoiding excludeID when there is a choice.
func (m *MainModel) randomDrill(excludeID string) *drill.Drill {
	pool := m.filteredDrills()
	if len(pool) == 0 {
		return nil
	}
	i := rand.Intn(len(pool))
	for len(pool) > 1 && pool[i].ID == excludeID {
		i = rand.Intn(len(pool))
	}
	return &pool[i]
}

func (m *MainModel) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, isKey := msg.(tea.KeyMsg); isKey {
		m.helpPress = nil
	}
	switch msg := msg.(type) {
	case factoryResetFinishedMsg:
		return m, m.finishFactoryReset(msg)
	case typingPreviewMsg:
		if msg.err != nil {
			m.typingNotice = "Preview unavailable: " + msg.err.Error()
		} else if msg.name != "" {
			m.typingNotice = "Previewed " + msg.name + "."
		}
		return m, nil
	case tea.WindowSizeMsg:
		m.cancelPress()
		m.width, m.height = msg.Width, msg.Height
		m.sizeList()
		m.resizeReferenceDiff()
		return m, nil
	case countdownTickMsg:
		return m, m.onCountdownTick(msg)
	case dragScrollMsg:
		return m, m.updateDragScroll(msg)
	case footerPressMsg:
		if m.helpPress == nil {
			return m, nil
		}
		key := m.helpPress.key
		m.helpPress = nil
		return m.Update(key)
	case sprintFinishedMsg:
		if cmd := m.onSprintFinished(msg); cmd != nil {
			return m, cmd
		}
		return m, m.mouseCmd()
	case evalFinishedMsg:
		m.onEvalFinished(msg)
		return m, m.mouseCmd()
	case tea.MouseMsg:
		return m, m.updateMouse(msg)
	}

	switch m.state {
	case StateMenu:
		return m, m.updateMenu(msg)
	case StateAddDrill:
		return m, m.updateAddDrill(msg)
	case StateEditDrill:
		return m, m.updateEditDrill(msg)
	}

	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch m.state {
	case StateSetupError:
		return m, m.updateSetupError(key)
	case StateStats:
		switch key.String() {
		case "esc", "q", "enter", "s":
			m.state = StateMenu
		}
	case StateBriefing:
		return m, m.updateBriefing(key)
	case StateCountdown:
		return m, m.updateCountdown(key)
	case StateLeaderboard:
		return m, m.updateLeaderboard(key)
	case StateEvaluating:
		return m, m.updateEvaluating(key)
	case StateResults:
		return m, m.updateResults(key)
	case StateSettings:
		return m, m.updateSettings(key)
	case StateTypingSounds:
		return m, m.updateTypingSounds(key)
	}
	return m, nil
}

func (m *MainModel) updateBriefing(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "enter":
		m.refreshBriefingReference(m.activeDrill)
		if m.briefingProblem() != "" {
			return nil
		}
		return m.triggerSprintStart()
	case " ":
		m.toggleBriefingOption()
	case "tab":
		m.focusBriefingOption(1)
	case "shift+tab":
		m.focusBriefingOption(-1)
	case "left", "←":
		m.cycleBriefingPreset(-1)
	case "right", "→":
		m.cycleBriefingPreset(1)
	case "1":
		m.applyBriefingPreset(mode.Study())
	case "2":
		m.applyBriefingPreset(mode.Copy())
	case "3":
		m.applyBriefingPreset(mode.Recall())
	case "4":
		m.applyBriefingPreset(mode.Exam())
	case "n", "p":
		delta := 1
		if key.String() == "p" {
			delta = -1
		}
		if d := m.stepDrill(*m.activeDrill, delta); d != nil {
			m.activeDrill = d
			if !m.loadBriefingOptions(d) {
				return nil
			}
		}
	case "r":
		if d := m.randomDrill(m.activeDrill.ID); d != nil {
			m.activeDrill = d
			if !m.loadBriefingOptions(d) {
				return nil
			}
		}
	case "l":
		m.openLeaderboard(StateBriefing)
	case "esc", "q":
		m.state = m.previousState
		m.previousState = StateMenu
	}
	return nil
}

func (m *MainModel) updateLeaderboard(key tea.KeyMsg) tea.Cmd {
	switch key.String() {
	case "enter", " ", "s":
		return m.triggerSprintStart()
	case "n", "p":
		delta := 1
		if key.String() == "p" {
			delta = -1
		}
		if d := m.stepDrill(*m.activeDrill, delta); d != nil {
			m.activeDrill = d
			m.leaderboardRecent = false
		}
	case "tab", "t":
		m.leaderboardRecent = !m.leaderboardRecent
	case "esc", "q", "b":
		m.leaderboardRecent = false
		if m.previousState == StateBriefing {
			m.state, m.previousState = StateBriefing, StateMenu
		} else {
			m.state = m.previousState
		}
	}
	return nil
}
