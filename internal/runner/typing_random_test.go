package runner

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/typingsound"
)

func TestRandomTypingPackRequiresBothToggles(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	on, off := true, false
	for _, tc := range []struct {
		name   string
		typing *storage.TypingSettings
		random bool
	}{
		{"default", nil, false},
		{"sounds off", &storage.TypingSettings{Enabled: &off, RandomPackPerDrill: true}, false},
		{"random off", &storage.TypingSettings{Enabled: &on, Pack: typingsound.Catalog[0].ID}, false},
		{"random on with silent saved", &storage.TypingSettings{Enabled: &on, RandomPackPerDrill: true, Pack: typingsound.SilentPackID}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := storage.SaveConfig(storage.Config{Typing: tc.typing}); err != nil {
				t.Fatal(err)
			}
			s, err := NewSession(drill.Drill{ID: "random-typing", EntryFile: "main.c"}, 0)
			if err != nil {
				t.Fatal(err)
			}
			s.SessionName = ""
			t.Cleanup(s.Cleanup)
			if _, found := typingsound.FindPack(s.typingPack); found != tc.random {
				t.Fatalf("drill pack = %q, want random catalog pack: %v", s.typingPack, tc.random)
			}
			if !tc.random && s.typingPack != "" {
				t.Fatalf("random disabled but drill chose %q", s.typingPack)
			}
			cfg, err := storage.LoadConfig()
			if err != nil || cfg.TypingPackID() != (storage.Config{Typing: tc.typing}).TypingPackID() {
				t.Fatalf("drill changed the manual pack: %+v, %v", cfg.Typing, err)
			}
		})
	}
}

func TestRandomTypingPackSurvivesEditorRestarts(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	t.Setenv("CODEDRILL_TYPING_SOUNDS", "")
	t.Setenv("CODEDRILL_TYPING_PACK", "inherited-from-another-drill")
	t.Setenv("CODEDRILL_EDITOR", "hx")
	bin := t.TempDir()
	for _, name := range []string{"bash", "dirname"} {
		real, err := exec.LookPath(name)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(real, filepath.Join(bin, name)); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(bin, "hx"), []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	log := filepath.Join(t.TempDir(), "packs")
	codedrill := filepath.Join(bin, "codedrill-stub")
	if err := os.WriteFile(codedrill, []byte("#!/bin/sh\nprintf '%s\\n' \"$CODEDRILL_TYPING_PACK\" >> "+shellQuote(log)+"\n"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	previous := executablePath
	executablePath = func() string { return codedrill }
	t.Cleanup(func() { executablePath = previous })
	on := true
	cfg := storage.Config{Editor: "hx", Shell: "bash", Typing: &storage.TypingSettings{
		Enabled: &on, Pack: typingsound.PrototypePackID, RandomPackPerDrill: true,
	}}
	if err := storage.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	newSession := func() *Session {
		t.Helper()
		s, err := NewSession(drill.Drill{ID: "random-restarts", EntryFile: "main.c"}, 0)
		if err != nil {
			t.Fatal(err)
		}
		s.SessionName = ""
		t.Cleanup(s.Cleanup)
		return s
	}
	s := newSession()
	if _, ok := typingsound.FindPack(s.typingPack); !ok {
		t.Fatalf("drill did not choose a catalog pack: %q", s.typingPack)
	}
	// Changing the default only affects new drills, including resumed direct launches.
	cfg.Typing.RandomPackPerDrill = false
	if err := storage.SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		cmd := s.GetSprintCmd()
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("direct editor: %v\n%s", err, output)
		}
	}
	for _, script := range []string{"start-editor.sh", "start-editor.sh", "start-terminal.sh"} {
		cmd := exec.Command("bash", filepath.Join(s.Workspace, script))
		cmd.Dir = s.Workspace
		cmd.Stdin = strings.NewReader("hx other.c\nhx main.c\nexit\n")
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%s: %v\n%s", script, err, output)
		}
	}
	// The shim must also retain the choice after changing directories in a pane.
	cmd := exec.Command(filepath.Join(s.Workspace, editorShimDir, "hx"), "other.c")
	cmd.Dir = t.TempDir()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("reopened editor: %v\n%s", err, output)
	}
	data, err := os.ReadFile(log)
	if err != nil || string(data) != strings.Repeat(s.typingPack+"\n", 11) {
		t.Fatalf("restarts changed the drill's pack: %q, %v", data, err)
	}
	manual := newSession()
	if manual.typingPack != "" {
		t.Fatal("new drill ignored the disabled random toggle")
	}
	if output, err := manual.GetSprintCmd().CombinedOutput(); err != nil {
		t.Fatalf("manual editor: %v\n%s", err, output)
	}
	data, err = os.ReadFile(log)
	if err != nil || string(data) != strings.Repeat(s.typingPack+"\n", 11)+"\n" {
		t.Fatalf("new drill kept the previous random pack: %q, %v", data, err)
	}
	saved, err := storage.LoadConfig()
	if err != nil || saved.TypingPackID() != typingsound.PrototypePackID || saved.Typing.RandomPackPerDrill {
		t.Fatalf("manual pack was not preserved: %+v, %v", saved.Typing, err)
	}
	entries, err := os.ReadDir(cache)
	if err != nil || len(entries) != 0 {
		t.Fatalf("choosing a pack prepared audio before the editor needed it: %v, %v", entries, err)
	}
}
