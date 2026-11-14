package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/compare"
	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/runner"
	"github.com/luvcie/codedrill/internal/solutions"
	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/typingsound"
	"github.com/luvcie/codedrill/internal/ui"
)

const Version = "0.2.0"

const helpText = `codedrill - terminal speedrun drill tool for code & algorithms

Usage:
  codedrill                  Launch interactive speedrun menu
  codedrill <id>             Shortcut to directly sprint a drill (e.g. codedrill ft_strlen)
  codedrill start <id>       Directly launch sprint for a drill
  codedrill daily [info]     Sprint today's daily challenge (+50 XP bonus), or show it
  codedrill random [filter]  Jump straight into a random drill sprint (e.g. random c, random 42)
  codedrill new              Create a new custom drill interactively
  codedrill export <id>      Export drill definition as JSON to stdout
  codedrill solution <id> [path...]  List attached reference solutions, or attach files
  codedrill match <id> [--workspace DIR] [--json]  Compare a workspace with its reference
  codedrill import <file>    Import drill definition from a JSON file
  codedrill rm <id>          Delete a custom drill, or hide a built-in one
  codedrill restore [id|all] List hidden built-in drills, or bring them back
  codedrill info <id>        View drill briefing, description & target paces
  codedrill top <id>         View Hall of Fame leaderboard for a drill
  codedrill recent <id>      View recent sprint history for a drill
  codedrill list [filter]    List all drills, or filter by category, language or difficulty
  codedrill search <query>   Search drills by id, title, language, category, difficulty or description
  codedrill stats            View aggregate sprint statistics & profile
  codedrill config [opt]     View or set config (e.g. config editor nvim, config shell zsh, config sound off, config countdown 0, config typing-pack <id>)
  codedrill config theme [id] List colour themes, or set one (e.g. config theme gruvbox)
  codedrill reset <id>       Reset stats and Personal Best for a specific drill
  codedrill reset all        Reset all personal bests, attempts, and streaks
  codedrill version          Show version information
  codedrill --help           Show this help message`

const rule = "--------------------------------------------------------------------------------"

var errUsage = errors.New("usage")

func usage(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{errUsage}, args...)...)
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	cfg, _ := storage.PeekConfig()
	ui.ApplyTheme(ui.ThemeOrDefault(cfg.Theme))
	c := &cli{out: stdout, errOut: stderr}
	if err := c.dispatch(args); err != nil {
		msg := err.Error()
		msg = strings.TrimPrefix(msg, errUsage.Error()+": ")
		fmt.Fprintln(stderr, "codedrill: "+msg)
		return 1
	}
	return 0
}

type cli struct {
	out, errOut io.Writer
}

func (c *cli) printf(format string, args ...any) { fmt.Fprintf(c.out, format, args...) }
func (c *cli) println(args ...any)               { fmt.Fprintln(c.out, args...) }

func needArg(args []string, what, example string) (string, error) {
	if len(args) < 2 {
		return "", usage("missing %s, e.g. codedrill %s", what, example)
	}
	return args[1], nil
}

