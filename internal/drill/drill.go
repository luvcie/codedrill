package drill

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/luvcie/codedrill/internal/storage"
)

//go:embed presets.json
var presetsJSON []byte

type Drill struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Category    string            `json:"category"`
	Language    string            `json:"language"`
	Description string            `json:"description"`
	Difficulty  string            `json:"difficulty,omitempty"`
	EntryFile   string            `json:"entry_file"`
	StarterCode string            `json:"starter_code"`
	TestCommand string            `json:"test_command"`
	AutoSubmit  bool              `json:"auto_submit"`
	ExtraFiles  map[string]string `json:"extra_files,omitempty"`

	IsPreset bool   `json:"-"`
	Dir      string `json:"-"`
}

// CategoryOf returns the drill's category, or "Custom" when it has none.
func CategoryOf(d Drill) string {
	if c := strings.TrimSpace(d.Category); c != "" {
		return c
	}
	return "Custom"
}

// Matches reports whether a drill belongs to a category, language or
// difficulty filter. Longer filters also match part of a category name.
func Matches(d Drill, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" || strings.EqualFold(filter, "ALL") {
		return true
	}
	cat := CategoryOf(d)
	return strings.EqualFold(cat, filter) ||
		strings.EqualFold(d.Language, filter) ||
		strings.EqualFold(d.Difficulty, filter) ||
		(len(filter) >= 3 && strings.Contains(strings.ToLower(cat), strings.ToLower(filter)))
}

// Filter returns the drills matching filter (see Matches).
func Filter(drills []Drill, filter string) []Drill {
	var out []Drill
	for _, d := range drills {
		if Matches(d, filter) {
			out = append(out, d)
		}
	}
	return out
}

// Find looks a drill up by ID or title, ignoring case.
func Find(drills []Drill, idOrTitle string) *Drill {
	for i, d := range drills {
		if strings.EqualFold(d.ID, idOrTitle) || strings.EqualFold(d.Title, idOrTitle) {
			return &drills[i]
		}
	}
	return nil
}

func GetUserDrillsDir() (string, error) {
	base, err := storage.ConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "drills")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// LoadDrills returns the built-in drills followed by the user's own; a user
// drill with a built-in's ID replaces it. Hidden built-ins are left out.
// Manifests that cannot be read are skipped and reported in the error, which
// is returned alongside every drill that did load.
func LoadDrills() ([]Drill, error) {
	var drills []Drill
	if err := json.Unmarshal(presetsJSON, &drills); err != nil {
		return nil, fmt.Errorf("built-in drills: %w", err)
	}
	for i := range drills {
		drills[i].IsPreset = true
	}

	dir, err := GetUserDrillsDir()
	if err != nil {
		return drills, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return drills, err
	}

	var problems []error
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		d, err := loadManifest(filepath.Join(dir, entry.Name()))
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if d.ID == "" {
			d.ID = entry.Name()
		}
		if i := slices.IndexFunc(drills, func(e Drill) bool { return e.ID == d.ID }); i >= 0 {
			drills[i] = d
		} else {
			drills = append(drills, d)
		}
	}

	hidden, err := loadHidden()
	if err != nil {
		problems = append(problems, err)
	}
	drills = slices.DeleteFunc(drills, func(d Drill) bool { return d.IsPreset && hidden[d.ID] })
	return drills, errors.Join(problems...)
}

func loadManifest(dir string) (Drill, error) {
	var d Drill
	data, err := os.ReadFile(filepath.Join(dir, "drill.json"))
	if errors.Is(err, os.ErrNotExist) {
		return d, fmt.Errorf("%s has no drill.json", dir)
	}
	if err != nil {
		return d, err
	}
	if err := json.Unmarshal(data, &d); err != nil {
		return d, fmt.Errorf("%s/drill.json: %w", dir, err)
	}
	d.Dir = dir
	// Drills saved before the rename still say "42 Exam".
	if strings.EqualFold(strings.TrimSpace(d.Category), "42 Exam") {
		d.Category = "42"
	}
	return d, nil
}

func hiddenPath() (string, error) {
	dir, err := storage.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hidden.json"), nil
}

// loadHidden returns the IDs of built-in drills the user removed from the list.
func loadHidden() (map[string]bool, error) {
	hidden := make(map[string]bool)
	path, err := hiddenPath()
	if err != nil {
		return hidden, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return hidden, nil
	}
	if err != nil {
		return hidden, err
	}
	var ids []string
	if err := json.Unmarshal(data, &ids); err != nil {
		return hidden, fmt.Errorf("%s: %w", path, err)
	}
	for _, id := range ids {
		hidden[id] = true
	}
	return hidden, nil
}

func saveHidden(hidden map[string]bool) error {
	path, err := hiddenPath()
	if err != nil {
		return err
	}
	ids := HiddenIDs(hidden)
	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return err
	}
	return storage.AtomicWriteFile(path, data, 0644)
}

