package drill

import "testing"

func hasDrill(t *testing.T, id string) bool {
	t.Helper()
	drills, err := LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range drills {
		if d.ID == id {
			return true
		}
	}
	return false
}

func TestRemoveCustomDrillDeletesIt(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SaveDrill(Drill{ID: "mine", Title: "Mine", Category: "Rust", EntryFile: "a.rs"}); err != nil {
		t.Fatal(err)
	}
	_, hidden, err := RemoveDrill("mine")
	if err != nil || hidden {
		t.Fatalf("RemoveDrill = %v, %v; want deleted", hidden, err)
	}
	if hasDrill(t, "mine") {
		t.Fatal("custom drill still listed after removal")
	}
}

func TestRemovePresetHidesAndRestores(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	_, hidden, err := RemoveDrill("ft_strlen")
	if err != nil || !hidden {
		t.Fatalf("RemoveDrill = %v, %v; want hidden", hidden, err)
	}
	if hasDrill(t, "ft_strlen") {
		t.Fatal("hidden preset still listed")
	}
	if ids, _ := HiddenDrillIDs(); len(ids) != 1 || ids[0] != "ft_strlen" {
		t.Fatalf("HiddenDrillIDs = %v", ids)
	}
	if err := RestoreDrill("ft_strlen"); err != nil {
		t.Fatal(err)
	}
	if !hasDrill(t, "ft_strlen") {
		t.Fatal("restored preset missing")
	}
	if err := RestoreDrill("ft_strlen"); err == nil {
		t.Fatal("restoring a drill that is not hidden should fail")
	}

	_, _, _ = RemoveDrill("ft_swap")
	_, _, _ = RemoveDrill("two_sum")
	if err := RestoreAll(); err != nil {
		t.Fatal(err)
	}
	if ids, _ := HiddenDrillIDs(); len(ids) != 0 {
		t.Fatalf("restore all left %v", ids)
	}
}

func TestLegacy42ExamCategoryIsRenamed(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := SaveDrill(Drill{ID: "old", Title: "Old", Category: "42 Exam", EntryFile: "a.c"}); err != nil {
		t.Fatal(err)
	}
	drills, _ := LoadDrills()
	if d := Find(drills, "old"); d == nil || d.Category != "42" {
		t.Fatalf("legacy drill not renamed: %+v", d)
	}
}
