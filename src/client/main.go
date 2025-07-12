package client

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/urfave/cli/v2"

	"nvrh/src/context"
	"nvrh/src/go_ssh_ext"
	"nvrh/src/logger"
	"nvrh/src/nvim_helpers"
	"nvrh/src/nvrh_base_ssh"
	"nvrh/src/nvrh_binary_ssh"
	"nvrh/src/nvrh_internal_ssh"
	"nvrh/src/ssh_endpoint"
	"nvrh/src/ssh_tunnel_info"
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
		&CliClientReconnectCommand,
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
			Usage:   "Path to SSH binary. 'binary' will use the default system SSH binary. 'internal' will use the internal SSH client. Anything else will be used as the path to the SSH binary",
			EnvVars: []string{"NVRH_CLIENT_SSH_PATH"},
			Value:   "binary",
		},

		&cli.BoolFlag{
			Name:    "use-ports",
			Usage:   "Use ports instead of sockets. Defaults to true on Windows",
			EnvVars: []string{"NVRH_CLIENT_USE_PORTS"},
			Value:   runtime.GOOS == "windows",
		},

		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "",
			EnvVars: []string{"NVRH_CLIENT_DEBUG"},
		},

		&cli.StringSliceFlag{
			Name:  "server-env",
			Usage: "Environment variables to set on the remote server",
		},

		&cli.StringSliceFlag{
			Name:    "local-editor",
			Usage:   "Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path",
			EnvVars: []string{"NVRH_CLIENT_LOCAL_EDITOR"},
			Value:   cli.NewStringSlice("nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui"),
		},
	},

	Action: func(c *cli.Context) error {
		isDebug := c.Bool("debug")
		logger.PrepareLogger(isDebug)

		// Prepare the context.
		server := c.Args().Get(0)
		if server == "" {
			return fmt.Errorf("<server> is required")
		}

		sessionId := fmt.Sprintf("%d", time.Now().Unix())
		sshPath := c.String("ssh-path")
		if sshPath == "binary" {
			sshPath = defaultSshPath()
		}

		endpoint, endpointErr := ssh_endpoint.ParseSshEndpoint(server)
		if endpointErr != nil {
			return endpointErr
		}

		nvrhContext := &context.NvrhContext{
			SessionId:       sessionId,
			Endpoint:        endpoint,
			RemoteDirectory: c.Args().Get(1),

			RemoteEnv:   c.StringSlice("server-env"),
			LocalEditor: c.StringSlice("local-editor"),

			ShouldUsePorts: c.Bool("use-ports"),

			RemoteSocketPath: fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId),
			LocalSocketPath:  filepath.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s", sessionId)),

			BrowserScriptPath: fmt.Sprintf("/tmp/nvrh-browser-%s", sessionId),

			SshPath: sshPath,
			Debug:   isDebug,
		}

		if nvrhContext.SshPath == "internal" {
			sshClient, err := go_ssh_ext.GetSshClientForEndpoint(endpoint)
			if err != nil {
				return err
			}

			nvrhContext.SshClient = nvrh_base_ssh.BaseNvrhSshClient(&nvrh_internal_ssh.NvrhInternalSshClient{
				Ctx:       nvrhContext,
				SshClient: sshClient,
			})
		} else {
			nvrhContext.SshClient = nvrh_base_ssh.BaseNvrhSshClient(&nvrh_binary_ssh.NvrhBinarySshClient{
				Ctx: nvrhContext,
			})
		}

		if nvrhContext.ShouldUsePorts {
			min := 1025
			max := 65535
			randomPort := rand.IntN(max-min) + min
			nvrhContext.LocalPortNumber = randomPort
			nvrhContext.RemotePortNumber = randomPort
		}

		var nv *nvim.Nvim

		doneChan := make(chan error)

		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)

		// Prepare remote instance.
		go func() {
			tunnelInfo := &ssh_tunnel_info.SshTunnelInfo{
				Mode:         "unix",
				LocalSocket:  nvrhContext.LocalSocketPath,
				RemoteSocket: nvrhContext.RemoteSocketPath,
				Public:       false,
			}

			if nvrhContext.ShouldUsePorts {
				tunnelInfo.Mode = "port"
				tunnelInfo.LocalSocket = fmt.Sprintf("%d", nvrhContext.LocalPortNumber)
				tunnelInfo.RemoteSocket = fmt.Sprintf("%d", nvrhContext.RemotePortNumber)
			}

			nvimCommandString := nvim_helpers.BuildRemoteCommandString(nvrhContext)
			nvimCommandString = fmt.Sprintf("$SHELL -i -c 'cd \"%s\" && %s'", nvrhContext.RemoteDirectory, nvimCommandString)
			slog.Info("Starting remote nvim", "nvimCommandString", nvimCommandString)

			nvrhContext.SshClient.Run(nvimCommandString, tunnelInfo)
		}()

		// Prepare client instance.
		nvChan := make(chan *nvim.Nvim, 1)
		go func() {
			nv, err := nvim_helpers.WaitForNvim(nvrhContext)

			if err != nil {
				slog.Error("Error connecting to nvim", "err", err)
				return
			}

			slog.Info("Connected to nvim")
			nvChan <- nv

			if err := prepareRemoteNvim(nvrhContext, nv); err != nil {
				slog.Warn("Error preparing remote nvim", "err", err)
			}

			clientCmd := BuildClientNvimCmd(nvrhContext)
			if nvrhContext.Debug {
				clientCmd.Stdout = os.Stdout
				clientCmd.Stderr = os.Stderr
				// clientCmd.Stdin = os.Stdin
			}

			nvrhContext.CommandsToKill = append(nvrhContext.CommandsToKill, clientCmd)

			if err := clientCmd.Start(); err != nil {
				slog.Error("Error starting local editor", "err", err)
				doneChan <- err
				return
			}

			if err := clientCmd.Wait(); err != nil {
				slog.Error("Error running local editor", "err", err)
				doneChan <- err
			} else {
				slog.Info("Local editor exited")
				doneChan <- nil
			}
		}()

		go func() {
			sig := <-signalChan
			slog.Debug("Received signal", "signal", sig)
			doneChan <- fmt.Errorf("Received signal: %s", sig)
		}()

		nv = <-nvChan

		err := <-doneChan

		slog.Info("Closing nvrh")
		closeNvimSocket(nv)
		killAllCmds(nvrhContext.CommandsToKill)
		os.Remove(nvrhContext.LocalSocketPath)
		if nvrhContext.SshClient != nil {
			nvrhContext.SshClient.Run(fmt.Sprintf("rm -f '%s'", nvrhContext.RemoteSocketPath), nil)
			nvrhContext.SshClient.Run(fmt.Sprintf("rm -f '%s'", nvrhContext.BrowserScriptPath), nil)

			nvrhContext.SshClient.Close()
		}

		if err != nil {
			return err
		}

		return nil
	},
}

