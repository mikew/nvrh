package ssh_helpers

import (
	"fmt"
	"io"
	"log/slog"
	"net"
	"nvrh/src/context"
	"os"
	"os/exec"
	"strings"

	"github.com/neovim/go-client/nvim"
	"golang.org/x/crypto/ssh"
)

func BuildRemoteNvimCmd(nvrhContext *context.NvrhContext) *exec.Cmd {
	nvimCommandString := BuildRemoteCommandString(nvrhContext)
	slog.Info("Starting remote nvim", "nvimCommandString", nvimCommandString)

	tunnel := fmt.Sprintf("%s:%s", nvrhContext.LocalSocketPath, nvrhContext.RemoteSocketPath)
	if nvrhContext.ShouldUsePorts {
		tunnel = fmt.Sprintf("%d:127.0.0.1:%d", nvrhContext.PortNumber, nvrhContext.PortNumber)
	}

	sshCommand := exec.Command(
		nvrhContext.SshPath,
		"-L",
		tunnel,
		"-t",
		nvrhContext.Server,
		// TODO Not really sure if this is better than piping it as exampled
		// below.
		fmt.Sprintf("$SHELL -i -c '%s'", nvimCommandString),
	)

	// Create a pipe to write to the command's stdin
	// stdinPipe, err := sshCommand.StdinPipe()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error creating stdin pipe: %v\n", err)
	// 	return
	// }
	// Write the predetermined string to the pipe
	// command := buildRemoteCommand(socketPath, directory)
	// if _, err := stdinPipe.Write([]byte(command)); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error writing to stdin pipe: %v\n", err)
	// 	return
	// }
	// Close the pipe after writing
	// stdinPipe.Close()

	return sshCommand
}

func BuildRemoteCommandString(nvrhContext *context.NvrhContext) string {
	envPairsString := ""
	if len(nvrhContext.RemoteEnv) > 0 {
		envPairsString = strings.Join(nvrhContext.RemoteEnv, " ")
	}

	return fmt.Sprintf(
		"%s nvim --headless --listen \"%s\" --cmd \"cd %s\"; rm -f \"%s\"; [ %t = true ] && rm -f \"%s\"",
		envPairsString,
		nvrhContext.RemoteSocketOrPort(),
		nvrhContext.RemoteDirectory,
		nvrhContext.BrowserScriptPath,
		!nvrhContext.ShouldUsePorts,
		nvrhContext.RemoteSocketPath,
	)
}

func MakeRpcTunnelHandler(nvrhContext *context.NvrhContext) func(*nvim.Nvim, []string) {
	return func(v *nvim.Nvim, args []string) {
		go TunnelSshSocket(nvrhContext, SshTunnelInfo{
			Mode:         "port",
			LocalSocket:  fmt.Sprintf("%s", args[0]),
			RemoteSocket: fmt.Sprintf("%s", args[0]),
		})
	}
}

type SshTunnelInfo struct {
	Mode         string
	LocalSocket  string
	RemoteSocket string
}

func (ti SshTunnelInfo) LocalListener() (net.Listener, error) {
	switch ti.Mode {
	case "unix":
		return net.Listen("unix", ti.LocalSocket)
	case "port":
		return net.Listen("tcp", fmt.Sprintf("localhost:%s", ti.LocalSocket))
	}

	return nil, fmt.Errorf("Invalid mode: %s", ti.Mode)
}

func (ti SshTunnelInfo) RemoteListener(sshClient *ssh.Client) (net.Conn, error) {
	switch ti.Mode {
	case "unix":
		return sshClient.Dial("unix", ti.RemoteSocket)
	case "port":
		return sshClient.Dial("tcp", fmt.Sprintf("localhost:%s", ti.RemoteSocket))
	}

	return nil, fmt.Errorf("Invalid mode: %s", ti.Mode)
}

func TunnelSshSocket(nvrhContext *context.NvrhContext, tunnelInfo SshTunnelInfo) {
	// Listen on the local Unix socket
	localListener, err := tunnelInfo.LocalListener()
	if err != nil {
		slog.Error("Failed to listen on local socket", "err", err)
		return
	}

	defer localListener.Close()

	defer func() {
		// Clean up local socket file
		os.Remove(tunnelInfo.LocalSocket)
	}()

	slog.Info("Tunneling SSH socket", "LocalSocke", tunnelInfo.LocalSocket, "RemoteSocket", tunnelInfo.RemoteSocket)

	for {
		// Accept incoming connections
		localConn, err := localListener.Accept()
		if err != nil {
			slog.Error("Failed to accept connection", "err", err)
			continue
		}

		// Establish a connection to the remote socket via SSH
		remoteConn, err := tunnelInfo.RemoteListener(nvrhContext.SshClient)
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

func RunCommand(nvrhContext *context.NvrhContext, command string) error {
	session, err := nvrhContext.SshClient.NewSession()

	if err != nil {
		slog.Error("Failed to create session", "err", err)
		return err
	}

	defer session.Close()

	if nvrhContext.Debug {
		session.Stdout = os.Stdout
		session.Stderr = os.Stderr
	}

	if err := session.Run(command); err != nil {
		slog.Error("Failed to run command", "err", err)
		return err
	}

	return nil
}
