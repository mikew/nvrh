package nvrh_binary_ssh

import (
	"context"
	"log/slog"
	"os"
	"os/exec"

	nvrh_context "nvrh/src/context"
	"nvrh/src/exec_helpers"
	"nvrh/src/ssh_tunnel_info"
)

type NvrhBinarySshClient struct {
	Ctx *nvrh_context.NvrhContext
}

func (c *NvrhBinarySshClient) Close() error {
	return nil
}

func (c *NvrhBinarySshClient) Run(ctx context.Context, command string, tunnelInfo *ssh_tunnel_info.SshTunnelInfo) error {
	args := []string{}

	if tunnelInfo != nil {
		args = append(args, "-L", tunnelInfo.BoundToIp())
	}

	if len(c.Ctx.SshArgs) > 0 {
		args = append(args, c.Ctx.SshArgs...)
	}

	args = append(args, "-t", c.Ctx.Endpoint.Given, "--", command)

	slog.Debug("Running command via SSH", "command", command)

	sshCommand := exec.CommandContext(
		ctx,
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

func (c *NvrhBinarySshClient) TunnelSocket(ctx context.Context, tunnelInfo *ssh_tunnel_info.SshTunnelInfo) {
	sshCommand := exec.CommandContext(
		ctx,
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

	defer exec_helpers.Kill(sshCommand)

	if err := sshCommand.Wait(); err != nil {
		return
	}
}
