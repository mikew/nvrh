---@class NvrhChannelClient
---@field name string
---@field attributes table<string, string>
---@field methods table<string, { NArgs: integer[], async: boolean }>

---@class NvrhChannel
---@field id integer
---@field client NvrhChannelClient

if should_initialize then
  _G._nvrh = {
    ---@type { [string]: boolean }
    mapped_ports = {},
  }

  function _G._nvrh.get_nvrh_channels()
    ---@type NvrhChannel[]
    local channels = {}

    for _, channel in ipairs(vim.api.nvim_list_chans()) do
      if channel.client ~= nil and channel.client.name == 'nvrh' then
        table.insert(channels, channel)
      end
    end

    return channels
  end

  vim.env.NVRH_SESSION_ID = session_id

  vim.api.nvim_create_autocmd('VimLeavePre', {
    callback = function()
      local browser_script_path =
        string.format('/tmp/nvrh-browser-%s', session_id)
      os.remove(browser_script_path)

      local socket_file = string.format('/tmp/nvrh-session-%s', session_id)
      os.remove(socket_file)
    end,
  })
end
