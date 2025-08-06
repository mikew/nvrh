package nvrh_base_ssh

import (
	"context"

	"nvrh/src/ssh_tunnel_info"
)

type BaseNvrhSshClient interface {
	Run(ctx context.Context, command string, tunnelInfo *ssh_tunnel_info.SshTunnelInfo) error
	TunnelSocket(ctx context.Context, tunnelInfo *ssh_tunnel_info.SshTunnelInfo)
	Close() error
}
