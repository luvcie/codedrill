package referenceview

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

type Model struct {
	paths    []string
	source   map[string]string
	current  int
	viewport viewport.Model
	score    string
	width    int
	height   int
	title    string
	wrap     bool
}

func New(files map[string][]byte, score string) Model {
	m := Model{source: make(map[string]string), viewport: viewport.New(80, 20), score: score, width: 80, height: 24}
	m.viewport.SetHorizontalStep(4)
	for path, data := range files {
		m.paths = append(m.paths, path)
		text := strings.ReplaceAll(safeText(string(data)), "\t", "    ")
		var highlighted bytes.Buffer
		language := "c"
		switch strings.ToLower(filepath.Ext(path)) {
		case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
			language = "cpp"
		}
		if err := quick.Highlight(&highlighted, text, language, "terminal256", "monokai"); err == nil {
			text = highlighted.String()
		}
		m.source[path] = text
	}
	sort.Strings(m.paths)
	m.selectFile()
	return m
}

func safeText(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return '\uFFFD'
		}
		return r
	}, text)
}

func (m *Model) selectFile() {
	if len(m.paths) == 0 {
		m.viewport.SetContent("No reference source files.")
		return
	}
	m.renderFile()
	m.viewport.GotoTop()
	m.viewport.SetXOffset(0)
}

func (m *Model) renderFile() {
	text := m.source[m.paths[m.current]]
	if m.wrap {
		text = ansi.Hardwrap(text, m.width, true)
	}
	m.viewport.SetContent(text)
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = max(1, msg.Width), max(1, msg.Height)
		m.viewport.Width = m.width
		m.viewport.Height = max(0, m.height-2)
		if m.wrap && len(m.paths) > 0 {
			m.renderFile()
		}
		m.viewport.SetYOffset(m.viewport.YOffset)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "home", "g":
			m.viewport.GotoTop()
			return m, nil
		case "end", "G":
			m.viewport.GotoBottom()
			return m, nil
		case "tab", "n", "shift+tab", "p":
			if len(m.paths) > 1 {
				delta := 1
				if msg.String() == "shift+tab" || msg.String() == "p" {
					delta = -1
				}
				m.current = (m.current + len(m.paths) + delta) % len(m.paths)
				m.selectFile()
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	title := m.title
	if title == "" {
		title = "REFERENCE"
	}
	header := title + ": " + m.CurrentFile()
	if m.score != "" {
		header += " | Match: " + m.score
	}
	header = ansi.Truncate(header, m.width, "")
	if m.height == 1 {
		return header
	}
	footer := fmt.Sprintf("q/Esc close | %d/%d | Tab files | arrows/PgUp/PgDn scroll", m.current+1, len(m.paths))
	footer = ansi.Truncate(footer, m.width, "")
	if m.height == 2 {
		return header + "\n" + footer
	}
	return header + "\n" + m.ContentView() + "\n" + footer
}

// CurrentFile identifies the selected file without terminal control sequences.
func (m Model) CurrentFile() string {
	if len(m.paths) == 0 {
		return "(none)"
	}
	return safeText(m.paths[m.current])
}

// ContentView allows the app to supply its own header and navigation footer.
func (m Model) ContentView() string { return m.viewport.View() }
