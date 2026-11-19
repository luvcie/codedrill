package mode

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/luvcie/codedrill/internal/solutions"
	"github.com/luvcie/codedrill/internal/storage"
)

type Preferences struct {
	Options
	ReferenceVariant solutions.Variant `json:"reference_variant,omitempty"`
}

// prefsFile maps drill IDs to their preferences. Preset names are not stored.
type prefsFile map[string]Preferences

// PrefsPath is the drill preferences file in the config directory.
func PrefsPath() (string, error) {
	dir, err := storage.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "drill_prefs.json"), nil
}

// Load returns a drill's options. A missing file or drill uses Study. Saved
// Recall selections open in Study. Other combinations are normalized; when
// neither pass gate is usable, only the tests gate is enabled.
func Load(drillID string) (Options, error) {
	prefs, err := LoadPreferences(drillID)
	return prefs.Options, err
}

func LoadPreferences(drillID string) (Preferences, error) {
	path, err := PrefsPath()
	if err != nil {
		return Preferences{}, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Preferences{Options: Default()}, nil
	}
	if err != nil {
		return Preferences{}, err
	}
	var prefs prefsFile
	if err := json.Unmarshal(data, &prefs); err != nil {
		return Preferences{}, fmt.Errorf("%s is not valid JSON: %w", path, err)
	}
	opts, ok := prefs[drillID]
	if !ok {
		return Preferences{Options: Default()}, nil
	}
	opts.Options = sanitize(opts.Options)
	return opts, nil
}

// Save stores one drill's options and leaves other drills alone. The four
// booleans are written as given; callers should Validate before saving so an
// unusable selection can be reported to the user.
func Save(drillID string, opts Options) error {
	prefs, err := LoadPreferences(drillID)
	if err != nil {
		return err
	}
	prefs.Options = opts
	return SavePreferences(drillID, prefs)
}

func SavePreferences(drillID string, selection Preferences) error {
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
	prefs[drillID] = selection
	data, err = json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return storage.AtomicWriteFile(path, data, 0644)
}

// sanitize replaces the retired Recall preset and repairs unusable gates.
func sanitize(opts Options) Options {
	opts = opts.Normalize()
	if opts == Recall() {
		return Study()
	}
	if !opts.RequireTests && !opts.RequireExact {
		opts.RequireTests = true
	}
	return opts
}
