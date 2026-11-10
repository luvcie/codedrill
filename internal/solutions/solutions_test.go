package solutions

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func rels(files []File) []string {
	out := make([]string, len(files))
	for i, f := range files {
		out[i] = f.Rel
	}
	return out
}

func TestOverlayDirIsUnderConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := OverlayDir("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(dir) != "ft_strlen" || filepath.Base(filepath.Dir(dir)) != "solutions" {
		t.Fatalf("overlay dir is %s", dir)
	}
}

func TestListFiltersAndKeepsRelativePaths(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"main.c", "a.h", "b.cpp", "c.cc", "d.cxx", "e.hpp", "f.hh", "g.hxx", "notes.txt", "Makefile"} {
		writeFile(t, dir, name, "")
	}
	writeFile(t, dir, "src/thing.h", "")
	files, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.h", "b.cpp", "c.cc", "d.cxx", "e.hpp", "f.hh", "g.hxx", "main.c", "src/thing.h"}
	if got := rels(files); !slices.Equal(got, want) {
		t.Fatalf("files are %v, want %v", got, want)
	}
	for _, f := range files {
		if !filepath.IsAbs(f.Path) {
			t.Fatalf("%s is not an absolute path", f.Path)
		}
	}
}

func TestListMissingDirectory(t *testing.T) {
	files, err := List(filepath.Join(t.TempDir(), "missing"))
	if err != nil || len(files) != 0 {
		t.Fatalf("missing directory returned %v, %v", files, err)
	}
}

func TestListSkipsSymlinks(t *testing.T) {
	dir := t.TempDir()
	outside := t.TempDir()
	writeFile(t, outside, "secret.c", "")
	writeFile(t, dir, "real.c", "")
	if err := os.Symlink(filepath.Join(outside, "secret.c"), filepath.Join(dir, "link.c")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	files, err := List(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := rels(files); !slices.Equal(got, []string{"real.c"}) {
		t.Fatalf("files are %v, want only real.c", got)
	}
}

func TestResolvePrefersOverlayWithoutMerging(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	overlay, err := OverlayDir("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, overlay, "solution.c", "overlay\n")
	drillDir := t.TempDir()
	writeFile(t, filepath.Join(drillDir, "solution"), "solution.c", "bundled\n")
	writeFile(t, filepath.Join(drillDir, "solution"), "extra.h", "bundled\n")

	src, err := Resolve("ft_strlen", drillDir)
	if err != nil {
		t.Fatal(err)
	}
	if src == nil || !src.Local || src.Dir != overlay {
		t.Fatalf("overlay did not win: %+v", src)
	}
	if got := rels(src.Files); !slices.Equal(got, []string{"solution.c"}) {
		t.Fatalf("overlay and bundled solutions were merged: %v", got)
	}
}

func TestResolveOverlayWorksForBuiltinDrill(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	overlay, err := OverlayDir("ft_strlen")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, overlay, "answer.c", "")
	src, err := Resolve("ft_strlen", "")
	if err != nil {
		t.Fatal(err)
	}
	if src == nil || !src.Local || !slices.Equal(rels(src.Files), []string{"answer.c"}) {
		t.Fatalf("built-in overlay not found: %+v", src)
	}
}

func TestResolveUsesBundledSolution(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	drillDir := t.TempDir()
	writeFile(t, filepath.Join(drillDir, "solution"), "src/ft_strlen.c", "")
	writeFile(t, filepath.Join(drillDir, "solution"), "include/ft.h", "")
	src, err := Resolve("custom", drillDir)
	if err != nil {
		t.Fatal(err)
	}
	if src == nil || src.Local || src.Dir != filepath.Join(drillDir, "solution") {
		t.Fatalf("bundled solution not used: %+v", src)
	}
	if got := rels(src.Files); !slices.Equal(got, []string{"include/ft.h", "src/ft_strlen.c"}) {
		t.Fatalf("relative paths were not preserved: %v", got)
	}
}

func TestResolveEmptyOverlayFallsBack(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	overlay, err := OverlayDir("custom")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, overlay, "notes.txt", "not a reference\n")
	drillDir := t.TempDir()
	writeFile(t, filepath.Join(drillDir, "solution"), "answer.c", "")

	src, err := Resolve("custom", drillDir)
	if err != nil {
		t.Fatal(err)
	}
	if src == nil || src.Local {
		t.Fatalf("unsupported overlay files counted as a solution: %+v", src)
	}
	if got := rels(src.Files); !slices.Equal(got, []string{"answer.c"}) {
		t.Fatalf("files are %v", got)
	}
}

func TestResolveMissingSolution(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	if src, err := Resolve("ft_strlen", ""); err != nil || src != nil {
		t.Fatalf("built-in without overlay returned %+v, %v", src, err)
	}
	if src, err := Resolve("custom", t.TempDir()); err != nil || src != nil {
		t.Fatalf("custom without solution returned %+v, %v", src, err)
	}
}

func TestResolveRejectsUnsafeDrillIDs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	drillDir := t.TempDir()
	for _, id := range []string{"", ".", "..", "../ft", "a/b", `a\b`, "/abs"} {
		if _, err := Resolve(id, drillDir); err == nil {
			t.Errorf("Resolve accepted drill id %q", id)
		}
		if _, err := OverlayDir(id); err == nil {
			t.Errorf("OverlayDir accepted drill id %q", id)
		}
	}
}
