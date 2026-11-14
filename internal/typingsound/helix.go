package typingsound

import (
	"os"
	"path/filepath"
	"strings"
)

// HelixArgs returns editor arguments that disable Helix auto-pairs for this
// session. The returned cleanup removes the temporary configuration and must be
// called after the editor exits. Other editors and sessions that already pass
// their own --config are left unchanged.
func HelixArgs(args []string) ([]string, func(), error) {
	noop := func() {}
	if !isHelix(args) || hasConfigFlag(args) {
		return args, noop, nil
	}
	source := ""
	if data, err := os.ReadFile(helixConfigPath()); err == nil {
		source = string(data)
	} else if !os.IsNotExist(err) {
		// Without the user's settings a replacement file would change Helix's
		// behavior, so keep the normal launch.
		return args, noop, err
	}
	dir, err := os.MkdirTemp("", "codedrill-helix-")
	if err != nil {
		return args, noop, err
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte(mergeAutoPairs(source)), 0600); err != nil {
		cleanup()
		return args, noop, err
	}
	out := append([]string{args[0], "-c", path}, args[1:]...)
	return out, cleanup, nil
}

func isHelix(args []string) bool {
	if len(args) == 0 {
		return false
	}
	switch strings.ToLower(filepath.Base(args[0])) {
	case "hx", "helix", "helix-term":
		return true
	}
	return false
}

func hasConfigFlag(args []string) bool {
	for _, arg := range args[1:] {
		if arg == "-c" || arg == "--config" || strings.HasPrefix(arg, "--config=") {
			return true
		}
	}
	return false
}

// helixConfigPath is the user configuration Helix loads by default.
func helixConfigPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "helix", "config.toml")
}

// mergeAutoPairs returns a copy of a Helix config with editor.auto-pairs set to
// false. Every other line is kept as it was, so themes, keymaps, language
// settings and comments survive.
func mergeAutoPairs(source string) string {
	lines := strings.Split(source, "\n")
	inEditor := false
	editorAt := -1
	for i, line := range lines {
		body := strings.TrimSpace(stripTomlComment(line))
		if strings.HasPrefix(body, "[") && strings.HasSuffix(body, "]") {
			inEditor = strings.ReplaceAll(body, " ", "") == "[editor]"
			if inEditor && editorAt < 0 {
				editorAt = i
			}
			continue
		}
		eq := strings.Index(body, "=")
		if eq < 0 {
			continue
		}
		key := strings.TrimSpace(body[:eq])
		if key == "auto-pairs" && inEditor || key == "editor.auto-pairs" {
			indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
			name := "auto-pairs"
			if key == "editor.auto-pairs" {
				name = "editor.auto-pairs"
			}
			lines[i] = indent + name + " = false"
			return strings.Join(lines, "\n")
		}
	}
	if editorAt < 0 {
		if source == "" {
			return "[editor]\nauto-pairs = false\n"
		}
		return strings.TrimRight(source, "\n") + "\n\n[editor]\nauto-pairs = false\n"
	}
	lines = append(lines[:editorAt+1], append([]string{"auto-pairs = false"}, lines[editorAt+1:]...)...)
	return strings.Join(lines, "\n")
}

// stripTomlComment removes a comment that starts outside a quoted string.
func stripTomlComment(line string) string {
	var quote byte
	for i := 0; i < len(line); i++ {
		switch c := line[i]; {
		case quote != 0:
			if c == quote && (i == 0 || line[i-1] != '\\') {
				quote = 0
			}
		case c == '"' || c == '\'':
			quote = c
		case c == '#':
			return line[:i]
		}
	}
	return line
}
