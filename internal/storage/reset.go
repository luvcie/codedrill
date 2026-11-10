package storage

import (
	"bytes"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

var resetNames = []string{
	"config.json", "drill_prefs.json", "history.json", "layout.json", "hidden.json",
	"drills", "solutions", "backups", "reference-backups",
}

type resetFile struct {
	mode fs.FileMode
	data []byte
	link string
}

// FactoryReset backs up and verifies all app-owned data before removing it.
// Unknown files are preserved. Root confines reads and removals to the config
// tree; symlinks are backed up as links, never traversed into external data.
// A nonempty backup path is returned even if a later step fails.
func FactoryReset() (backup string, err error) {
	dir, err := configDirPath()
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(dir) {
		return "", fmt.Errorf("refusing to reset a relative config path: %s", dir)
	}
	info, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("refusing to reset a linked or non-directory config path: %s", dir)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return "", err
	}
	defer root.Close()
	if _, err := root.Lstat(".git"); !os.IsNotExist(err) {
		return "", fmt.Errorf("refusing to reset a source repository: %s", dir)
	}
	entries, err := fs.ReadDir(root.FS(), ".")
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		name := entry.Name()
		owned := slices.Contains(resetNames, name) || strings.HasPrefix(name, "history.json.corrupt-")
		for _, base := range resetNames[:5] {
			owned = owned || (strings.HasPrefix(name, "."+base+"-") && strings.HasSuffix(name, ".tmp"))
		}
		if owned {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return "", nil
	}
	before, err := resetSnapshot(root, names)
	if err != nil {
		return "", fmt.Errorf("nothing reset; could not read user data: %w", err)
	}
	backup, err = os.MkdirTemp(filepath.Dir(dir), "codedrill-backup-"+time.Now().UTC().Format("20060102T150405Z")+"-")
	if err != nil {
		return "", fmt.Errorf("nothing reset; could not create backup: %w", err)
	}
	destination := filepath.Join(backup, "data")
	if err := os.Mkdir(destination, 0700); err != nil {
		return backup, err
	}
	copyRoot, err := os.OpenRoot(destination)
	if err != nil {
		return backup, err
	}
	defer copyRoot.Close()
	paths := slices.Sorted(maps.Keys(before))
	for _, path := range paths {
		file := before[path]
		switch {
		case file.mode.IsDir():
			err = copyRoot.MkdirAll(path, 0700)
		case file.mode&os.ModeSymlink != 0:
			err = copyRoot.Symlink(file.link, path)
		default:
			err = copyRoot.WriteFile(path, file.data, file.mode.Perm())
			if err == nil {
				err = copyRoot.Chmod(path, file.mode.Perm())
			}
		}
		if err != nil {
			return backup, fmt.Errorf("nothing reset; backup incomplete: %w", err)
		}
	}
	for _, path := range slices.Backward(paths) {
		if before[path].mode.IsDir() {
			if err := copyRoot.Chmod(path, before[path].mode.Perm()); err != nil {
				return backup, err
			}
		}
	}
	readme := fmt.Sprintf("Factory Reset backup\n\nOriginal data directory: %s\n\nTo restore, quit Codedrill and copy the contents of data/ back into that\ndirectory, preserving file permissions and symlinks. This backup is outside\nthe reset scope and is not removed by later resets.\n", dir)
	if err := AtomicWriteFile(filepath.Join(backup, "RESTORE.txt"), []byte(readme), 0600); err != nil {
		return backup, err
	}
	for _, tree := range []*os.Root{copyRoot, root} {
		after, err := resetSnapshot(tree, names)
		if err != nil || !sameResetFiles(before, after) {
			return backup, fmt.Errorf("nothing reset; backup verification failed or user data changed: %v", err)
		}
	}
	for _, name := range names {
		if err := root.RemoveAll(name); err != nil {
			return backup, fmt.Errorf("reset incomplete; restore from %s if needed: %w", backup, err)
		}
	}
	return backup, nil
}

func resetSnapshot(root *os.Root, names []string) (map[string]resetFile, error) {
	files := make(map[string]resetFile)
	for _, name := range names {
		err := fs.WalkDir(root.FS(), name, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			file := resetFile{mode: info.Mode()}
			switch {
			case file.mode.IsDir():
			case file.mode.IsRegular():
				file.data, err = root.ReadFile(path)
			case file.mode&os.ModeSymlink != 0:
				file.link, err = root.Readlink(path)
			default:
				err = fmt.Errorf("unsupported data file: %s", path)
			}
			files[path] = file
			return err
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func sameResetFiles(left, right map[string]resetFile) bool {
	if len(left) != len(right) {
		return false
	}
	for path, a := range left {
		b, ok := right[path]
		if !ok || a.mode != b.mode || a.link != b.link || !bytes.Equal(a.data, b.data) {
			return false
		}
	}
	return true
}
