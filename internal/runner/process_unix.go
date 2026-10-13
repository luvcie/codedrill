//go:build unix

package runner

import (
	"os/exec"
	"syscall"
)

func configureTestProcess(cmd *exec.Cmd) {
	// Killing only bash leaves its children holding the output pipes open.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
