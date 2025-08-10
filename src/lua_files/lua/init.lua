_G._nvrh = {
  ---@type { [string]: boolean }
  mapped_ports = {},
}

---@class NvrhChannelClient
---@field name string
---@field attributes table<string, string>
---@field methods table<string, { NArgs: integer[], async: boolean }>

---@class NvrhChannel
---@field id integer
---@field client NvrhChannelClient

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
