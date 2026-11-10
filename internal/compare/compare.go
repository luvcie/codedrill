// Package compare checks a sprint workspace against a drill's reference
// solution. It resolves the solution through internal/solutions and delegates
// each file to the token comparison engine in internal/reference.
package compare

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/reference"
	"github.com/luvcie/codedrill/internal/solutions"
)

// FileResult is the comparison of one reference file with the workspace file
// at the same relative path. Missing is true when no workspace file exists.
type FileResult struct {
	Rel     string
	Missing bool
	Result  reference.Result
}

// Result is the comparison of the whole reference solution. Score is matched
// tokens divided by the sum of each file's larger token count, so large files
// weigh more than small ones.
type Result struct {
	Exact           bool
	Score           float64
	ReferenceTokens int
	StudentTokens   int
	Matched         int
	Files           []FileResult
}

// Percent rounds Score to one decimal and stays below 100 unless the whole
// comparison is exact.
func (r Result) Percent() float64 {
	return reference.Percent(r.Score, r.Exact)
}

// Compare compares a drill's workspace with its selected reference solution.
// The entry file comes first, followed by remaining files in alphabetical order.
// A reference file without a workspace counterpart always makes the result inexact.
func Compare(d drill.Drill, workspace string) (Result, error) {
	language := strings.ToLower(strings.TrimSpace(d.Language))
	if !reference.SupportedLanguage(language) {
		return Result{}, fmt.Errorf("reference comparison is not supported for language %q", d.Language)
	}

	info, err := os.Stat(workspace)
	if err != nil {
		return Result{}, fmt.Errorf("workspace: %w", err)
	}
	if !info.IsDir() {
		return Result{}, fmt.Errorf("workspace %s is not a directory", workspace)
	}

	src, err := solutions.Resolve(d.ID, d.Dir)
	if err != nil {
		return Result{}, err
	}
	if src == nil || len(src.Files) == 0 {
		return Result{}, fmt.Errorf("no reference solution for %s", d.ID)
	}

	files := src.Files
	if language != "c" {
		files, err = cppFiles(d, files)
		if err != nil {
			return Result{}, err
		}
	}
	res := Result{Exact: true, Files: make([]FileResult, 0, len(files))}
	denominator := 0
	for _, f := range orderedFiles(files, d.EntryFile) {
		switch ext := strings.ToLower(filepath.Ext(f.Rel)); ext {
		case ".c", ".h":
		case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
			if language == "c" {
				return Result{}, fmt.Errorf("%s: C++ reference file in a C drill", f.Rel)
			}
		default:
			return Result{}, fmt.Errorf("%s: unsupported reference file", f.Rel)
		}
		clean := filepath.Clean(filepath.FromSlash(f.Rel))
		if !filepath.IsLocal(clean) || clean == "." {
			return Result{}, fmt.Errorf("unsafe reference path %q", f.Rel)
		}

		if err := referenceOutsideWorkspace(workspace, f.Path); err != nil {
			return Result{}, err
		}
		refData, err := os.ReadFile(f.Path)
		if err != nil {
			return Result{}, fmt.Errorf("read reference %s: %w", f.Rel, err)
		}
		student, exists, err := readStudentFile(workspace, f.Rel)
		if err != nil {
			return Result{}, err
		}

		cmp, err := reference.CompareLanguage(language, string(refData), string(student))
		if err != nil {
			return Result{}, fmt.Errorf("%s: %w", f.Rel, err)
		}
		if !exists {
			// A missing file never passes, even when the reference is empty.
			cmp.Exact = false
			if cmp.Score >= 1 {
				cmp.Score = 0
			}
		}

		res.Files = append(res.Files, FileResult{Rel: f.Rel, Missing: !exists, Result: cmp})
		res.ReferenceTokens += cmp.ReferenceTokens
		res.StudentTokens += cmp.StudentTokens
		res.Matched += cmp.Matched
		denominator += max(cmp.ReferenceTokens, cmp.StudentTokens)
		if !cmp.Exact {
			res.Exact = false
		}
	}

	if denominator > 0 {
		res.Score = float64(res.Matched) / float64(denominator)
	} else if res.Exact {
		res.Score = 1
	}
	if !res.Exact && res.Score >= 1 {
		res.Score = 0.9999
	}
	return res, nil
}

