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

	LocalSocketPath  string
	RemoteSocketPath string
	ShouldUsePorts   bool
	LocalPortNumber  int
	RemotePortNumber int
	AutomapPorts     bool

	RemoteEnv   []string
	LocalEditor []string

	BrowserScriptPath string

	CommandsToKill []*exec.Cmd

	Debug bool

	SshPath   string
	SshClient nvrh_base_ssh.BaseNvrhSshClient
	SshArgs   []string

	NvimCmd []string

	TunneledPorts map[string]bool

	ServerInfo *NvrhServerInfo
}

type NvrhServerInfo struct {
	Os        string `json:"os"`
	Arch      string `json:"arch"`
	Username  string `json:"username"`
	Homedir   string `json:"homedir"`
	Tmpdir    string `json:"tmpdir"`
	ShellName string `json:"shell_name"`
}
