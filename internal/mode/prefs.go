package mode

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luvcie/codedrill/internal/storage"
)

// prefsFile maps drill IDs to their options. Preset names are not stored.
type prefsFile map[string]Options

// PrefsPath is the drill preferences file in the config directory.
func PrefsPath() (string, error) {
	dir, err := storage.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "drill_prefs.json"), nil
}

// Load returns a drill's options. A missing file or drill uses Study. Saved
// combinations are normalized; when neither pass gate is usable, only the
// tests gate is enabled and the other normalized values are kept.
func Load(drillID string) (Options, error) {
	path, err := PrefsPath()
	if err != nil {
		return Options{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Options{}, err
	}
	var prefs prefsFile
	if err := json.Unmarshal(data, &prefs); err != nil {
		return Options{}, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	opts, ok := prefs[drillID]
	if !ok {
		return Default(), nil
	}
	return sanitize(opts), nil
}

// Save stores one drill's options and leaves other drills alone. The four
// booleans are written as given; callers should Validate before saving so an
// unusable selection can be reported to the user.
func Save(drillID string, opts Options) error {
	if strings.TrimSpace(drillID) == "" {
		return errors.New("missing drill id")
	}
	path, err := PrefsPath()
	if err != nil {
		return err
	}
	prefs := prefsFile{}
	data, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(data, &prefs); err != nil {
			return fmt.Errorf("%s is not valid JSON: %w", path, err)
		}
	case errors.Is(err, os.ErrNotExist):
	default:
		return err
	}
	if prefs == nil {
		prefs = prefsFile{}
	}
	prefs[drillID] = opts
	data, err = json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return storage.AtomicWriteFile(path, data, 0644)
}

// sanitize repairs saved options without touching valid gate choices. A saved
// combination with no gate keeps its normalized reference options and gains
// only the tests gate, so loading never enables showing or exact copy.
func sanitize(opts Options) Options {
	opts = opts.Normalize()
	if !opts.RequireTests && !opts.RequireExact {
		opts.RequireTests = true
	}
	return opts
}