func (c *cli) dispatch(args []string) error {
	if len(args) == 0 {
		return launch(ui.NewMainModel())
	}
	rest := strings.Join(args[1:], " ")
	switch args[0] {
	case "__typing-sound-editor":
		return runSoundEditor(args[1:])
	case "list", "ls":
		return c.list(rest)
	case "search", "find":
		if rest == "" {
			return usage("missing search query, e.g. codedrill search string")
		}
		return c.search(rest)
	case "info", "show":
		id, err := needArg(args, "drill id", "info ft_strlen")
		if err != nil {
			return err
		}
		return c.info(id)
	case "top", "leaderboard", "hof":
		id, err := needArg(args, "drill id", "top ft_strlen")
		if err != nil {
			return err
		}
		return c.runs(id, false)
	case "recent", "history", "runs":
		id, err := needArg(args, "drill id", "recent ft_strlen")
		if err != nil {
			return err
		}
		return c.runs(id, true)
	case "stats":
		return c.stats()
	case "config":
		return c.config(args[1:])
	case "reset":
		id, err := needArg(args, "drill id or 'all'", "reset ft_strlen")
		if err != nil {
			return err
		}
		if id == "all" || id == "--all" {
			return c.resetAll()
		}
		return c.reset(id)
	case "delete", "rm", "remove":
		id, err := needArg(args, "drill id", "rm my_drill")
		if err != nil {
			return err
		}
		return c.remove(id)
	case "restore", "unhide":
		return c.restore(args[1:])
	case "export":
		id, err := needArg(args, "drill id", "export ft_strlen")
		if err != nil {
			return err
		}
		data, err := drill.ExportDrill(id)
		if err != nil {
			return err
		}
		c.println(string(data))
		return nil
	case "import":
		path, err := needArg(args, "file path", "import drill.json")
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		d, err := drill.ImportDrill(data)
		if err != nil {
			return err
		}
		c.printf("Imported %q as %s.\n", d.Title, d.ID)
		return nil
	case "solution", "solutions":
		return c.solution(args[1:])
	case "match":
		return c.match(args[1:])
	case "reference-view":
		return c.referenceView(args[1:])
	case "reference-diff":
		return c.referenceDiff(args[1:])
	case "version", "--version", "-v":
		c.printf("codedrill v%s\n", Version)
		return nil
	case "help", "--help", "-h":
		c.println(helpText)
		return nil
	case "new", "add", "create":
		return launch(ui.NewMainModelWithAddDrill())
	case "start":
		id, err := needArg(args, "drill id", "start ft_strlen")
		if err != nil {
			return err
		}
		return launch(ui.NewMainModelWithTarget(id))
	case "daily", "today":
		if rest == "info" || rest == "-i" || rest == "--info" {
			drills, err := c.loadDrills()
			if err != nil {
				return err
			}
			d := drill.GetDailyDrill(drills, time.Now())
			if d == nil {
				return errors.New("no drills available")
			}
			return c.info(d.ID)
		}
		return launch(ui.NewMainModelWithDaily())
	case "random", "rand":
		return launch(ui.NewMainModelWithRandomFiltered(rest))
	}

	m, err := ui.NewMainModelWithTarget(args[0])
	if err != nil {
		return usage("unknown command or drill %q; run 'codedrill list' or 'codedrill --help'", args[0])
	}
	return launch(m, nil)
}

// launch runs the TUI and always removes the sprint workspace afterwards.
func launch(m *ui.MainModel, err error) error {
	if err != nil {
		return err
	}
	defer m.Cleanup()
	_, err = tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}

// loadDrills returns the drills, warning about any custom drill that failed to
// load. It only fails when nothing could be loaded.
func (c *cli) loadDrills() ([]drill.Drill, error) {
	drills, err := drill.LoadDrills()
	if drills == nil {
		return nil, err
	}
	if err != nil {
		fmt.Fprintln(c.errOut, "codedrill: warning:", err)
	}
	return drills, nil
}

func (c *cli) findDrill(idOrTitle string) (*drill.Drill, error) {
	drills, err := c.loadDrills()
	if err != nil {
		return nil, err
	}
	d := drill.Find(drills, idOrTitle)
	if d == nil {
		return nil, fmt.Errorf("drill %q not found; run 'codedrill list' to see available drills", idOrTitle)
	}
	return d, nil
}

func pbText(hist *storage.HistoryStore, d drill.Drill) string {
	pb := hist.GetPB(d.ID)
	if pb <= 0 {
		return "-"
	}
	grade, _ := ui.CalculateGrade(pb, d.Difficulty)
	return fmt.Sprintf("%s (%s)", storage.FormatDuration(pb), grade)
}

func (c *cli) drillTable(header string, drills []drill.Drill, hist *storage.HistoryStore, footer string) {
	c.println(header)
	c.println(rule)
	for _, d := range drills {
		difficulty := strings.ToUpper(d.Difficulty)
		switch difficulty {
		case "":
			difficulty = "EASY"
		case "MEDIUM":
			difficulty = "MED"
		}
		c.printf("  %-18s [%-6s] [%-10s] [%-4s]  PB: %-14s  %s\n",
			d.ID, strings.ToUpper(d.Language), drill.CategoryOf(d), difficulty, pbText(hist, d), d.Title)
	}
	c.println(rule)
	c.println(footer)
}

func (c *cli) list(filter string) error {
	drills, err := c.loadDrills()
	if err != nil {
		return err
	}
	hist, err := storage.LoadHistory()
	if err != nil {
		return err
	}
	filter = strings.TrimSpace(filter)
	matches := drill.Filter(drills, filter)
	if len(matches) == 0 {
		return fmt.Errorf("no drills match %q", filter)
	}
	header := "Drills:"
	if filter != "" {
		header = fmt.Sprintf("Drills matching %q:", filter)
	}
	c.drillTable(header, matches, hist, "Run 'codedrill start <id>' or 'codedrill random' to sprint.")
	return nil
}

