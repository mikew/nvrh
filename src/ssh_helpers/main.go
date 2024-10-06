package ssh_helpers

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

func StartRemoteNvim(server string, socketPath string, directory string, envPairs []string) {
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
