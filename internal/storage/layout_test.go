package storage

import (
	"os"
	"reflect"
	"testing"
)

func TestLayoutRoundTrip(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	empty, err := LoadLayout()
	if err != nil {
		t.Fatal(err)
	}
	if len(empty.Categories) != 0 {
		t.Fatalf("missing layout loaded %v", empty.Categories)
	}

	want := MenuLayout{Categories: map[string][]LayoutBlock{
		"42": {
			{Group: "Exam 05", Drills: []string{"exam05a", "exam05b"}},
			{Drill: "ft_strlen"},
		},
	}}
	if err := SaveLayout(want); err != nil {
		t.Fatal(err)
	}
	got, err := LoadLayout()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("layout = %+v, want %+v", got, want)
	}
}

func TestCorruptLayoutIsReported(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := GetLayoutPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{oops"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadLayout(); err == nil {
		t.Fatal("a corrupt layout should be reported")
	}
}