func (c *cli) search(query string) error {
	drills, err := c.loadDrills()
	if err != nil {
		return err
	}
	hist, err := storage.LoadHistory()
	if err != nil {
		return err
	}
	q := strings.ToLower(strings.TrimSpace(query))
	var matches []drill.Drill
	for _, d := range drills {
		fields := []string{d.ID, d.Title, d.Language, d.Category, d.Difficulty, d.Description}
		if slices.ContainsFunc(fields, func(f string) bool { return strings.Contains(strings.ToLower(f), q) }) {
			matches = append(matches, d)
		}
	}
	if len(matches) == 0 {
		return fmt.Errorf("no drills match %q", query)
	}
	c.drillTable(fmt.Sprintf("Search results for %q (%d):", query, len(matches)), matches, hist, "Run 'codedrill <id>' to sprint.")
	return nil
}

func (c *cli) info(idOrTitle string) error {
	d, err := c.findDrill(idOrTitle)
	if err != nil {
		return err
	}
	hist, err := storage.LoadHistory()
	if err != nil {
		return err
	}
	difficulty := d.Difficulty
	if difficulty == "" {
		difficulty = "easy"
	}
	s, a, b := drill.PaceTargets(difficulty)
	pb := "None yet"
	if p := hist.GetPB(d.ID); p > 0 {
		grade, _ := ui.CalculateGrade(p, d.Difficulty)
		pb = fmt.Sprintf("%s (%s-RANK)", storage.FormatDuration(p), grade)
	}

	c.println(rule)
	c.printf("Drill:         %s (%s)\n", d.Title, d.ID)
	c.printf("Category:      %s\n", drill.CategoryOf(*d))
	c.printf("Language:      %s\n", strings.ToUpper(d.Language))
	c.printf("Difficulty:    %s\n", strings.ToUpper(difficulty))
	c.printf("Target paces:  S ≤ %v · A ≤ %v · B ≤ %v\n", s, a, b)
	c.printf("Personal best: %s\n", pb)
	c.printf("Entry file:    %s\n", d.EntryFile)
	c.println(rule)
	c.println(d.Description)
	c.println(rule)
	c.printf("Run 'codedrill %s' to sprint.\n", d.ID)
	return nil
}

func (c *cli) runs(idOrTitle string, recent bool) error {
	d, err := c.findDrill(idOrTitle)
	if err != nil {
		return err
	}
	hist, err := storage.LoadHistory()
	if err != nil {
		return err
	}

	title, attempts := "Hall of fame", hist.GetTopAttempts(d.ID, 10)
	if recent {
		title, attempts = "Recent runs", hist.GetRecentAttempts(d.ID, 10)
	}
	c.println(rule)
	c.printf("%s: %s (%s)\n", title, d.Title, strings.ToUpper(d.Language))
	if st := hist.GetStats(d.ID); st != nil && st.TotalRuns > 0 {
		streak := ""
		if st.CurrentStreak > 0 {
			streak = fmt.Sprintf(" · Streak: %d (best %d)", st.CurrentStreak, st.BestStreak)
		}
		c.printf("Runs: %d · Passed: %d (%d%%) · PB: %s%s\n",
			st.TotalRuns, st.Successful, st.Successful*100/st.TotalRuns, pbText(hist, *d), streak)
	}
	c.println(rule)
	if len(attempts) == 0 {
		c.println("  No runs yet.")
	} else {
		first := "RANK"
		if recent {
			first = "STATUS"
		}
		c.printf("  %-8s %-16s %-8s %-10s %s\n", first, "TIME", "GRADE", "SUBMITS", "DATE")
		for i, att := range attempts {
			label := "#" + strconv.Itoa(i+1)
			grade := "-"
			if att.Passed {
				g, _ := ui.CalculateGrade(att.Duration, d.Difficulty)
				grade = "[" + string(g) + "]"
			}
			if recent {
				label = "PASS"
				if !att.Passed {
					label = "FAIL"
				}
			}
			c.printf("  %-8s %-16s %-8s %-10d %s\n",
				label, storage.FormatDuration(att.Duration), grade, att.Attempts, att.Timestamp.Format("2006-01-02 15:04"))
		}
	}
	c.println(rule)
	c.printf("Run 'codedrill %s' to sprint.\n", d.ID)
	return nil
}

