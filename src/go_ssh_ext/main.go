package go_ssh_ext

import (
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/kevinburke/ssh_config"
	"github.com/skeema/knownhosts"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/term"

	"nvrh/src/ssh_endpoint"
)

func GetSshClientForEndpoint(endpoint *ssh_endpoint.SshEndpoint) (*ssh.Client, error) {
	knownhostsPath := filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts")

	if _, err := os.Stat(knownhostsPath); os.IsNotExist(err) {
		f, ferr := os.Create(knownhostsPath)
		os.OpenFile(knownhostsPath, os.O_CREATE|os.O_RDONLY, 0600)
		if ferr != nil {
			return nil, ferr
		}

		f.Close()
	}

	kh, err := knownhosts.NewDB(knownhostsPath)

	if err != nil {
		return nil, err
	}

	hostKeyCallback := ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		slog.Debug("Checking host key", "hostname", hostname, "remote", remote, "key", key)
		err := kh.HostKeyCallback()(hostname, remote, key)

		if knownhosts.IsHostKeyChanged(err) {
			return fmt.Errorf("REMOTE HOST IDENTIFICATION HAS CHANGED for host %s! This may indicate a MitM attack.", hostname)
		}

		if knownhosts.IsHostUnknown(err) {
			fmt.Print(fmt.Sprintf(`The authenticity of host '%s (%s)' can't be established.
%s key fingerprint is %s.
Are you sure you want to continue connecting (yes/no)? `,
				hostname, remote, key.Type(), ssh.FingerprintSHA256(key),
			))
			var response string
			fmt.Scanf("%v", &response)

			if response != "yes" {
				return fmt.Errorf("Host key verification failed.")
			}

			f, ferr := os.OpenFile(knownhostsPath, os.O_APPEND|os.O_WRONLY, 0600)
			if ferr == nil {
				defer f.Close()
				ferr = knownhosts.WriteKnownHost(f, hostname, remote, key)
			}
			if ferr == nil {
				slog.Info("Added host to known_hosts\n", "hostname", hostname)
			} else {
				slog.Error("Failed to add host to known_hosts\n", "hostname", hostname, "ferr", ferr)
				return ferr
			}

			// permit previously-unknown hosts (warning: may be insecure)
			return nil
		}

		return err
	})

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
		HostKeyCallback:   hostKeyCallback,
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

	if runtime.GOOS == "windows" && sshAuthSock == "" {
		sshAuthSock = `\\.\pipe\openssh-ssh-agent`
	}

	if sshAuthSock == "" {
		return nil, nil
	}

	sshAuthSock = CleanupSshConfigValue(sshAuthSock)

	conn, err := getConnectionForAgent(sshAuthSock)
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
	password, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()

	if err != nil {
		return nil, err
	}

	return password, nil
}
