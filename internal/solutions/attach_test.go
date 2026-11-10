package solutions

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestAttachOneFile(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := t.TempDir()
	writeFile(t, src, "answer.c", "int answer(void) { return 1; }\n")

	att, err := Attach("qa", []string{filepath.Join(src, "answer.c")})
	if err != nil {
		t.Fatal(err)
	}
	want, _ := OverlayDir("qa")
	if att.Dir != want || !slices.Equal(att.Files, []string{"answer.c"}) {
		t.Fatalf("attachment is %+v, want dir %s", att, want)
	}
	data, err := os.ReadFile(filepath.Join(want, "answer.c"))
	if err != nil || !strings.Contains(string(data), "return 1") {
		t.Fatalf("attached file is %q, %v", data, err)
	}
}

func TestAttachMultipleFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := t.TempDir()
	writeFile(t, src, "main.c", "int main(void) { return 0; }\n")
	writeFile(t, src, "defs.h", "#define N 1\n")

	att, err := Attach("qa", []string{filepath.Join(src, "main.c"), filepath.Join(src, "defs.h")})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(att.Files, []string{"defs.h", "main.c"}) {
		t.Fatalf("attached files are %v", att.Files)
	}
}

func TestAttachDirectoryKeepsNestedPaths(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := t.TempDir()
	writeFile(t, src, "src/ft_strlen.c", "int ft_strlen(char *s) { return 0; }\n")
	writeFile(t, src, "include/ft.h", "int ft_strlen(char *s);\n")
	writeFile(t, src, "README.md", "not a solution\n")

	att, err := Attach("custom", []string{src})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(att.Files, []string{"include/ft.h", "src/ft_strlen.c"}) {
		t.Fatalf("attached files are %v", att.Files)
	}
	if _, err := os.Stat(filepath.Join(att.Dir, "README.md")); !os.IsNotExist(err) {
		t.Fatalf("unsupported file was attached: %v", err)
	}
	if _, err := os.Stat(filepath.Join(att.Dir, "src", "ft_strlen.c")); err != nil {
		t.Fatalf("nested file missing: %v", err)
	}
}

func TestAttachOverwriteKeepsUnrelatedFiles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	src := t.TempDir()
	writeFile(t, src, "a.c", "one\n")
	writeFile(t, src, "b.c", "keep\n")
	if _, err := Attach("qa", []string{filepath.Join(src, "a.c"), filepath.Join(src, "b.c")}); err != nil {
		t.Fatal(err)
	}

	writeFile(t, src, "a.c", "two\n")
	att, err := Attach("qa", []string{filepath.Join(src, "a.c")})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(att.Files, []string{"a.c"}) {
		t.Fatalf("second attachment is %v", att.Files)
	}
	overlay, _ := OverlayDir("qa")
	if data, _ := os.ReadFile(filepath.Join(overlay, "a.c")); string(data) != "two\n" {
		t.Fatalf("a.c was not overwritten: %q", data)
	}
	if data, _ := os.ReadFile(filepath.Join(overlay, "b.c")); string(data) != "keep\n" {
		t.Fatalf("b.c was changed or deleted: %q", data)
	}
}

func TestAttachRejectsUnsafeInput(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	outside := t.TempDir()
	src := t.TempDir()
	writeFile(t, src, "answer.c", "int answer;\n")
	writeFile(t, outside, "outside.c", "int outside;\n")
	if err := os.Symlink(filepath.Join(outside, "outside.c"), filepath.Join(src, "link.c")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	text := filepath.Join(src, "notes.txt")
	writeFile(t, src, "notes.txt", "notes\n")

	cases := []struct {
		name    string
		drillID string
		path    string
		want    string
	}{
		{"drill id", "../qa", filepath.Join(src, "answer.c"), "invalid drill id"},
		{"symlink file", "qa", filepath.Join(src, "link.c"), "symbolic link"},
		{"symlink in directory", "qa", src, "symbolic link"},
		{"unsupported file", "qa", text, "unsupported solution file"},
		{"missing path", "qa", filepath.Join(src, "missing.c"), "no such file"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Attach(tc.drillID, []string{tc.path}); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error is %v, want one containing %q", err, tc.want)
			}
		})
	}
}

func TestAttachRejectsSymlinkedDestination(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	overlay, err := OverlayDir("qa")
	if err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.MkdirAll(overlay, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(overlay, "lib")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	src := t.TempDir()
	writeFile(t, src, "lib/util.c", "int util;\n")
	if _, err := Attach("qa", []string{src}); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("error is %v, want a symlink error", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "util.c")); !os.IsNotExist(err) {
		t.Fatalf("attachment escaped the overlay: %v", err)
	}
}