func (c *cli) stats() error {
	hist, err := storage.LoadHistory()
	if err != nil {
		return err
	}
	drills, err := c.loadDrills()
	if err != nil {
		return err
	}
	ids := make([]string, len(drills))
	byID := make(map[string]drill.Drill, len(drills))
	for i, d := range drills {
		ids[i] = d.ID
		byID[d.ID] = d
	}
	lvl := storage.GetLevelInfo(hist.GetTotalXP())
	runsToday, passedToday := hist.RunsToday()

	const short = "--------------------------------------------------"
	c.println("Speedrunner profile:")
	c.println(short)
	c.printf("  Runner Level:      Level %d [%s]\n", lvl.Level, lvl.Title)
	c.printf("  Total XP:          %d / %d XP\n", lvl.TotalXP, lvl.NextLevelXP)
	c.printf("  Total Sprints:     %d\n", hist.GetTotalRuns())
	c.printf("  Passed:            %d (%d%% win rate)\n", hist.GetPassedRuns(), hist.GetOverallWinRate())
	c.printf("  Win Streak:        %d (best %d)\n", hist.CurrentStreak, hist.BestStreak)
	c.printf("  Sprints Today:     %d (%d passed)\n", runsToday, passedToday)
	if day, best := hist.GetDailyStreak(), hist.GetBestDailyStreak(); day > 0 || best > 0 {
		c.printf("  Daily Streak:      %d days (best %d)\n", day, best)
	}
	c.printf("  Practice Time:     %s\n", storage.FormatDuration(hist.GetTotalPracticeTime()))
	c.printf("  Drills Cleared:    %d / %d\n", hist.ClearedCount(ids), len(drills))
	c.println(short)

	var cleared []string
	for id, st := range hist.Stats {
		if st.Successful > 0 {
			cleared = append(cleared, id)
		}
	}
	if len(cleared) == 0 {
		return nil
	}
	slices.Sort(cleared)
	c.println("Personal bests:")
	for _, id := range cleared {
		st := hist.Stats[id]
		pb := storage.FormatDuration(st.PersonalBest)
		if d, ok := byID[id]; ok {
			grade, _ := ui.CalculateGrade(st.PersonalBest, d.Difficulty)
			pb += " [" + string(grade) + "]"
		}
		streak := ""
		if st.CurrentStreak > 1 {
			streak = fmt.Sprintf(", streak: %d", st.CurrentStreak)
		}
		c.printf("  %-18s  PB: %-14s (runs: %d, passed: %d%s)\n", id, pb, st.TotalRuns, st.Successful, streak)
	}
	c.println(short)
	return nil
}

func (c *cli) reset(idOrTitle string) error {
	hist, err := storage.LoadHistory()
	if err != nil {
		return err
	}
	id := idOrTitle
	if d, err := c.findDrill(idOrTitle); err == nil {
		id = d.ID
	} else if hist.GetStats(idOrTitle) == nil {
		return err
	}
	if err := hist.ResetDrillStats(id); err != nil {
		return err
	}
	c.printf("Reset the runs and PB for %s.\n", id)
	return nil
}

func (c *cli) resetAll() error {
	hist, err := storage.LoadHistory()
	if err != nil {
		return err
	}
	if err := hist.ResetAll(); err != nil {
		return err
	}
	c.println("Reset all personal bests, runs and streaks.")
	return nil
}

func (c *cli) remove(idOrTitle string) error {
	d, hidden, err := drill.RemoveDrill(idOrTitle)
	if err != nil {
		return err
	}
	if hidden {
		c.printf("Hid built-in drill %s. Bring it back with: codedrill restore %s\n", d.ID, d.ID)
		return nil
	}
	hist, err := storage.LoadHistory()
	if err == nil {
		err = hist.ResetDrillStats(d.ID)
	}
	if err != nil {
		return fmt.Errorf("deleted %s, but could not clear its runs: %w", d.ID, err)
	}
	c.printf("Deleted custom drill %s.\n", d.ID)
	return nil
}

func (c *cli) restore(args []string) error {
	if len(args) == 0 {
		ids, err := drill.HiddenDrillIDs()
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			c.println("No hidden drills.")
			return nil
		}
		c.println("Hidden drills (restore with: codedrill restore <id> | all):")
		for _, id := range ids {
			c.printf("  %s\n", id)
		}
		return nil
	}
	if args[0] == "all" {
		if err := drill.RestoreAll(); err != nil {
			return err
		}
		c.println("Restored all hidden drills.")
		return nil
	}
	if err := drill.RestoreDrill(args[0]); err != nil {
		return err
	}
	c.printf("Restored %s.\n", args[0])
	return nil
}

