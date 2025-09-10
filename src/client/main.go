package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/neovim/go-client/nvim"
	"github.com/urfave/cli/v3"

	"nvrh/src/bridge_files"
	nvrh_context "nvrh/src/context"
	"nvrh/src/exec_helpers"
	"nvrh/src/go_ssh_ext"
	"nvrh/src/logger"
	"nvrh/src/nvim_helpers"
	"nvrh/src/nvrh_base_ssh"
	"nvrh/src/nvrh_binary_ssh"
	"nvrh/src/nvrh_config"
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

	Commands: []*cli.Command{
		&CliClientOpenCommand,
		&CliClientReconnectCommand,
	},
}

var CliClientOpenCommand = cli.Command{
	Name:      "open",
	Usage:     "Open a remote nvim instance in a local editor",
	Category:  "client",
	ArgsUsage: "<server> [remote-directory]",

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "ssh-path",
			Usage:   "Path to SSH binary. 'binary' will use the default system SSH binary. 'internal' will use the internal SSH client. Anything else will be used as the path to the SSH binary",
			Sources: cli.EnvVars("NVRH_CLIENT_SSH_PATH"),
			Value:   "binary",
		},

		&cli.BoolFlag{
			Name:  "use-ports",
			Usage: "Use ports instead of sockets. Defaults to true on Windows [$NVRH_CLIENT_USE_PORTS]",
			// Sources: cli.EnvVars("NVRH_CLIENT_USE_PORTS"),
			Value: runtime.GOOS == "windows",
		},

		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "",
			Sources: cli.EnvVars("NVRH_CLIENT_DEBUG"),
		},

		&cli.StringSliceFlag{
			Name:    "server-env",
			Usage:   "Environment variables to set on the remote server",
			Sources: cli.EnvVars("NVRH_CLIENT_SERVER_ENV"),
		},

		&cli.StringSliceFlag{
			Name:    "local-editor",
			Usage:   "Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path",
			Sources: cli.EnvVars("NVRH_CLIENT_LOCAL_EDITOR"),
			Value:   []string{"nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui"},
		},

		&cli.StringSliceFlag{
			Name:  "nvim-cmd",
			Usage: "Command to run nvim with. Defaults to `nvim` [$NVRH_CLIENT_NVIM_CMD]",
			// Sources: cli.EnvVars("NVRH_CLIENT_NVIM_CMD"),
			Value: []string{"nvim"},
		},

		&cli.StringSliceFlag{
			Name:  "ssh-arg",
			Usage: "Additional arguments to pass to the SSH command [$NVRH_CLIENT_SSH_ARG]",
			// Sources: cli.EnvVars("NVRH_CLIENT_SSH_ARG"),
		},

		&cli.BoolFlag{
			Name:    "enable-automap-ports",
			Usage:   "Enable automatic port mapping",
			Sources: cli.EnvVars("NVRH_CLIENT_AUTOMAP_PORTS"),
			Value:   true,
		},
	},

	Action: func(ctx context.Context, cmd *cli.Command) error {
		cfg, err := nvrh_config.LoadConfig(nvrh_config.DefaultConfigPath())
		if err != nil {
			return err
		}

		isDebug := cmd.Bool("debug")
		logger.PrepareLogger(isDebug)

		server := cmd.Args().Get(0)
		if server == "" {
			return fmt.Errorf("<server> is required")
		}

		sessionId := fmt.Sprintf("%d", time.Now().Unix())
		sshPath := getSshPath(cmd.String("ssh-path"))

		endpoint, endpointErr := ssh_endpoint.ParseSshEndpoint(server)
		if endpointErr != nil {
			return endpointErr
		}

		serverConfig := cfg.Servers[endpoint.GivenHost]
		if err := nvrh_config.ApplyPrecedence(cmd, serverConfig); err != nil {
			return err
		}

		// Context with cancellation on SIGINT
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()
		done := make(chan error, 1)

		// Prepare the context.
		nvrhContext := &nvrh_context.NvrhContext{
			SessionId:       sessionId,
			Endpoint:        endpoint,
			RemoteDirectory: cmd.Args().Get(1),

			AutomapPorts: cmd.Bool("enable-automap-ports"),

			Debug: isDebug,

			TunneledPorts: make(map[string]bool),

			NvimCmd: cmd.StringSlice("nvim-cmd"),
		}

		remoteEnv := cmd.StringSlice("server-env")
		localEditor := cmd.StringSlice("local-editor")
		sshArgs := cmd.StringSlice("ssh-arg")

		shouldUsePorts := cmd.Bool("use-ports")
		remoteSocketPath := fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId)
		localSocketPath := filepath.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s", sessionId))

		randomPort := getRandomPort()
		localPortNumber := randomPort
		remotePortNumber := randomPort

		var tunnelInfo *ssh_tunnel_info.SshTunnelInfo

		// Setup SSH client
		sshClient, sshClientErr := getSshClient(nvrhContext, endpoint, sshPath, sshArgs)
		if sshClientErr != nil {
			return sshClientErr
		}
		nvrhContext.SshClient = sshClient

		var nv *nvim.Nvim
		didClientFail := false

		// Cleanup on exit
		defer func() {
			slog.Info("Cleaning up")
			closeNvimSocket(nv, didClientFail)
			killAllCmds(nvrhContext.CommandsToKill)
			os.Remove(localSocketPath)
			if nvrhContext.SshClient != nil {
				nvrhContext.SshClient.Close()
			}
		}()

		siDone := make(chan error, 1)

		siTunnelInfo := &ssh_tunnel_info.SshTunnelInfo{
			Mode:         "port",
			Public:       false,
			LocalSocket:  fmt.Sprintf("%d", randomPort),
			RemoteSocket: fmt.Sprintf("%d", randomPort),
		}

		// Start server info nvim instance.
		slog.Info("Starting server info nvim instance")
		go func() {
			// Not quoting here because Powershell doesn't like it without the
			// preceding ampersand, and we don't know what shell we're using at this
			// point.
			nvimCmd := strings.Join(nvrhContext.NvimCmd, " ")

			siDone <- nvrhContext.SshClient.Run(
				fmt.Sprintf("%s -u NONE --headless --listen \"%s\"", nvimCmd, siTunnelInfo.RemoteBoundToIp()),
				siTunnelInfo,
			)
		}()

		// Grab server info and potentially prepare Windows.
		go func() {
			siNv, err := nvim_helpers.WaitForNvim(ctx, siTunnelInfo)

			if err != nil {
				siDone <- err
				return
			}

			var serverInfoString string
			var serverInfo *nvrh_context.NvrhServerInfo
			siNv.ExecLua(bridge_files.ReadFileWithoutError("lua/determine_server_info.lua"), &serverInfoString, nil)
			json.Unmarshal([]byte(serverInfoString), &serverInfo)

			nvrhContext.ServerInfo = serverInfo

			if nvrhContext.ServerInfo.Os == "windows" {
				shouldUsePorts = true

				nvrhContext.WindowsLauncherPath = fmt.Sprintf(
					`%s\nvim-launcher-%s.bat`,
					nvrhContext.ServerInfo.Tmpdir,
					nvrhContext.SessionId,
				)

				tunnelInfo = &ssh_tunnel_info.SshTunnelInfo{
					Mode:         "port",
					LocalSocket:  fmt.Sprintf("%d", localPortNumber),
					RemoteSocket: fmt.Sprintf("%d", remotePortNumber),
					Public:       false,
				}

				nvimCmd := nvim_helpers.BuildRemoteCommandString(
					nvrhContext.NvimCmd,
					"bat",
					nvrhContext.RemoteDirectory,
					remoteEnv,
					tunnelInfo,
				)

				err := siNv.ExecLua(
					bridge_files.ReadFileWithoutError("lua/setup_nvim_launcher.lua"),
					nil,
					nvimCmd,
					nvrhContext.WindowsLauncherPath,
				)

				if err != nil {
					siDone <- err
					return
				}
			}

			siNv.ExecLua("vim.cmd('qall!')", nil, nil)
			siNv.Close()

			siDone <- nil
		}()

		// Wait for server info process to finish.
		select {
		case <-ctx.Done():
			slog.Warn("Interrupted by user")
			return ctx.Err()
		case err := <-siDone:
			if err != nil {
				slog.Error("Error while getting server info", "err", err)
				return err
			}
		}

		// Even though this happens in the Windows Server path, we still need a
		// check here in case that path isn't hit.
		if tunnelInfo == nil {
			tunnelInfo = &ssh_tunnel_info.SshTunnelInfo{
				Mode:         "unix",
				LocalSocket:  localSocketPath,
				RemoteSocket: remoteSocketPath,
				Public:       false,
			}
		}

		if shouldUsePorts {
			tunnelInfo.SwitchToPorts(localPortNumber, remotePortNumber)
		}

		// Start remote nvim
		go func() {
			var nvimCommandString string
			if nvrhContext.ServerInfo.Os == "windows" {
				if nvrhContext.ServerInfo.ShellName == "bash" {
					nvimCommandString = fmt.Sprintf("/tmp/nvim-launcher-%s.bat", nvrhContext.SessionId)
				} else {
					nvimCommandString = nvrhContext.WindowsLauncherPath
				}
			} else {
				nvimCommandString = nvim_helpers.BuildRemoteCommandString(
					nvrhContext.NvimCmd,
					nvrhContext.ServerInfo.ShellName,
					nvrhContext.RemoteDirectory,
					remoteEnv,
					tunnelInfo,
				)
			}

			slog.Info("Starting remote nvim", "nvimCommandString", nvimCommandString)
			done <- nvrhContext.SshClient.Run(nvimCommandString, tunnelInfo)
			// Call stop so WaitForNvim can exit.
			stop()
		}()

		// Wait for remote nvim
		nv, err = nvim_helpers.WaitForNvim(ctx, tunnelInfo)
		if err != nil {
			return fmt.Errorf("failed to connect to remote nvim: %w", err)
		}

		// Prepare remote nvim
		if err := prepareRemoteNvim(nvrhContext, nv, cmd.Root().Version, tunnelInfo); err != nil {
			slog.Warn("Error preparing remote nvim", "err", err)
		}

		// Start local client
		clientCmd := BuildClientNvimCmd(ctx, localEditor, tunnelInfo)
		if nvrhContext.Debug {
			clientCmd.Stdout = os.Stdout
			clientCmd.Stderr = os.Stderr
		}
		nvrhContext.CommandsToKill = append(nvrhContext.CommandsToKill, clientCmd)

		if err := clientCmd.Start(); err != nil {
			didClientFail = true
			return fmt.Errorf("failed to start local nvim: %w", err)
		}

		go func() {
			err := clientCmd.Wait()
			didClientFail = err != nil
			done <- err
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
	ArgsUsage: "<server> <session-id>",

	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "ssh-path",
			Usage:   "Path to SSH binary. 'binary' will use the default system SSH binary. 'internal' will use the internal SSH client. Anything else will be used as the path to the SSH binary",
			Sources: cli.EnvVars("NVRH_CLIENT_SSH_PATH"),
			Value:   "binary",
		},

		&cli.BoolFlag{
			Name:  "use-ports",
			Usage: "Use ports instead of sockets. Defaults to true on Windows [$NVRH_CLIENT_USE_PORTS]",
			// Sources: cli.EnvVars("NVRH_CLIENT_USE_PORTS"),
			Value: runtime.GOOS == "windows",
		},

		&cli.BoolFlag{
			Name:    "debug",
			Usage:   "",
			Sources: cli.EnvVars("NVRH_CLIENT_DEBUG"),
		},

		&cli.StringSliceFlag{
			Name:    "local-editor",
			Usage:   "Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path",
			Sources: cli.EnvVars("NVRH_CLIENT_LOCAL_EDITOR"),
			Value:   []string{"nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui"},
		},

		&cli.StringSliceFlag{
			Name:  "ssh-arg",
			Usage: "Additional arguments to pass to the SSH command [$NVRH_CLIENT_SSH_ARG]",
			// Sources: cli.EnvVars("NVRH_CLIENT_SSH_ARG"),
		},
	},

	Action: func(ctx context.Context, cmd *cli.Command) error {
		cfg, err := nvrh_config.LoadConfig(nvrh_config.DefaultConfigPath())
		if err != nil {
			return err
		}

		isDebug := cmd.Bool("debug")
		logger.PrepareLogger(isDebug)

		// Prepare the context.
		server := cmd.Args().Get(0)
		if server == "" {
			return fmt.Errorf("<server> is required")
		}

		sessionId := cmd.Args().Get(1)
		if sessionId == "" {
			return fmt.Errorf("<session-id> is required")
		}

		sshPath := getSshPath(cmd.String("ssh-path"))

		endpoint, endpointErr := ssh_endpoint.ParseSshEndpoint(server)
		if endpointErr != nil {
			return endpointErr
		}

		serverConfig := cfg.Servers[endpoint.GivenHost]
		if err := nvrh_config.ApplyPrecedence(cmd, serverConfig); err != nil {
			return err
		}

		// Context with cancellation on SIGINT
		ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
		defer stop()
		done := make(chan error, 1)

		// Prepare the context.
		randomId := fmt.Sprintf("%d", time.Now().Unix())
		nvrhContext := &nvrh_context.NvrhContext{
			SessionId: sessionId,
			Endpoint:  endpoint,
			// RemoteDirectory: c.Args().Get(1),

			// TODO Handle mapping ports better with multiple clients.
			// AutomapPorts: c.Bool("enable-automap-ports"),

			Debug: isDebug,

			TunneledPorts: make(map[string]bool),

			// NvimCmd: c.StringSlice("nvim-cmd"),
		}

		localEditor := cmd.StringSlice("local-editor")
		sshArgs := cmd.StringSlice("ssh-arg")

		shouldUsePorts := cmd.Bool("use-ports")
		remoteSocketPath := fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId)
		localSocketPath := filepath.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s-%s", sessionId, randomId))

		randomPort := getRandomPort()
		localPortNumber := randomPort
		remotePortNumber := randomPort

		// Setup SSH client
		sshClient, sshClientErr := getSshClient(nvrhContext, endpoint, sshPath, sshArgs)
		if sshClientErr != nil {
			return sshClientErr
		}
		nvrhContext.SshClient = sshClient

		if shouldUsePorts {
			portNumberString := cmd.Args().Get(2)
			portNumber := 0
			if portNumberString != "" {
				converted, err := strconv.Atoi(portNumberString)

				if err != nil {
					return fmt.Errorf("invalid port number: %w", err)
				}

				portNumber = converted
			}

			if portNumber != 0 {
				remotePortNumber = portNumber
			}
		}

		var nv *nvim.Nvim

		// Cleanup on exit
		defer func() {
			slog.Info("Cleaning up")
			closeNvimSocket(nv, false)
			killAllCmds(nvrhContext.CommandsToKill)
			os.Remove(localSocketPath)
			if nvrhContext.SshClient != nil {
				nvrhContext.SshClient.Close()
			}
		}()

		// Setup SSH tunnel
		tunnelInfo := &ssh_tunnel_info.SshTunnelInfo{
			Mode:         "unix",
			LocalSocket:  localSocketPath,
			RemoteSocket: remoteSocketPath,
			Public:       false,
		}

		if shouldUsePorts {
			tunnelInfo.SwitchToPorts(localPortNumber, remotePortNumber)
		}

		go func() {
			nvrhContext.SshClient.TunnelSocket(tunnelInfo)
			// TODO needed?
			stop()
		}()

		// Wait for remote nvim
		nv, err = nvim_helpers.WaitForNvim(ctx, tunnelInfo)
		if err != nil {
			return fmt.Errorf("failed to connect to remote nvim: %w", err)
		}

		var serverInfoString string
		var serverInfo *nvrh_context.NvrhServerInfo
		nv.ExecLua("return vim.json.encode(_G._nvrh.server_info)", &serverInfoString, nil)
		json.Unmarshal([]byte(serverInfoString), &serverInfo)
		nvrhContext.ServerInfo = serverInfo

		// Prepare remote nvim
		if err := prepareRemoteNvim(nvrhContext, nv, cmd.Root().Version, tunnelInfo); err != nil {
			slog.Warn("Error preparing remote nvim", "err", err)
		}

		// Start local client
		clientCmd := BuildClientNvimCmd(ctx, localEditor, tunnelInfo)
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

