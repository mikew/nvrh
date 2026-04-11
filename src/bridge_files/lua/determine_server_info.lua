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
  local shell = (vim.env.SHELL or passwd.shell or ''):lower()
  if shell:match('bash') then
    return 'bash'
  elseif shell:match('zsh') then
    return 'zsh'
  end

  if vim.env.PSMODULEPATH then
    return 'powershell'
  end

  if vim.env.COMSPEC and vim.env.PROMPT then
    return 'cmd'
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
