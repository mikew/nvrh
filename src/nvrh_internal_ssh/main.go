package nvrh_internal_ssh

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"

	nvrhcontext "nvrh/src/context"
	"nvrh/src/ssh_tunnel_info"
)

type NvrhInternalSshClient struct {
	Ctx       *nvrhcontext.NvrhContext
	SshClient *ssh.Client
	tunnelCtx context.Context
	cancelTunnel context.CancelFunc
	tunnelMutex sync.Mutex
}

func (c *NvrhInternalSshClient) Close() error {
	c.tunnelMutex.Lock()
	defer c.tunnelMutex.Unlock()
	
	// Cancel any active tunnels
	if c.cancelTunnel != nil {
		c.cancelTunnel()
	}
	
	if c.SshClient == nil {
		return fmt.Errorf("ssh client not initialized")
	}

	return c.SshClient.Close()
}

func (c *NvrhInternalSshClient) Run(command string, tunnelInfo *ssh_tunnel_info.SshTunnelInfo) error {
	if c.SshClient == nil {
		return fmt.Errorf("ssh client not initialized")
	}

	slog.Debug("Running command via SSH", "command", command)

	if tunnelInfo != nil {
		go c.TunnelSocket(tunnelInfo)
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
	c.TunnelSocketWithTimeout(tunnelInfo, 30*time.Second, 3)
}

// TunnelSocketWithTimeout creates an SSH tunnel with automatic cleanup after timeout or repeated errors
func (c *NvrhInternalSshClient) TunnelSocketWithTimeout(tunnelInfo *ssh_tunnel_info.SshTunnelInfo, timeout time.Duration, maxErrors int) {
	if c.SshClient == nil {
		slog.Error("SSH client not initialized")
		return
	}

	c.tunnelMutex.Lock()
	c.tunnelCtx, c.cancelTunnel = context.WithTimeout(context.Background(), timeout)
	ctx := c.tunnelCtx
	cancel := c.cancelTunnel
	c.tunnelMutex.Unlock()
	
	defer cancel()

	errorCount := 0
	
	for errorCount < maxErrors {
		select {
		case <-ctx.Done():
			slog.Warn("SSH tunnel timeout reached", "timeout", timeout)
			return
		default:
		}

		// Listen on the local socket
		localListener, err := LocalListenerFromTunnelInfo(tunnelInfo)
		if err != nil {
			slog.Error("Failed to listen on local socket", "error", err, "attempt", errorCount+1)
			errorCount++
			if errorCount < maxErrors {
				time.Sleep(2 * time.Second)
				continue
			}
			break
		}

		// Clean up local socket file
		defer func() {
			localListener.Close()
			if tunnelInfo.Mode == "unix" {
				os.Remove(tunnelInfo.LocalSocket)
			}
		}()

		slog.Info("Tunneling SSH socket", "tunnelInfo", tunnelInfo, "timeout", timeout, "attempt", errorCount+1)

		// Accept connections with timeout
		connectionErrors := 0
		for {
			select {
			case <-ctx.Done():
				slog.Warn("SSH tunnel context cancelled", "reason", ctx.Err())
				return
			default:
			}

			// Set a deadline for accepting connections
			if tcpListener, ok := localListener.(*net.TCPListener); ok {
				tcpListener.SetDeadline(time.Now().Add(1 * time.Second))
			} else if unixListener, ok := localListener.(*net.UnixListener); ok {
				unixListener.SetDeadline(time.Now().Add(1 * time.Second))
			}

			localConn, err := localListener.Accept()
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					// Timeout is expected, continue the loop
					continue
				}
				slog.Error("Failed to accept connection", "error", err)
				connectionErrors++
				if connectionErrors >= 5 {
					slog.Error("Too many connection errors, restarting listener")
					localListener.Close()
					errorCount++
					break
				}
				continue
			}

			// Reset connection error count on successful accept
			connectionErrors = 0

			// Establish a connection to the remote socket via SSH
			remoteConn, err := RemoteListenerFromTunnelInfo(tunnelInfo, c.SshClient)
			if err != nil {
				slog.Error("Failed to dial remote socket", "error", err)
				localConn.Close()
				continue
			}

			// Start a goroutine to handle the connection with context
			go c.handleConnectionWithContext(ctx, localConn, remoteConn)
		}

		if errorCount >= maxErrors {
			break
		}
		
		time.Sleep(2 * time.Second) // Wait before retry
	}

	slog.Error("SSH tunnel failed after maximum attempts", "max_errors", maxErrors, "timeout", timeout)
}

// handleConnectionWithContext handles a connection with context cancellation support
func (c *NvrhInternalSshClient) handleConnectionWithContext(ctx context.Context, localConn net.Conn, remoteConn net.Conn) {
	defer localConn.Close()
	defer remoteConn.Close()

	// Create a context that gets cancelled when the parent context is cancelled
	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Channel to signal when copying is done
	done := make(chan struct{}, 2)

	// Copy data from local to remote
	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(remoteConn, localConn)
	}()

	// Copy data from remote to local
	go func() {
		defer func() { done <- struct{}{} }()
		io.Copy(localConn, remoteConn)
	}()

	// Wait for either context cancellation or connection completion
	select {
	case <-connCtx.Done():
		slog.Debug("Connection cancelled due to context")
		return
	case <-done:
		// One direction finished, wait for the other or timeout
		select {
		case <-done:
			slog.Debug("Connection completed normally")
		case <-time.After(5 * time.Second):
			slog.Debug("Connection cleanup timeout")
		case <-connCtx.Done():
			slog.Debug("Connection cancelled during cleanup")
		}
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
