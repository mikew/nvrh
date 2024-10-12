package client

import (
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/urfave/cli/v2"

	"nvrh/src/context"
	"nvrh/src/exec_helpers"
	"nvrh/src/nvim_helpers"
	"nvrh/src/ssh_helpers"
)

func defaultSshPath() string {
	if runtime.GOOS == "windows" {
		return "C:\\Windows\\System32\\OpenSSH\\ssh.exe"
	}

	return "ssh"
}

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
		&cli.StringFlag{
			Name:    "ssh-path",
			Usage:   "Path to SSH binary. Defaults to ssh on Unix, C:\\Windows\\System32\\OpenSSH\\ssh.exe on Windows",
			EnvVars: []string{"NVRH_CLIENT_SSH_PATH"},
			Value:   defaultSshPath(),
		},

		&cli.BoolFlag{
			Name:    "use-ports",
			Usage:   "Use ports instead of sockets. Defaults to true on Windows",
			EnvVars: []string{"NVRH_CLIENT_USE_PORTS"},
			Value:   runtime.GOOS == "windows",
		},

		&cli.StringSliceFlag{
			Name:  "server-env",
			Usage: "Environment variables to set on the remote server",
		},

		&cli.StringSliceFlag{
			Name:  "local-editor",
			Usage: "Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path",
			Value: cli.NewStringSlice("nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui"),
		},
	},

	Action: func(c *cli.Context) error {
		// Prepare the context.
		sessionId := fmt.Sprintf("%d", time.Now().Unix())
		nvrhContext := context.NvrhContext{
			SessionId:       sessionId,
			Server:          c.Args().Get(0),
			RemoteDirectory: c.Args().Get(1),

			RemoteEnv:   c.StringSlice("server-env"),
			LocalEditor: c.StringSlice("local-editor"),

			ShouldUsePorts: c.Bool("use-ports"),

			RemoteSocketPath: fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId),
			LocalSocketPath:  path.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s", sessionId)),

			BrowserScriptPath: fmt.Sprintf("/tmp/nvrh-browser-%s", sessionId),

			SshPath: c.String("ssh-path"),
		}

		if nvrhContext.ShouldUsePorts {
			min := 1025
			max := 65535
			nvrhContext.PortNumber = rand.IntN(max-min) + min
		}

		if nvrhContext.Server == "" {
			return fmt.Errorf("<server> is required")
		}

		var nv *nvim.Nvim

		doneChan := make(chan error)

		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)

		// Prepare remote instance.
		go func() {
			remoteCmd := ssh_helpers.BuildRemoteNvimCmd(&nvrhContext)
			nvrhContext.CommandsToKill = append(nvrhContext.CommandsToKill, remoteCmd)

			// We don't want the ssh process ending too early, if it does we can't
			// clean up the remote nvim instance.
			exec_helpers.PrepareForForking(remoteCmd)

			if err := remoteCmd.Run(); err != nil {
				log.Printf("Error running ssh: %v", err)
			} else {
				log.Printf("Remote nvim exited")
			}
		}()

		// Prepare client instance.
		nvChan := make(chan *nvim.Nvim, 1)
		go func() {
			nv, err := nvim_helpers.WaitForNvim(&nvrhContext)

			if err != nil {
				log.Printf("Error connecting to nvim: %v", err)
				return
			}

			log.Print("Connected to nvim")
			nvChan <- nv

			if err := prepareRemoteNvim(&nvrhContext, nv); err != nil {
				log.Printf("Error preparing remote nvim: %v", err)
			}

			clientCmd := BuildClientNvimCmd(&nvrhContext)
			nvrhContext.CommandsToKill = append(nvrhContext.CommandsToKill, clientCmd)

			if err := clientCmd.Run(); err != nil {
				log.Printf("Error running local editor: %v", err)
				doneChan <- err
			} else {
				log.Printf("Local editor exited")
				doneChan <- nil
			}
		}()

		nv = <-nvChan

		go func() {
			select {
			case sig := <-signalChan:
				doneChan <- fmt.Errorf("Received signal: %s", sig)
			}
		}()

		err := <-doneChan

		log.Printf("Closing nvrh")
		closeNvimSocket(nv)
		killAllCmds(nvrhContext.CommandsToKill)

		if err != nil {
			return err
		}

		return nil
	},
}

func BuildClientNvimCmd(nvrhContext *context.NvrhContext) *exec.Cmd {
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

	return editorCommand
}

func prepareRemoteNvim(nvrhContext *context.NvrhContext, nv *nvim.Nvim) error {
	nv.RegisterHandler("tunnel-port", ssh_helpers.MakeRpcTunnelHandler(nvrhContext))
	nv.RegisterHandler("open-url", RpcHandleOpenUrl)

	batch := nv.NewBatch()

	// Let nvim know the channel id so it can send us messages.
	batch.Command(fmt.Sprintf(`let $NVRH_CHANNEL_ID="%d"`, nv.ChannelID()))
	// Set $BROWSER so the remote machine can open a browser locally.
	batch.Command(fmt.Sprintf(`let $BROWSER="%s"`, nvrhContext.BrowserScriptPath))

	// Add command to tunnel port.
	// TODO use `vim.api.nvim_create_user_command`, and check to see if the
	// port is already mapped somehow.
	batch.ExecLua(`
vim.api.nvim_create_user_command(
	'NvrhTunnelPort',
	function(args)
		vim.rpcnotify(tonumber(os.getenv('NVRH_CHANNEL_ID')), 'tunnel-port', { args.args })
	end,
	{
		nargs = 1,
		force = true,
	}
)
return true
	`, nil, nil)

	// Add command to open url.
	batch.ExecLua(`
vim.api.nvim_create_user_command(
	'NvrhOpenUrl',
	function(args)
		vim.rpcnotify(tonumber(os.getenv('NVRH_CHANNEL_ID')), 'open-url', { args.args })
	end,
	{
		nargs = 1,
		force = true,
	}
)
	`, nil, nil)

	// Prepare the browser script.
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
	`, nil, nvrhContext.BrowserScriptPath, nvrhContext.RemoteSocketOrPort(), nv.ChannelID())






	if err := batch.Execute(); err != nil {
		return err
	}

	return nil
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
		exec.Command("cmd", "/c", "start", url).Run()
	} else {
		log.Printf("Don't know how to open url on %s", goos)
	}
}

func killAllCmds(cmds []*exec.Cmd) {
	for _, cmd := range cmds {
		log.Printf("Killing command: %v", cmd)
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				log.Printf("Error killing command: %v", err)
			}
		}
	}
}

func closeNvimSocket(nv *nvim.Nvim) {
	if nv == nil {
		return
	}

	log.Print("Closing nvim")
	if err := nv.ExecLua("vim.cmd('qall!')", nil, nil); err != nil {
		log.Printf("Error closing remote nvim: %v", err)
	}
	nv.Close()
}
