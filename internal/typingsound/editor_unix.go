//go:build unix

package typingsound

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// RunEditor connects an editor to a PTY and observes its input. emit must not
// block; input bytes are sent to the editor before they are observed.
func RunEditor(cmd *exec.Cmd, wordCompletion bool, emit func(Event)) error {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return fmt.Errorf("editor input is not a terminal")
	}
	state, err := term.MakeRaw(fd)
	if err != nil {
		return err
	}
	defer term.Restore(fd, state)

	window, err := pty.GetsizeFull(os.Stdin)
	if err != nil {
		return err
	}
	master, err := pty.StartWithSize(cmd, window)
	if err != nil {
		return err
	}
	defer master.Close()

	resize := make(chan os.Signal, 1)
	signal.Notify(resize, syscall.SIGWINCH)
	defer signal.Stop(resize)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-resize:
				_ = pty.InheritSize(os.Stdin, master)
			case <-done:
				return
			}
		}
	}()
	outputDone := make(chan struct{})
	go func() {
		_, _ = io.Copy(os.Stdout, master)
		close(outputDone)
	}()
	inputDone := make(chan struct{})
	go func() {
		defer close(inputDone)
		input := NewInput(wordCompletion)
		buf := make([]byte, 4096)
		for {
			select {
			case <-done:
				return
			default:
			}
			fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
			ready, pollErr := unix.Poll(fds, 100)
			if pollErr == unix.EINTR || ready == 0 && pollErr == nil {
				continue
			}
			if pollErr != nil {
				return
			}
			select {
			case <-done:
				return
			default:
			}
			n, err := unix.Read(fd, buf)
			if n > 0 {
				for written := 0; written < n; {
					count, writeErr := master.Write(buf[written:n])
					if writeErr != nil || count == 0 {
						return
					}
					written += count
				}
				input.Observe(buf[:n], emit)
			}
			if err != nil {
				return
			}
		}
	}()
	err = cmd.Wait()
	close(done)
	<-inputDone
	select {
	case <-outputDone:
	case <-time.After(250 * time.Millisecond):
		// A background descendant can keep the PTY slave open after the editor exits.
		_ = master.Close()
		<-outputDone
	}
	return err
}
