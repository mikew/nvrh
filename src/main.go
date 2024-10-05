package main

import (
	// "errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "nvrh",
		Usage: "Helps work with a remote nvim instance",

		Commands: []*cli.Command{
			{
				Name: "client",

				Subcommands: []*cli.Command{
					{
						Name:      "open",
						Usage:     "Open a remote nvim instance in a local editor",
						Category:  "client",
						Args:      true,
						ArgsUsage: "<server> [directory]",

						Flags: []cli.Flag{
							&cli.StringSliceFlag{
								Name:  "server-env",
								Usage: "Environment variables to set on the remote server",
							},

							&cli.StringSliceFlag{
								Name:  "local-editor",
								Usage: "Local editor to use. {{SOCKET_PATH}}",
							},
						},

						Action: func(c *cli.Context) error {
							sessionId := time.Now().Unix()
							socketPath := fmt.Sprintf("/tmp/nvrh-socket-%d", sessionId)
							browserScriptPath := fmt.Sprintf("/tmp/nvrh-browser-%d", sessionId)
							server := c.Args().Get(0)
							directory := c.Args().Get(1)

							serverEnvPairs := c.StringSlice("server-env")
							log.Printf("serverEnvPairs: %v", serverEnvPairs)

							localEditor := c.StringSlice("server-env")
							log.Printf("localEditor: %v", localEditor)

							if server == "" {
								return fmt.Errorf("<server> is required")
							}

							go startRemoteNvim(server, socketPath, directory, serverEnvPairs)

							go func() {
								nv, err := waitForNvim(socketPath)

								if err != nil {
									log.Printf("Error connecting to nvim: %v", err)
									return
								}

								defer func() {
									log.Print("Closing nvim")
									nv.Close()
								}()

								nv.RegisterHandler("tunnel-port", makeTunnelHandler(server))
								nv.RegisterHandler("open-url", handleOpenUrl)

								batch := nv.NewBatch()

								// Let nvim know the channel id so it can send us messages.
								batch.Command(fmt.Sprintf(`let $NVRH_CHANNEL_ID="%d"`, nv.ChannelID()))
								// Set $BROWSER so the remote machine can open a browser locally.
								batch.Command(fmt.Sprintf(`let $BROWSER="%s"`, browserScriptPath))

								// Add command to tunnel port.
								batch.Command("command! -nargs=1 NvrhTunnelPort call rpcnotify(str2nr($NVRH_CHANNEL_ID), 'tunnel-port', [<f-args>])")
								// Add command to open url.
								batch.Command("command! -nargs=1 NvrhOpenUrl call rpcnotify(str2nr($NVRH_CHANNEL_ID), 'open-url', [<f-args>])")

								// Prepare the browser script.
								var output any
								batch.ExecLua(`
local browser_script_path, socket_path, channel_id = ...

local script_contents = [[
#!/bin/sh

SOCKET_PATH="%s"
CHANNEL_ID="%s"

exec nvim --server "$SOCKET_PATH" --remote-expr "rpcnotify(str2nr($CHANNEL_ID), 'open-url', ['$1'])" > /dev/null
]]
script_contents = string.format(script_contents, socket_path, channel_id)

vim.fn.writefile(vim.fn.split(script_contents, '\n'), browser_script_path)
os.execute('chmod +x ' .. browser_script_path)

return true
								`, &output, browserScriptPath, socketPath, nv.ChannelID())

								if err := batch.Execute(); err != nil {
									log.Fatalf("Error while preparing remote nvim: %v", err)
								}

								log.Print("Connected to nvim")
								startLocalEditor(socketPath, localEditor)
							}()

							select {}
						},
					},
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func startRemoteNvim(server string, socketPath string, directory string, envPairs []string) {
	nvimCommand := buildRemoteCommand(socketPath, directory, envPairs)
	log.Printf("Starting remote nvim: %s", nvimCommand)

	sshCommand := exec.Command(
		"ssh",
		"-L",
		fmt.Sprintf("%s:%s", socketPath, socketPath),
		server,
		"-t",
		// TODO Not really sure if this is better than piping it as exampled
		// below.
		fmt.Sprintf("$SHELL -i -c '%s'", nvimCommand),
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
		log.Printf("Error starting command: %v", err)
		return
	}

	defer sshCommand.Process.Kill()

	if err := sshCommand.Wait(); err != nil {
		log.Printf("Error waiting for command: %v", err)
	}
}

func startLocalEditor(socketPath string, args []string) {
	replacedArgs := make([]string, len(args))
	for i, arg := range args {
		replacedArgs[i] = strings.Replace(arg, "{{SOCKET_PATH}}", socketPath, -1)
	}

	if len(replacedArgs) == 0 {
		replacedArgs = []string{"nvim", "--server", socketPath, "--remote-ui"}
	}

	log.Printf("Starting local editor: %v", replacedArgs)

	// editorCommand := exec.Command("nvim-qt", "--server", socketPath)
	editorCommand := exec.Command(replacedArgs[0], replacedArgs[1:]...)
	if replacedArgs[0] == "nvim" {
		editorCommand.Stdin = os.Stdin
		editorCommand.Stdout = os.Stdout
		editorCommand.Stderr = os.Stderr
	}

	if err := editorCommand.Run(); err != nil {
		log.Printf("Error running editor: %v", err)
		return
	}
}

func makeTunnelHandler(server string) func(*nvim.Nvim, []string) {
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

func handleOpenUrl(v *nvim.Nvim, args []string) {
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

func waitForNvim(socketPath string) (*nvim.Nvim, error) {
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

func buildRemoteCommand(socketPath string, directory string, envPairs []string) string {
	envPairsString := ""
	if len(envPairs) > 0 {
		envPairsString = strings.Join(envPairs, " ")
	}

	return fmt.Sprintf(
		"%s nvim --headless --listen \"%s\" --cmd \"cd %s\"",
		envPairsString,
		socketPath,
		directory,
	)
}
