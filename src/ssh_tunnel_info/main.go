package ssh_tunnel_info

import (
	"fmt"
)

type SshTunnelInfo struct {
	Mode         string
	LocalSocket  string
	RemoteSocket string
	Public       bool
}

func (ti *SshTunnelInfo) LocalBoundToIp() string {
	if ti.Mode == "unix" {
		return ti.LocalSocket
	}

	ip := "localhost"
	if ti.Public {
		ip = "0.0.0.0"
	}

	return fmt.Sprintf("%s:%s", ip, ti.LocalSocket)
}

func (ti *SshTunnelInfo) RemoteBoundToIp() string {
	if ti.Mode == "unix" {
		return ti.RemoteSocket
	}

	ip := "localhost"
	if ti.Public {
		ip = "0.0.0.0"
	}

	return fmt.Sprintf("%s:%s", ip, ti.RemoteSocket)
}
