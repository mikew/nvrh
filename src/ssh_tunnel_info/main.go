package ssh_tunnel_info

import (
	"fmt"
)

type SshTunnelInfo struct {
	Mode              string
	LocalSocket       string
	RemoteSocket      string
	Public            bool
	DirectConnectHost string
}

func (ti *SshTunnelInfo) LocalBoundToIp() string {
	if ti.Mode == "unix" {
		return ti.LocalSocket
	}

	ip := "localhost"
	if ti.Public {
		ip = "0.0.0.0"
	}

	if ti.DirectConnectHost != "" {
		return fmt.Sprintf("%s:%s", ti.DirectConnectHost, ti.LocalSocket)
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

	if ti.DirectConnectHost != "" {
		return fmt.Sprintf("%s:%s", "0.0.0.0", ti.LocalSocket)
	}

	return fmt.Sprintf("%s:%s", ip, ti.RemoteSocket)
}

func (ti *SshTunnelInfo) SwitchToPorts(localPort int, remotePort int) {
	ti.Mode = "port"
	ti.LocalSocket = fmt.Sprintf("%d", localPort)
	ti.RemoteSocket = fmt.Sprintf("%d", remotePort)
}

func (ti *SshTunnelInfo) SwitchToSockets(localSocket string, remoteSocket string) {
	ti.Mode = "unix"
	ti.LocalSocket = localSocket
	ti.RemoteSocket = remoteSocket
}
