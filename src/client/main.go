package client

import (
	"context"
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

	nvrh_context "nvrh/src/context"
	"nvrh/src/exec_helpers"
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
			Name:    "server-env",
			Usage:   "Environment variables to set on the remote server",
			EnvVars: []string{"NVRH_CLIENT_SERVER_ENV"},
		},

		&cli.StringSliceFlag{
			Name:    "local-editor",
			Usage:   "Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path",
			EnvVars: []string{"NVRH_CLIENT_LOCAL_EDITOR"},
			Value:   cli.NewStringSlice("nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui"),
		},

		&cli.StringSliceFlag{
			Name:    "nvim-cmd",
			Usage:   "Command to run nvim with. Defaults to `nvim`",
			EnvVars: []string{"NVRH_CLIENT_NVIM_CMD"},
			Value:   cli.NewStringSlice("nvim"),
		},

		&cli.StringSliceFlag{
			Name:    "ssh-arg",
			Usage:   "Additional arguments to pass to the SSH command",
			EnvVars: []string{"NVRH_CLIENT_SSH_ARG"},
		},

		&cli.BoolFlag{
			Name:    "enable-automap-ports",
			Usage:   "Enable automatic port mapping",
			EnvVars: []string{"NVRH_CLIENT_AUTOMAP_PORTS"},
			Value:   true,
		},
	},

	Action: func(c *cli.Context) error {
		isDebug := c.Bool("debug")
		logger.PrepareLogger(isDebug)

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

		// Context with cancellation on SIGINT
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		done := make(chan error, 1)

		// Prepare the context.
		nvrhContext := &nvrh_context.NvrhContext{
			SessionId:       sessionId,
			Endpoint:        endpoint,
			RemoteDirectory: c.Args().Get(1),

			RemoteEnv:   c.StringSlice("server-env"),
			LocalEditor: c.StringSlice("local-editor"),

			ShouldUsePorts: c.Bool("use-ports"),

			RemoteSocketPath: fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId),
			LocalSocketPath:  filepath.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s", sessionId)),
			AutomapPorts:     c.Bool("enable-automap-ports"),

			BrowserScriptPath: fmt.Sprintf("/tmp/nvrh-browser-%s", sessionId),

			SshPath: sshPath,
			Debug:   isDebug,

			TunneledPorts: make(map[string]bool),

			NvimCmd: c.StringSlice("nvim-cmd"),

			SshArgs: c.StringSlice("ssh-arg"),
		}

		// Setup SSH client
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

		// Cleanup on exit
		defer func() {
			slog.Info("Cleaning up")
			closeNvimSocket(nv, true)
			killAllCmds(nvrhContext.CommandsToKill)
			os.Remove(nvrhContext.LocalSocketPath)
			if nvrhContext.SshClient != nil {
				if nv != nil {
					nvrhContext.SshClient.Run(fmt.Sprintf("rm -f '%s'", nvrhContext.RemoteSocketPath), nil)
					nvrhContext.SshClient.Run(fmt.Sprintf("rm -f '%s'", nvrhContext.BrowserScriptPath), nil)
				}

				nvrhContext.SshClient.Close()
			}
		}()

		// Start remote nvim
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

			nvimCommandString := fmt.Sprintf(
				"exec $SHELL -i -c 'cd \"%s\" && %s'",
				nvrhContext.RemoteDirectory,
				nvim_helpers.BuildRemoteCommandString(nvrhContext),
			)

			slog.Info("Starting remote nvim", "nvimCommandString", nvimCommandString)
			done <- nvrhContext.SshClient.Run(nvimCommandString, tunnelInfo)
			// Call stop so WaitForNvim can exit.
			stop()
		}()

		// Wait for remote nvim
		nv, err := nvim_helpers.WaitForNvim(ctx, nvrhContext)
		if err != nil {
			return fmt.Errorf("failed to connect to remote nvim: %w", err)
		}

		// Prepare remote nvim
		if err := prepareRemoteNvim(nvrhContext, nv, "primary"); err != nil {
			slog.Warn("Error preparing remote nvim", "err", err)
		}

		// Start local client
		clientCmd := BuildClientNvimCmd(ctx, nvrhContext)
		if nvrhContext.Debug {
			clientCmd.Stdout = os.Stdout
			clientCmd.Stderr = os.Stderr
		}
		nvrhContext.CommandsToKill = append(nvrhContext.CommandsToKill, clientCmd)

		if err := clientCmd.Start(); err != nil {
			return fmt.Errorf("failed to start local nvim: %w", err)
		}

		go func() {
			done <- clientCmd.Wait()
		}()

		select {
		case <-ctx.Done():
			slog.Warn("Interrupted by user")
			return ctx.Err()
		case err := <-done:
			if err != nil {
				slog.Error("Local nvim exited with error", "err", err)
				return err
			}
			slog.Info("Local nvim exited cleanly")
			return nil
		}
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

		&cli.StringSliceFlag{
			Name:    "ssh-arg",
			Usage:   "Additional arguments to pass to the SSH command",
			EnvVars: []string{"NVRH_CLIENT_SSH_ARG"},
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

		// Context with cancellation on SIGINT
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
		defer stop()
		done := make(chan error, 1)

		// Prepare the context.
		randomId := fmt.Sprintf("%d", time.Now().Unix())
		nvrhContext := &nvrh_context.NvrhContext{
			SessionId: sessionId,
			Endpoint:  endpoint,
			// RemoteDirectory: c.Args().Get(1),

			// RemoteEnv:   c.StringSlice("server-env"),
			LocalEditor: c.StringSlice("local-editor"),

			ShouldUsePorts: c.Bool("use-ports"),

			RemoteSocketPath: fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId),
			LocalSocketPath:  filepath.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s-%s", sessionId, randomId)),
			// TODO Handle mapping ports better with multiple clients.
			// AutomapPorts:     c.Bool("enable-automap-ports"),

			BrowserScriptPath: fmt.Sprintf("/tmp/nvrh-browser-%s", sessionId),

			SshPath: sshPath,
			Debug:   isDebug,

			TunneledPorts: make(map[string]bool),

			// NvimCmd: c.StringSlice("nvim-cmd"),

			SshArgs: c.StringSlice("ssh-arg"),
		}

		// Setup SSH client
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

			portNumberString := c.Args().Get(2)
			portNumber := 0
			if portNumberString != "" {
				converted, err := strconv.Atoi(portNumberString)

				if err != nil {
					return fmt.Errorf("invalid port number: %w", err)
				}

				portNumber = converted
			}

			nvrhContext.LocalPortNumber = randomPort
			if portNumber != 0 {
				nvrhContext.RemotePortNumber = portNumber
			} else {
				nvrhContext.RemotePortNumber = randomPort
			}
		}

		var nv *nvim.Nvim

		// Cleanup on exit
		defer func() {
			slog.Info("Cleaning up")
			closeNvimSocket(nv, false)
			killAllCmds(nvrhContext.CommandsToKill)
			os.Remove(nvrhContext.LocalSocketPath)
			if nvrhContext.SshClient != nil {
				nvrhContext.SshClient.Close()
			}
		}()

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
			// TODO needed?
			stop()
		}()

		// Wait for remote nvim
		nv, err := nvim_helpers.WaitForNvim(ctx, nvrhContext)
		if err != nil {
			return fmt.Errorf("failed to connect to remote nvim: %w", err)
		}

		// Prepare remote nvim
		if err := prepareRemoteNvim(nvrhContext, nv, "secondary"); err != nil {
			slog.Warn("Error preparing remote nvim", "err", err)
		}

		// Start local client
		clientCmd := BuildClientNvimCmd(ctx, nvrhContext)
		if nvrhContext.Debug {
			clientCmd.Stdout = os.Stdout
			clientCmd.Stderr = os.Stderr
		}
		nvrhContext.CommandsToKill = append(nvrhContext.CommandsToKill, clientCmd)

		if err := clientCmd.Start(); err != nil {
			return fmt.Errorf("failed to start local nvim: %w", err)
		}

		go func() {
			done <- clientCmd.Wait()
		}()

		select {
		case <-ctx.Done():
			slog.Warn("Interrupted by user")
			return ctx.Err()
		case err := <-done:
			if err != nil {
				slog.Error("Local nvim exited with error", "err", err)
				return err
			}
			slog.Info("Local nvim exited cleanly")
			return nil
		}
	},
}

