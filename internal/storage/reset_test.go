package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFactoryResetBackupAndScope(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	contents := map[string]string{
		"config.json": "broken JSON is backed up too", "history.json": "progress",
		"drill_prefs.json": "preferences", "layout.json": "layout", "hidden.json": "hidden presets",
		"drills/custom/drill.json": "custom drill", "solutions/custom/answer.cpp": "private solution",
		"backups/old.json": "older backup", "reference-backups/old/answer.c": "old reference",
		"history.json.corrupt-123": "corrupt history", ".config.json-leftover.tmp": "interrupted save",
	}
	for name, text := range contents {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0640); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "personal-notes.txt"), []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	external := t.TempDir()
	if err := os.WriteFile(filepath.Join(external, "source.cpp"), []byte("external"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(external, filepath.Join(dir, "drills", "linked")); err != nil {
		t.Fatal(err)
	}
	backup, err := FactoryReset()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Dir(backup) != filepath.Dir(dir) || backup == dir {
		t.Fatalf("backup is not a sibling of the reset directory: %s", backup)
	}
	if info, err := os.Stat(backup); err != nil || info.Mode().Perm() != 0700 {
		t.Fatalf("backup permissions are not private: %v, %v", info, err)
	}
	for name, text := range contents {
		path := filepath.Join(backup, "data", name)
		data, err := os.ReadFile(path)
		if err != nil || string(data) != text {
			t.Fatalf("backup of %s: %q, %v", name, data, err)
		}
		if info, err := os.Stat(path); err != nil || info.Mode().Perm() != 0640 {
			t.Fatalf("backup permissions changed for %s: %v, %v", name, info, err)
		}
		if _, err := os.Lstat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("app-owned data remains: %s, %v", name, err)
		}
	}
	link, err := os.Readlink(filepath.Join(backup, "data", "drills", "linked"))
	if err != nil || link != external {
		t.Fatalf("backup did not preserve symlink: %q, %v", link, err)
	}
	for path, want := range map[string]string{filepath.Join(external, "source.cpp"): "external", filepath.Join(dir, "personal-notes.txt"): "keep"} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != want {
			t.Fatalf("unrelated file changed: %s, %q, %v", path, data, err)
		}
	}
	if _, err := os.Stat(filepath.Join(backup, "RESTORE.txt")); err != nil {
		t.Fatal(err)
	}
	// A second reset has nothing to remove and preserves the first backup.
	if next, err := FactoryReset(); err != nil || next != "" {
		t.Fatalf("second reset: %q, %v", next, err)
	}
	if _, err := os.Stat(filepath.Join(backup, "data", "history.json")); err != nil {
		t.Fatal(err)
	}
	restored := filepath.Join(t.TempDir(), "codedrill")
	if err := os.Rename(filepath.Join(backup, "data"), restored); err != nil {
		t.Fatal(err)
	}
	for name, text := range contents {
		if data, err := os.ReadFile(filepath.Join(restored, name)); err != nil || string(data) != text {
			t.Fatalf("restoring %s: %q, %v", name, data, err)
		}
	}
}

func TestFactoryResetBackupFailurePreservesData(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks require an unprivileged user")
	}
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	dir, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(base, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(base, 0700) })
	if backup, err := FactoryReset(); err == nil || backup != "" {
		t.Fatalf("backup failure did not stop reset: %q, %v", backup, err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "preserve" {
		t.Fatalf("data changed without a backup: %q, %v", data, err)
	}
}

func TestFactoryResetFreshAndPartialData(t *testing.T) {
	for _, files := range []map[string]string{nil, {"config.json": "{"}, {"hidden.json": "[]", "history.json": "{}"}} {
		t.Run("partial", func(t *testing.T) {
			t.Setenv("XDG_CONFIG_HOME", t.TempDir())
			dir, err := configDirPath()
			if err != nil {
				t.Fatal(err)
			}
			for name, text := range files {
				if err := os.MkdirAll(dir, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := FactoryReset(); err != nil {
				t.Fatal(err)
			}
			cfg, err := LoadConfig()
			if err != nil || cfg.Theme != "" || !cfg.MouseOn() || cfg.GetCountdown() != 3 {
				t.Fatalf("default settings not restored: %+v, %v", cfg, err)
			}
			history, err := LoadHistory()
			if err != nil || len(history.Attempts) != 0 || len(history.Stats) != 0 {
				t.Fatalf("history not empty: %+v, %v", history, err)
			}
		})
	}
}

func TestFactoryResetRefusesRepositoryAndLinkedRoot(t *testing.T) {
	for _, linked := range []bool{false, true} {
		t.Run("unsafe root", func(t *testing.T) {
			base := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", base)
			dir := filepath.Join(base, "codedrill")
			if linked {
				if err := os.Symlink(t.TempDir(), dir); err != nil {
					t.Fatal(err)
				}
			} else {
				if err := os.MkdirAll(filepath.Join(dir, ".git"), 0700); err != nil {
					t.Fatal(err)
				}
			}
			path := filepath.Join(dir, "config.json")
			if err := os.WriteFile(path, []byte("keep"), 0600); err != nil {
				t.Fatal(err)
			}
			if backup, err := FactoryReset(); err == nil || backup != "" {
				t.Fatalf("unsafe reset was accepted: %q, %v", backup, err)
			}
			if data, err := os.ReadFile(path); err != nil || string(data) != "keep" {
				t.Fatalf("unsafe reset changed data: %q, %v", data, err)
			}
		})
	}
}
