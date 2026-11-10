package mode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingPreferencesDefaultsToStudy(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := Load("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	if got != Study() {
		t.Fatalf("got %+v, want Study", got)
	}
}

func TestSaveAndLoadPerDrill(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Save("first", Recall()); err != nil {
		t.Fatal(err)
	}
	if err := Save("second", Exam()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id   string
		want Options
	}{
		{"first", Recall()},
		{"second", Exam()},
		{"third", Study()},
	} {
		got, err := Load(tc.id)
		if err != nil {
			t.Fatalf("%s: %v", tc.id, err)
		}
		if got != tc.want {
			t.Fatalf("%s is %+v, want %+v", tc.id, got, tc.want)
		}
	}

	path, err := PrefsPath()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lower := strings.ToLower(string(data))
	for _, name := range []string{"study", "copy", "recall", "exam", "preset"} {
		if strings.Contains(lower, name) {
			t.Fatalf("preferences persist the preset name %q:\n%s", name, data)
		}
	}
}

func TestSaveStoresGivenBooleansAndLoadNormalizes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	odd := Options{ShowReference: true, RequireExact: true}
	if err := Save("odd", odd); err != nil {
		t.Fatal(err)
	}
	path, _ := PrefsPath()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`"show_reference": true`, `"require_exact": true`} {
		if !strings.Contains(string(data), want) {
			t.Fatalf("saved file lacks %q:\n%s", want, data)
		}
	}
	got, err := Load("odd")
	if err != nil {
		t.Fatal(err)
	}
	want := Options{RequireTests: true}
	if got != want {
		t.Fatalf("unusable saved combination loaded as %+v, want %+v", got, want)
	}
}

func TestLoadNormalizesSavedCombinations(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := PrefsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data := `{
  "no-gates": {"check_reference": true, "show_reference": true, "require_exact": false, "require_tests": false},
  "no-show": {"check_reference": true, "show_reference": false, "require_exact": false, "require_tests": false},
  "inconsistent": {"check_reference": false, "show_reference": true, "require_exact": true, "require_tests": true},
  "reference-off": {"check_reference": false, "show_reference": true, "require_exact": true, "require_tests": false}
}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		id   string
		want Options
	}{
		{"no-gates", Study()},
		{"no-show", Options{CheckReference: true, RequireTests: true}},
		{"inconsistent", Exam()},
		{"reference-off", Exam()},
	} {
		got, err := Load(tc.id)
		if err != nil {
			t.Fatalf("%s: %v", tc.id, err)
		}
		if got != tc.want {
			t.Fatalf("%s is %+v, want %+v", tc.id, got, tc.want)
		}
	}
}

func TestLoadEnablesOnlyTestsWhenNoGate(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := PrefsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	data := `{"no-gate": {"check_reference": true, "show_reference": false, "require_exact": false, "require_tests": false}}`
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
	got, err := Load("no-gate")
	if err != nil {
		t.Fatal(err)
	}
	want := Options{CheckReference: true, RequireTests: true}
	if got != want {
		t.Fatalf("loaded %+v, want %+v", got, want)
	}
	if got == Study() {
		t.Fatal("loader fell back to Study instead of enabling only tests")
	}
}

func TestCorruptPreferencesAreNotOverwritten(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	path, err := PrefsPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load("ft_strlen"); err == nil {
		t.Fatal("loading corrupt preferences did not fail")
	}
	if err := Save("ft_strlen", Study()); err == nil {
		t.Fatal("saving over corrupt preferences did not fail")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "{" {
		t.Fatalf("corrupt file was overwritten: %q, %v", data, err)
	}
}

func TestSaveRejectsEmptyDrillID(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if err := Save("  ", Study()); err == nil {
		t.Fatal("empty drill id accepted")
	}
}

func TestPreferencesConfigErrors(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/dev/null")
	if _, err := Load("ft_strlen"); err == nil {
		t.Fatal("load with an unusable config directory did not fail")
	}
	if err := Save("ft_strlen", Study()); err == nil {
		t.Fatal("save with an unusable config directory did not fail")
	}
}
