//go:build !windows
// +build !windows

package exec_helpers

import (
	"os/exec"
	"syscall"
)

func PrepareForForking(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func Kill(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	// On Unix-like systems, we can use syscall.Kill to send a signal to the process group.
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		// Handle error if needed
	}
}
