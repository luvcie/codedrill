package solutions

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExam05ReferenceOverride(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := OverlayDir("exam05-vect2")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dir, "private.cpp", "private reference")
	source, err := Resolve("exam05-vect2", "")
	if err != nil || source == nil || !source.Local || len(source.Files) != 1 {
		t.Fatalf("local reference did not win outright: %+v, %v", source, err)
	}
	data, err := source.Files[0].Read()
	if err != nil || string(data) != "private reference" {
		t.Fatalf("local reference changed: %q, %v", data, err)
	}
	// A custom drill with the same ID does not inherit an incompatible reference.
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	custom := t.TempDir()
	source, err = Resolve("exam05-vect2", custom)
	if err != nil || source != nil {
		t.Fatalf("custom override inherited the built-in reference: %+v, %v", source, err)
	}
	writeFile(t, filepath.Join(custom, "solution"), "custom.cpp", "custom bundled reference")
	source, err = Resolve("exam05-vect2", custom)
	if err != nil || source == nil || len(source.Files) != 1 || source.Files[0].Rel != "custom.cpp" {
		t.Fatalf("custom bundled reference did not win: %+v, %v", source, err)
	}
}
