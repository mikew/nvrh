---@class NvrhChannelClient
---@field name? string
---@field attributes? table<string, string>
---@field methods? table<string, { NArgs: [integer, integer], async: boolean }>

---@class NvrhChannel
---@field id integer
---@field client NvrhChannelClient?

if _G._nvrh_is_initialized ~= true then
  local nvrh_server_info, session_id, browser_script_path, editor_script_path, socket_path, windows_launcher_path =
    ...

  _G._nvrh = {
    ---@type { [string]: boolean }
    mapped_ports = {},

    ---@type NvrhServerInfo
    server_info = vim.json.decode(nvrh_server_info),
  }

  function _G._nvrh.get_nvrh_channels()
    ---@type NvrhChannel[]
    local nvrh_channels = {}

    ---@type NvrhChannel[]
    local nvim_channels = vim.api.nvim_list_chans()
    for _, channel in ipairs(nvim_channels) do
      if channel.client ~= nil and channel.client.name == 'nvrh' then
        table.insert(nvrh_channels, channel)
      end
    end

    return nvrh_channels
  end

  vim.env.NVRH_SESSION_ID = session_id

  local function cleanup()
    os.remove(browser_script_path)
    os.remove(editor_script_path)
    os.remove(socket_path)

    if
      _G._nvrh.server_info.os == 'windows'
      and windows_launcher_path
      and windows_launcher_path ~= ''
    then
      os.remove(windows_launcher_path)
    end
  end

  -- Cleanup when exiting Neovim.
  vim.api.nvim_create_autocmd('VimLeavePre', {
    callback = function()
      cleanup()
    end,
  })

  -- Exit if last client disconnects.
  vim.api.nvim_create_autocmd('UILeave', {
    callback = function(args)
      if #vim.api.nvim_list_uis() == 0 then
        -- TODO No idea why cleanup is needed here, it should be handled by
        -- VimLeavePre, and seems to work fine when `qall` is used in nvrh.
        cleanup()
        vim.cmd('qall')
      end
    end,
  })
end
