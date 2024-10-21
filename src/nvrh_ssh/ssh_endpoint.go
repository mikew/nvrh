package nvrh_ssh

import (
	"fmt"
	"log/slog"
	"net/url"
	"os/user"

	"github.com/kevinburke/ssh_config"
)

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
