package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/luvcie/codedrill/internal/storage"
)

// MixerModel runs in a drill's terminal pane without suspending the editor.
type MixerModel struct {
	volumes [2]int
	off     [2]bool
	cursor  int
	width   int
	err     error
}

func NewMixerModel() (*MixerModel, error) {
	cfg, err := storage.LoadConfig()
	if err != nil {
		return nil, err
	}
	return &MixerModel{
		volumes: [2]int{int(cfg.MusicVolume()*100 + .5), int(cfg.TypingVolume()*100 + .5)},
		off:     [2]bool{cfg.Music == nil || !cfg.Music.Enabled, !cfg.TypingSoundsOn()},
		width:   80,
	}, nil
}

func (m *MixerModel) Init() tea.Cmd { return nil }

func (m *MixerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "enter", "ctrl+c":
			return m, tea.Quit
		case "up", "k", "down", "j", "tab", "shift+tab":
			m.cursor = 1 - m.cursor
		case "left", "h":
			m.changeVolume(-5)
		case "right", "l":
			m.changeVolume(5)
		}
	}
	return m, nil
}

func (m *MixerModel) changeVolume(delta int) {
	cfg, err := storage.LoadConfig()
	if err != nil {
		m.err = err
		return
	}
	current := cfg.MusicVolume()
	if m.cursor == 1 {
		current = cfg.TypingVolume()
	}
	next := min(100, max(0, int(current*100+.5)+delta))
	if m.cursor == 0 {
		if cfg.Music == nil {
			cfg.Music = &storage.MusicSettings{}
		}
		cfg.Music.Volume = &next
	} else {
		if cfg.Typing == nil {
			cfg.Typing = &storage.TypingSettings{}
		}
		cfg.Typing.Volume = &next
	}
	if m.err = storage.SaveConfig(cfg); m.err == nil {
		m.volumes[m.cursor] = next
	}
}

func (m *MixerModel) View() string {
	w := max(1, m.width)
	lines := []string{"", "  " + LogoStyle.Render("codedrill") + "  " + SubtextStyle.Render("volume"), ""}
	for i, name := range []string{"Music", "SFX"} {
		status := ""
		if m.off[i] {
			status = " · off"
		}
		line := fmt.Sprintf("%-5s %s%s", name, renderProgressBar(float64(m.volumes[i])/100, max(1, min(32, w-18-len(status)))), status)
		lines = append(lines, cursorRow(i == m.cursor, w, line))
	}
	if m.err != nil {
		lines = append(lines, "  "+DangerStyle.Render("Could not save: "+m.err.Error()))
	}
	lines = append(lines, "", renderHelp("[↑/↓] choose  [←/→] volume  [q/Esc/Enter] return", w, ""))
	return truncateBlock(strings.Join(lines, "\n"), w)
}
