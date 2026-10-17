package drill

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestLoadDrillsPresets(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	drills, err := LoadDrills()
	if err != nil {
		t.Fatalf("failed to load drills: %v", err)
	}

	if len(drills) == 0 {
		t.Fatalf("expected presets to be loaded, got 0 drills")
	}

	for _, d := range drills {
		if !d.IsPreset {
			t.Errorf("expected drill %s to have IsPreset=true", d.ID)
		}
		if d.ID == "" {
			t.Errorf("drill has empty ID")
		}
		if d.Title == "" {
			t.Errorf("drill %s has empty Title", d.ID)
		}
		if d.Category == "" {
			t.Errorf("drill %s has empty Category", d.ID)
		}
		if d.Language == "" {
			t.Errorf("drill %s has empty Language", d.ID)
		}
		if d.EntryFile == "" {
			t.Errorf("drill %s has empty EntryFile", d.ID)
		}
		if d.StarterCode == "" {
			t.Errorf("drill %s has empty StarterCode", d.ID)
		}
		if d.TestCommand == "" {
			t.Errorf("drill %s has empty TestCommand", d.ID)
		}
		diff := strings.ToLower(d.Difficulty)
		if diff != "easy" && diff != "medium" && diff != "hard" {
			t.Errorf("drill %s has invalid difficulty %s", d.ID, d.Difficulty)
		}
	}
}

func TestSaveAndLoadCustomDrill(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	custom := Drill{
		ID:          "custom-drill-1",
		Title:       "Custom Test Drill",
		Category:    "CUSTOM",
		Language:    "c",
		Description: "A custom drill for testing",
		Difficulty:  "easy",
		EntryFile:   "solution.c",
		StarterCode: "int solution(void) { return 42; }\n",
		TestCommand: "echo ok",
	}

	if err := SaveDrill(custom); err != nil {
		t.Fatalf("failed to save custom drill: %v", err)
	}

	drillDir, _ := GetUserDrillsDir()
	if _, err := os.Stat(filepath.Join(drillDir, "custom-drill-1", "drill.json")); err != nil {
		t.Fatalf("manifest not written: %v", err)
	}

	loaded, err := LoadDrills()
	if err != nil {
		t.Fatalf("failed to load drills: %v", err)
	}

	var found *Drill
	for i := range loaded {
		if loaded[i].ID == "custom-drill-1" {
			found = &loaded[i]
			break
		}
	}

	if found == nil {
		t.Fatalf("custom drill not found in loaded list")
	}
	if found.Title != "Custom Test Drill" {
		t.Errorf("expected Title %q, got %q", "Custom Test Drill", found.Title)
	}
	if found.Category != "CUSTOM" {
		t.Errorf("expected Category CUSTOM, got %s", found.Category)
	}
	if found.Difficulty != "easy" {
		t.Errorf("expected Difficulty easy, got %s", found.Difficulty)
	}
	if found.IsPreset {
		t.Errorf("expected custom drill to have IsPreset=false")
	}
}

func TestRemoveDrillErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, _, err := RemoveDrill("ghost-drill-999"); err == nil {
		t.Error("removing a missing drill should fail")
	}
}