// HiddenIDs returns the keys of a hidden set, sorted.
func HiddenIDs(hidden map[string]bool) []string {
	ids := make([]string, 0, len(hidden))
	for id := range hidden {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	return ids
}

// HiddenDrillIDs lists hidden built-in drills, sorted.
func HiddenDrillIDs() ([]string, error) {
	hidden, err := loadHidden()
	return HiddenIDs(hidden), err
}

// RestoreDrill unhides a built-in drill.
func RestoreDrill(id string) error {
	hidden, err := loadHidden()
	if err != nil {
		return err
	}
	for hid := range hidden {
		if strings.EqualFold(hid, id) {
			delete(hidden, hid)
			return saveHidden(hidden)
		}
	}
	return fmt.Errorf("drill %q is not hidden", id)
}

// RestoreAll unhides every built-in drill.
func RestoreAll() error {
	return saveHidden(map[string]bool{})
}

// RemoveDrill deletes a custom drill or hides a built-in one, and returns it.
// hidden reports which of the two happened.
func RemoveDrill(idOrTitle string) (removed Drill, hidden bool, err error) {
	drills, err := LoadDrills()
	if err != nil && drills == nil {
		return Drill{}, false, err
	}
	d := Find(drills, idOrTitle)
	if d == nil {
		return Drill{}, false, fmt.Errorf("drill %q not found", idOrTitle)
	}
	if d.IsPreset {
		set, err := loadHidden()
		if err != nil {
			return *d, true, err
		}
		set[d.ID] = true
		return *d, true, saveHidden(set)
	}

	// Only ever delete a directory directly inside the drills directory.
	base, err := GetUserDrillsDir()
	if err != nil {
		return *d, false, err
	}
	if filepath.Dir(d.Dir) != base {
		return *d, false, fmt.Errorf("refusing to delete %s: not inside %s", d.Dir, base)
	}
	if err := os.RemoveAll(d.Dir); err != nil {
		return *d, false, fmt.Errorf("delete drill directory: %w", err)
	}
	return *d, false, nil
}

func SanitizeDrillID(titleOrID string) string {
	safeID := strings.Map(func(r rune) rune {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		case r == ' ' || r == '/' || r == '\\' || r == ':' || r == '#' || r == '.':
			return '-'
		}
		return -1
	}, titleOrID)

	safeID = strings.Trim(safeID, "-_")
	for strings.Contains(safeID, "--") {
		safeID = strings.ReplaceAll(safeID, "--", "-")
	}
	if safeID == "" {
		safeID = fmt.Sprintf("drill-%d", time.Now().Unix())
	}
	return safeID
}

// UniqueID turns a title into an ID that no existing drill uses.
func UniqueID(drills []Drill, title string) string {
	base := SanitizeDrillID(title)
	id := base
	for n := 2; slices.ContainsFunc(drills, func(d Drill) bool { return d.ID == id }); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

// SaveDrill writes a user drill's manifest, replacing any drill with that ID.
func SaveDrill(d Drill) error {
	baseDir, err := GetUserDrillsDir()
	if err != nil {
		return err
	}
	if d.ID == "" {
		d.ID = d.Title
	}
	d.ID = SanitizeDrillID(d.ID)
	if d.EntryFile != "" && !filepath.IsLocal(d.EntryFile) {
		return fmt.Errorf("entry file %q must be a relative path inside the drill", d.EntryFile)
	}

	drillDir := filepath.Join(baseDir, d.ID)
	if err := os.MkdirAll(drillDir, 0755); err != nil {
		return fmt.Errorf("create drill directory: %w", err)
	}
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return storage.AtomicWriteFile(filepath.Join(drillDir, "drill.json"), data, 0644)
}

// GetDailyDrill selects the same drill for a given date and pool.
func GetDailyDrill(drills []Drill, t time.Time) *Drill {
	if len(drills) == 0 {
		return nil
	}
	sorted := slices.Clone(drills)
	slices.SortFunc(sorted, func(a, b Drill) int { return strings.Compare(a.ID, b.ID) })

	sum := 0
	for _, r := range t.Format("2006-01-02") {
		sum = sum*31 + int(r)
	}
	return &sorted[sum%len(sorted)]
}

func ExportDrill(idOrTitle string) ([]byte, error) {
	drills, err := LoadDrills()
	if err != nil && drills == nil {
		return nil, err
	}
	d := Find(drills, idOrTitle)
	if d == nil {
		return nil, fmt.Errorf("drill %q not found", idOrTitle)
	}
	return json.MarshalIndent(d, "", "  ")
}

func ImportDrill(data []byte) (*Drill, error) {
	var d Drill
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, fmt.Errorf("invalid drill JSON: %w", err)
	}
	if strings.TrimSpace(d.Title) == "" {
		return nil, errors.New("drill title cannot be empty")
	}
	if strings.TrimSpace(d.EntryFile) == "" {
		return nil, errors.New("drill entry_file cannot be empty")
	}
	if d.ID == "" {
		d.ID = d.Title
	}
	d.ID = SanitizeDrillID(d.ID)
	if err := SaveDrill(d); err != nil {
		return nil, err
	}
	return &d, nil
}
