package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

type Config struct {
	Editor    string `json:"editor"`
	Sound     *bool  `json:"sound,omitempty"`
	Countdown *int   `json:"countdown,omitempty"`
	Theme     string `json:"theme,omitempty"`
	Mouse     *bool  `json:"mouse,omitempty"`
}

func (c Config) IsSoundEnabled() bool {
	if c.Sound == nil {
		return true
	}
	return *c.Sound
}

func (c Config) MouseOn() bool {
	if c.Mouse == nil {
		return true
	}
	return *c.Mouse
}

func (c Config) GetCountdown() int {
	if c.Countdown == nil {
		return 3
	}
	return min(max(*c.Countdown, 0), MaxCountdown)
}

const MaxCountdown = 10

func RingBell()  { ring("bell") }
func RingChime() { ring("chime") }

func ring(kind string) {
	if cfg, _ := LoadConfig(); cfg.IsSoundEnabled() {
		os.Stdout.Write([]byte("\a"))
		go playSystemAudio(kind)
	}
}

func playSystemAudio(kind string) {
	if runtime.GOOS == "darwin" {
		file := "/System/Library/Sounds/Ping.aiff"
		if kind == "chime" {
			file = "/System/Library/Sounds/Glass.aiff"
		}
		if _, err := os.Stat(file); err == nil {
			_ = exec.Command("afplay", file).Run()
			return
		}
	}

	if _, err := exec.LookPath("canberra-gtk-play"); err == nil {
		soundID := "bell"
		if kind == "chime" {
			soundID = "complete"
		}
		if err := exec.Command("canberra-gtk-play", "-i", soundID).Run(); err == nil {
			return
		}
	}

	var player string
	for _, p := range []string{"pw-play", "paplay", "aplay"} {
		if path, err := exec.LookPath(p); err == nil {
			player = path
			break
		}
	}
	if player == "" {
		return
	}

	var candidateFiles []string
	if kind == "chime" {
		candidateFiles = []string{
			"/run/current-system/sw/share/sounds/freedesktop/stereo/complete.oga",
			"/usr/share/sounds/freedesktop/stereo/complete.oga",
			"/run/current-system/sw/share/sounds/freedesktop/stereo/message-new-instant.oga",
			"/usr/share/sounds/freedesktop/stereo/message-new-instant.oga",
			"/usr/share/sounds/gnome/default/alerts/glass.ogg",
		}
	} else {
		candidateFiles = []string{
			"/run/current-system/sw/share/sounds/freedesktop/stereo/bell.oga",
			"/usr/share/sounds/freedesktop/stereo/bell.oga",
			"/run/current-system/sw/share/sounds/freedesktop/stereo/audio-volume-change.oga",
			"/usr/share/sounds/gnome/default/alerts/bark.ogg",
		}
	}

	for _, f := range candidateFiles {
		if _, err := os.Stat(f); err == nil {
			_ = exec.Command(player, f).Run()
			return
		}
	}
}

func configDirPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "codedrill"), nil
}

func ConfigDir() (string, error) {
	dir, err := configDirPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func GetConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LoadConfig() (Config, error) {
	dir, err := ConfigDir()
	if err != nil {
		return Config{}, err
	}
	return readConfig(filepath.Join(dir, "config.json"))
}

func PeekConfig() (Config, error) {
	dir, err := configDirPath()
	if err != nil {
		return Config{}, err
	}
	return readConfig(filepath.Join(dir, "config.json"))
}

func readConfig(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	return cfg, nil
}

func SaveConfig(cfg Config) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWriteFile(path, data, 0644)
}
