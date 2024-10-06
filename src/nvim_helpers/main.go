package nvim_helpers

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
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

func HandleOpenUrl(v *nvim.Nvim, args []string) {
	goos := runtime.GOOS
	url := args[0]

	// if url == "" || !strings.HasPrefix(url, "http://") || !strings.HasPrefix(url, "https://") {
	// 	log.Printf("Invalid url: %s", url)
	// 	return
	// }

	log.Printf("Opening url: %s", url)

	if goos == "darwin" {
		exec.Command("open", url).Run()
	} else if goos == "linux" {
		exec.Command("xdg-open", url).Run()
	} else if goos == "windows" {
		exec.Command("start", "", url).Run()
	} else {
		log.Printf("Don't know how to open url on %s", goos)
	}
}

func MakeTunnelHandler(server string) func(*nvim.Nvim, []string) {
	return func(v *nvim.Nvim, args []string) {
		go func() {
			log.Printf("Tunneling %s:%s", server, args[0])

			sshCommand := exec.Command(
				"ssh",
				"-NL",
				fmt.Sprintf("%s:0.0.0.0:%s", args[0], args[0]),
				server,
			)

			if err := sshCommand.Start(); err != nil {
				log.Printf("Error starting command: %v", err)
				return
			}

			defer sshCommand.Process.Kill()

			if err := sshCommand.Wait(); err != nil {
				log.Printf("Error waiting for command: %v", err)
			}
		}()
	}
}
