package drill

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestStudentPathsDefaultsAndValidation(t *testing.T) {
	d := Drill{EntryFile: "src/main.cpp", StudentFiles: []string{"src/main.cpp", "src/part.cpp"}, OptionalStudentFiles: []string{"include/extra.hpp"}}
	required, optional, err := d.StudentPaths()
	if err != nil || !reflect.DeepEqual(required, []string{"src/main.cpp", "src/part.cpp"}) || !reflect.DeepEqual(optional, []string{"include/extra.hpp"}) {
		t.Fatalf("ownership paths: %v %v %v", required, optional, err)
	}
	for _, d := range []Drill{
		{EntryFile: "../outside"},
		{EntryFile: "main.c", StudentFiles: []string{"."}},
		{EntryFile: "main.c", OptionalStudentFiles: []string{"main.c"}},
		{EntryFile: "src/main.c", StudentFiles: []string{"src"}},
		{EntryFile: "main.c", OptionalStudentFiles: []string{"/absolute.h"}},
	} {
		if _, _, err := d.StudentPaths(); err == nil {
			t.Fatalf("invalid ownership accepted: %+v", d)
		}
	}
}

func TestOwnershipImportExportCompatibility(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	for _, input := range []string{
		`{"id":"legacy","title":"Legacy","entry_file":"main.c","starter_code":"legacy source","extra_files":{"test.c":"provided test"}}`,
		`{"id":"owned","title":"Owned","entry_file":"main.cpp","starter_code":"legacy source","student_files":["part.cpp","part.hpp"],"optional_student_files":["extra.hpp"],"extra_files":{"main.cpp":"legacy entry","test.cpp":"provided test"}}`,
	} {
		imported, err := ImportDrill([]byte(input))
		if err != nil {
			t.Fatal(err)
		}
		data, err := ExportDrill(imported.ID)
		if err != nil {
			t.Fatal(err)
		}
		var roundtrip Drill
		if err := json.Unmarshal(data, &roundtrip); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(roundtrip, *imported) {
			t.Fatalf("export lost legacy data, resources or ownership: %+v != %+v", roundtrip, *imported)
		}
		required, _, err := roundtrip.StudentPaths()
		if err != nil || required[0] != imported.EntryFile {
			t.Fatalf("entry is not implicitly student-owned: %v %v", required, err)
		}
	}
}
