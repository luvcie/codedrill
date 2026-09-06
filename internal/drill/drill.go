package drill

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Drill struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Category    string `json:"category"`
	Language    string `json:"language"`
	Description string `json:"description"`
	EntryFile   string `json:"entry_file"`
	StarterCode string `json:"starter_code"`
	TestCommand string `json:"test_command"`
	AutoSubmit  bool   `json:"auto_submit"`
	Dir         string `json:"-"`
}

func GetUserDrillsDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	dir := filepath.Join(configDir, "codedrill", "drills")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

func LoadDrills() ([]Drill, error) {
	dir, err := GetUserDrillsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var drills []Drill
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		drillPath := filepath.Join(dir, entry.Name())
		manifestFile := filepath.Join(drillPath, "drill.json")
		data, err := os.ReadFile(manifestFile)
		if err != nil {
			continue
		}

		var d Drill
		if err := json.Unmarshal(data, &d); err != nil {
			continue
		}
		if d.ID == "" {
			d.ID = entry.Name()
		}
		d.Dir = drillPath
		drills = append(drills, d)
	}

	return drills, nil
}

func SaveDrill(d Drill) error {
	baseDir, err := GetUserDrillsDir()
	if err != nil {
		return err
	}

	drillDir := filepath.Join(baseDir, d.ID)
	if err := os.MkdirAll(drillDir, 0755); err != nil {
		return fmt.Errorf("failed to create drill dir: %w", err)
	}

	d.Dir = drillDir
	manifestData, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal drill: %w", err)
	}

	manifestPath := filepath.Join(drillDir, "drill.json")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write drill.json: %w", err)
	}

	if d.EntryFile != "" && d.StarterCode != "" {
		starterPath := filepath.Join(drillDir, d.EntryFile)
		if _, err := os.Stat(starterPath); os.IsNotExist(err) {
			_ = os.WriteFile(starterPath, []byte(d.StarterCode), 0644)
		}
	}

	return nil
}
