# nvrh

https://github.com/user-attachments/assets/aad16d20-cc78-44cd-8e9f-8412c87087eb

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
   --server-env value [ --server-env value ]                          Environment variables to set on the remote server
   --local-editor {{SOCKET_PATH}} [ --local-editor {{SOCKET_PATH}} ]  Local editor to use. {{SOCKET_PATH}} will be replaced with the socket path
   --help, -h                                                         show help
```

By default it runs `nvim`, but you can run something else with

```sh
nvrh client open \
  --local-editor nvim-qt \
  --local-editor --nofork \
  --local-editor --server \
  --local-editor {{SOCKET_PATH}}
```

### `:NvrhTunnelPort`

https://github.com/user-attachments/assets/4a8c302e-4e49-4f74-81a3-ac86ba33016a

nvrh adds a `:NvrhTunnelPort` command to Neovim to tunnel a port between your
local and remote machines.

```vim
:NvrhTunnelPort 8080
:NvrhTunnelPort 4000
```

### `:NvrhOpenUrl`

https://github.com/user-attachments/assets/04f9eea3-58a6-4bff-a155-8134ecdeaf2b

nvrh adds a `:NvrhOpenUrl` command to Neovim to open a URL on your local machine.

```vim
:NvrhOpenUrl https://example.com
```

In addition to this command, it also sets the `BROWSER` environment variable,
so commands can open a browser on your local machine.
