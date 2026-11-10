package drill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExam05LocalOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	local := Drill{ID: "exam05-vect2", Title: "My vector", Language: "cpp", EntryFile: "mine.cpp", Description: "private", TestCommand: "true"}
	if err := SaveDrill(local); err != nil {
		t.Fatal(err)
	}
	dir, err := GetUserDrillsDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, local.ID, "drill.json")
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	drills, err := LoadDrills()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, d := range drills {
		if d.ID == local.ID {
			count++
			if d.IsPreset || d.Title != local.Title || d.EntryFile != local.EntryFile || d.Description != "private" {
				t.Fatalf("local drill was replaced: %+v", d)
			}
		}
	}
	if count != 1 {
		t.Fatalf("got %d copies of local override", count)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(before) != string(after) {
		t.Fatalf("loading rewrote the local drill: %v", err)
	}
}

func TestFindPrefersIDOverTitle(t *testing.T) {
	drills := []Drill{{ID: "exam05-vect2", Title: "vect2"}, {ID: "vect2", Title: "Custom vector"}}
	if d := Find(drills, "VECT2"); d == nil || d.ID != "vect2" {
		t.Fatalf("built-in title shadows a local ID: %+v", d)
	}
}