var CliClientReconnectCommand = cli.Command{
	Name:      "reconnect",
	Usage:     "Reconnect to an existing remote nvim instance",
	Category:  "client",
	Args:      true,
	ArgsUsage: "<server> <session-id>",

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "ssh-path",
			Usage:   "Path to SSH binary. 'binary' will use the default system SSH binary. 'internal' will use the internal SSH client. Anything else will be used as the path to the SSH binary",
			EnvVars: []string{"NVRH_CLIENT_SSH_PATH"},
			Value:   "binary",
		},

		&cli.BoolFlag{
			Name:    "use-ports",
			Usage:   "Use ports instead of sockets. Defaults to true on Windows",
			EnvVars: []string{"NVRH_CLIENT_USE_PORTS"},
			Value:   runtime.GOOS == "windows",
		},

		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "",
			EnvVars: []string{"NVRH_CLIENT_DEBUG"},
		},

		&cli.StringSliceFlag{
			Name:    "local-editor",
			Usage:   "Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path",
			EnvVars: []string{"NVRH_CLIENT_LOCAL_EDITOR"},
			Value:   cli.NewStringSlice("nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui"),
		},
	},

	Action: func(c *cli.Context) error {
		isDebug := c.Bool("debug")
		logger.PrepareLogger(isDebug)

		// Prepare the context.
		server := c.Args().Get(0)
		if server == "" {
			return fmt.Errorf("<server> is required")
		}

		sessionId := c.Args().Get(1)
		if sessionId == "" {
			return fmt.Errorf("<session-id> is required")
		}

		sshPath := c.String("ssh-path")
		if sshPath == "binary" {
			sshPath = defaultSshPath()
		}

		endpoint, endpointErr := ssh_endpoint.ParseSshEndpoint(server)
		if endpointErr != nil {
			return endpointErr
		}

		randomId := fmt.Sprintf("%d", time.Now().Unix())

		nvrhContext := &context.NvrhContext{
			SessionId:         sessionId,
			Endpoint:          endpoint,
			LocalEditor:       c.StringSlice("local-editor"),
			ShouldUsePorts:    c.Bool("use-ports"),
			RemoteSocketPath:  fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId),
			LocalSocketPath:   filepath.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s-%s", sessionId, randomId)),
			BrowserScriptPath: fmt.Sprintf("/tmp/nvrh-browser-%s", sessionId),
			SshPath:           sshPath,
			Debug:             isDebug,
		}

		portNumberString := c.Args().Get(2)
		portNumber := 0
		if portNumberString != "" {
			portNumber, _ = strconv.Atoi(portNumberString)
		}

		if nvrhContext.SshPath == "internal" {
			sshClient, err := go_ssh_ext.GetSshClientForEndpoint(endpoint)
			if err != nil {
				return err
			}

			nvrhContext.SshClient = nvrh_base_ssh.BaseNvrhSshClient(&nvrh_internal_ssh.NvrhInternalSshClient{
				Ctx:       nvrhContext,
				SshClient: sshClient,
			})
		} else {
			nvrhContext.SshClient = nvrh_base_ssh.BaseNvrhSshClient(&nvrh_binary_ssh.NvrhBinarySshClient{
				Ctx: nvrhContext,
			})
		}

		if nvrhContext.ShouldUsePorts {
			min := 1025
			max := 65535
			randomPort := rand.IntN(max-min) + min
			nvrhContext.LocalPortNumber = randomPort
			if portNumber != 0 {
				nvrhContext.RemotePortNumber = portNumber
			} else {
				nvrhContext.RemotePortNumber = randomPort

			}
		}

		var nv *nvim.Nvim
		doneChan := make(chan error)
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)

		// Setup SSH tunnel
		go func() {
			tunnelInfo := &ssh_tunnel_info.SshTunnelInfo{
				Mode:         "unix",
				LocalSocket:  nvrhContext.LocalSocketPath,
				RemoteSocket: nvrhContext.RemoteSocketPath,
				Public:       false,
			}

			if nvrhContext.ShouldUsePorts {
				tunnelInfo.Mode = "port"
				tunnelInfo.LocalSocket = fmt.Sprintf("%d", nvrhContext.LocalPortNumber)
				tunnelInfo.RemoteSocket = fmt.Sprintf("%d", nvrhContext.RemotePortNumber)
			}

			nvrhContext.SshClient.TunnelSocket(tunnelInfo)
		}()

		// Connect client instance
		nvChan := make(chan *nvim.Nvim, 1)
		go func() {
			nv, err := nvim_helpers.WaitForNvim(nvrhContext)

			if err != nil {
				slog.Error("Error connecting to nvim", "err", err)
				doneChan <- err
				return
			}

			slog.Info("Connected to nvim")
			nvChan <- nv

			if err := prepareRemoteNvim(nvrhContext, nv); err != nil {
				slog.Warn("Error preparing remote nvim", "err", err)
			}

			clientCmd := BuildClientNvimCmd(nvrhContext)
			if nvrhContext.Debug {
				clientCmd.Stdout = os.Stdout
				clientCmd.Stderr = os.Stderr
			}

			nvrhContext.CommandsToKill = append(nvrhContext.CommandsToKill, clientCmd)

			if err := clientCmd.Start(); err != nil {
				slog.Error("Error starting local editor", "err", err)
				doneChan <- err
				return
			}

			if err := clientCmd.Wait(); err != nil {
				slog.Error("Error running local editor", "err", err)
				doneChan <- err
			} else {
				slog.Info("Local editor exited")
				doneChan <- nil
			}
		}()

		go func() {
			sig := <-signalChan
			slog.Debug("Received signal", "signal", sig)
			doneChan <- fmt.Errorf("received signal: %s", sig)
		}()

		nv = <-nvChan

		err := <-doneChan

		slog.Info("Closing nvrh")
		closeNvimSocket(nv)
		killAllCmds(nvrhContext.CommandsToKill)
		os.Remove(nvrhContext.LocalSocketPath)
		if nvrhContext.SshClient != nil {
			nvrhContext.SshClient.Close()
		}

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

	slog.Info("Starting local editor", "cmd", replacedArgs)

	editorCommand := exec.Command(replacedArgs[0], replacedArgs[1:]...)
	if replacedArgs[0] == "nvim" {
		editorCommand.Stdin = os.Stdin
		editorCommand.Stdout = os.Stdout
		editorCommand.Stderr = os.Stderr
	}

	return editorCommand
}

