---@param port string|integer
function _G._nvrh.tunnel_port(port)
  for _, channel in ipairs(_G._nvrh.get_nvrh_channels()) do
    _G._nvrh._tunnel_port_with_channel(channel.id, port)
  end

  if not _G._nvrh.mapped_ports[port] then
    _G._nvrh.mapped_ports[port] = true
  end
end

---@param channel_id integer
---@param port string|integer
function _G._nvrh._tunnel_port_with_channel(channel_id, port)
  local channel = vim.api.nvim_get_chan_info(channel_id)
  if channel.client ~= nil and channel.client.methods['tunnel-port'] then
    pcall(
      vim.rpcnotify,
      tonumber(channel_id),
      'tunnel-port',
      { tostring(port) }
    )
  end
end

vim.api.nvim_create_user_command('NvrhTunnelPort', function(args)
  _G._nvrh.tunnel_port(args.args)
end, {
  nargs = 1,
  force = true,
})