func BuildClientNvimCmd(
	ctx context.Context,
	cmd []string,
	ti *ssh_tunnel_info.SshTunnelInfo,
) *exec.Cmd {
	replacedArgs := make([]string, len(cmd))
	for i, arg := range cmd {
		replacedArgs[i] = strings.ReplaceAll(arg, "{{SOCKET_PATH}}", ti.LocalBoundToIp())
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

func prepareRemoteNvim(
	nvrhContext *nvrh_context.NvrhContext,
	nv *nvim.Nvim,
	version string,
	ti *ssh_tunnel_info.SshTunnelInfo,
) error {
	slog.Info("Preparing remote nvim", "sessionId", nvrhContext.SessionId)

	//Setup channel info for the remote nvim instance.
	currentUser, _ := user.Current()
	hostname, _ := os.Hostname()

	nv.SetClientInfo(
		"nvrh",
		nvim.ClientVersion{},
		"rpc",
		map[string]*nvim.ClientMethod{
			"tunnel-port": {
				Async: true,
				NArgs: nvim.ClientMethodNArgs{
					Min: 1,
					Max: 1,
				},
			},

			"open-url": {
				Async: true,
				NArgs: nvim.ClientMethodNArgs{
					Min: 1,
					Max: 1,
				},
			},
		},
		nvim.ClientAttributes{
			"nvrh_version":         version,
			"nvrh_client_username": currentUser.Username,
			"nvrh_client_hostname": hostname,
			"nvrh_client_os":       runtime.GOOS,
			// Assume the UI channel is the next channel.
			"nvrh_assumed_ui_channel": fmt.Sprintf("%d", nv.ChannelID()+1),
		},
	)

	// Register RPC handlers.
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

	// Prepare bridge code.
	browserShellScript, browserScriptPath := getBrowserScript(nvrhContext)
	allScripts := []string{
		bridge_files.ReadFileWithoutError("lua/init_bridge.lua"),
		bridge_files.ReadFileWithoutError("lua/init_nvrh.lua"),

		bridge_files.ReadFileWithoutError("lua/rpc_open_url.lua"),
		fmt.Sprintf(
			bridge_files.ReadFileWithoutError("lua/setup_browser_script.lua"),
			fmt.Sprintf(
				browserShellScript,
				ti.RemoteBoundToIp(),
			),
		),

		bridge_files.ReadFileWithoutError("lua/rpc_tunnel_port.lua"),
		bridge_files.ReadFileWithoutError("lua/setup_port_scanner.lua"),
		bridge_files.ReadFileWithoutError("lua/session_automap_ports.lua"),
	}
	scriptsJoined := strings.Join(allScripts, "\n\n")

	marshalled, err := json.Marshal(nvrhContext.ServerInfo)
	if err != nil {
		return err
	}

	err = nv.ExecLua(
		scriptsJoined,
		nil,
		nvrhContext.SessionId,
		nv.ChannelID(),
		ti.RemoteBoundToIp(),
		browserScriptPath,
		nvrhContext.AutomapPorts,
		string(marshalled),
		nvrhContext.WindowsLauncherPath,
	)

	if err != nil {
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
		slog.Error("Don't know how to open url", "url", url, "os", goos)
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

func getBrowserScript(nvrhContext *nvrh_context.NvrhContext) (string, string) {
	var browserShellScript string
	if nvrhContext.ServerInfo.Os == "windows" {
		browserShellScript = bridge_files.ReadFileWithoutError("shell/nvrh-browser.bat")
	} else {
		browserShellScript = bridge_files.ReadFileWithoutError("shell/nvrh-browser")
	}

	var browserScriptPath string
	if nvrhContext.ServerInfo.Os == "windows" {
		browserScriptPath = strings.Join(
			[]string{
				nvrhContext.ServerInfo.Tmpdir,
				fmt.Sprintf("nvrh-browser-%s.bat", nvrhContext.SessionId),
			},
			`\`,
		)
	} else {
		browserScriptPath = strings.Join(
			[]string{
				nvrhContext.ServerInfo.Tmpdir,
				fmt.Sprintf("nvrh-browser-%s", nvrhContext.SessionId),
			},
			`/`,
		)
	}

	return browserShellScript, browserScriptPath
}

func getSshPath(given string) string {
	if given == "binary" {
		return defaultSshPath()
	}

	return given
}

func getSshClient(
	nvrhContext *nvrh_context.NvrhContext,
	endpoint *ssh_endpoint.SshEndpoint,
	sshPath string,
	sshArgs []string,
) (nvrh_base_ssh.BaseNvrhSshClient, error) {
	if sshPath == "internal" {
		sshClient, err := go_ssh_ext.GetSshClientForEndpoint(endpoint)
		if err != nil {
			return nil, err
		}

		return nvrh_base_ssh.BaseNvrhSshClient(&nvrh_internal_ssh.NvrhInternalSshClient{
			Ctx:       nvrhContext,
			SshClient: sshClient,
		}), nil
	}

	return nvrh_base_ssh.BaseNvrhSshClient(&nvrh_binary_ssh.NvrhBinarySshClient{
		Ctx:     nvrhContext,
		SshPath: sshPath,
		SshArgs: sshArgs,
	}), nil
}

func getRandomPort() int {
	min := 1025
	max := 65535
	return rand.IntN(max-min) + min
}
