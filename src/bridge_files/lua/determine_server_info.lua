-- https://github.com/neovim/neovim/blob/c12701d4e1404a67fef6da01a8a9d7e2d48d78d6/runtime/lua/uv/_meta.lua#L4297-L4301
--- @class uv.os_uname.info
--- @field sysname string
--- @field release string
--- @field version string
--- @field machine string

-- https://github.com/neovim/neovim/blob/c12701d4e1404a67fef6da01a8a9d7e2d48d78d6/runtime/lua/uv/_meta.lua#L4367-L4372
--- @class uv.os_get_passwd.passwd
--- @field username string
--- @field uid integer?
--- @field gid integer?
--- @field shell string?
--- @field homedir string

---@type uv.os_uname.info
local uname = vim.uv.os_uname()
---@type uv.os_get_passwd.passwd
local passwd = vim.uv.os_get_passwd()

local function get_os()
  local os = uname.sysname:lower()

  if os == 'linux' then
    return 'linux'
  elseif os == 'darwin' then
    return 'macos'
  elseif os == 'windows_nt' then
    return 'windows'
  elseif os:match('^cygwin') or os:match('^mingw') or os:match('^msys') then
    return 'windows'
  else
    return 'unknown'
  end
end

local function get_arch()
  local arch = uname.machine:lower()

  if arch == 'x86_64' or arch == 'amd64' then
    return 'amd64'
  elseif arch == 'aarch64' or arch == 'arm64' then
    return 'arm64'
  elseif arch:match('^armv8') then
    return 'arm64'
  elseif arch:match('^armv7') or arch:match('^armv6') then
    return 'arm'
  elseif
    arch == 'i386'
    or arch == 'i486'
    or arch == 'i586'
    or arch == 'i686'
    or arch == 'i786'
    or arch == 'x86'
  then
    return '386'
  elseif arch == 'ppc64le' then
    return 'ppc64le'
  elseif arch == 's390x' then
    return 's390x'
  else
    return 'unknown'
  end
end

local function get_shell_name()
  if vim.env.COMSPEC and vim.env.PROMPT then
    return 'cmd'
  end

  if vim.env.PSMODULEPATH then
    return 'powershell'
  end

  local shell = (vim.env.SHELL or passwd.shell or ''):lower()
  if shell:match('bash') then
    return 'bash'
  elseif shell:match('zsh') then
    return 'zsh'
  end

  return 'unknown'
end

local server_info = {
  os = get_os(),
  arch = get_arch(),
  username = passwd.username,
  homedir = passwd.homedir,
  tmpdir = vim.uv.os_tmpdir(),
  shell_name = get_shell_name(),
}

return vim.json.encode(server_info)
