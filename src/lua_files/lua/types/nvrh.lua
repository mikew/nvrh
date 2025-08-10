---@meta

---@class Nvrh
---@field client_channels number[] A list of channels used by the client.
---@field mapped_ports { [string]: boolean } A map of ports that have been mapped.
---@field tunnel_port fun(port: number|string)
---@field open_url fun(url: string)
---@field _tunnel_port_with_channel fun(channel_id: string|number, port: number|string)

--- These are bridged from the Go code
---@type "primary"|"secondary"
nvrh_mode = ""
session_id = ""
channel_id = -1
socket_path = ""
browser_script_path = ""
should_map_ports = false

---@type Nvrh
-- nvrh = {}