func prepareRemoteNvim(nvrhContext *context.NvrhContext, nv *nvim.Nvim) error {
	nv.RegisterHandler("tunnel-port", func(v *nvim.Nvim, args []string) {
		go nvrhContext.SshClient.TunnelSocket(&ssh_tunnel_info.SshTunnelInfo{
			Mode:         "port",
			LocalSocket:  args[0],
			RemoteSocket: args[0],
			Public:       true,
		})
	})
	nv.RegisterHandler("open-url", RpcHandleOpenUrl)

	batch := nv.NewBatch()

	// Set $NVRH_SESSION_ID so we can identify the session.
	batch.Command(fmt.Sprintf(`let $NVRH_SESSION_ID="%s"`, nvrhContext.SessionId))
	// Let nvim know the channel id so it can send us messages.
	batch.Command(fmt.Sprintf(`let $NVRH_CHANNEL_ID="%d"`, nv.ChannelID()))
	// Set $BROWSER so the remote machine can open a browser locally.
	batch.Command(fmt.Sprintf(`let $BROWSER="%s"`, nvrhContext.BrowserScriptPath))

	// Add command to tunnel port.
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
		slog.Error("Invalid url", "url", url)
		return
	}

	slog.Info("Opening url", "url", url)

	if goos == "darwin" {
		exec.Command("open", url).Run()
	} else if goos == "linux" {
		exec.Command("xdg-open", url).Run()
	} else if goos == "windows" {
		exec.Command("cmd", "/c", "start", url).Run()
	} else {
		slog.Error("Don't know how to open url", "url", url)
	}
}

func killAllCmds(cmds []*exec.Cmd) {
	for _, cmd := range cmds {
		slog.Debug("Killing command", "cmd", cmd.Args)
		if cmd.Process != nil {
			if err := cmd.Process.Kill(); err != nil {
				slog.Warn("Error killing command", "err", err)
			}
		}
	}
}

func closeNvimSocket(nv *nvim.Nvim) {
	if nv == nil {
		return
	}

	slog.Info("Closing nvim")
	if err := nv.ExecLua("vim.cmd('qall!')", nil, nil); err != nil {
		slog.Warn("Error closing remote nvim", "err", err)
	}
	nv.Close()
}
