package nvim_helpers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"

	nvrh_context "nvrh/src/context"
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

func BuildRemoteCommandString(nvrhContext *nvrh_context.NvrhContext, ti *ssh_tunnel_info.SshTunnelInfo) string {
	if nvrhContext.ServerInfo.Os == "windows" {
		if nvrhContext.ServerInfo.ShellName == "powershell" || nvrhContext.ServerInfo.ShellName == "cmd" {
			return nvrhContext.WindowsLauncherPath
		} else {
			return fmt.Sprintf(`/tmp/`)
		}
	}

	envPairsString := ""
	if len(nvrhContext.RemoteEnv) > 0 {
		var formattedEnvPairs []string
		for _, envPair := range nvrhContext.RemoteEnv {
			if nvrhContext.ServerInfo.ShellName == "powershell" {
				// FOO=BAR -> $env:FOO='BAR';
				formattedEnvPairs = append(formattedEnvPairs, fmt.Sprintf("$env:%s", strings.Replace(envPair, "=", "='", 1)+"';"))
			} else if nvrhContext.ServerInfo.ShellName == "cmd" {
				// FOO=BAR -> set FOO=BAR&&
				formattedEnvPairs = append(formattedEnvPairs, fmt.Sprintf("set %s&&", envPair))
			} else {
				// FOO=BAR -> 'FOO=BAR'
				formattedEnvPairs = append(formattedEnvPairs, fmt.Sprintf("'%s'", envPair))
			}
		}
		envPairsString = strings.Join(formattedEnvPairs, " ")
	}

	nvimCmd := `"` + strings.Join(nvrhContext.NvimCmd, `" "`) + `"`
	if nvrhContext.ServerInfo.ShellName == "powershell" {
		nvimCmd = "& " + nvimCmd
	}

	return fmt.Sprintf(
		"%s %s --headless --listen \"%s\"",
		envPairsString,
		nvimCmd,
		ti.RemoteBoundToIp(),
	)
}
