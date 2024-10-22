package nvrh_internal_ssh

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	"golang.org/x/crypto/ssh"

	"nvrh/src/context"
	"nvrh/src/ssh_tunnel_info"
)

type NvrhInternalSshClient struct {
	Ctx       *context.NvrhContext
	SshClient *ssh.Client
}

func (c *NvrhInternalSshClient) Close() error {
	if c.SshClient == nil {
		return fmt.Errorf("ssh client not initialized")
	}

	return c.SshClient.Close()
}

func (c *NvrhInternalSshClient) Run(command string) error {
	if c.SshClient == nil {
		return fmt.Errorf("ssh client not initialized")
	}

	session, err := c.SshClient.NewSession()

	if err != nil {
		return err
	}

	defer session.Close()

	if c.Ctx.Debug {
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
	}

	if err := session.Run(command); err != nil {
		return err
	}

	return nil
}

func (c *NvrhInternalSshClient) TunnelSocket(tunnelInfo *ssh_tunnel_info.SshTunnelInfo) {
	if c.SshClient == nil {
		return
	}

	// Listen on the local Unix socket
	localListener, err := tunnelInfo.LocalListener(tunnelInfo.Public)
	if err != nil {
		slog.Error("Failed to listen on local socket", "err", err)
		return
	}

	defer localListener.Close()

	// Clean up local socket file
	defer func() {
		if tunnelInfo.Mode == "unix" {
			os.Remove(tunnelInfo.LocalSocket)
		}
	}()

	slog.Info("Tunneling SSH socket", "tunnelInfo", tunnelInfo)

	for {
		// Accept incoming connections
		localConn, err := localListener.Accept()
		if err != nil {
			slog.Error("Failed to accept connection", "err", err)
			continue
		}

		// Establish a connection to the remote socket via SSH
		remoteConn, err := tunnelInfo.RemoteListener(c.SshClient)
		if err != nil {
			slog.Error("Failed to dial remote socket", "err", err)
			localConn.Close()
			continue
		}

		// Start a goroutine to handle the connection
		go handleConnection(localConn, remoteConn)
	}

}

func handleConnection(localConn net.Conn, remoteConn net.Conn) {
	// Close connections when done
	defer localConn.Close()
	defer remoteConn.Close()

	// Copy data from local to remote
	go io.Copy(remoteConn, localConn)
	// Copy data from remote to local
	io.Copy(localConn, remoteConn)
}
