package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Editor string `json:"editor"`
}

func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	dir := filepath.Join(configDir, "codedrill")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func LoadConfig() Config {
	cfg := Config{
		Editor: "",
	}
	path, err := GetConfigPath()
	if err != nil {
		return cfg
	}
	data, err := os.ReadFile(path)
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	return cfg
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
	return os.WriteFile(path, data, 0644)
}
