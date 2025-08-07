//go:build !windows
// +build !windows

package exec_helpers

import (
	"log/slog"
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

	slog.Debug("Killing command", "cmd", cmd.Args, "pid", cmd.Process.Pid)

	// On Unix-like systems, we can use syscall.Kill to send a signal to the process group.
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		if err := cmd.Process.Kill(); err != nil {
			slog.Error("Failed to kill process group", "error", err, "pid", cmd.Process.Pid)
			// Handle error if needed
		}
	}
}
