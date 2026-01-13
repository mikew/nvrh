package nvrh_config

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type DirectConnectValue struct {
	Enabled bool
	Address string
}

func (dcv *DirectConnectValue) UnmarshalYAML(value *yaml.Node) error {
	var b bool
	if err := value.Decode(&b); err == nil {
		dcv.Enabled = b
		dcv.Address = ""

		return nil
	}

	var s string
	if err := value.Decode(&s); err == nil {
		dcv.Enabled = true
		dcv.Address = s

		return nil
	}

	return fmt.Errorf("direct-connect must be a boolean or a string")
}

type NvrhConfigServer struct {
	NvimCmd       []string           `yaml:"nvim-cmd,omitempty"`
	UsePorts      *bool              `yaml:"use-ports,omitempty"`
	SshArg        []string           `yaml:"ssh-arg,omitempty"`
	SshPath       string             `yaml:"ssh-path,omitempty"`
	LocalEditor   []string           `yaml:"local-editor,omitempty"`
	ServerEnv     []string           `yaml:"server-env,omitempty"`
	DirectConnect DirectConnectValue `yaml:"direct-connect,omitempty"`
}

type NvrhConfig struct {
	Default NvrhConfigServer            `yaml:"default,omitempty"`
	Servers map[string]NvrhConfigServer `yaml:"servers"`
}

func DefaultConfigPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nvrh", "config.yml")
}

func LoadConfig(path string) (*NvrhConfig, error) {
	bytes, err := os.ReadFile(path)

	if err != nil {
		if os.IsNotExist(err) {
			return &NvrhConfig{Servers: map[string]NvrhConfigServer{}}, nil
		}

		return nil, err
	}

	var cconfig NvrhConfig
	if err := yaml.Unmarshal(bytes, &cconfig); err != nil {
		return nil, err
	}

	if cconfig.Servers == nil {
		cconfig.Servers = map[string]NvrhConfigServer{}
	}

	return &cconfig, nil
}

var envIndex = map[string][]string{
	"nvim-cmd":     {"NVRH_CLIENT_NVIM_CMD"},
	"use-ports":    {"NVRH_CLIENT_USE_PORTS"},
	"ssh-arg":      {"NVRH_CLIENT_SSH_ARG"},
	"ssh-path":     {"NVRH_CLIENT_SSH_PATH"},
	"local-editor": {"NVRH_CLIENT_LOCAL_EDITOR"},
	"server-env":   {"NVRH_CLIENT_SERVER_ENV"},
}

type shouldSetFunc func(name string) bool

func ApplyPrecedence(
	c *cli.Command,
	defaultServerConfig NvrhConfigServer,
	serverConfig NvrhConfigServer,
) error {
	flagNames := getFlagNames(c)
	shouldSet := func(name string) bool {
		return slices.Contains(flagNames, name) && !c.IsSet(name)
	}

	// First apply the specific server config, then the default config.
	err := applyServerConfig(c, serverConfig, shouldSet)
	if err != nil {
		return err
	}

	err = applyServerConfig(c, defaultServerConfig, shouldSet)
	if err != nil {
		return err
	}

	// Fall back to environment variables if still not set.
	for name, keys := range envIndex {
		if c.IsSet(name) {
			continue
		}

		if raw, ok := lookupFirst(keys); ok {
			if err := c.Set(name, raw); err != nil {
				return err
			}
		}
	}

	return nil
}

func applyServerConfig(c *cli.Command, serverConfig NvrhConfigServer, shouldSet shouldSetFunc) error {
	if shouldSet("ssh-path") && serverConfig.SshPath != "" {
		if err := c.Set("ssh-path", serverConfig.SshPath); err != nil {
			return err
		}
	}

	if shouldSet("local-editor") && len(serverConfig.LocalEditor) > 0 {
		for _, v := range serverConfig.LocalEditor {
			if err := c.Set("local-editor", v); err != nil {
				return err
			}
		}
	}

	if shouldSet("server-env") && len(serverConfig.ServerEnv) > 0 {
		for _, v := range serverConfig.ServerEnv {
			if err := c.Set("server-env", v); err != nil {
				return err
			}
		}
	}

	if shouldSet("nvim-cmd") && len(serverConfig.NvimCmd) > 0 {
		for _, v := range serverConfig.NvimCmd {
			if err := c.Set("nvim-cmd", v); err != nil {
				return err
			}
		}
	}

	if shouldSet("use-ports") && serverConfig.UsePorts != nil {
		if err := c.Set("use-ports", fmt.Sprintf("%v", *serverConfig.UsePorts)); err != nil {
			return err
		}
	}

	if shouldSet("ssh-arg") && len(serverConfig.SshArg) > 0 {
		for _, v := range serverConfig.SshArg {
			if err := c.Set("ssh-arg", v); err != nil {
				return err
			}
		}
	}

	if shouldSet("direct-connect") && serverConfig.DirectConnect.Enabled {
		val := serverConfig.DirectConnect.Address

		if val == "" {
			val = "true"
		}

		if err := c.Set("direct-connect", val); err != nil {
			return err
		}
	}

	return nil
}

func getFlagNames(c *cli.Command) []string {
	flagNames := []string{}

	for _, f := range c.Flags {
		flagNames = append(flagNames, f.Names()...)
	}

	return flagNames
}

func lookupFirst(keys []string) (string, bool) {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok && strings.TrimSpace(v) != "" {
			return v, true
		}
	}

	return "", false
}
