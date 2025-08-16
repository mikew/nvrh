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

	timeout := time.After(10 * time.Second) // optional timeout
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
	envPairsString := ""
	if len(nvrhContext.RemoteEnv) > 0 {
		envPairsString = strings.Join(nvrhContext.RemoteEnv, " ")
	}

	nvimCmd := strings.Join(nvrhContext.NvimCmd, " ")

	return fmt.Sprintf(
		"%s %s --headless --listen \"%s\"",
		envPairsString,
		nvimCmd,
		ti.RemoteBoundToIp(),
	)
}
