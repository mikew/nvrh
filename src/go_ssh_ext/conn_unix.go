//go:build !windows
// +build !windows

package go_ssh_ext

import (
	"net"
)

func getConnectionForAgent(sshAuthSock string) (net.Conn, error) {
	return net.Dial("unix", sshAuthSock)
}
