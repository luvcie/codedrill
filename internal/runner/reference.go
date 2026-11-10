package runner

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"

	"github.com/luvcie/codedrill/internal/drill"
	"github.com/luvcie/codedrill/internal/mode"
)

// ReferenceSession contains metadata only. It lives outside the student workspace.
type ReferenceSession struct {
	Version              int               `json:"version"`
	Workspace            string            `json:"workspace"`
	DrillID              string            `json:"drill_id"`
	DrillDir             string            `json:"drill_dir"`
	EntryFile            string            `json:"entry_file"`
	Language             string            `json:"language"`
	StudentFiles         []string          `json:"student_files,omitempty"`
	OptionalStudentFiles []string          `json:"optional_student_files,omitempty"`
	ExtraFiles           map[string]string `json:"extra_files,omitempty"`
	Options              *mode.Options     `json:"options"`
}

var ErrReferencePolicy = errors.New("reference session policy is missing or invalid")

func LoadReferenceSession(path string) (ReferenceSession, error) {
	var session ReferenceSession
	data, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(data, &session) != nil {
		return ReferenceSession{}, ErrReferencePolicy
	}
	if session.Version != 1 || !filepath.IsAbs(session.Workspace) || session.DrillID == "" || session.Options == nil {
		return ReferenceSession{}, ErrReferencePolicy
	}
	return session, nil
}

func (s *Session) writeReferenceSession(options *mode.Options) error {
	dir, err := os.MkdirTemp("", "codedrill-reference-session-*")
	if err != nil {
		return err
	}
	s.referenceDir = dir
	s.referencePolicy = filepath.Join(dir, "session.json")
	// Only resource paths are needed to preserve ownership for legacy manifests.
	provided := make(map[string]string, len(s.drill.ExtraFiles))
	for path := range s.drill.ExtraFiles {
		provided[path] = ""
	}
	data, err := json.Marshal(ReferenceSession{
		StudentFiles: s.drill.StudentFiles, OptionalStudentFiles: s.drill.OptionalStudentFiles, ExtraFiles: provided,
		Version: 1, Workspace: s.Workspace, DrillID: s.drill.ID,
		DrillDir: s.drill.Dir, EntryFile: s.EntryFile, Language: s.drill.Language, Options: options,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(s.referencePolicy, data, 0600)
}

func (s *Session) referenceKeybind() string {
	return "\nkeybinds {\n    normal {\n        bind \"Alt v\" {\n            Run " +
		strconv.Quote(executablePath()) + " \"reference-view\" \"--session\" " + strconv.Quote(s.referencePolicy) + ` {
                floating true
                close_on_exit true
                name "Reference"
            }
        }
        bind "Alt b" {
            Run ` + strconv.Quote(executablePath()) + ` "reference-view" "--session" ` + strconv.Quote(s.referencePolicy) + ` "--toggle-pane" {
                floating true
                close_on_exit true
                name "Toggle reference"
            }
        }
    }
}
`
}

func (s ReferenceSession) Drill() drill.Drill {
	return drill.Drill{ID: s.DrillID, Dir: s.DrillDir, EntryFile: s.EntryFile, Language: s.Language, StudentFiles: s.StudentFiles, OptionalStudentFiles: s.OptionalStudentFiles, ExtraFiles: s.ExtraFiles}
}
