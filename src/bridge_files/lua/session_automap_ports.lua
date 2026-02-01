local channel_id = ...

for port, _ in pairs(_G._nvrh.mapped_ports) do
  _G._nvrh._tunnel_port_with_channel(channel_id, port)
end
