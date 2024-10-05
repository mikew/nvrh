package main

import (
	// "errors"
	"fmt"
	"os"
	"os/exec"
	// "strings"
	"time"

	"github.com/neovim/go-client/nvim"
)

func main() {
	sessionId := time.Now().Unix()
	socketPath := fmt.Sprintf("/tmp/nvim-socket-%d", sessionId)
	server := os.Args[1]
	directory := os.Args[2]

	go startRemoteNvim(server, socketPath, directory)

	go func() {
		nv, err := waitForNvim(socketPath)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error connecting to nvim: %v\n", err)
			return
		}

		defer func() {
			fmt.Println("Closing nvim")
			nv.Close()
		}()

		nv.RegisterHandler("tunnel-port", makeTunnelHandler(server))

		batch := nv.NewBatch()

		// Let nvim know the channel id so it can send us messages.
		batch.Command(fmt.Sprintf("let $NVIM_REMOTE_HELPER_CHANNEL_ID=%d", nv.ChannelID()))
		// Set $BROWSER so the remote machine can open a browser locally.
		// TODO Actually get this script to work.
		batch.Command(fmt.Sprintf("let $BROWSER='/tmp/nvim-remote-helper-browser-%d'", sessionId))

		// Add NvimRemoteHelperTunnelPort command to nvim.
		batch.Command("command! -nargs=1 NvimRemoteHelperTunnelPort call rpcnotify(str2nr($NVIM_REMOTE_HELPER_CHANNEL_ID), 'tunnel-port', [<f-args>])")

		if err := batch.Execute(); err != nil {
			panic(err)
		}

		fmt.Println("Connected to nvim")
		startLocalEditor(socketPath)
	}()

	select {}
}

func startRemoteNvim(server string, socketPath string, directory string) {
	sshCommand := exec.Command(
		"ssh",
		"-L",
		fmt.Sprintf("%s:%s", socketPath, socketPath),
		server,
		"-t",
		// TODO Not really sure if this is better than piping it as exampled
		// below.
		fmt.Sprintf("$SHELL -i -c '%s'", buildRemoteCommand(socketPath, directory)),
	)

	// sshCommand.Stdout = os.Stdout
	// sshCommand.Stderr = os.Stderr
	// sshCommand.Stdin = os.Stdin

	// Create a pipe to write to the command's stdin
	// stdinPipe, err := sshCommand.StdinPipe()
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error creating stdin pipe: %v\n", err)
	// 	return
	// }
	// Write the predetermined string to the pipe
	// command := buildRemoteCommand(socketPath, directory)
	// if _, err := stdinPipe.Write([]byte(command)); err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error writing to stdin pipe: %v\n", err)
	// 	return
	// }
	// Close the pipe after writing
	// stdinPipe.Close()

	if err := sshCommand.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", err)
		return
	}

	defer sshCommand.Process.Kill()

	if err := sshCommand.Wait(); err != nil {
		fmt.Fprintf(os.Stderr, "Error waiting for command: %v\n", err)
	}
}

func startLocalEditor(socketPath string) {
	editorCommand := exec.Command("nvim-qt", "--server", socketPath)
	// editorCommand.Stdout = os.Stdout
	// editorCommand.Stderr = os.Stderr

	if err := editorCommand.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running editor: %v\n", err)
		return
	}
}

func makeTunnelHandler(server string) func(*nvim.Nvim, []string) {
	return func(v *nvim.Nvim, args []string) {
		go func() {
			fmt.Printf("Tunneling %s:%s\n", server, args[0])

			sshCommand := exec.Command(
				"ssh",
				"-NL",
				fmt.Sprintf("%s:0.0.0.0:%s", args[0], args[0]),
				server,
			)

			if err := sshCommand.Start(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting command: %v\n", err)
				return
			}

			defer sshCommand.Process.Kill()

			if err := sshCommand.Wait(); err != nil {
				fmt.Fprintf(os.Stderr, "Error waiting for command: %v\n", err)
			}
		}()
	}
}

func waitForNvim(socketPath string) (*nvim.Nvim, error) {
	for {
		nv, err := nvim.Dial(socketPath)

		if err == nil {
			_, err := nv.APIInfo()

			if err == nil {
				return nv, nil
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	// return nil, errors.New("Timed out waiting for nvim")
}

func buildRemoteCommand(socketPath string, directory string) string {
	return fmt.Sprintf(
		"NVIM_FORCE_OS=macos NVIM_FORCE_UI=nvim-qt nvim --headless --listen \"%s\" --cmd \"cd %s\"",
		socketPath,
		directory,
	)
}