func (c *cli) solution(args []string) error {
	if len(args) == 0 {
		return usage("missing drill id, e.g. codedrill solution ft_strlen answer.c")
	}
	d, err := c.findDrill(args[0])
	if err != nil {
		return err
	}
	if len(args) == 1 {
		return c.listSolutions(*d)
	}
	att, err := solutions.Attach(d.ID, args[1:])
	if err != nil {
		return err
	}
	noun := "files"
	if len(att.Files) == 1 {
		noun = "file"
	}
	c.printf("Attached %d solution %s for %s:\n", len(att.Files), noun, d.ID)
	for _, rel := range att.Files {
		c.printf("  %s\n", rel)
	}
	c.printf("Destination: %s\n", att.Dir)
	return nil
}

func (c *cli) listSolutions(d drill.Drill) error {
	src, err := solutions.Resolve(d.ID, d.Dir)
	if err != nil {
		return err
	}
	if src == nil || len(src.Files) == 0 {
		c.printf("No solution files attached for %s.\n", d.ID)
		c.printf("Attach one with: codedrill solution %s <path>\n", d.ID)
		return nil
	}
	c.printf("Solution files for %s:\n", d.ID)
	for _, f := range src.Files {
		c.printf("  %s\n", f.Rel)
	}
	if src.Local {
		c.printf("Destination: %s\n", src.Dir)
	} else {
		c.printf("Source: %s\n", src.Dir)
	}
	return nil
}

type matchReport struct {
	Drill           string            `json:"drill"`
	Title           string            `json:"title"`
	Language        string            `json:"language"`
	Workspace       string            `json:"workspace"`
	Exact           bool              `json:"exact"`
	Percent         float64           `json:"percent"`
	MatchedTokens   int               `json:"matched_tokens"`
	MissingTokens   int               `json:"missing_tokens"`
	ExtraTokens     int               `json:"extra_tokens"`
	ReferenceTokens int               `json:"reference_tokens"`
	StudentTokens   int               `json:"student_tokens"`
	FirstDifference *matchLocation    `json:"first_difference,omitempty"`
	Files           []matchFileReport `json:"files"`
}

type matchLocation struct {
	Path    string `json:"path"`
	Line    int    `json:"line,omitempty"`
	Missing bool   `json:"missing,omitempty"`
}

type matchFileReport struct {
	Path               string  `json:"path"`
	Missing            bool    `json:"missing"`
	Exact              bool    `json:"exact"`
	Percent            float64 `json:"percent"`
	MatchedTokens      int     `json:"matched_tokens"`
	MissingTokens      int     `json:"missing_tokens"`
	ExtraTokens        int     `json:"extra_tokens"`
	ReferenceTokens    int     `json:"reference_tokens"`
	StudentTokens      int     `json:"student_tokens"`
	FirstDifferingLine int     `json:"first_differing_line,omitempty"`
}

func (c *cli) match(args []string) error {
	workspace := ""
	asJSON := false
	var drillArgs []string
	for i := 0; i < len(args); i++ {
		switch arg := args[i]; {
		case arg == "--json":
			asJSON = true
		case arg == "--workspace":
			if i+1 >= len(args) {
				return usage("missing directory after --workspace")
			}
			i++
			workspace = args[i]
		case strings.HasPrefix(arg, "--workspace="):
			workspace = strings.TrimPrefix(arg, "--workspace=")
			if workspace == "" {
				return usage("missing directory after --workspace")
			}
		case strings.HasPrefix(arg, "-"):
			return usage("unknown option %q; use --workspace DIR or --json", arg)
		default:
			drillArgs = append(drillArgs, arg)
		}
	}
	if len(drillArgs) == 0 {
		return usage("missing drill id, e.g. codedrill match ft_strlen")
	}
	d, err := c.findDrill(strings.Join(drillArgs, " "))
	if err != nil {
		return err
	}
	if workspace == "" {
		workspace, err = os.Getwd()
		if err != nil {
			return err
		}
	}
	res, err := compare.Compare(*d, workspace)
	if err != nil {
		return err
	}
	if asJSON {
		return c.printMatchJSON(*d, workspace, res)
	}
	c.printMatch(*d, res)
	return nil
}

func matchCounts(res compare.Result) (missing, extra int) {
	return res.ReferenceTokens - res.Matched, res.StudentTokens - res.Matched
}