func orderedFiles(files []solutions.File, entryFile string) []solutions.File {
	sorted := slices.Clone(files)
	slices.SortFunc(sorted, func(a, b solutions.File) int { return strings.Compare(a.Rel, b.Rel) })
	entry := ""
	if clean := filepath.Clean(filepath.FromSlash(entryFile)); clean != "." {
		entry = filepath.ToSlash(clean)
	}
	out := make([]solutions.File, 0, len(sorted))
	for _, f := range sorted {
		if f.Rel == entry {
			out = append(out, f)
		}
	}
	for _, f := range sorted {
		if f.Rel != entry {
			out = append(out, f)
		}
	}
	return out
}

// readStudentFile reads rel inside root without following symbolic links.
// exists is false when any part of the path is missing.
func readStudentFile(root, rel string) (data []byte, exists bool, err error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if !filepath.IsLocal(clean) || clean == "." {
		return nil, false, fmt.Errorf("unsafe workspace path %q", rel)
	}
	parts := strings.Split(clean, string(filepath.Separator))
	current := root
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if errors.Is(err, os.ErrNotExist) {
			return nil, false, nil
		}
		if err != nil {
			return nil, false, fmt.Errorf("read workspace %s: %w", rel, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, false, fmt.Errorf("%s: symbolic links are not allowed", rel)
		}
		last := i == len(parts)-1
		if last && !info.Mode().IsRegular() {
			return nil, false, fmt.Errorf("%s: not a regular file", rel)
		}
		if !last && !info.IsDir() {
			return nil, false, fmt.Errorf("%s: not a directory", rel)
		}
	}
	data, err = os.ReadFile(filepath.Join(root, clean))
	if err != nil {
		return nil, false, fmt.Errorf("read workspace %s: %w", rel, err)
	}
	return data, true, nil
}

// Explicit ownership excludes harnesses and supplied interfaces. Legacy C++
// manifests keep additional reference files unless they are provided resources.
// Optional student files are optional to create, but a selected reference that
// contains one still requires its counterpart for an exact copy.
func cppFiles(d drill.Drill, files []solutions.File) ([]solutions.File, error) {
	required, optional, err := d.StudentPaths()
	if err != nil {
		return nil, err
	}
	owned := make(map[string]bool)
	for _, paths := range [][]string{required, optional} {
		for _, path := range paths {
			owned[filepath.ToSlash(path)] = true
		}
	}
	explicit := len(d.StudentFiles)+len(d.OptionalStudentFiles) > 0
	selected := make(map[string]bool)
	var out []solutions.File
	for _, file := range files {
		if !owned[file.Rel] {
			if explicit {
				continue
			}
			if _, provided := d.ExtraFiles[filepath.FromSlash(file.Rel)]; provided {
				continue
			}
			if d.Dir != "" {
				_, err := os.Lstat(filepath.Join(d.Dir, filepath.FromSlash(file.Rel)))
				if err == nil {
					continue
				}
				if !errors.Is(err, os.ErrNotExist) {
					return nil, fmt.Errorf("inspect provided file %s: %w", file.Rel, err)
				}
			}
		}
		out = append(out, file)
		selected[file.Rel] = true
	}
	for _, path := range required {
		if !selected[filepath.ToSlash(path)] {
			return nil, fmt.Errorf("reference is missing required student file %s", path)
		}
	}
	return out, nil
}

func referenceOutsideWorkspace(workspace, path string) error {
	root, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	resolved, err = filepath.EvalSymlinks(resolved)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return err
	}
	if rel == "." || filepath.IsLocal(rel) {
		return fmt.Errorf("reference file must be outside the workspace: %s", path)
	}
	return nil
}
