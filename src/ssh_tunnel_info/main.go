package ssh_tunnel_info

import (
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

type SshTunnelInfo struct {
	Mode         string
	LocalSocket  string
	RemoteSocket string
	Public       bool
}

func (ti *SshTunnelInfo) LocalListener(public bool) (net.Listener, error) {
	switch ti.Mode {
	case "unix":
		return net.Listen("unix", ti.LocalSocket)
	case "port":
		ip := "localhost"
		if public {
			ip = "0.0.0.0"
		}

		return net.Listen("tcp", fmt.Sprintf("%s:%s", ip, ti.LocalSocket))
	}

	return nil, fmt.Errorf("Invalid mode: %s", ti.Mode)
}

func (ti *SshTunnelInfo) RemoteListener(sshClient *ssh.Client) (net.Conn, error) {
	switch ti.Mode {
	case "unix":
		return sshClient.Dial("unix", ti.RemoteSocket)
	case "port":
		return sshClient.Dial("tcp", fmt.Sprintf("localhost:%s", ti.RemoteSocket))
	}

	return nil, fmt.Errorf("Invalid mode: %s", ti.Mode)
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
