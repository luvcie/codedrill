package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/luvcie/codedrill/internal/drill"
)

func TestTerminalBannerUsesLogoColour(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	d := drill.Drill{ID: "banner", EntryFile: "solution.sh", TestCommand: "true"}
	for _, tc := range []struct {
		name string
		logo string
		want string
	}{
		{"given colour", "1;2;3", "LOGO_RGB='1;2;3'"},
		{"default colour", "", "LOGO_RGB='127;160;90'"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sess, err := NewSessionWithLogo(d, 0, tc.logo)
			if err != nil {
				t.Fatal(err)
			}
			defer sess.Cleanup()
			data, err := os.ReadFile(filepath.Join(sess.Workspace, "start-terminal.sh"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), tc.want) {
				t.Fatalf("banner script lacks %s:\n%s", tc.want, data)
			}
			if strings.Contains(string(data), "38;2;127;160;90") {
				t.Fatal("banner script hard-codes the croc colour")
			}
		})
	}
}
