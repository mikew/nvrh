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

	AutomapPorts bool

	CommandsToKill []*exec.Cmd

	Debug bool

	SshClient nvrh_base_ssh.BaseNvrhSshClient

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
