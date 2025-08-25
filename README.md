# nvrh

https://github.com/user-attachments/assets/a7ab3f59-f931-4a47-ac28-2339d0cd25e9

nvrh (Neovim Remote Helper) aims to provide a simple way of working with a
remote Neovim instance, like you would with VSCode Remote.

## Installation

Download the `nvrh` binary for your platform / architecture from [the latest
release](https://github.com/mikew/nvrh/releases/latest).

Rename it to `nvrh` and put it somewhere on your `PATH` for convenience.

## Features

- Start Neovim on a remote machine.
- Tunnel the connection between your local and remote machines.
- Start your editor locally, talking to your remote Neovim instance.
- Provide an easy way to tunnel ports.
- Provide an easy way to open URLs on your local machine.

## Usage

### `nvrh client open`

This will open a new Neovim instance on your remote machine and connect to it
from your local machine.

```
NAME:
   nvrh client open - Open a remote nvim instance in a local editor

USAGE:
   nvrh client open [command options] <server> [remote-directory]

CATEGORY:
   client

OPTIONS:
   --ssh-path value                               Path to SSH binary. 'binary' will use the default system SSH binary. 'internal' will use the internal SSH client. Anything else will be used as the path to the SSH binary (default: "binary") [$NVRH_CLIENT_SSH_PATH]
   --use-ports                                    Use ports instead of sockets. Defaults to true on Windows (default: false) [$NVRH_CLIENT_USE_PORTS]
   --debug                                        (default: false) [$NVRH_CLIENT_DEBUG]
   --server-env value [ --server-env value ]      Environment variables to set on the remote server [$NVRH_CLIENT_SERVER_ENV]
   --local-editor value [ --local-editor value ]  Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path (default: "nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui") [$NVRH_CLIENT_LOCAL_EDITOR]
   --nvim-cmd nvim [ --nvim-cmd nvim ]            Command to run nvim with. Defaults to nvim (default: "nvim") [$NVRH_CLIENT_NVIM_CMD]
   --ssh-arg value [ --ssh-arg value ]            Additional arguments to pass to the SSH command [$NVRH_CLIENT_SSH_ARG]
   --enable-automap-ports                         Enable automatic port mapping (default: true) [$NVRH_CLIENT_AUTOMAP_PORTS]
   --help, -h                                     show help
```

### `nvrh client reconnect`

Reconnect to an existing nvrh session.

```
NAME:
   nvrh client reconnect - Reconnect to an existing remote nvim instance

USAGE:
   nvrh client reconnect [command options] <server> <session-id>

CATEGORY:
   client

OPTIONS:
   --ssh-path value                               Path to SSH binary. 'binary' will use the default system SSH binary. 'internal' will use the internal SSH client. Anything else will be used as the path to the SSH binary (default: "binary") [$NVRH_CLIENT_SSH_PATH]
   --use-ports                                    Use ports instead of sockets. Defaults to true on Windows (default: false) [$NVRH_CLIENT_USE_PORTS]
   --debug                                        (default: false) [$NVRH_CLIENT_DEBUG]
   --local-editor value [ --local-editor value ]  Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path (default: "nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui") [$NVRH_CLIENT_LOCAL_EDITOR]
   --ssh-arg value [ --ssh-arg value ]            Additional arguments to pass to the SSH command [$NVRH_CLIENT_SSH_ARG]
   --help, -h                                     show help
```

### Launch a different editor

By default it runs `nvim`, but you can run something else with
`--local-editor`. This example runs `nvim-qt`:

```sh
nvrh client open \
  --local-editor nvim-qt,--nofork,--server,--local-editor,{{SOCKET_PATH}}
```

### Tunneling Ports

https://github.com/user-attachments/assets/f84c9fb5-f757-489d-aee5-fcdfe44317b0

nvrh can tunnel ports between your local and remote machine. It does this
either automatically by scanning the output of terminal buffers, or manually
with the `:NvrhTunnelPort` command.

```vim
:NvrhTunnelPort 8080
:NvrhTunnelPort 4000
```

### Opening URLs

https://github.com/user-attachments/assets/d6205b5d-179e-46cd-83ec-b0de278c81f6

nvrh can open URLs on your local machine from your remote Neovim instance. There's a few ways to do this:

- It patches `vim.ui.open`, so `gx` and `:Open https://example.com` will work.
- It sets the `BROWSER` environment variable, so anything that runs in a Neovim terminal can open a URL.
- It creates an `:NvrhOpenUrl` command to open a URL.
