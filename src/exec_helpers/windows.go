//go:build windows
// +build windows

package exec_helpers

import (
	"log/slog"
	"os/exec"
)

func PrepareForForking(cmd *exec.Cmd) {
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	// 	Setpgid: true,
	// }
}

func Kill(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}

	slog.Debug("Killing command", "cmd", cmd.Args, "pid", cmd.Process.Pid)

	// On Windows, we can use the Process.Kill method directly.
	if err := cmd.Process.Kill(); err != nil {
		// Handle error if needed
	}
}
