package context

import (
	"os/exec"

	"nvrh/src/nvrh_base_ssh"
	"nvrh/src/ssh_endpoint"
)

type NvrhContext struct {
	SessionId       string
	Endpoint        *ssh_endpoint.SshEndpoint
	RemoteDirectory string

	// Deprecated: Might not need to be on the context.
	LocalSocketPath string
	// Deprecated: Might not need to be on the context.
	RemoteSocketPath string
	// Deprecated: Might not need to be on the context.
	ShouldUsePorts bool
	// Deprecated: Might not need to be on the context.
	LocalPortNumber int
	// Deprecated: Might not need to be on the context.
	RemotePortNumber int

	AutomapPorts bool

	RemoteEnv   []string
	LocalEditor []string

	CommandsToKill []*exec.Cmd

	Debug bool

	SshClient nvrh_base_ssh.BaseNvrhSshClient
	// Deprecated: Might not need to be on the context.
	SshArgs []string

	NvimCmd []string

	TunneledPorts map[string]bool

	ServerInfo *NvrhServerInfo

	WindowsLauncherPath string
}

type NvrhServerInfo struct {
	Os        string `json:"os"`
	Arch      string `json:"arch"`
	Username  string `json:"username"`
	Homedir   string `json:"homedir"`
	Tmpdir    string `json:"tmpdir"`
	ShellName string `json:"shell_name"`
}
