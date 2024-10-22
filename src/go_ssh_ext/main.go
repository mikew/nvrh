package go_ssh_ext

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"

	"github.com/kevinburke/ssh_config"
	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"

	"nvrh/src/ssh_endpoint"
)

func GetSshClientForEndpoint(endpoint *ssh_endpoint.SshEndpoint) (*ssh.Client, error) {
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
		password, err := askForPassword(fmt.Sprintf("%s's password: ", endpoint))
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

	identityFile = CleanupSshConfigValue(identityFile)

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
			passPhrase, _ := askForPassword(fmt.Sprintf("Enter passphrase for key '%s': ", identityFile))

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

	sshAuthSock = CleanupSshConfigValue(sshAuthSock)

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

func askForPassword(message string) ([]byte, error) {
	fmt.Print(message)
	password, err := term.ReadPassword(0)
	fmt.Println()

	if err != nil {
		return nil, err
	}

	return password, nil
}
