package nvrh_config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
	"gopkg.in/yaml.v3"
)

type NvrhConfigServer struct {
	NvimCmd  []string `yaml:"nvim-cmd"`
	UsePorts *bool    `yaml:"use-ports,omitempty"`
}

type NvrhConfig struct {
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
	"nvim-cmd":  {"NVRH_CLIENT_NVIM_CMD"},
	"use-ports": {"NVRH_CLIENT_USE_PORTS"},
}

func ApplyPrecedence(c *cli.Command, sc NvrhConfigServer) error {
	// Use values from YAML if not set in command.
	if !c.IsSet("nvim-cmd") && len(sc.NvimCmd) > 0 {
		for _, v := range sc.NvimCmd {
			if err := c.Set("nvim-cmd", v); err != nil {
				return err
			}
		}
	}

	if !c.IsSet("use-ports") && sc.UsePorts != nil {
		if err := c.Set("use-ports", fmt.Sprintf("%v", *sc.UsePorts)); err != nil {
			return err
		}
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

func lookupFirst(keys []string) (string, bool) {
	for _, k := range keys {
		if v, ok := os.LookupEnv(k); ok && strings.TrimSpace(v) != "" {
			return v, true
		}
	}

	return "", false
}