func firstMatchDifference(res compare.Result) (matchLocation, bool) {
	for _, f := range res.Files {
		if f.Missing {
			return matchLocation{Path: f.Rel, Missing: true}, true
		}
		if !f.Result.Exact {
			return matchLocation{Path: f.Rel, Line: f.Result.FirstDifferingLine}, true
		}
	}
	return matchLocation{}, false
}

func (c *cli) printMatch(d drill.Drill, res compare.Result) {
	title := d.Title
	if title == "" {
		title = d.ID
	}
	missing, extra := matchCounts(res)
	c.printf("Match: %s (%s)\n", title, strings.ToUpper(d.Language))
	c.printf("Score: %.1f%%\n", res.Percent())
	c.printf("Tokens: %d matched, %d missing, %d extra\n", res.Matched, missing, extra)
	if loc, ok := firstMatchDifference(res); ok {
		switch {
		case loc.Missing:
			c.printf("First difference: %s (missing from workspace)\n", loc.Path)
		case loc.Line > 0:
			c.printf("First difference: %s:%d\n", loc.Path, loc.Line)
		default:
			c.printf("First difference: %s\n", loc.Path)
		}
	}
	if len(res.Files) > 1 {
		c.println("Files:")
		for _, f := range res.Files {
			status := "exact"
			switch {
			case f.Missing:
				status = "missing from workspace"
			case !f.Result.Exact:
				status = fmt.Sprintf("%d missing, %d extra", f.Result.Missing, f.Result.Extra)
			}
			c.printf("  %-24s %5.1f%%  %s\n", f.Rel, f.Result.Percent(), status)
		}
	}
}

