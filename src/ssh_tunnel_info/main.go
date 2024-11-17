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

func (ti *SshTunnelInfo) BoundToIp() string {
	if ti.Mode == "unix" {
		return fmt.Sprintf("%s:%s", ti.LocalSocket, ti.RemoteSocket)
	}

	ip := "localhost"
	if ti.Public {
		ip = "0.0.0.0"
	}

	return fmt.Sprintf("%s:%s:%s", ti.LocalSocket, ip, ti.RemoteSocket)
}
