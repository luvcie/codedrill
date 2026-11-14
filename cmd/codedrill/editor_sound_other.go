//go:build !unix

package main

import (
	"errors"
	"os"
	"os/exec"
)

func runSoundEditor(args []string) error {
	if len(args) == 0 {
		return errors.New("missing editor")
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
