package client

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/kevinburke/ssh_config"
	"github.com/neovim/go-client/nvim"
	"github.com/skeema/knownhosts"
	"github.com/urfave/cli/v2"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"

	"nvrh/src/context"
	"nvrh/src/logger"
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
			Name:  "local-editor",
			Usage: "Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path",
			Value: cli.NewStringSlice("nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui"),
		},
	},

	Action: func(c *cli.Context) error {
		isDebug := c.Bool("debug")
		logger.PrepareLogger(isDebug)

		// Prepare the context.
		sessionId := fmt.Sprintf("%d", time.Now().Unix())

		endpoint, err := ParseSshEndpoint(c.Args().Get(0))
		if err != nil {
			return err
		}

		sshClient, err := getSshClientForServer(endpoint)
		if err != nil {
			return err
		}

		nvrhContext := &context.NvrhContext{
			SessionId:       sessionId,
			Server:          c.Args().Get(0),
			RemoteDirectory: c.Args().Get(1),

			RemoteEnv:   c.StringSlice("server-env"),
			LocalEditor: c.StringSlice("local-editor"),

			ShouldUsePorts: c.Bool("use-ports"),

			RemoteSocketPath: fmt.Sprintf("/tmp/nvrh-socket-%s", sessionId),
			LocalSocketPath:  filepath.Join(os.TempDir(), fmt.Sprintf("nvrh-socket-%s", sessionId)),

			BrowserScriptPath: fmt.Sprintf("/tmp/nvrh-browser-%s", sessionId),

			SshPath: c.String("ssh-path"),
			Debug:   isDebug,

			SshClient: sshClient,
		}

		if nvrhContext.ShouldUsePorts {
			min := 1025
			max := 65535
			nvrhContext.PortNumber = rand.IntN(max-min) + min
		}

		if nvrhContext.Server == "" {
			return fmt.Errorf("<server> is required")
		}

		defer nvrhContext.SshClient.Close()

		var nv *nvim.Nvim

		doneChan := make(chan error)

		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt)

		// Prepare remote instance.
		go func() {
			go ssh_helpers.TunnelSshSocket(nvrhContext, ssh_helpers.SshTunnelInfo{
				Mode:         "unix",
				LocalSocket:  nvrhContext.LocalSocketPath,
				RemoteSocket: nvrhContext.RemoteSocketPath,
			})

			nvimCommandString := ssh_helpers.BuildRemoteCommandString(nvrhContext)
			nvimCommandString = fmt.Sprintf("$SHELL -i -c '%s'", nvimCommandString)
			slog.Info("Starting remote nvim", "nvimCommandString", nvimCommandString)
			if err := ssh_helpers.RunCommand(nvrhContext, nvimCommandString); err != nil {
				doneChan <- err
			}

			ssh_helpers.RunCommand(nvrhContext, fmt.Sprintf("rm -f '%s'", nvrhContext.RemoteSocketPath))
			ssh_helpers.RunCommand(nvrhContext, fmt.Sprintf("rm -f '%s'", nvrhContext.BrowserScriptPath))
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

		nv = <-nvChan

		go func() {
			select {
			case sig := <-signalChan:
				slog.Debug("Received signal", "signal", sig)
				doneChan <- fmt.Errorf("Received signal: %s", sig)
			}
		}()

		err = <-doneChan

		slog.Info("Closing nvrh")
		closeNvimSocket(nv)
		killAllCmds(nvrhContext.CommandsToKill)
		os.Remove(nvrhContext.LocalSocketPath)

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
	nv.RegisterHandler("tunnel-port", ssh_helpers.MakeRpcTunnelHandler(nvrhContext))
	nv.RegisterHandler("open-url", RpcHandleOpenUrl)

	batch := nv.NewBatch()

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

func getSshClientForServer(endpoint *SshEndpoint) (*ssh.Client, error) {
	kh, err := knownhosts.NewDB(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return nil, err
	}

	slog.Debug("Connecting to server", "endpoint", endpoint)

	authMethods := []ssh.AuthMethod{}

	authMethods = append(authMethods, ssh.PublicKeysCallback(func() ([]ssh.Signer, error) {
		allSigners := []ssh.Signer{}

		if agentSigners, _ := getSignersForIdentityAgent(endpoint.GivenHost); agentSigners != nil {
			allSigners = append(allSigners, agentSigners...)
		}

		if identitySigner, _ := getSignerForIdentityFile(endpoint.GivenHost); identitySigner != nil {
			allSigners = append(allSigners, identitySigner)
		}

		return allSigners, nil
	}))

	authMethods = append(authMethods, ssh.PasswordCallback(func() (string, error) {
		password, err := askForPassword(fmt.Sprintf("Password for %s: ", endpoint))
		if err != nil {
			slog.Error("Error reading password", "err", err)
			return "", err
		}

		return string(password), nil
	}))

	config := &ssh.ClientConfig{
		User:              endpoint.FinalUser(),
		Auth:              authMethods,
		HostKeyCallback:   kh.HostKeyCallback(),
		HostKeyAlgorithms: kh.HostKeyAlgorithms(fmt.Sprintf("%s:%s", endpoint.FinalHost(), endpoint.FinalPort())),
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", endpoint.FinalHost(), endpoint.FinalPort()), config)
	if err != nil {
		slog.Error("Failed to dial", "err", err)
		return nil, err
	}

	return client, nil
}

func getSignerForIdentityFile(hostname string) (ssh.Signer, error) {
	identityFile := ssh_config.Get(hostname, "IdentityFile")

	if identityFile == "" {
		return nil, nil
	}

	identityFile = cleanupSshConfigValue(identityFile)

	if _, err := os.Stat(identityFile); os.IsNotExist(err) {
		slog.Error("Identity file does not exist", "identityFile", identityFile)
		return nil, err
	}

	slog.Info("Using identity file", "identityFile", identityFile)

	key, err := os.ReadFile(identityFile)
	if err != nil {
		slog.Error("Unable to read private key", "err", err)
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		if _, ok := err.(*ssh.PassphraseMissingError); ok {
			passPhrase, _ := askForPassword(fmt.Sprintf("Passphrase for %s: ", identityFile))

			signer, signerErr := ssh.ParsePrivateKeyWithPassphrase(key, passPhrase)
			if signerErr != nil {
				slog.Error("Unable to parse private key", "err", signerErr)
				return nil, signerErr
			}

			return signer, nil
		}

		slog.Error("Unable to parse private key", "err", err)
		return nil, err
	}

	return signer, nil
}

func getSignersForIdentityAgent(hostname string) ([]ssh.Signer, error) {
	sshAuthSock := ssh_config.Get(hostname, "IdentityAgent")

	if sshAuthSock == "" {
		sshAuthSock = os.Getenv("SSH_AUTH_SOCK")
	}

	if sshAuthSock == "" {
		return nil, nil
	}

	sshAuthSock = cleanupSshConfigValue(sshAuthSock)

	conn, err := net.Dial("unix", sshAuthSock)
	if err != nil {
		slog.Error("Failed to open SSH auth socket", "err", err)
		return nil, err
	}

	slog.Info("Using ssh agent", "socket", sshAuthSock)
	agentClient := agent.NewClient(conn)
	agentSigners, err := agentClient.Signers()
	if err != nil {
		slog.Error("Error getting signers from agent", "err", err)
		return nil, err
	}

	return agentSigners, nil
}

func cleanupSshConfigValue(value string) string {
	replaced := strings.Trim(value, "\"")

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		slog.Warn("Error getting user home dir", "err", err)
		return replaced
	}

	replaced = strings.ReplaceAll(replaced, "$HOME", userHomeDir)
	if strings.HasPrefix(replaced, "~/") {
		replaced = strings.Replace(replaced, "~", userHomeDir, 1)
	}

	return replaced
}

// TODO Really needs "GivenHostName", "ResolvedHostName", etc
type SshEndpoint struct {
	GivenUser     string
	SshConfigUser string
	FallbackUser  string

	GivenHost     string
	SshConfigHost string

	GivenPort     string
	SshConfigPort string
}

func (e *SshEndpoint) String() string {
	return fmt.Sprintf("%s@%s:%s", e.FinalUser(), e.GivenHost, e.FinalPort())
}

func (e *SshEndpoint) FinalUser() string {
	if e.GivenUser != "" {
		return e.GivenUser
	}

	if e.SshConfigUser != "" {
		return e.SshConfigUser
	}

	return e.FallbackUser
}

func (e *SshEndpoint) FinalHost() string {
	if e.SshConfigHost != "" {
		return e.SshConfigHost
	}

	return e.GivenHost
}

func (e *SshEndpoint) FinalPort() string {
	if e.GivenPort != "" {
		return e.GivenPort
	}

	if e.SshConfigPort != "" {
		return e.SshConfigPort
	}

	return "22"
}

func ParseSshEndpoint(server string) (*SshEndpoint, error) {
	currentUser, err := user.Current()

	if err != nil {
		slog.Error("Error getting current user", "err", err)
		return nil, err
	}

	parsed, err := url.Parse(fmt.Sprintf("ssh://%s", server))
	if err != nil {
		return nil, err
	}

	return &SshEndpoint{
		GivenUser:     parsed.User.Username(),
		SshConfigUser: ssh_config.Get(parsed.Hostname(), "User"),
		FallbackUser:  currentUser.Username,

		GivenHost:     parsed.Hostname(),
		SshConfigHost: ssh_config.Get(parsed.Hostname(), "HostName"),

		GivenPort:     parsed.Port(),
		SshConfigPort: ssh_config.Get(parsed.Hostname(), "Port"),
	}, nil
}

func askForPassword(message string) ([]byte, error) {
	fmt.Print(message)
	password, err := term.ReadPassword(0)
	fmt.Println()

	if err != nil {
		return nil, err
	}

	return password, nil
}