func (c *cli) printMatchJSON(d drill.Drill, workspace string, res compare.Result) error {
	title := d.Title
	if title == "" {
		title = d.ID
	}
	missing, extra := matchCounts(res)
	report := matchReport{
		Drill:           d.ID,
		Title:           title,
		Language:        strings.ToLower(d.Language),
		Workspace:       workspace,
		Exact:           res.Exact,
		Percent:         res.Percent(),
		MatchedTokens:   res.Matched,
		MissingTokens:   missing,
		ExtraTokens:     extra,
		ReferenceTokens: res.ReferenceTokens,
		StudentTokens:   res.StudentTokens,
		Files:           make([]matchFileReport, 0, len(res.Files)),
	}
	if loc, ok := firstMatchDifference(res); ok {
		report.FirstDifference = &loc
	}
	for _, f := range res.Files {
		report.Files = append(report.Files, matchFileReport{
			Path:               f.Rel,
			Missing:            f.Missing,
			Exact:              f.Result.Exact,
			Percent:            f.Result.Percent(),
			MatchedTokens:      f.Result.Matched,
			MissingTokens:      f.Result.Missing,
			ExtraTokens:        f.Result.Extra,
			ReferenceTokens:    f.Result.ReferenceTokens,
			StudentTokens:      f.Result.StudentTokens,
			FirstDifferingLine: f.Result.FirstDifferingLine,
		})
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	c.println(string(data))
	return nil
}

func (c *cli) config(args []string) error {
	cfg, err := storage.LoadConfig()
	if err != nil {
		return fmt.Errorf("%w; fix or remove it before changing settings", err)
	}
	if len(args) == 0 {
		return c.printConfig(cfg)
	}

	var msg string
	switch args[0] {
	case "editor":
		if len(args) < 2 {
			return usage("missing editor name, e.g. codedrill config editor nvim")
		}
		cfg.Editor = args[1]
		msg = fmt.Sprintf("Editor set to %s.", cfg.Editor)
	case "shell":
		if len(args) < 2 {
			return usage("missing shell name, e.g. codedrill config shell zsh")
		}
		shell := strings.TrimSpace(args[1])
		if shell == "" {
			return usage("shell cannot be empty; use bash, zsh or a path")
		}
		cfg.Shell = shell
		msg = fmt.Sprintf("Shell set to %s.", shell)
	case "sound", "bell":
		if len(args) < 2 {
			return usage("missing value, e.g. codedrill config sound off")
		}
		var on bool
		switch strings.ToLower(args[1]) {
		case "on", "true", "yes", "1", "enable":
			on, msg = true, "Sound on."
		case "off", "false", "no", "0", "disable":
			on, msg = false, "Sound off."
		default:
			return usage("sound must be on or off, not %q", args[1])
		}
		cfg.Sound = &on
	case "mouse":
		if len(args) < 2 {
			return usage("missing value, e.g. codedrill config mouse off")
		}
		var on bool
		switch strings.ToLower(args[1]) {
		case "on", "true", "yes", "1", "enable":
			on, msg = true, "Mouse on."
		case "off", "false", "no", "0", "disable":
			on, msg = false, "Mouse off."
		default:
			return usage("mouse must be on or off, not %q", args[1])
		}
		cfg.Mouse = &on
	case "theme":
		if len(args) < 2 {
			c.printThemes(cfg)
			return nil
		}
		t, ok := ui.ThemeByID(args[1])
		if !ok {
			return usage("unknown theme %q; valid themes are %s", args[1], themeIDs())
		}
		cfg.Theme = t.ID
		msg = fmt.Sprintf("Theme set to %s.", t.Name)
	case "countdown":
		if len(args) < 2 {
			return usage("missing value, e.g. codedrill config countdown 3")
		}
		n, err := strconv.Atoi(args[1])
		switch strings.ToLower(args[1]) {
		case "off", "disable", "none", "instant":
			n, err = 0, nil
		}
		if err != nil || n < 0 || n > storage.MaxCountdown {
			return usage("countdown must be 0-%d or off, not %q", storage.MaxCountdown, args[1])
		}
		cfg.Countdown = &n
		msg = fmt.Sprintf("Countdown set to %ds.", n)
		if n == 0 {
			msg = "Countdown off; sprints start immediately."
		}
	case "typing":
		if len(args) < 2 {
			return usage("missing value, e.g. codedrill config typing on")
		}
		on, err := onOff(args[1])
		if err != nil {
			return usage("typing must be on or off, not %q", args[1])
		}
		typing := ensureTyping(&cfg)
		typing.Enabled = &on
		if on {
			msg = fmt.Sprintf("Typing sounds on (%s).", typingsound.PackLabel(cfg.TypingPackID()))
		} else {
			msg = "Typing sounds off."
		}
	case "typing-pack":
		if len(args) < 2 {
			c.printTypingPacks(cfg)
			return nil
		}
		id := strings.TrimSpace(args[1])
		if id != typingsound.SilentPackID && id != typingsound.PrototypePackID {
			if _, ok := typingsound.FindPack(id); !ok {
				return usage("unknown sound pack %q; run 'codedrill config typing-pack' to list packs", id)
			}
		}
		typing := ensureTyping(&cfg)
		typing.Pack = id
		msg = fmt.Sprintf("Typing sound pack set to %s.", typingsound.PackLabel(id))
	case "typing-volume":
		if len(args) < 2 {
			return usage("missing value, e.g. codedrill config typing-volume 70")
		}
		n, err := strconv.Atoi(args[1])
		if err != nil || n < 0 || n > 100 {
			return usage("typing volume must be 0-100, not %q", args[1])
		}
		typing := ensureTyping(&cfg)
		typing.Volume = &n
		msg = fmt.Sprintf("Typing volume set to %d%%.", n)
	case "typing-words":
		if len(args) < 2 {
			return usage("missing value, e.g. codedrill config typing-words on")
		}
		on, err := onOff(args[1])
		if err != nil {
			return usage("typing-words must be on or off, not %q", args[1])
		}
		typing := ensureTyping(&cfg)
		typing.WordCompletion = &on
		if on {
			msg = "Word completion sounds on."
		} else {
			msg = "Word completion sounds off."
		}
	case "typing-keys":
		if len(args) < 4 {
			return usage("expected lock, combo and quad key lists, e.g. codedrill config typing-keys '{,enter' ';' '}'")
		}
		if _, err := typingsound.ParseMapping(args[1], args[2], args[3]); err != nil {
			return usage("%v", err)
		}
		typing := ensureTyping(&cfg)
		typing.LockKeys, typing.ComboKeys, typing.QuadKeys = args[1], args[2], args[3]
		msg = "Typing key mapping saved."
	default:
		return usage("unknown config option %q; use editor, shell, sound, mouse, countdown, theme, typing, typing-pack, typing-volume, typing-words or typing-keys", args[0])
	}
	if err := storage.SaveConfig(cfg); err != nil {
		return err
	}
	c.println(msg)
	return nil
}

// onOff parses the on/off words the config commands share.
func onOff(value string) (bool, error) {
	switch strings.ToLower(value) {
	case "on", "true", "yes", "1", "enable":
		return true, nil
	case "off", "false", "no", "0", "disable":
		return false, nil
	}
	return false, errors.New("expected on or off")
}

func ensureTyping(cfg *storage.Config) *storage.TypingSettings {
	if cfg.Typing == nil {
		cfg.Typing = &storage.TypingSettings{}
	}
	return cfg.Typing
}

// printTypingPacks lists the packs the settings screen and config command use.
func (c *cli) printTypingPacks(cfg storage.Config) {
	current := cfg.TypingPackID()
	if current == "" {
		current = typingsound.SilentPackID
	}
	c.println("Typing sound packs:")
	c.printf("  %s %s\n", mark(current, typingsound.SilentPackID), typingsound.PackLabel(typingsound.SilentPackID))
	c.printf("  %s %s\n", mark(current, typingsound.PrototypePackID), typingsound.PackLabel(typingsound.PrototypePackID))
	for _, pack := range typingsound.Catalog {
		c.printf("  %s %-60s %s\n", mark(current, pack.ID), pack.ID, pack.Name)
	}
	c.println("Set with: codedrill config typing-pack <id>")
}

func mark(current, id string) string {
	if current == id {
		return "*"
	}
	return " "
}

func (c *cli) printConfig(cfg storage.Config) error {
	path, err := storage.GetConfigPath()
	if err != nil {
		return err
	}
	drillsDir, err := drill.GetUserDrillsDir()
	if err != nil {
		return err
	}
	editor := cfg.Editor
	if editor == "" {
		editor = runner.DetectEditor() + " (auto-detected)"
	}
	sound := "on"
	if !cfg.IsSoundEnabled() {
		sound = "off"
	}
	mouse := "on"
	if !cfg.MouseOn() {
		mouse = "off"
	}
	shell := cfg.ShellName()
	if cfg.Shell == "" {
		shell += " (default)"
	}
	countdown := fmt.Sprintf("%ds", cfg.GetCountdown())
	if cfg.GetCountdown() == 0 {
		countdown = "off"
	}
	theme := ui.ThemeOrDefault(cfg.Theme)
	themeText := fmt.Sprintf("%s (%s)", theme.ID, theme.Name)
	if _, ok := ui.ThemeByID(cfg.Theme); !ok && cfg.Theme != "" {
		themeText = fmt.Sprintf("%s (unknown, using %s)", cfg.Theme, ui.Themes[0].ID)
	}
	typingSound := "off"
	if cfg.TypingSoundsOn() {
		typingSound = "on"
	}
	typingVolume := int(cfg.TypingVolume()*100 + 0.5)
	typingWords := "off"
	if cfg.TypingWordSoundsOn() {
		typingWords = "on"
	}
	const short = "--------------------------------------------------"
	c.println("Configuration:")
	c.println(short)
	c.printf("  Config File:    %s\n", path)
	c.printf("  Drills Dir:     %s\n", drillsDir)
	c.printf("  Editor:         %s\n", editor)
	c.printf("  Shell:          %s\n", shell)
	c.printf("  Sound:          %s\n", sound)
	c.printf("  Countdown:      %s\n", countdown)
	c.printf("  Mouse:          %s\n", mouse)
	c.printf("  Theme:          %s\n", themeText)
	c.printf("  Typing sounds:  %s\n", typingSound)
	c.printf("  Typing pack:    %s\n", typingsound.PackLabel(cfg.TypingPackID()))
	c.printf("  Typing volume:  %d%%\n", typingVolume)
	c.printf("  Typing words:   %s\n", typingWords)
	c.println(short)
	c.println("Change with: codedrill config editor <name> | shell <bash|zsh|path> | sound <on|off> | mouse <on|off> | countdown <0-10|off> | theme <id> | typing <on|off> | typing-pack <id> | typing-volume <0-100> | typing-words <on|off>")
	return nil
}

func themeIDs() string {
	ids := make([]string, len(ui.Themes))
	for i, t := range ui.Themes {
		ids[i] = t.ID
	}
	return strings.Join(ids, ", ")
}

func (c *cli) printThemes(cfg storage.Config) {
	current := ui.ThemeOrDefault(cfg.Theme).ID
	width := 0
	for _, t := range ui.Themes {
		width = max(width, len(t.ID))
	}
	for _, t := range ui.Themes {
		mark := " "
		if t.ID == current {
			mark = "*"
		}
		c.printf("%s %-*s  %s\n", mark, width, t.ID, t.Name)
	}
	c.println("Set with: codedrill config theme <id>")
}
