// Package solutions locates reference solutions for drills. A reference lives
// either in a local overlay under the config directory or, for custom drills,
// in a solution directory beside drill.json. Reference files are only located,
// never copied into a sprint workspace.
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

// File is one reference source file.
type File struct {
	Path string // absolute path on disk
	Rel  string // slash-separated path inside the solution directory
}

// Source is the reference solution set for one drill. Local is true when the
// files came from the config overlay instead of the drill's own directory.
type Source struct {
	Dir   string
	Local bool
	Files []File
}

var extensions = map[string]bool{
	".c":   true,
	".h":   true,
	".cpp": true,
	".cc":  true,
	".cxx": true,
	".hpp": true,
	".hh":  true,
	".hxx": true,
}

// OverlayDir is where a drill's local reference overlay lives. The directory
// is not created; an absent overlay simply has no reference solution.
func OverlayDir(drillID string) (string, error) {
	if err := checkDrillID(drillID); err != nil {
		return "", err
	}
	base, err := storage.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "solutions", drillID), nil
}

// Resolve finds the reference solution for a drill. drillDir is the custom
// drill's directory and is empty for a built-in drill. When the local overlay
// holds at least one supported file it wins outright; the drill's own
// solution directory is used only when the overlay has none. A missing
// solution is (nil, nil), not an error.
func Resolve(drillID, drillDir string) (*Source, error) {
	overlay, err := OverlayDir(drillID)
	if err != nil {
		return nil, err
	}
	fromOverlay, err := sourceFor(overlay)
	if err != nil {
		return nil, err
	}
	if len(fromOverlay.Files) > 0 {
		fromOverlay.Local = true
		return fromOverlay, nil
	}
	if drillDir == "" {
		return nil, nil
	}
	bundled, err := sourceFor(filepath.Join(drillDir, "solution"))
	if err != nil {
		return nil, err
	}
	if len(bundled.Files) == 0 {
		return nil, nil
	}
	return bundled, nil
}

// List returns the supported reference files under dir, sorted by relative
// path. A missing directory is empty, not an error.
func List(dir string) ([]File, error) {
	src, err := sourceFor(dir)
	if err != nil {
		return nil, err
	}
	return src.Files, nil
}

// sourceFor reads the supported files under dir. Symbolic links are skipped,
// so a link cannot pull a file in from outside the solution directory.
func sourceFor(dir string) (*Source, error) {
	src := &Source{Dir: dir, Files: []File{}}
	info, err := os.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return src, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read solution directory: %w", err)
	}
	if !info.IsDir() {
		return src, nil
	}
	err = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if !filepath.IsLocal(rel) {
			return fmt.Errorf("solution path escapes %s: %q", dir, rel)
		}
		if !entry.Type().IsRegular() || !supported(rel) {
			return nil
		}
		src.Files = append(src.Files, File{Path: path, Rel: filepath.ToSlash(rel)})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read solution directory %s: %w", dir, err)
	}
	slices.SortFunc(src.Files, func(a, b File) int { return strings.Compare(a.Rel, b.Rel) })
	return src, nil
}

func supported(name string) bool {
	return extensions[strings.ToLower(filepath.Ext(name))]
}

// checkDrillID keeps a drill ID to a single path element so joining it to a
// directory cannot escape that directory.
func checkDrillID(id string) error {
	if id == "" || id == "." || id == ".." ||
		strings.ContainsAny(id, `/\`) || strings.IndexByte(id, 0) >= 0 ||
		!filepath.IsLocal(id) || id != filepath.Base(id) {
		return fmt.Errorf("invalid drill id %q", id)
	}
	return nil
}
