//go:build windows
// +build windows

package exec_helpers

import (
	"os/exec"
)

func PrepareForForking(cmd *exec.Cmd) {
	// cmd.SysProcAttr = &syscall.SysProcAttr{
	// 	Setpgid: true,
	// }
}