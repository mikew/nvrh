package nvim_helpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"

	"nvrh/src/ssh_tunnel_info"
)

func WaitForNvim(ctx context.Context, ti *ssh_tunnel_info.SshTunnelInfo) (*nvim.Nvim, error) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(10 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()

		case <-timeout:
			return nil, errors.New("Timed out waiting for nvim")

		case <-ticker.C:
			nv, err := nvim.Dial(ti.LocalBoundToIp())
			if err != nil {
				continue
			}

			if _, err := nv.APIInfo(); err != nil {
				continue
			}

			return nv, nil
		}
	}
}

func BuildRemoteCommandString(
	nvimCmd []string,
	shellName string,
	remoteDirectory string,
	remoteEnv []string,
	ti *ssh_tunnel_info.SshTunnelInfo,
) string {
	nvimCmdWithAdditions := append(nvimCmd, "--headless", "--listen", ti.RemoteBoundToIp())
	nvimCmdQuoted := `"` + strings.Join(nvimCmdWithAdditions, `" "`) + `"`

	switch shellName {
	case "powershell":
		parts := []string{}

		if remoteDirectory != "" {
			parts = append(parts, fmt.Sprintf(
				"cd '%s'",
				remoteDirectory,
			))
		}

		if len(remoteEnv) > 0 {
			parts = append(parts, BuildRemoteEnvString(remoteEnv, shellName))
		}

		parts = append(parts, fmt.Sprintf(
			"& %s",
			nvimCmdQuoted,
		))

		return strings.Join(parts, "; ")

	case "cmd":
		parts := []string{}

		if remoteDirectory != "" {
			parts = append(parts, fmt.Sprintf(
				`cd /d "%s"`,
				remoteDirectory,
			))
		}

		if len(remoteEnv) > 0 {
			parts = append(parts, BuildRemoteEnvString(remoteEnv, shellName))
		}

		parts = append(parts, nvimCmdQuoted)

		return strings.Join(parts, " && ")

	case "bat":
		parts := []string{
			"@echo off",
		}

		if remoteDirectory != "" {
			parts = append(parts, fmt.Sprintf(
				`cd /d "%s"`,
				remoteDirectory,
			))
		}

		if len(remoteEnv) > 0 {
			parts = append(parts, BuildRemoteEnvString(remoteEnv, shellName))
		}

		parts = append(parts, fmt.Sprintf(`start "" /WAIT %s`, nvimCmdQuoted))

		return strings.Join(parts, "\n\n")
	}

	parts := []string{}

	if remoteDirectory != "" {
		parts = append(parts, fmt.Sprintf(
			`cd "%s"`,
			remoteDirectory,
		))
	}

	parts = append(parts, fmt.Sprintf(
		"%s %s",
		BuildRemoteEnvString(remoteEnv, shellName),
		nvimCmdQuoted,
	))

	return fmt.Sprintf(
		`exec "$SHELL" -i -c '%s'`,
		strings.Join(parts, " && "),
	)
}

func BuildRemoteEnvString(envPairs []string, shellName string) string {
	if len(envPairs) == 0 {
		return ""
	}

	switch shellName {
	case "powershell":
		var formattedEnvPairs []string

		for _, envPair := range envPairs {
			// FOO=BAR -> $env:FOO='BAR'
			parts := strings.SplitN(envPair, "=", 2)

			// Skip malformed env pairs
			if len(parts) != 2 {
				continue
			}

			formattedEnvPairs = append(formattedEnvPairs, fmt.Sprintf("$env:%s='%s'", parts[0], parts[1]))
		}

		return strings.Join(formattedEnvPairs, "; ")

	case "cmd":
		var formattedEnvPairs []string

		for _, envPair := range envPairs {
			// FOO=BAR -> set FOO='BAR'
			parts := strings.SplitN(envPair, "=", 2)

			// Skip malformed env pairs
			if len(parts) != 2 {
				continue
			}

			formattedEnvPairs = append(formattedEnvPairs, fmt.Sprintf("set %s='%s'", parts[0], parts[1]))
		}

		return strings.Join(formattedEnvPairs, " && ")

	case "bat":
		var formattedEnvPairs []string

		for _, envPair := range envPairs {
			// FOO=BAR -> set FOO='BAR'
			parts := strings.SplitN(envPair, "=", 2)

			// Skip malformed env pairs
			if len(parts) != 2 {
				continue
			}

			formattedEnvPairs = append(formattedEnvPairs, fmt.Sprintf("set %s='%s'", parts[0], parts[1]))
		}

		return strings.Join(formattedEnvPairs, "\n")
	}

	var formattedEnvPairs []string

	for _, envPair := range envPairs {
		// FOO=BAR -> FOO="BAR"
		parts := strings.SplitN(envPair, "=", 2)

		// Skip malformed env pairs
		if len(parts) != 2 {
			continue
		}

		formattedEnvPairs = append(formattedEnvPairs, fmt.Sprintf(`%s="%s"`, parts[0], parts[1]))
	}

	return strings.Join(formattedEnvPairs, " ")
}