func BuildClientNvimCmd(ctx context.Context, nvrhContext *nvrh_context.NvrhContext) *exec.Cmd {
	replacedArgs := make([]string, len(nvrhContext.LocalEditor))
	for i, arg := range nvrhContext.LocalEditor {
		replacedArgs[i] = strings.ReplaceAll(arg, "{{SOCKET_PATH}}", nvrhContext.LocalSocketOrPort())
	}

	slog.Info("Starting local editor", "cmd", replacedArgs)

	editorCommand := exec.CommandContext(ctx, replacedArgs[0], replacedArgs[1:]...)
	if replacedArgs[0] == "nvim" {
		editorCommand.Stdin = os.Stdin
		editorCommand.Stdout = os.Stdout
		editorCommand.Stderr = os.Stderr
	}

	return editorCommand
}

func prepareRemoteNvim(nvrhContext *nvrh_context.NvrhContext, nv *nvim.Nvim, mode string) error {
	nv.RegisterHandler("tunnel-port", func(v *nvim.Nvim, args []string) {
		if _, ok := nvrhContext.TunneledPorts[args[0]]; ok {
			return
		}

		nvrhContext.TunneledPorts[args[0]] = true

		go nvrhContext.SshClient.TunnelSocket(&ssh_tunnel_info.SshTunnelInfo{
			Mode:         "port",
			LocalSocket:  args[0],
			RemoteSocket: args[0],
			Public:       true,
		})
	})
	nv.RegisterHandler("open-url", RpcHandleOpenUrl)

	batch := nv.NewBatch()

	slog.Info("Preparing remote nvim", "mode", mode, "sessionId", nvrhContext.SessionId)

	if mode == "primary" {
		batch.ExecLua(`
local session_id = ...

_G._nvrh = {
	session_id = session_id,
	client_channels = {},

	-- Only really used when connecting secondary sessions, the actual Go code
	-- already checks if a port is already mapped.
	mapped_ports = {},

	tunnel_port = function(port)
		for _, channel_id in ipairs(_G._nvrh.client_channels) do
			_G._nvrh._tunnel_port_with_channel(channel_id, port)
		end

		if not _G._nvrh.mapped_ports[port] then
			_G._nvrh.mapped_ports[port] = true
		end
	end,

	_tunnel_port_with_channel = function(channel_id, port)
		pcall(vim.rpcnotify, tonumber(channel_id), 'tunnel-port', { port })
	end,

	open_url = function(url)
		for _, channel_id in ipairs(_G._nvrh.client_channels) do
			pcall(vim.rpcnotify, tonumber(channel_id), 'open-url', { url })
		end
	end,
}

vim.print(_G._nvrh)

vim.api.nvim_create_user_command(
	'NvrhTunnelPort',
	function(args)
		_G._nvrh.tunnel_port(args.args)
	end,
	{
		nargs = 1,
		force = true,
	}
)

vim.api.nvim_create_user_command(
	'NvrhOpenUrl',
	function(args)
		_G._nvrh.open_url(args.args)
	end,
	{
		nargs = 1,
		force = true,
	}
)

local original_open = vim.ui.open
vim.ui.open = function(uri, opts)
	if type(uri) == 'string' and uri:match('^https?://') then
		_G._nvrh.open_url(uri)
		return nil, nil
	else
		return original_open(uri, opts)
	end
end
		`, nil, nvrhContext.SessionId)
	}

	batch.ExecLua(`
		local channel_id = ...
		table.insert(_G._nvrh.client_channels, channel_id)
	`, nil, nv.ChannelID())

	if mode == "secondary" {
		batch.ExecLua(`
local channel_id = ...
for i, port in ipairs(_G._nvrh.mapped_ports) do
	_G._nvrh._tunnel_port_with_channel(channel_id, port)
end
		`, nil, nv.ChannelID())
	}

	batch.Command(fmt.Sprintf(`let $BROWSER="%s"`, nvrhContext.BrowserScriptPath))

	// Prepare the browser script.
	batch.ExecLua(`
local browser_script_path, socket_path = ...

local script_contents = [[
#!/bin/sh

SOCKET_PATH="%s"

exec nvim --server "$SOCKET_PATH" --remote-expr "v:lua._nvrh.open_url('$1')" > /dev/null
]]
script_contents = string.format(script_contents, socket_path)

vim.fn.writefile(vim.fn.split(script_contents, '\n'), browser_script_path)
os.execute('chmod +x ' .. browser_script_path)
	`, nil, nvrhContext.BrowserScriptPath, nvrhContext.RemoteSocketOrPort(), nv.ChannelID())

	if nvrhContext.AutomapPorts {
		batch.ExecLua(`
local nvrh_port_scanner = {
	active_watchers = {},

	patterns = {
		-- "port 3000"
		"port%s+(%d+)",
		-- "localhost:3000"
		"localhost:(%d+)",
		-- "0.0.0.0:3000"
		"%d+%.%d+%.%d+%.%d+:(%d+)",
		-- ":3000" at start of line
		"^:(%d+)",
		-- ":3000" but avoid eslint errors (error in foo.tsx:3)
		"%s+:(%d+)",
		-- http://some.domain.com:3000 / https://some.domain.com:3000
		"https?://[^/]+:(%d+)",
	},
}

function nvrh_port_scanner.attach_port_watcher(bufnr)
	if vim.bo[bufnr].buftype ~= "terminal" then
		return
	end

	-- Already watching?
	if nvrh_port_scanner.active_watchers[bufnr] then
		return
	end

	local function on_lines(_, _, _, lastline, new_lastline, _)
		local lines = vim.api.nvim_buf_get_lines(bufnr, lastline, new_lastline, false)

		for _, line in ipairs(lines) do
			for _, pattern in ipairs(nvrh_port_scanner.patterns) do
				local port = string.match(line, pattern)
				if port then
					_G._nvrh.tunnel_port(port)
					break
				end
			end
		end
	end

	vim.api.nvim_buf_attach(bufnr, false, {
		on_lines = on_lines,
	})

	nvrh_port_scanner.active_watchers[bufnr] = true
end

-- Attach watcher on TermOpen
vim.api.nvim_create_autocmd("TermOpen", {
	callback = function(args)
		nvrh_port_scanner.attach_port_watcher(args.buf)
	end,
})
		`, nil)
	}

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

	switch goos {
	case "darwin":
		exec.Command("open", url).Run()
	case "linux":
		exec.Command("xdg-open", url).Run()
	case "windows":
		exec.Command("cmd", "/c", "start", url).Run()
	default:
		slog.Error("Don't know how to open url", "url", url)
	}
}

func killAllCmds(cmds []*exec.Cmd) {
	for _, cmd := range cmds {
		exec_helpers.Kill(cmd)
	}
}

func closeNvimSocket(nv *nvim.Nvim, quitAll bool) {
	if nv == nil {
		return
	}

	if quitAll {
		if err := nv.ExecLua("vim.cmd('qall!')", nil, nil); err != nil {
			slog.Warn("Error closing remote nvim", "err", err)
		}
	}

	slog.Info("Closing nvim socket")
	nv.Close()
}
