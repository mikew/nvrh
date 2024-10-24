package nvim_helpers

import (
	"fmt"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"

	"nvrh/src/context"
)

func WaitForNvim(nvrhContext *context.NvrhContext) (*nvim.Nvim, error) {
	for {
		nv, err := nvim.Dial(nvrhContext.LocalSocketOrPort())

		if err == nil {
			// TODO Can probably trim down the data passed over the wire by
			// using another method.
			_, err := nv.APIInfo()

			if err == nil {
				return nv, nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	// return nil, errors.New("Timed out waiting for nvim")
}

func BuildRemoteCommandString(nvrhContext *context.NvrhContext) string {
	envPairsString := ""
	if len(nvrhContext.RemoteEnv) > 0 {
		envPairsString = strings.Join(nvrhContext.RemoteEnv, " ")
	}

	return fmt.Sprintf(
		"%s nvim --headless --listen \"%s\" --cmd \"cd %s\"",
		envPairsString,
		nvrhContext.RemoteSocketOrPort(),
		nvrhContext.RemoteDirectory,
	)
}
