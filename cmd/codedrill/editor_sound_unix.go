//go:build unix

package main

import (
	"errors"
	"os"
	"os/exec"
	"sync"

	"github.com/luvcie/codedrill/internal/storage"
	"github.com/luvcie/codedrill/internal/typingsound"
)

// typingKeyer is the subset of a sound player the editor wrapper needs.
type typingKeyer interface {
	Key(typingsound.Event)
	Close()
}

// typingSink hands events to the player once it has finished loading. Events
// that arrive before then are silent, so the editor never waits for audio.
type typingSink struct {
	mu     sync.Mutex
	keyer  typingKeyer
	closed bool
}

func (s *typingSink) set(keyer typingKeyer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		keyer.Close()
		return
	}
	s.keyer = keyer
}

func (s *typingSink) key(event typingsound.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.keyer != nil {
		s.keyer.Key(event)
	}
}

func (s *typingSink) close() {
	s.mu.Lock()
	s.closed = true
	keyer := s.keyer
	s.keyer = nil
	s.mu.Unlock()
	if keyer != nil {
		keyer.Close()
	}
}

// typingSoundMode decides which player an editor launch needs. The prototype
// environment variable wins. A saved prototype pack needs the normal on switch,
// and an unknown pack id must not start a silent wrapper.
func typingSoundMode(cfg storage.Config) (prototype bool, pack typingsound.PackInfo, usePack bool) {
	if os.Getenv("CODEDRILL_TYPING_SOUNDS") == "prototype" {
		return true, typingsound.PackInfo{}, false
	}
	if !cfg.TypingSoundsOn() {
		return false, typingsound.PackInfo{}, false
	}
	if cfg.TypingPackID() == typingsound.PrototypePackID {
		return true, typingsound.PackInfo{}, false
	}
	pack, ok := typingsound.FindPack(cfg.TypingPackID())
	return false, pack, ok
}

// editorResult ignores the editor's own exit status. Only a failure to start
// the editor (no process) is reported.
func editorResult(cmd *exec.Cmd, err error) error {
	if err != nil && cmd.Process != nil {
		return nil
	}
	return err
}

func runSoundEditor(args []string) error {
	if len(args) == 0 {
		return errors.New("missing editor")
	}
	cfg, _ := storage.LoadConfig()
	prototype, pack, usePack := typingSoundMode(cfg)
	if !prototype && !usePack {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return editorResult(cmd, cmd.Run())
	}

	// Only Helix needs this, and only for a session that already plays typing
	// sounds. The temporary config copies the user's file and changes one key.
	args, cleanup, _ := typingsound.HelixArgs(args)
	defer cleanup()
	cmd := exec.Command(args[0], args[1:]...)

	wordSounds := cfg.TypingWordSoundsOn()
	sink := &typingSink{}
	go func() {
		var keyer typingKeyer
		var err error
		if prototype {
			keyer, err = typingsound.NewPrototype(cfg.TypingVolume())
		} else {
			mapping, mapErr := typingsound.ParseMapping(cfg.TypingKeySettings())
			if mapErr != nil {
				mapping = typingsound.DefaultMapping()
			}
			keyer, err = typingsound.LoadPack(pack, cfg.TypingVolume(), mapping)
		}
		if err == nil && keyer != nil {
			sink.set(keyer)
		}
	}()

	err := typingsound.RunEditor(cmd, wordSounds, sink.key)
	sink.close()
	if err != nil && cmd.Process == nil {
		cmd = exec.Command(args[0], args[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
		return editorResult(cmd, cmd.Run())
	}
	return nil
}
