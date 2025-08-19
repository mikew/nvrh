package nvrh_binary_ssh

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"

	"nvrh/src/context"
	"nvrh/src/ssh_tunnel_info"
)

type NvrhBinarySshClient struct {
	Ctx     *context.NvrhContext
	SshPath string
	SshArgs []string
}

func (c *NvrhBinarySshClient) Close() error {
	return nil
}

func (c *NvrhBinarySshClient) Run(command string, tunnelInfo *ssh_tunnel_info.SshTunnelInfo) error {
	args := []string{}

	if tunnelInfo != nil {
		args = append(args, "-L", bindTunnelInfo(tunnelInfo))
	}

	if len(c.SshArgs) > 0 {
		args = append(args, c.SshArgs...)
	}

	args = append(args, "-t", c.Ctx.Endpoint.Given, "--", command)

	slog.Debug("Running command via SSH", "command", command)

	sshCommand := exec.Command(
		c.SshPath,
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
		c.SshPath,
		"-nNTL",
		bindTunnelInfo(tunnelInfo),
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

	if err := sshCommand.Wait(); err != nil {
		return
	}
}

func bindTunnelInfo(ti *ssh_tunnel_info.SshTunnelInfo) string {
	if ti == nil {
		return ""
	}

	if ti.Mode == "unix" {
		return fmt.Sprintf("%s:%s", ti.LocalSocket, ti.RemoteSocket)
	}

	ip := "localhost"
	if ti.Public {
		ip = "0.0.0.0"
	}

	return fmt.Sprintf("%s:%s:%s", ti.LocalSocket, ip, ti.RemoteSocket)
}
