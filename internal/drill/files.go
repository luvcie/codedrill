package drill

import (
	"fmt"
	"path/filepath"
	"strings"
)

// InitialFile prefers the required .hpp matching the entry file, then the first
// required .hpp, and otherwise the entry file.
func (d Drill) InitialFile() string {
	matchingHeader := strings.TrimSuffix(filepath.Base(d.EntryFile), filepath.Ext(d.EntryFile)) + ".hpp"
	firstHeader := ""
	for _, file := range d.StudentFiles {
		if filepath.Ext(file) != ".hpp" {
			continue
		}
		if filepath.Base(file) == matchingHeader {
			return file
		}
		if firstHeader == "" {
			firstHeader = file
		}
	}
	if firstHeader != "" {
		return firstHeader
	}
	return d.EntryFile
}

// StudentPaths includes the entry even for manifests written before ownership
// metadata existed. Unlisted attachments remain exercise-provided resources.
func (d Drill) StudentPaths() (required, optional []string, err error) {
	seen := make(map[string]bool)
	add := func(path string, isOptional bool) error {
		if !filepath.IsLocal(path) || filepath.Clean(path) == "." {
			return fmt.Errorf("invalid student filename: %q", path)
		}
		path = filepath.Clean(path)
		if oldOptional, exists := seen[path]; exists {
			if oldOptional != isOptional {
				return fmt.Errorf("student file %q is both required and optional", path)
			}
			return nil
		}
		for other := range seen {
			if strings.HasPrefix(path, other+string(filepath.Separator)) || strings.HasPrefix(other, path+string(filepath.Separator)) {
				return fmt.Errorf("student file paths overlap: %q and %q", other, path)
			}
		}
		seen[path] = isOptional
		if isOptional {
			optional = append(optional, path)
		} else {
			required = append(required, path)
		}
		return nil
	}
	if err := add(d.EntryFile, false); err != nil {
		return nil, nil, err
	}
	for _, group := range []struct {
		paths    []string
		optional bool
	}{{d.StudentFiles, false}, {d.OptionalStudentFiles, true}} {
		for _, path := range group.paths {
			if err := add(path, group.optional); err != nil {
				return nil, nil, err
			}
		}
	}
	return required, optional, nil
}
