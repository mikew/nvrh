package nvrh_base_ssh

import (
	"nvrh/src/ssh_tunnel_info"
)

type BaseNvrhSshClient interface {
	Run(command string) error
	TunnelSocket(tunnelInfo *ssh_tunnel_info.SshTunnelInfo)
	Close() error
}
