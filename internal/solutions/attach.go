package solutions

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/luvcie/codedrill/internal/storage"
)

// Attachment describes the files copied into a drill's local overlay.
type Attachment struct {
	Dir   string
	Files []string // slash-separated relative paths, sorted
}

// Attach copies solution files from paths into the drill's local overlay. A
// file is stored under its base name; a directory keeps its internal layout.
// Existing files with the same relative path are replaced and other files are
// left alone. Symbolic links and unsafe paths are rejected.
func Attach(drillID string, paths []string) (Attachment, error) {
	dir, err := OverlayDir(drillID)
	if err != nil {
		return Attachment{}, err
	}
	if len(paths) == 0 {
		return Attachment{}, errors.New("no solution paths given")
	}

	type source struct {
		path string
		rel  string
	}
	var files []source
	seen := make(map[string]bool)
	add := func(path, rel string) error {
		rel = filepath.Clean(rel)
		if !filepath.IsLocal(rel) || rel == "." {
			return fmt.Errorf("unsafe solution path %q", rel)
		}
		rel = filepath.ToSlash(rel)
		if seen[rel] {
			return nil
		}
		seen[rel] = true
		files = append(files, source{path: path, rel: rel})
		return nil
	}

	for _, path := range paths {
		info, err := os.Lstat(path)
		if err != nil {
			return Attachment{}, err
		}
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			return Attachment{}, fmt.Errorf("%s: symbolic links are not allowed", path)
		case info.Mode().IsRegular():
			if !supported(path) {
				return Attachment{}, unsupportedFile(path)
			}
			if err := add(path, filepath.Base(path)); err != nil {
				return Attachment{}, err
			}
		case info.IsDir():
			if err := collectDir(path, add); err != nil {
				return Attachment{}, err
			}
		default:
			return Attachment{}, fmt.Errorf("%s: not a regular file or directory", path)
		}
	}
	if len(files) == 0 {
		return Attachment{}, errors.New("no supported solution files found")
	}

	if err := prepareRoot(dir); err != nil {
		return Attachment{}, err
	}
	names := make([]string, 0, len(files))
	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			return Attachment{}, err
		}
		dest, err := makeParent(dir, f.rel)
		if err != nil {
			return Attachment{}, err
		}
		if err := storage.AtomicWriteFile(dest, data, 0644); err != nil {
			return Attachment{}, err
		}
		names = append(names, f.rel)
	}
	slices.Sort(names)
	return Attachment{Dir: dir, Files: names}, nil
}

// collectDir adds the supported files under dir, keeping their paths relative
// to dir. Symlinks are an error because they may point outside the tree.
func collectDir(dir string, add func(path, rel string) error) error {
	return filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == dir {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("%s: symbolic links are not allowed", path)
		}
		if entry.IsDir() || !entry.Type().IsRegular() || !supported(path) {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		return add(path, rel)
	})
}

func unsupportedFile(name string) error {
	return fmt.Errorf("%s: unsupported solution file (use .c, .h, .cpp, .cc, .cxx, .hpp, .hh or .hxx)", name)
}

// prepareRoot creates the overlay directory, refusing a symlink in its place.
func prepareRoot(dir string) error {
	info, err := os.Lstat(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return os.MkdirAll(dir, 0755)
	case err != nil:
		return err
	case info.Mode()&os.ModeSymlink != 0:
		return fmt.Errorf("solution directory %s is a symbolic link", dir)
	case !info.IsDir():
		return fmt.Errorf("solution directory %s is not a directory", dir)
	}
	return nil
}

// makeParent creates the destination's parent directories, refusing to write
// through a symlink inside the overlay. It returns the destination path.
func makeParent(root, rel string) (string, error) {
	dest := filepath.Join(root, filepath.FromSlash(rel))
	prefix := root + string(filepath.Separator)
	if !strings.HasPrefix(dest, prefix) {
		return "", fmt.Errorf("unsafe solution path %q", rel)
	}
	parent := filepath.Dir(dest)
	if parent == root {
		return dest, nil
	}
	current := root
	for _, part := range strings.Split(strings.TrimPrefix(parent, prefix), string(filepath.Separator)) {
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		switch {
		case errors.Is(err, os.ErrNotExist):
			if err := os.Mkdir(current, 0755); err != nil {
				return "", err
			}
		case err != nil:
			return "", err
		case info.Mode()&os.ModeSymlink != 0:
			return "", fmt.Errorf("solution path %s is a symbolic link", current)
		case !info.IsDir():
			return "", fmt.Errorf("solution path %s is not a directory", current)
		}
	}
	return dest, nil
}
