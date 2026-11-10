package drill

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/luvcie/codedrill/internal/exercises"
	"github.com/luvcie/codedrill/internal/storage"
)

//go:embed presets.json
var presetsJSON []byte

type Drill struct {
	ID                   string            `json:"id"`
	Title                string            `json:"title"`
	Category             string            `json:"category"`
	Language             string            `json:"language"`
	Description          string            `json:"description"`
	Difficulty           string            `json:"difficulty,omitempty"`
	EntryFile            string            `json:"entry_file"`
	StarterCode          string            `json:"starter_code"` // Legacy import/export data; not used to initialize workspaces.
	TestCommand          string            `json:"test_command"`
	AutoSubmit           bool              `json:"auto_submit"`
	ExtraFiles           map[string]string `json:"extra_files,omitempty"`
	StudentFiles         []string          `json:"student_files,omitempty"`
	OptionalStudentFiles []string          `json:"optional_student_files,omitempty"`

	IsPreset bool   `json:"-"`
	Dir      string `json:"-"`
}

// CategorySeparator divides a category into a parent and a subcategory. A
// drill whose category is "42/Exam 05" belongs to the 42 tab and is listed in
// a collapsible "Exam 05" group there.
const CategorySeparator = "/"

func CategoryOf(d Drill) string {
	if c := NormalizeCategory(d.Category); c != "" {
		return c
	}
	return "Custom"
}

// NormalizeCategory cleans up a category name. It maps the old "42 Exam" and
// "42 Exam NN" names onto the current 42 category, keeping the exam part as a
// subcategory, and trims the parts of a "parent/subcategory" path.
func NormalizeCategory(category string) string {
	c := strings.TrimSpace(category)
	if c == "" {
		return ""
	}
	if i := strings.Index(c, CategorySeparator); i >= 0 {
		parent := strings.TrimSpace(c[:i])
		sub := strings.TrimSpace(c[i+1:])
		switch {
		case parent == "":
			return sub
		case sub == "":
			return parent
		default:
			return parent + CategorySeparator + sub
		}
	}
	const legacy = "42 Exam"
	if len(c) > len(legacy) && strings.EqualFold(c[:len(legacy)], legacy) && (c[len(legacy)] == ' ' || c[len(legacy)] == '\t') {
		return "42/Exam " + strings.TrimSpace(c[len(legacy):])
	}
	if strings.EqualFold(c, legacy) {
		return "42"
	}
	return c
}

// ParentCategory is the tab a category belongs to: "42" for "42/Exam 05".
func ParentCategory(category string) string {
	c := NormalizeCategory(category)
	if i := strings.Index(c, CategorySeparator); i >= 0 {
		return strings.TrimSpace(c[:i])
	}
	return c
}

// ParentCategoryOf is ParentCategory with the empty-category fallback.
func ParentCategoryOf(d Drill) string {
	return ParentCategory(CategoryOf(d))
}

// SubcategoryOf returns the part after the separator, or "" when the category
// has no subcategory.
func SubcategoryOf(d Drill) string {
	c := CategoryOf(d)
	if i := strings.Index(c, CategorySeparator); i >= 0 {
		return strings.TrimSpace(c[i+1:])
	}
	return ""
}

func SortCategories(cats []string) {
	slices.SortStableFunc(cats, func(a, b string) int {
		if ra, rb := categoryRank(a), categoryRank(b); ra != rb {
			return ra - rb
		}
		return strings.Compare(strings.ToUpper(a), strings.ToUpper(b))
	})
}

func categoryRank(c string) int {
	switch {
	case isCustomCategory(c):
		return 2
	case isLastCategory(c):
		return 1
	default:
		return 0
	}
}

func isLastCategory(c string) bool {
	return strings.EqualFold(ParentCategory(c), "42")
}

func isCustomCategory(c string) bool {
	return strings.EqualFold(ParentCategory(c), "Custom")
}

func Matches(d Drill, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" || strings.EqualFold(filter, "ALL") {
		return true
	}
	cat := CategoryOf(d)
	return strings.EqualFold(cat, filter) ||
		strings.EqualFold(ParentCategory(cat), filter) ||
		strings.EqualFold(d.Language, filter) ||
		strings.EqualFold(d.Difficulty, filter) ||
		(len(filter) >= 3 && strings.Contains(strings.ToLower(cat), strings.ToLower(filter)))
}

func Filter(drills []Drill, filter string) []Drill {
	var out []Drill
	for _, d := range drills {
		if Matches(d, filter) {
			out = append(out, d)
		}
	}
	return out
}

func Find(drills []Drill, idOrTitle string) *Drill {
	for i, d := range drills {
		if strings.EqualFold(d.ID, idOrTitle) {
			return &drills[i]
		}
	}
	for i, d := range drills {
		if strings.EqualFold(d.Title, idOrTitle) {
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

// LoadBuiltins reads only the exercises shipped in this executable.
func LoadBuiltins() ([]Drill, error) {
	var drills []Drill
	if err := json.Unmarshal(presetsJSON, &drills); err != nil {
		return nil, fmt.Errorf("built-in drills: %w", err)
	}
	entries, err := fs.ReadDir(exercises.Files, "exam05")
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		root := "exam05/" + entry.Name()
		data, err := exercises.Files.ReadFile(root + "/drill.json")
		if err != nil {
			return nil, err
		}
		var d Drill
		if err := json.Unmarshal(data, &d); err != nil {
			return nil, fmt.Errorf("%s: %w", root, err)
		}
		statement, err := exercises.Files.ReadFile(root + "/statement.md")
		if err != nil {
			return nil, err
		}
		d.Description = string(statement)
		d.ExtraFiles = make(map[string]string)
		if err := fs.WalkDir(exercises.Files, root+"/resources", func(path string, e fs.DirEntry, err error) error {
			if err != nil || e.IsDir() {
				return err
			}
			content, err := exercises.Files.ReadFile(path)
			if err == nil {
				d.ExtraFiles[strings.TrimPrefix(path, root+"/resources/")] = string(content)
			}
			return err
		}); err != nil {
			return nil, err
		}
		drills = append(drills, d)
	}
	for i := range drills {
		drills[i].IsPreset = true
		drills[i].Category = NormalizeCategory(drills[i].Category)
	}
	return drills, nil
}

func LoadDrills() ([]Drill, error) {
	drills, err := LoadBuiltins()
	if err != nil {
		return nil, err
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
	d.Category = NormalizeCategory(d.Category)
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

func UniqueID(drills []Drill, title string) string {
	base := SanitizeDrillID(title)
	id := base
	for n := 2; slices.ContainsFunc(drills, func(d Drill) bool { return d.ID == id }); n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

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
	if d.EntryFile != "" || len(d.StudentFiles)+len(d.OptionalStudentFiles) > 0 {
		if _, _, err := d.StudentPaths(); err != nil {
			return err
		}
	}
	d.Category = NormalizeCategory(d.Category)

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
	d.Category = NormalizeCategory(d.Category)
	if err := SaveDrill(d); err != nil {
		return nil, err
	}
	return &d, nil
}
