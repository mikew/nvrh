package nvrh_internal_ssh

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"

	"golang.org/x/crypto/ssh"

	nvrh_context "nvrh/src/context"
	"nvrh/src/ssh_tunnel_info"
)

type NvrhInternalSshClient struct {
	Ctx       *nvrh_context.NvrhContext
	SshClient *ssh.Client
}

func (c *NvrhInternalSshClient) Close() error {
	if c.SshClient == nil {
		return fmt.Errorf("ssh client not initialized")
	}

	return c.SshClient.Close()
}

func (c *NvrhInternalSshClient) Run(ctx context.Context, command string, tunnelInfo *ssh_tunnel_info.SshTunnelInfo) error {
	if c.SshClient == nil {
		return fmt.Errorf("ssh client not initialized")
	}

	slog.Debug("Running command via SSH", "command", command)

	if tunnelInfo != nil {
		go c.TunnelSocket(ctx, tunnelInfo)
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

func (c *NvrhInternalSshClient) TunnelSocket(ctx context.Context, tunnelInfo *ssh_tunnel_info.SshTunnelInfo) {
	if c.SshClient == nil {
		return
	}

	// Listen on the local Unix socket
	localListener, err := LocalListenerFromTunnelInfo(tunnelInfo)
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
		remoteConn, err := RemoteListenerFromTunnelInfo(tunnelInfo, c.SshClient)
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

func LocalListenerFromTunnelInfo(ti *ssh_tunnel_info.SshTunnelInfo) (net.Listener, error) {
	switch ti.Mode {
	case "unix":
		return net.Listen("unix", ti.LocalSocket)
	case "port":
		ip := "localhost"
		if ti.Public {
			ip = "0.0.0.0"
		}

		return net.Listen("tcp", fmt.Sprintf("%s:%s", ip, ti.LocalSocket))
	}

	return nil, fmt.Errorf("Invalid mode: %s", ti.Mode)
}

func RemoteListenerFromTunnelInfo(ti *ssh_tunnel_info.SshTunnelInfo, sshClient *ssh.Client) (net.Conn, error) {
	switch ti.Mode {
	case "unix":
		return sshClient.Dial("unix", ti.RemoteSocket)
	case "port":
		return sshClient.Dial("tcp", fmt.Sprintf("localhost:%s", ti.RemoteSocket))
	}

	return nil, fmt.Errorf("Invalid mode: %s", ti.Mode)
}
