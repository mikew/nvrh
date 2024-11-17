//go:build windows
// +build windows

package go_ssh_ext

import (
	"net"

	"github.com/Microsoft/go-winio"
)

func getConnectionForAgent(sshAuthSock string) (net.Conn, error) {
	return winio.DialPipe(sshAuthSock, nil)
}
