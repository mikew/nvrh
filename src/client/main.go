package client

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"nvrh/src/nvim_helpers"
	"nvrh/src/ssh_helpers"
)

var CliClientCommand = cli.Command{
	Name: "client",

	Subcommands: []*cli.Command{
		&CliClientOpenCommand,
	},
}

var CliClientOpenCommand = cli.Command{
	Name:      "open",
	Usage:     "Open a remote nvim instance in a local editor",
	Category:  "client",
	Args:      true,
	ArgsUsage: "<server> [remote-directory]",

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
		localEditor := c.StringSlice("server-env")

		if server == "" {
			return fmt.Errorf("<server> is required")
		}

		go ssh_helpers.StartRemoteNvim(server, socketPath, directory, serverEnvPairs)

		go func() {
			nv, err := nvim_helpers.WaitForNvim(socketPath)

			if err != nil {
				log.Printf("Error connecting to nvim: %v", err)
				return
			}

			defer func() {
				log.Print("Closing nvim")
				nv.Close()
			}()

			nv.RegisterHandler("tunnel-port", nvim_helpers.MakeTunnelHandler(server))
			nv.RegisterHandler("open-url", nvim_helpers.HandleOpenUrl)

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