// A hand-written manifest can carry an ID that differs from its folder name.
// Removal must delete the folder it was loaded from.
func TestRemoveDrillUsesItsDirectory(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	base, _ := GetUserDrillsDir()
	dir := filepath.Join(base, "two-sum-mine")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"id": "two_sum_mine", "title": "Mine", "entry_file": "a.go"}`
	if err := os.WriteFile(filepath.Join(dir, "drill.json"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if _, hidden, err := RemoveDrill("two_sum_mine"); err != nil || hidden {
		t.Fatalf("RemoveDrill: hidden=%v err=%v", hidden, err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("drill directory still exists: %v", err)
	}
}

func TestBrokenManifestIsReported(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	base, _ := GetUserDrillsDir()
	dir := filepath.Join(base, "broken")
	_ = os.MkdirAll(dir, 0755)
	_ = os.WriteFile(filepath.Join(dir, "drill.json"), []byte("{oops"), 0644)
	drills, err := LoadDrills()
	if err == nil || !strings.Contains(err.Error(), "broken") {
		t.Fatalf("expected the broken manifest to be reported, got %v", err)
	}
	if len(drills) == 0 {
		t.Fatal("a broken user drill should not hide the built-ins")
	}
}

func TestPresetFlagIsNotReadFromManifests(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if _, err := ImportDrill([]byte(`{"id": "sneaky", "title": "Sneaky", "entry_file": "a.c", "is_preset": true}`)); err != nil {
		t.Fatal(err)
	}
	drills, _ := LoadDrills()
	if d := Find(drills, "sneaky"); d == nil || d.IsPreset {
		t.Fatalf("custom drill loaded as preset: %+v", d)
	}
}

func TestUniqueID(t *testing.T) {
	drills := []Drill{{ID: "two-sum"}, {ID: "two-sum-2"}}
	if got := UniqueID(drills, "Two Sum"); got != "two-sum-3" {
		t.Fatalf("UniqueID = %q", got)
	}
	if got := UniqueID(drills, "Other"); got != "other" {
		t.Fatalf("UniqueID = %q", got)
	}
}

func TestMatchesAndFind(t *testing.T) {
	d := Drill{ID: "ft_strlen", Title: "ft_strlen", Category: "42", Language: "c", Difficulty: "easy"}
	for _, f := range []string{"", "ALL", "42", "c", "C", "easy"} {
		if !Matches(d, f) {
			t.Errorf("Matches(%q) = false", f)
		}
	}
	if Matches(d, "go") || Matches(Drill{}, "42") {
		t.Error("unexpected match")
	}
	if !Matches(Drill{}, "custom") {
		t.Error("drills without a category belong to Custom")
	}
	if Find([]Drill{d}, "FT_STRLEN") == nil || Find([]Drill{d}, "nope") != nil {
		t.Error("Find is wrong")
	}
}

func TestSanitizeDrillID(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"ft_strlen", "ft_strlen"},
		{"LeetCode #1: Two Sum!", "leetcode-1-two-sum"},
		{"42 / ft_putchar", "42-ft_putchar"},
		{"../../../etc/passwd", "etc-passwd"},
		{"Hello   World", "hello-world"},
		{"---crazy---dashes---", "crazy-dashes"},
	}

	for _, c := range cases {
		got := SanitizeDrillID(c.input)
		if got != c.expected {
			t.Errorf("SanitizeDrillID(%q) = %q; want %q", c.input, got, c.expected)
		}
	}

	empty := SanitizeDrillID("")
	if !strings.HasPrefix(empty, "drill-") {
		t.Errorf("expected fallback drill ID starting with drill-, got %q", empty)
	}
}

func TestGetDailyDrill(t *testing.T) {
	drills := []Drill{
		{ID: "ft_strlen", Title: "ft_strlen"},
		{ID: "ft_swap", Title: "ft_swap"},
		{ID: "two_sum", Title: "Two Sum"},
	}

	if GetDailyDrill(nil, time.Now()) != nil {
		t.Errorf("expected nil for empty drills list")
	}

	t1 := time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 9, 6, 23, 59, 59, 0, time.UTC)
	d1 := GetDailyDrill(drills, t1)
	d2 := GetDailyDrill(drills, t2)
	if d1 == nil || d2 == nil || d1.ID != d2.ID {
		t.Errorf("expected identical daily drill on same day, got %v vs %v", d1, d2)
	}

	t3 := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	seen := map[string]bool{}
	for day := 0; day < 30; day++ {
		seen[GetDailyDrill(drills, t3.AddDate(0, 0, day)).ID] = true
	}
	if len(seen) < 2 {
		t.Errorf("daily drill never changed over 30 days: %v", seen)
	}
}

func TestExportAndImportDrill(t *testing.T) {
	tempConfig := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempConfig)

	exported, err := ExportDrill("ft_strlen")
	if err != nil {
		t.Fatalf("failed to export ft_strlen: %v", err)
	}
	if !strings.Contains(string(exported), "ft_strlen") {
		t.Errorf("expected exported JSON to contain ft_strlen")
	}

	_, err = ExportDrill("non-existent-xyz")
	if err == nil {
		t.Errorf("expected error exporting non-existent drill")
	}

	newDrillJSON := []byte(`{
		"id": "imported-drill",
		"title": "Imported Drill",
		"language": "python",
		"entry_file": "imported.py",
		"starter_code": "def run(): pass\n"
	}`)

	imported, err := ImportDrill(newDrillJSON)
	if err != nil {
		t.Fatalf("failed to import drill: %v", err)
	}
	if imported.ID != "imported-drill" {
		t.Errorf("expected ID imported-drill, got %s", imported.ID)
	}

	_, err = ImportDrill([]byte(`{not valid json`))
	if err == nil {
		t.Errorf("expected error importing invalid JSON")
	}

	_, err = ImportDrill([]byte(`{"entry_file": "test.c"}`))
	if err == nil {
		t.Errorf("expected error importing drill without title")
	}
}

func TestSortCategoriesPutsFortyTwoLast(t *testing.T) {
	cats := []string{"LeetCode", "42", "Algorithms", "Custom", "Rust"}
	SortCategories(cats)
	want := []string{"Algorithms", "LeetCode", "Rust", "42", "Custom"}
	if !slices.Equal(cats, want) {
		t.Fatalf("SortCategories = %q, want %q", cats, want)
	}

	// Names compare ignoring case, and 42 stays last even when it is first.
	cats = []string{"42", "rust", "Algorithms", "LeetCode"}
	SortCategories(cats)
	if want := []string{"Algorithms", "LeetCode", "rust", "42"}; !slices.Equal(cats, want) {
		t.Fatalf("SortCategories = %q, want %q", cats, want)
	}
}

func TestSortCategoriesPutsCustomLast(t *testing.T) {
	// Custom is last whatever its case, and after 42. "Customs" is an ordinary
	// category name and sorts alphabetically.
	cats := []string{"custom", "42", "Zig", "Customs", "Algorithms"}
	SortCategories(cats)
	want := []string{"Algorithms", "Customs", "Zig", "42", "custom"}
	if !slices.Equal(cats, want) {
		t.Fatalf("SortCategories = %q, want %q", cats, want)
	}

	// Without 42, Custom still goes after the other categories.
	cats = []string{"Custom", "Rust", "Algorithms"}
	SortCategories(cats)
	if want := []string{"Algorithms", "Rust", "Custom"}; !slices.Equal(cats, want) {
		t.Fatalf("SortCategories = %q, want %q", cats, want)
	}

	// Without Custom, 42 is still last.
	cats = []string{"42", "Rust", "Algorithms"}
	SortCategories(cats)
	if want := []string{"Algorithms", "Rust", "42"}; !slices.Equal(cats, want) {
		t.Fatalf("SortCategories = %q, want %q", cats, want)
	}
}
