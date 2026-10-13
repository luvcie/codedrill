//go:build !unix

package runner

import "os/exec"

// Other platforms retain CommandContext's direct-process cancellation.
func configureTestProcess(cmd *exec.Cmd) {}
