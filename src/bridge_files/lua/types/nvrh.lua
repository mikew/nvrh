---@meta

---@class NvrhServerInfo
---@field os string The operating system, e.g., "linux", "macos", "windows"
---@field arch string The architecture, e.g., "amd64", "arm64", "386"
---@field username string The username of the user
---@field homedir string The home directory of the user
---@field tmpdir string The temporary directory path
---@field shell_name string The shell name, e.g., "bash", "zsh", "cmd", "powershell"

--- These are bridged from the Go code
session_id = ''
channel_id = -1
socket_path = ''
browser_script_path = ''
should_map_ports = false
nvrh_server_info = ''
windows_launcher_path = ''

should_initialize = false
