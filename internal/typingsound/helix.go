package typingsound

import (
	"os"
	"path/filepath"
	"strconv"
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
// settings and comments survive. The parser understands quoted table names,
// quoted keys, and dotted editor keys, which keeps the edit from creating a
// duplicate key or table in valid TOML.
func mergeAutoPairs(source string) string {
	lines := strings.Split(source, "\n")
	var table []string
	editorAt := -1
	for i, line := range lines {
		body := strings.TrimSpace(stripTomlComment(line))
		if path, ok := parseTomlTable(body); ok {
			table = path
			if len(path) == 1 && path[0] == "editor" && editorAt < 0 {
				editorAt = i
			}
			continue
		}
		eq := unquotedIndex(body, '=')
		if eq < 0 {
			continue
		}
		key, ok := parseTomlKeyPath(body[:eq])
		if !ok {
			continue
		}
		if len(table) == 1 && table[0] == "editor" && len(key) == 1 && key[0] == "auto-pairs" {
			return replaceTomlLine(lines, i, "auto-pairs")
		}
		if len(table) == 0 && len(key) == 2 && key[0] == "editor" && key[1] == "auto-pairs" {
			return replaceTomlLine(lines, i, "editor.auto-pairs")
		}
		if len(table) == 0 && len(key) == 1 && key[0] == "editor" {
			if updated, ok := setInlineAutoPairs(body, eq); ok {
				indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
				lines[i] = indent + updated + line[len(body):]
				return strings.Join(lines, "\n")
			}
		}
	}
	if editorAt < 0 {
		if source == "" {
			return "[editor]\nauto-pairs = false\n"
		}
		return strings.TrimRight(source, "\n") + "\n\n[editor]\nauto-pairs = false\n"
	}
	inserted := make([]string, 0, len(lines)+1)
	inserted = append(inserted, lines[:editorAt+1]...)
	inserted = append(inserted, "auto-pairs = false")
	inserted = append(inserted, lines[editorAt+1:]...)
	return strings.Join(inserted, "\n")
}

// setInlineAutoPairs edits a single-line inline editor table without changing
// its other keys. It returns false for a value that is not an inline table.
func setInlineAutoPairs(body string, eq int) (string, bool) {
	start := eq + 1
	for start < len(body) && (body[start] == ' ' || body[start] == '\t') {
		start++
	}
	if start >= len(body) || body[start] != '{' {
		return "", false
	}
	end := inlineTableEnd(body, start)
	if end < 0 {
		return "", false
	}
	inner := body[start+1 : end]
	parts := splitInlineTable(inner)
	found := false
	for i, part := range parts {
		partEq := unquotedIndex(part, '=')
		if partEq < 0 {
			continue
		}
		key, ok := parseTomlKeyPath(part[:partEq])
		if ok && len(key) == 1 && key[0] == "auto-pairs" {
			leading := part[:len(part)-len(strings.TrimLeft(part, " \t"))]
			trailing := part[len(strings.TrimRight(part, " \t")):]
			parts[i] = leading + strings.TrimSpace(part[:partEq]) + " = false" + trailing
			found = true
			break
		}
	}
	if found {
		inner = strings.Join(parts, ",")
	} else if strings.TrimSpace(inner) == "" {
		inner = " auto-pairs = false "
	} else {
		inner = strings.TrimRight(inner, " \t") + ", auto-pairs = false "
	}
	return body[:start+1] + inner + body[end:], true
}

func inlineTableEnd(source string, start int) int {
	depth := 0
	var quote byte
	for i := start; i < len(source); i++ {
		c := source[i]
		if quote != 0 {
			if quote == '"' && c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func splitInlineTable(source string) []string {
	var parts []string
	start, depth := 0, 0
	var quote byte
	for i := 0; i < len(source); i++ {
		c := source[i]
		if quote != 0 {
			if quote == '"' && c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		switch c {
		case '"', '\'':
			quote = c
		case '{', '[':
			depth++
		case '}', ']':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, source[start:i])
				start = i + 1
			}
		}
	}
	return append(parts, source[start:])
}

func replaceTomlLine(lines []string, i int, key string) string {
	line := lines[i]
	indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
	body := stripTomlComment(line)
	comment := line[len(body):]
	if comment != "" {
		gap := body[len(strings.TrimRight(body, " \t")):]
		if gap == "" {
			gap = " "
		}
		comment = gap + comment
	}
	lines[i] = indent + key + " = false" + comment
	return strings.Join(lines, "\n")
}

// parseTomlTable reads a single table header such as [editor], ["editor"] or
// ['editor']. Array-of-table headers are not relevant to Helix config and are
// rejected so they cannot be mistaken for editor.
func parseTomlTable(body string) ([]string, bool) {
	if len(body) < 2 || body[0] != '[' || body[len(body)-1] != ']' || strings.HasPrefix(body, "[[") {
		return nil, false
	}
	return parseTomlKeyPath(strings.TrimSpace(body[1 : len(body)-1]))
}

// parseTomlKeyPath parses bare and quoted key segments separated by dots.
func parseTomlKeyPath(source string) ([]string, bool) {
	source = strings.TrimSpace(source)
	if source == "" {
		return nil, false
	}
	var parts []string
	for {
		source = strings.TrimSpace(source)
		if source == "" {
			return nil, false
		}
		var part string
		switch source[0] {
		case '"', '\'':
			value, rest, ok := parseTomlString(source)
			if !ok {
				return nil, false
			}
			part, source = value, rest
		default:
			i := 0
			for i < len(source) && source[i] != '.' && source[i] != ' ' && source[i] != '\t' {
				i++
			}
			if i == 0 || !validBareKey(source[:i]) {
				return nil, false
			}
			part, source = source[:i], source[i:]
		}
		parts = append(parts, part)
		source = strings.TrimSpace(source)
		if source == "" {
			return parts, true
		}
		if source[0] != '.' {
			return nil, false
		}
		source = strings.TrimSpace(source[1:])
	}
}

func parseTomlString(source string) (value, rest string, ok bool) {
	quote := source[0]
	for i := 1; i < len(source); i++ {
		if quote == '"' && source[i] == '\\' {
			i++
			continue
		}
		if source[i] != quote {
			continue
		}
		raw := source[:i+1]
		if quote == '\'' {
			return raw[1 : len(raw)-1], source[i+1:], true
		}
		value, err := strconv.Unquote(raw)
		if err != nil {
			return "", "", false
		}
		return value, source[i+1:], true
	}
	return "", "", false
}

func validBareKey(key string) bool {
	for i := 0; i < len(key); i++ {
		c := key[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-' {
			continue
		}
		return false
	}
	return key != ""
}

func unquotedIndex(source string, want byte) int {
	var quote byte
	for i := 0; i < len(source); i++ {
		c := source[i]
		if quote != 0 {
			if quote == '"' && c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if c == want {
			return i
		}
	}
	return -1
}

// stripTomlComment removes a comment that starts outside a quoted string.
func stripTomlComment(line string) string {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		if quote != 0 {
			if quote == '"' && c == '\\' {
				i++
				continue
			}
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
			continue
		}
		if c == '#' {
			return line[:i]
		}
	}
	return line
}
