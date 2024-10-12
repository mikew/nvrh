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
