//go:build unix

package typingsound

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/creack/pty"
)

func TestEditorBridgeHelper(t *testing.T) {
	if os.Getenv("CODEDRILL_PTY_TEST") != "1" {
		return
	}
	cmd := exec.Command("sh", "-c", "stty raw -echo; printf READY; dd bs=1 count=1 2>/dev/null")
	if err := RunEditor(cmd, false, func(Event) {}); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}

func TestEditorBridgeForwardsInputAndExits(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestEditorBridgeHelper$")
	cmd.Env = append(os.Environ(), "CODEDRILL_PTY_TEST=1")
	terminal, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	result := make(chan error, 1)
	go func() {
		buf := make([]byte, 5)
		if _, err := io.ReadFull(terminal, buf); err != nil {
			result <- err
			return
		}
		if !bytes.Equal(buf, []byte("READY")) {
			result <- &unexpectedOutput{string(buf)}
			return
		}
		if _, err := terminal.Write([]byte("{")); err != nil {
			result <- err
			return
		}
		buf = buf[:1]
		_, err := io.ReadFull(terminal, buf)
		if err == nil && buf[0] != '{' {
			err = &unexpectedOutput{string(buf)}
		}
		result <- err
	}()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("PTY bridge did not forward input")
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("PTY bridge did not exit")
	}
}

// The bridge is created once per editor launch. This helper runs it twice in a
// row, as reopening an editor after closing it does, and requires each session
// to observe its key.
func TestEditorBridgeRepeatHelper(t *testing.T) {
	if os.Getenv("CODEDRILL_PTY_REPEAT_TEST") != "1" {
		return
	}
	for i := 0; i < 2; i++ {
		cmd := exec.Command("sh", "-c", "stty raw -echo; printf READY; dd bs=1 count=1 2>/dev/null")
		seen := 0
		if err := RunEditor(cmd, false, func(Event) { seen++ }); err != nil || seen != 1 {
			os.Exit(1)
		}
		os.Stdout.WriteString("DONE")
	}
	os.Exit(0)
}

func TestEditorBridgeEmitsEventsAcrossRepeatedSessions(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestEditorBridgeRepeatHelper$")
	cmd.Env = append(os.Environ(), "CODEDRILL_PTY_REPEAT_TEST=1")
	terminal, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	result := make(chan error, 1)
	go func() {
		for i := 0; i < 2; i++ {
			ready := make([]byte, 5)
			if _, err := io.ReadFull(terminal, ready); err != nil {
				result <- err
				return
			}
			if string(ready) != "READY" {
				result <- &unexpectedOutput{string(ready)}
				return
			}
			if _, err := terminal.Write([]byte("{")); err != nil {
				result <- err
				return
			}
			echo := make([]byte, 1)
			if _, err := io.ReadFull(terminal, echo); err != nil {
				result <- err
				return
			}
			if echo[0] != '{' {
				result <- &unexpectedOutput{string(echo)}
				return
			}
			done := make([]byte, 4)
			if _, err := io.ReadFull(terminal, done); err != nil {
				result <- err
				return
			}
			if string(done) != "DONE" {
				result <- &unexpectedOutput{string(done)}
				return
			}
		}
		result <- nil
	}()
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("repeated editor sessions did not forward input")
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("repeated bridge helper did not exit")
	}
}

type unexpectedOutput struct{ output string }

func (e *unexpectedOutput) Error() string { return "unexpected PTY output: " + e.output }
