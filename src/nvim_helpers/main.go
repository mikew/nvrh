package nvim_helpers

import (
	"time"

	"github.com/neovim/go-client/nvim"
)

func WaitForNvim(socketPath string) (*nvim.Nvim, error) {
	for {
		nv, err := nvim.Dial(socketPath)

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