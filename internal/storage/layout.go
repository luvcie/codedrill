package storage

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// MenuLayout stores the order rows were arranged in, per category. A group
// block is a subcategory header with its drills; a drill block is one drill
// directly under the category.
type MenuLayout struct {
	Categories map[string][]LayoutBlock `json:"categories,omitempty"`
}

// LayoutBlock is either a subcategory group or one loose drill. Only one of
// Group and Drill is set.
type LayoutBlock struct {
	Group  string   `json:"group,omitempty"`
	Drills []string `json:"drills,omitempty"`
	Drill  string   `json:"drill,omitempty"`
}

func GetLayoutPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "layout.json"), nil
}

// LoadLayout reads the menu arrangement. A missing file is an empty layout.
func LoadLayout() (MenuLayout, error) {
	layout := MenuLayout{Categories: make(map[string][]LayoutBlock)}
	path, err := GetLayoutPath()
	if err != nil {
		return layout, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return layout, nil
	}
	if err != nil {
		return layout, err
	}
	if err := json.Unmarshal(data, &layout); err != nil {
		return MenuLayout{Categories: make(map[string][]LayoutBlock)},
			fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	if layout.Categories == nil {
		layout.Categories = make(map[string][]LayoutBlock)
	}
	return layout, nil
}

func SaveLayout(layout MenuLayout) error {
	path, err := GetLayoutPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(layout, "", "  ")
	if err != nil {
		return err
	}
	return AtomicWriteFile(path, data, 0644)
}
