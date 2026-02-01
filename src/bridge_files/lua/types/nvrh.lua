---@meta

---@class NvrhServerInfo
---@field os string The operating system, e.g., "linux", "macos", "windows"
---@field arch string The architecture, e.g., "amd64", "arm64", "386"
---@field username string The username of the user
---@field homedir string The home directory of the user
---@field tmpdir string The temporary directory path
---@field shell_name string The shell name, e.g., "bash", "zsh", "cmd", "powershell"

---@type boolean|nil
_G._nvrh_is_initialized = nil

---@type fun(filename: string, line?: number, col?: number): nil
_G.nvrh_open_file_handler = nil

should_initialize = false
