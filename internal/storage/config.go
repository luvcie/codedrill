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
}

func (c Config) IsSoundEnabled() bool {
	if c.Sound == nil {
		return true
	}
	return *c.Sound
}

func (c Config) GetCountdown() int {
	if c.Countdown == nil {
		return 3
	}
	return min(max(*c.Countdown, 0), MaxCountdown)
}

// MaxCountdown is the longest countdown the config accepts, in seconds.
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

// ConfigDir returns codedrill's config directory, creating it if needed.
func ConfigDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "codedrill")
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

// LoadConfig reads config.json. A missing file gives the defaults. On any
// other error it still returns the defaults, so callers that only read
// settings can ignore the error, but nothing should save over the file.
func LoadConfig() (Config, error) {
	var cfg Config
	path, err := GetConfigPath()
	if err != nil {
		return cfg, err
	}
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
