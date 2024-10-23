package nvrh_binary_ssh

import (
	"log/slog"
	"os"
	"os/exec"

	"nvrh/src/context"
	"nvrh/src/ssh_tunnel_info"
)

type NvrhBinarySshClient struct {
	Ctx *context.NvrhContext
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
	sshCommand := exec.Command(
		c.Ctx.SshPath,
		"-NL",
		tunnelInfo.BoundToIp(),
		c.Ctx.Endpoint.Given,
	)

	slog.Info("Tunneling SSH socket", "tunnelInfo", tunnelInfo)

	c.Ctx.CommandsToKill = append(c.Ctx.CommandsToKill, sshCommand)
	if c.Ctx.Debug {
		sshCommand.Stdout = os.Stdout
		sshCommand.Stderr = os.Stderr
	}

	if err := sshCommand.Start(); err != nil {
		return
	}

	defer sshCommand.Process.Kill()

	if err := sshCommand.Wait(); err != nil {
		return
	}
}
