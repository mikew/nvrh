package ssh_helpers

import (
	"fmt"
	"log/slog"
	"nvrh/src/context"
	"os/exec"
	"strings"

	"github.com/neovim/go-client/nvim"
)

func BuildRemoteNvimCmd(nvrhContext *context.NvrhContext) *exec.Cmd {
	nvimCommandString := buildRemoteCommandString(nvrhContext)
	slog.Info("Starting remote nvim", "nvimCommandString", nvimCommandString)

	tunnel := fmt.Sprintf("%s:%s", nvrhContext.LocalSocketPath, nvrhContext.RemoteSocketPath)
	if nvrhContext.ShouldUsePorts {
		tunnel = fmt.Sprintf("%d:0.0.0.0:%d", nvrhContext.PortNumber, nvrhContext.PortNumber)
	}

	sshCommand := exec.Command(
		nvrhContext.SshPath,
		"-L",
		tunnel,
		"-t",
		nvrhContext.Server,
		// TODO Not really sure if this is better than piping it as exampled
		// below.
		fmt.Sprintf("$SHELL -i -c '%s'", nvimCommandString),
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

	return sshCommand
}

func buildRemoteCommandString(nvrhContext *context.NvrhContext) string {
	envPairsString := ""
	if len(nvrhContext.RemoteEnv) > 0 {
		envPairsString = strings.Join(nvrhContext.RemoteEnv, " ")
	}

	return fmt.Sprintf(
		"%s nvim --headless --listen \"%s\" --cmd \"cd %s\"",
		envPairsString,
		nvrhContext.RemoteSocketOrPort(),
		nvrhContext.RemoteDirectory,
	)
}

func MakeRpcTunnelHandler(nvrhContext *context.NvrhContext) func(*nvim.Nvim, []string) {
	return func(v *nvim.Nvim, args []string) {
		go func() {
			slog.Info("Tunneling", "server", nvrhContext.Server, "port", args[0])

			sshCommand := exec.Command(
				nvrhContext.SshPath,
				"-NL",
				fmt.Sprintf("%s:0.0.0.0:%s", args[0], args[0]),
				nvrhContext.Server,
			)

			// sshCommand.Stdout = os.Stdout
			// sshCommand.Stderr = os.Stderr
			// sshCommand.Stdin = os.Stdin
			nvrhContext.CommandsToKill = append(nvrhContext.CommandsToKill, sshCommand)

			if err := sshCommand.Start(); err != nil {
				slog.Error("Error starting tunnel", "err", err)
				return
			}

			defer sshCommand.Process.Kill()

			if err := sshCommand.Wait(); err != nil {
				slog.Error("Error running tunnel", "err", err)
			}
		}()
	}
}
