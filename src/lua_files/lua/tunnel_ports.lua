if nvrh_mode == "primary" then
  ---@param port string|integer
  function _G._nvrh.tunnel_port(port)
    for _, channel_id in ipairs(_G._nvrh.client_channels) do
      _G._nvrh._tunnel_port_with_channel(channel_id, port)
    end

    if not _G._nvrh.mapped_ports[port] then
      _G._nvrh.mapped_ports[port] = true
    end
  end

  ---@param channel_id integer
  ---@param port string|integer
  function _G._nvrh._tunnel_port_with_channel(channel_id, port)
    pcall(vim.rpcnotify, tonumber(channel_id), 'tunnel-port', { tostring(port) })
  end

  vim.api.nvim_create_user_command(
    'NvrhTunnelPort',
    function(args)
      _G._nvrh.tunnel_port(args.args)
    end,
    {
      nargs = 1,
      force = true,
    }
  )
end
