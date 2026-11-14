//go:build unix

package main

import (
	"os/exec"
	"testing"

	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/typingsound"
)

func TestTypingSoundMode(t *testing.T) {
	on, off := true, false
	enabled := func(pack string) storage.Config {
		return storage.Config{Typing: &storage.TypingSettings{Enabled: &on, Pack: pack}}
	}
	tests := []struct {
		name      string
		cfg       storage.Config
		env       string
		prototype bool
		usePack   bool
		pack      string
	}{
		{name: "saved prototype", cfg: enabled(typingsound.PrototypePackID), prototype: true},
		{name: "saved prototype off", cfg: storage.Config{Typing: &storage.TypingSettings{Enabled: &off, Pack: typingsound.PrototypePackID}}},
		{name: "environment prototype", cfg: storage.Config{}, env: "prototype", prototype: true},
		{name: "real pack", cfg: enabled("sfx-Starcat_JP-8Bit_Puzzler"), usePack: true, pack: "sfx-Starcat_JP-8Bit_Puzzler"},
		{name: "unknown pack", cfg: enabled("missing")},
		{name: "silent", cfg: enabled(typingsound.SilentPackID)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("CODEDRILL_TYPING_SOUNDS", tc.env)
			prototype, pack, usePack := typingSoundMode(tc.cfg)
			if prototype != tc.prototype || usePack != tc.usePack || pack.ID != tc.pack {
				t.Fatalf("got prototype=%v usePack=%v pack=%q, want %v %v %q",
					prototype, usePack, pack.ID, tc.prototype, tc.usePack, tc.pack)
			}
		})
	}
}

func TestEditorResultIgnoresExitStatusAndReportsStartFailure(t *testing.T) {
	cmd := exec.Command("sh", "-c", "exit 7")
	err := cmd.Run()
	if err == nil {
		t.Fatal("sh exit 7 succeeded")
	}
	if got := editorResult(cmd, err); got != nil {
		t.Fatalf("editor exit returned %v, want nil", got)
	}

	missing := exec.Command("/nonexistent/codedrill-test-editor")
	err = missing.Run()
	if got := editorResult(missing, err); got == nil {
		t.Fatal("a failure to start the editor was hidden")
	}
}
