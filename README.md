# nvrh

https://github.com/user-attachments/assets/3cc9be58-ddce-4eaa-97bc-fd6ee8b0b942

nvrh (Neovim Remote Helper) aims to provide a simple way of working with a
remote Neovim instance, like you would with VSCode Remote.

## Installation

Download the `nvrh` binary for your platform / architecture from [the latest
release](https://github.com/mikew/nvrh/releases/latest).

Rename the downloaded file to `nvrh` and put it somewhere on your `PATH` for
convenience.

## Features

- Start Neovim on a remote machine.
- Tunnel the connection between your local and remote machines.
- Start your editor locally, talking to your remote Neovim instance.
- Provide an easy way to tunnel ports.
- Provide an easy way to open URLs on your local machine.

## Usage

### `nvrh client open`

This will open a new Neovim instance on your remote machine and tunnel the
socket to your local machine.

```
NAME:
   nvrh client open - Open a remote nvim instance in a local editor

USAGE:
   nvrh client open [options] <server> [remote-directory]

CATEGORY:
   client

OPTIONS:
   --ssh-path string                                Path to SSH binary. 'binary' will use the default system SSH binary. 'internal' will use the internal SSH client. Anything else will be used as the path to the SSH binary [$NVRH_CLIENT_SSH_PATH] (default: "binary")
   --use-ports                                      Use ports instead of sockets. Defaults to true on Windows [$NVRH_CLIENT_USE_PORTS] (default: false)
   --debug                                          (default: false) [$NVRH_CLIENT_DEBUG]
   --server-env string [ --server-env string ]      Environment variables to set on the remote server [$NVRH_CLIENT_SERVER_ENV]
   --local-editor string [ --local-editor string ]  Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path [$NVRH_CLIENT_LOCAL_EDITOR] (default: "nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui")
   --nvim-cmd nvim [ --nvim-cmd nvim ]              Command to run nvim with. Defaults to nvim [$NVRH_CLIENT_NVIM_CMD] (default: "nvim")
   --ssh-arg string [ --ssh-arg string ]            Additional arguments to pass to the SSH command [$NVRH_CLIENT_SSH_ARG]
   --enable-automap-ports                           Enable automatic port mapping (default: true) [$NVRH_CLIENT_AUTOMAP_PORTS]
   --insecure-direct-connect string                 Opens a public port on the server and connects directly to it. Use 'true' to connect to the server you're already passing
   --help, -h                                       show help
```

### `nvrh client reconnect`

Reconnect to an existing nvrh session.

```
NAME:
   nvrh client reconnect - Reconnect to an existing remote nvim instance

USAGE:
   nvrh client reconnect [options] <server> <session-id>

CATEGORY:
   client

OPTIONS:
   --ssh-path string                                Path to SSH binary. 'binary' will use the default system SSH binary. 'internal' will use the internal SSH client. Anything else will be used as the path to the SSH binary [$NVRH_CLIENT_SSH_PATH] (default: "binary")
   --use-ports                                      Use ports instead of sockets. Defaults to true on Windows [$NVRH_CLIENT_USE_PORTS] (default: false)
   --debug                                          (default: false) [$NVRH_CLIENT_DEBUG]
   --local-editor string [ --local-editor string ]  Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path [$NVRH_CLIENT_LOCAL_EDITOR] (default: "nvim", "--server", "{{SOCKET_PATH}}", "--remote-ui")
   --ssh-arg string [ --ssh-arg string ]            Additional arguments to pass to the SSH command [$NVRH_CLIENT_SSH_ARG]
   --insecure-direct-connect string                 Opens a public port on the server and connects directly to it. Use 'true' to connect to the server you're already passing
   --help, -h                                       show help
```

### Launch a different editor

By default nvrh runs `nvim`, but you can run something else with
`--local-editor`. This example runs `nvim-qt`:

```sh
nvrh client open \
  --local-editor nvim-qt,--nofork,--server,{{SOCKET_PATH}}
```

### Tunneling Ports

https://github.com/user-attachments/assets/6de3dfdc-d9bc-4668-be66-cbcf2071fa82

nvrh can tunnel ports between your local and remote machine. This happens
either automatically by scanning the output of terminal buffers, or manually
with the `:NvrhTunnelPort` command.

```vim
:NvrhTunnelPort 8080
:NvrhTunnelPort 4000
```

### Opening URLs

https://github.com/user-attachments/assets/7a0f8418-828d-4a5f-86cb-026d5d6fd182

nvrh can open URLs on your local machine from your remote Neovim instance.
There's a few ways to trigger this:

- `vim.ui.open`, so `gx` and `:Open https://example.com` will work.
- The `BROWSER` environment variable for process started from Neovim.
- The `:NvrhOpenUrl` command.

### Editing Files

nvrh sets the `$EDITOR` environment variable on the remote machine, so
something like `git commit` will open the file in your current remote session.
This also works for UIs that have an "Open in Editor" feature.

### Windows Support

https://github.com/user-attachments/assets/e3e542db-4858-40c6-bb90-a6f3fc642087

nvrh supports Windows both locally and remote. If running on Windows, or
if nvrh detects the remote machine is Windows `--use-ports` will default to
true.
