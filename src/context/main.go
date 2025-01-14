package context

import (
	"fmt"
	"os/exec"

	"nvrh/src/nvrh_base_ssh"
	"nvrh/src/ssh_endpoint"
)

type NvrhContext struct {
	SessionId       string
	Endpoint        *ssh_endpoint.SshEndpoint
	RemoteDirectory string

	LocalSocketPath  string
	RemoteSocketPath string
	ShouldUsePorts   bool
	LocalPortNumber  int
	RemotePortNumber int

	RemoteEnv   []string
	LocalEditor []string

	BrowserScriptPath string

	CommandsToKill []*exec.Cmd

	SshPath string
	Debug   bool

	SshClient nvrh_base_ssh.BaseNvrhSshClient
}

func (nc *NvrhContext) LocalSocketOrPort() string {
	if nc.ShouldUsePorts {
		// nvim-qt, at least on Windows (and might have something to do with
		// running in a VM) seems to prefer `127.0.0.1` to `0.0.0.0`, and I think
		// that's safe on other OSes.
		return fmt.Sprintf("localhost:%d", nc.LocalPortNumber)
	}

	return nc.LocalSocketPath
}

func (nc *NvrhContext) RemoteSocketOrPort() string {
	if nc.ShouldUsePorts {
		return fmt.Sprintf("localhost:%d", nc.RemotePortNumber)
	}

	return nc.RemoteSocketPath
}
