package client

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/urfave/cli/v2"

	"nvrh/src/context"
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
			Usage: "Local editor to use. `{{SOCKET_PATH}}` will be replaced with the socket path",
			Value: cli.NewStringSlice("nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui"),
		},
	},

	Action: func(c *cli.Context) error {
		sessionId := fmt.Sprintf("%d", time.Now().Unix())
		nvrhContext := context.NvrhContext{
			SessionId:       sessionId,
			Server:          c.Args().Get(0),
			RemoteDirectory: c.Args().Get(1),

			RemoteEnv:   c.StringSlice("server-env"),
			LocalEditor: c.StringSlice("local-editor"),


			RemoteSocketPath: fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId),
			LocalSocketPath:  path.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s", sessionId)),

			BrowserScriptPath: fmt.Sprintf("/tmp/nvrh-browser-%s", sessionId),
		}

		if nvrhContext.Server == "" {
			return fmt.Errorf("<server> is required")
		}

		go ssh_helpers.StartRemoteNvim(nvrhContext)

		go func() {
			nv, err := nvim_helpers.WaitForNvim(nvrhContext)

			if err != nil {
				log.Printf("Error connecting to nvim: %v", err)
				return
			}

			defer func() {
				log.Print("Closing nvim")
				nv.Close()
			}()

			nv.RegisterHandler("tunnel-port", ssh_helpers.MakeRpcTunnelHandler(nvrhContext.Server))
			nv.RegisterHandler("open-url", RpcHandleOpenUrl)

			batch := nv.NewBatch()

			// Let nvim know the channel id so it can send us messages.
			batch.Command(fmt.Sprintf(`let $NVRH_CHANNEL_ID="%d"`, nv.ChannelID()))
			// Set $BROWSER so the remote machine can open a browser locally.
			batch.Command(fmt.Sprintf(`let $BROWSER="%s"`, nvrhContext.BrowserScriptPath))

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
			`, &output, nvrhContext.BrowserScriptPath, nvrhContext.LocalSocketOrPort(), nv.ChannelID())

			if err := batch.Execute(); err != nil {
				log.Fatalf("Error while preparing remote nvim: %v", err)
			}

			log.Print("Connected to nvim")
			startLocalEditor(nvrhContext)
		}()

		select {}
	},
}

func startLocalEditor(nvrhContext context.NvrhContext) {
	replacedArgs := make([]string, len(nvrhContext.LocalEditor))
	for i, arg := range nvrhContext.LocalEditor {
		replacedArgs[i] = strings.Replace(arg, "{{SOCKET_PATH}}", nvrhContext.LocalSocketOrPort(), -1)
	}

	log.Printf("Starting local editor: %v", replacedArgs)

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

func RpcHandleOpenUrl(v *nvim.Nvim, args []string) {
	goos := runtime.GOOS
	url := args[0]

	if url == "" || !(strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")) {
		log.Printf("Invalid url: %s", url)
		return
	}

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
