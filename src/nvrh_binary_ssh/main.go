package nvrh_binary_ssh

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"

	nvrhcontext "nvrh/src/context"
	"nvrh/src/ssh_tunnel_info"
)

type NvrhBinarySshClient struct {
	Ctx *nvrhcontext.NvrhContext
}

func (c *NvrhBinarySshClient) Close() error {
	return nil
}

func (c *NvrhBinarySshClient) Run(command string, tunnelInfo *ssh_tunnel_info.SshTunnelInfo) error {
	args := []string{}

	if tunnelInfo != nil {
		args = append(args, "-L", tunnelInfo.BoundToIp())
	}

	args = append(args, "-t", c.Ctx.Endpoint.Given, command)

	slog.Debug("Running command via SSH", "command", command)

	sshCommand := exec.Command(
		c.Ctx.SshPath,
		args...,
	)

	c.Ctx.CommandsToKill = append(c.Ctx.CommandsToKill, sshCommand)
	if c.Ctx.Debug {
		sshCommand.Stdout = os.Stdout
		sshCommand.Stderr = os.Stderr
	}

	if err := sshCommand.Start(); err != nil {
		return err
	}

	if err := sshCommand.Wait(); err != nil {
		return err
	}

	return nil
}

func (c *NvrhBinarySshClient) TunnelSocket(tunnelInfo *ssh_tunnel_info.SshTunnelInfo) {
	c.TunnelSocketWithTimeout(tunnelInfo, 30*time.Second, 3)
}

// TunnelSocketWithTimeout creates an SSH tunnel with automatic cleanup after timeout or repeated errors
func (c *NvrhBinarySshClient) TunnelSocketWithTimeout(tunnelInfo *ssh_tunnel_info.SshTunnelInfo, timeout time.Duration, maxErrors int) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	errorCount := 0
	
	for errorCount < maxErrors {
		sshCommand := exec.CommandContext(ctx,
			c.Ctx.SshPath,
			"-NL",
			tunnelInfo.BoundToIp(),
			c.Ctx.Endpoint.Given,
		)

		slog.Info("Tunneling SSH socket", "tunnelInfo", tunnelInfo, "timeout", timeout, "attempt", errorCount+1)

		c.Ctx.CommandsToKill = append(c.Ctx.CommandsToKill, sshCommand)
		if c.Ctx.Debug {
			sshCommand.Stdout = os.Stdout
			sshCommand.Stderr = os.Stderr
		}

		if err := sshCommand.Start(); err != nil {
			slog.Error("Failed to start SSH tunnel", "error", err, "attempt", errorCount+1)
			errorCount++
			time.Sleep(1 * time.Second) // Brief delay before retry
			continue
		}

		// Monitor for context cancellation or command completion
		done := make(chan error, 1)
		go func() {
			done <- sshCommand.Wait()
		}()

		select {
		case <-ctx.Done():
			slog.Warn("SSH tunnel timeout reached, killing process", "timeout", timeout)
			if sshCommand.Process != nil {
				sshCommand.Process.Kill()
			}
			return
		case err := <-done:
			if err != nil {
				slog.Error("SSH tunnel process exited with error", "error", err, "attempt", errorCount+1)
				errorCount++
				if errorCount < maxErrors {
					slog.Info("Retrying SSH tunnel", "remaining_attempts", maxErrors-errorCount)
					time.Sleep(2 * time.Second) // Longer delay before retry on error
				}
			} else {
				slog.Info("SSH tunnel process completed successfully")
				return
			}
		}
	}

	slog.Error("SSH tunnel failed after maximum attempts", "max_errors", maxErrors, "timeout", timeout)
}
