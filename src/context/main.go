package context

import (
	"fmt"
)

type NvrhContext struct {
	SessionId       string
	Server          string
	RemoteDirectory string

	LocalSocketPath  string
	RemoteSocketPath string
	ShouldUsePorts   bool
	PortNumber       int

	RemoteEnv   []string
	LocalEditor []string

	BrowserScriptPath string
}

func (nc NvrhContext) LocalSocketOrPort() string {
	if nc.ShouldUsePorts {
		// nvim-qt, at least on Windows (and might have something to do with
		// running in a VM) seems to prefer `127.0.0.1` to `0.0.0.0`, and I think
		// that's safe on other OSes.
		return fmt.Sprintf("127.0.0.1:%d", nc.PortNumber)
	}

	return nc.LocalSocketPath
}

func (nc NvrhContext) RemoteSocketOrPort() string {
	if nc.ShouldUsePorts {
		return fmt.Sprintf("0.0.0.0:%d", nc.PortNumber)
	}

	return nc.RemoteSocketPath
}
